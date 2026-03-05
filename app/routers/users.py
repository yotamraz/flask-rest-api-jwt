"""User management & authentication endpoints — mirrors Flask /user/ blueprint."""

from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.responses import JSONResponse
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from ..auth import (
    create_access_token,
    create_refresh_token,
    get_current_user,
    get_current_user_from_refresh,
    get_validated_access_token,
    hash_password,
    token_blocklist,
    verify_password,
)
from ..database import get_db
from ..models import User
from ..schemas import UserLoginRequest, UserRegisterRequest, UserResponse

router = APIRouter(prefix="/user", tags=["users"])


@router.post("/register", status_code=status.HTTP_201_CREATED)
async def register(
    data: UserRegisterRequest,
    db: AsyncSession = Depends(get_db),
):
    """Register a new user. Returns the created user (without password)."""
    result = await db.execute(
        select(User).where(User.username == data.username)
    )
    if result.scalar_one_or_none() is not None:
        return JSONResponse(
            status_code=status.HTTP_400_BAD_REQUEST,
            content={"message": "User exists"},
        )

    user = User(
        username=data.username,
        password_hash=hash_password(data.password),
    )
    db.add(user)
    await db.flush()  # populate user.id before serialising
    await db.refresh(user)
    return UserResponse.model_validate(user)


@router.post("/login")
async def login(
    data: UserLoginRequest,
    db: AsyncSession = Depends(get_db),
):
    """Authenticate and return access + refresh tokens."""
    result = await db.execute(
        select(User).where(User.username == data.username)
    )
    user = result.scalar_one_or_none()
    if user is None or not verify_password(data.password, user.password_hash):
        return JSONResponse(
            status_code=status.HTTP_401_UNAUTHORIZED,
            content={"message": "Invalid credentials"},
        )

    access_token = create_access_token(user.id)
    refresh_token = create_refresh_token(user.id)
    return {"access_token": access_token, "refresh_token": refresh_token}


@router.post("/logout")
async def logout(
    payload: dict = Depends(get_validated_access_token),
):
    """Revoke the current access token (add its JTI to the blocklist)."""
    jti = payload.get("jti")
    if jti:
        token_blocklist.add(jti)
    return {"message": "Logged out"}


@router.post("/refresh")
async def refresh(
    identity: dict = Depends(get_current_user_from_refresh),
):
    """Issue a new access token using a valid refresh token."""
    access_token = create_access_token(int(identity["user_id"]))
    return {"access_token": access_token}


@router.get("/{user_id}")
async def get_user(
    user_id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Get a user by ID. Users can only view their own profile."""
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found",
        )
    if current_user.id != user.id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Unauthorized",
        )
    return UserResponse.model_validate(user)


@router.delete("/{user_id}")
async def delete_user(
    user_id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Delete a user by ID. Users can only delete their own account."""
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found",
        )
    if current_user.id != user.id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Unauthorized",
        )
    await db.delete(user)
    return {"message": "Deleted"}
