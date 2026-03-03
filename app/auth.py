"""JWT authentication utilities and FastAPI dependencies.

Replaces Flask-JWT-Extended usage with python-jose for token management
and bcrypt for password hashing.
"""

from __future__ import annotations

import uuid
from datetime import datetime, timedelta, timezone
from typing import Optional

from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from jose import JWTError, jwt
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from .config import settings
from .database import get_db
from .models import User

# ---------------------------------------------------------------------------
# Security scheme (auto_error=False so we control the 401 response format)
# ---------------------------------------------------------------------------

bearer_scheme = HTTPBearer(auto_error=False)

# ---------------------------------------------------------------------------
# Token blocklist (in-memory set, matching Flask source behaviour)
# ---------------------------------------------------------------------------

token_blocklist: set[str] = set()

# ---------------------------------------------------------------------------
# Custom exception for missing/invalid auth (matches Flask-JWT-Extended format)
# ---------------------------------------------------------------------------


class MissingAuthError(Exception):
    """Raised when the Authorization header is missing."""

    def __init__(self, msg: str = "Missing Authorization Header"):
        self.msg = msg


# ---------------------------------------------------------------------------
# JWT helpers
# ---------------------------------------------------------------------------

ALGORITHM = "HS256"


def create_access_token(user_id: int) -> str:
    """Create a JWT access token for the given user ID."""
    now = datetime.now(timezone.utc)
    payload = {
        "sub": str(user_id),
        "jti": str(uuid.uuid4()),
        "type": "access",
        "exp": now + timedelta(minutes=settings.ACCESS_TOKEN_EXPIRES_MINUTES),
        "iat": now,
    }
    return jwt.encode(payload, settings.SECRET_KEY, algorithm=ALGORITHM)


def create_refresh_token(user_id: int) -> str:
    """Create a JWT refresh token for the given user ID."""
    now = datetime.now(timezone.utc)
    payload = {
        "sub": str(user_id),
        "jti": str(uuid.uuid4()),
        "type": "refresh",
        "exp": now + timedelta(days=settings.REFRESH_TOKEN_EXPIRES_DAYS),
        "iat": now,
    }
    return jwt.encode(payload, settings.SECRET_KEY, algorithm=ALGORITHM)


def decode_token(token: str) -> dict:
    """Decode and validate a JWT token. Raises HTTPException on failure."""
    try:
        payload = jwt.decode(token, settings.SECRET_KEY, algorithms=[ALGORITHM])
    except JWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired token",
        )

    # Check blocklist
    jti = payload.get("jti")
    if jti and jti in token_blocklist:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token has been revoked",
        )

    return payload


# ---------------------------------------------------------------------------
# Auth token dependency (checks bearer is present)
# ---------------------------------------------------------------------------


async def require_auth_token(
    credentials: Optional[HTTPAuthorizationCredentials] = Depends(bearer_scheme),
) -> str:
    """Dependency that ensures a Bearer token is present.

    Raises MissingAuthError (-> 401 with {"msg": "Missing Authorization Header"})
    if no token is provided, matching Flask-JWT-Extended behaviour.
    """
    if credentials is None:
        raise MissingAuthError()
    return credentials.credentials


# ---------------------------------------------------------------------------
# FastAPI dependencies
# ---------------------------------------------------------------------------


async def get_current_user(
    token: str = Depends(require_auth_token),
    db: AsyncSession = Depends(get_db),
) -> User:
    """Dependency that extracts and validates an **access** JWT, then loads the user."""
    payload = decode_token(token)

    if payload.get("type") != "access":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token type",
        )

    user_id = payload.get("sub")
    if user_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token payload",
        )

    result = await db.execute(select(User).where(User.id == int(user_id)))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User not found",
        )

    return user


async def get_current_user_from_refresh(
    token: str = Depends(require_auth_token),
    db: AsyncSession = Depends(get_db),
) -> tuple[User, dict]:
    """Dependency that extracts and validates a **refresh** JWT, then loads the user."""
    payload = decode_token(token)

    if payload.get("type") != "refresh":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token type",
        )

    user_id = payload.get("sub")
    if user_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token payload",
        )

    result = await db.execute(select(User).where(User.id == int(user_id)))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User not found",
        )

    return user, payload
