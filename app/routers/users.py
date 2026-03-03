"""Users router — registration, login, logout, refresh, get, and delete.

Replaces app/resources/users.py Flask Blueprint with a FastAPI APIRouter.
"""

from fastapi import APIRouter, Depends, Request
from fastapi.responses import JSONResponse
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from ..auth import (
    create_access_token,
    create_refresh_token,
    decode_token,
    get_current_user,
    get_current_user_from_refresh,
    require_auth_token,
    token_blocklist,
)
from ..database import get_db
from ..models import User

router = APIRouter(prefix="/user", tags=["users"])


@router.post("/register", status_code=201)
async def register(request: Request, db: AsyncSession = Depends(get_db)):
    """Register a new user. Returns 400 if username already exists.

    Accepts raw JSON body (not Pydantic-validated) to match Flask behaviour
    where a missing 'password' field causes an unhandled KeyError -> 500.
    """
    data = await request.json()

    result = await db.execute(select(User).where(User.username == data["username"]))
    if result.scalar_one_or_none() is not None:
        return JSONResponse(
            status_code=400,
            content={"message": "User exists"},
        )

    user = User(username=data["username"])
    user.set_password(data["password"])  # KeyError -> 500 if password missing (matches Flask)
    db.add(user)
    await db.flush()
    await db.refresh(user)
    return JSONResponse(
        status_code=201,
        content={"id": user.id, "username": user.username},
    )


@router.post("/login")
async def login(request: Request, db: AsyncSession = Depends(get_db)):
    """Authenticate user and return access + refresh tokens."""
    data = await request.json()

    result = await db.execute(select(User).where(User.username == data.get("username")))
    user = result.scalar_one_or_none()

    if user is None or not user.check_password(data.get("password", "")):
        return JSONResponse(
            status_code=401,
            content={"message": "Invalid credentials"},
        )

    access_token = create_access_token(user.id)
    refresh_token = create_refresh_token(user.id)
    return JSONResponse(
        status_code=200,
        content={"access_token": access_token, "refresh_token": refresh_token},
    )


@router.post("/logout")
async def logout(
    token: str = Depends(require_auth_token),
):
    """Revoke the current access token by adding its jti to the blocklist."""
    payload = decode_token(token)
    jti = payload.get("jti")
    if jti:
        token_blocklist.add(jti)
    return JSONResponse(status_code=200, content={"message": "Logged out"})


@router.post("/refresh")
async def refresh(
    user_and_payload: tuple = Depends(get_current_user_from_refresh),
):
    """Issue a new access token using a valid refresh token."""
    user, _payload = user_and_payload
    access_token = create_access_token(user.id)
    return JSONResponse(status_code=200, content={"access_token": access_token})


@router.get("/{id}")
async def get_user(
    id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Get user info. First checks if user exists (404), then authorization (401).

    Matches Flask's get_or_404() then identity check pattern.
    """
    result = await db.execute(select(User).where(User.id == id))
    user = result.scalar_one_or_none()
    if user is None:
        return JSONResponse(status_code=404, content={"message": "User not found"})
    if current_user.id != id:
        return JSONResponse(status_code=401, content={"message": "Unauthorized"})
    return JSONResponse(
        status_code=200,
        content={"id": user.id, "username": user.username},
    )


@router.delete("/{id}")
async def delete_user(
    id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Delete user. First checks if user exists (404), then authorization (401).

    Matches Flask's get_or_404() then identity check pattern.
    """
    result = await db.execute(select(User).where(User.id == id))
    user = result.scalar_one_or_none()
    if user is None:
        return JSONResponse(status_code=404, content={"message": "User not found"})
    if current_user.id != id:
        return JSONResponse(status_code=401, content={"message": "Unauthorized"})
    await db.delete(user)
    return JSONResponse(status_code=200, content={"message": "Deleted"})
