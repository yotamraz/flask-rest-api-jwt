"""User management endpoints — mirrors the Flask ``/user`` blueprint."""
from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.responses import JSONResponse
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from ..auth import (
    create_access_token,
    create_refresh_token,
    decode_token,
    get_current_user,
    get_current_user_refresh,
    hash_password,
    revoke_token,
    verify_password,
    bearer_scheme,
)
from ..database import get_db
from ..models import User
from ..schemas import UserCreate, UserLogin, UserResponse

router = APIRouter(prefix="/user", tags=["users"])


@router.post("/register", status_code=201)
async def register(payload: UserCreate, db: AsyncSession = Depends(get_db)):
    """Register a new user."""
    result = await db.execute(select(User).where(User.username == payload.username))
    if result.scalars().first() is not None:
        return JSONResponse(
            status_code=400,
            content={"message": "User exists"},
        )

    user = User(
        username=payload.username,
        password_hash=hash_password(payload.password),
    )
    db.add(user)
    await db.commit()
    await db.refresh(user)
    return UserResponse.model_validate(user)


@router.post("/login")
async def login(payload: UserLogin, db: AsyncSession = Depends(get_db)):
    """Authenticate and return access + refresh tokens."""
    result = await db.execute(select(User).where(User.username == payload.username))
    user = result.scalars().first()

    if user is None or not verify_password(payload.password, user.password_hash):
        return JSONResponse(
            status_code=401,
            content={"message": "Invalid credentials"},
        )

    access_token = create_access_token(user.id)
    refresh_token = create_refresh_token(user.id)
    return {"access_token": access_token, "refresh_token": refresh_token}


@router.post("/logout")
async def logout(
    current_user: User = Depends(get_current_user),
    credentials=Depends(bearer_scheme),
):
    """Revoke the current access token."""
    payload = decode_token(credentials.credentials)
    revoke_token(payload["jti"])
    return {"message": "Logged out"}


@router.post("/refresh")
async def refresh(
    user_and_payload: tuple[User, dict] = Depends(get_current_user_refresh),
):
    """Issue a new access token using a valid refresh token."""
    user, _payload = user_and_payload
    access_token = create_access_token(user.id)
    return {"access_token": access_token}


@router.get("/{user_id}")
async def get_user(
    user_id: int,
    current_user: User = Depends(get_current_user),
):
    """Get a user by ID (only own profile)."""
    if current_user.id != user_id:
        return JSONResponse(
            status_code=401,
            content={"message": "Unauthorized"},
        )
    return UserResponse.model_validate(current_user)


@router.delete("/{user_id}")
async def delete_user(
    user_id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Delete a user by ID (only own account)."""
    if current_user.id != user_id:
        return JSONResponse(
            status_code=401,
            content={"message": "Unauthorized"},
        )
    await db.delete(current_user)
    await db.commit()
    return {"message": "Deleted"}
