"""Users router — registration, login, logout, refresh, get, and delete.

Replaces app/resources/users.py Flask Blueprint with a FastAPI APIRouter.
"""

from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from ..auth import (
    bearer_scheme,
    create_access_token,
    create_refresh_token,
    decode_token,
    get_current_user,
    get_current_user_from_refresh,
    token_blocklist,
)
from ..database import get_db
from ..models import User
from ..schemas import (
    AccessTokenResponse,
    LoginRequest,
    MessageResponse,
    TokenResponse,
    UserCreate,
    UserResponse,
)

router = APIRouter(prefix="/user", tags=["users"])


@router.post("/register", response_model=UserResponse, status_code=201)
async def register(data: UserCreate, db: AsyncSession = Depends(get_db)):
    """Register a new user. Returns 400 if username already exists."""
    result = await db.execute(select(User).where(User.username == data.username))
    if result.scalar_one_or_none() is not None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="User exists",
        )

    user = User(username=data.username)
    user.set_password(data.password)
    db.add(user)
    await db.flush()
    await db.refresh(user)
    return user


@router.post("/login", response_model=TokenResponse)
async def login(data: LoginRequest, db: AsyncSession = Depends(get_db)):
    """Authenticate user and return access + refresh tokens."""
    result = await db.execute(select(User).where(User.username == data.username))
    user = result.scalar_one_or_none()

    if user is None or not user.check_password(data.password):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid credentials",
        )

    access_token = create_access_token(user.id)
    refresh_token = create_refresh_token(user.id)
    return TokenResponse(access_token=access_token, refresh_token=refresh_token)


@router.post("/logout", response_model=MessageResponse)
async def logout(
    credentials: HTTPAuthorizationCredentials = Depends(bearer_scheme),
):
    """Revoke the current access token by adding its jti to the blocklist."""
    payload = decode_token(credentials.credentials)
    jti = payload.get("jti")
    if jti:
        token_blocklist.add(jti)
    return MessageResponse(message="Logged out")


@router.post("/refresh", response_model=AccessTokenResponse)
async def refresh(
    user_and_payload: tuple = Depends(get_current_user_from_refresh),
):
    """Issue a new access token using a valid refresh token."""
    user, _payload = user_and_payload
    access_token = create_access_token(user.id)
    return AccessTokenResponse(access_token=access_token)


@router.get("/{id}", response_model=UserResponse)
async def get_user(
    id: int,
    current_user: User = Depends(get_current_user),
):
    """Get user info. Only the user themselves can access their profile."""
    if current_user.id != id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Unauthorized",
        )
    return current_user


@router.delete("/{id}", response_model=MessageResponse)
async def delete_user(
    id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Delete user. Only the user themselves can delete their account."""
    if current_user.id != id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Unauthorized",
        )
    await db.delete(current_user)
    return MessageResponse(message="Deleted")
