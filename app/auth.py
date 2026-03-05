"""JWT authentication and password hashing utilities.

Replaces Flask-JWT-Extended with python-jose and passlib[bcrypt].
"""

from __future__ import annotations

import uuid
from datetime import datetime, timedelta, timezone

from fastapi import Depends, HTTPException, status
from fastapi.security import OAuth2PasswordBearer
from jose import JWTError, jwt
from passlib.context import CryptContext
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from .config import settings
from .database import get_db
from .models import User

# ---------------------------------------------------------------------------
# Password hashing
# ---------------------------------------------------------------------------

pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")


def hash_password(password: str) -> str:
    """Hash a plaintext password using bcrypt."""
    return pwd_context.hash(password)


def verify_password(plain_password: str, hashed_password: str) -> bool:
    """Verify a plaintext password against a bcrypt hash."""
    return pwd_context.verify(plain_password, hashed_password)


# ---------------------------------------------------------------------------
# In-memory token blocklist (mirrors Flask-JWT-Extended's blocklist)
# ---------------------------------------------------------------------------

token_blocklist: set[str] = set()


# ---------------------------------------------------------------------------
# JWT token creation
# ---------------------------------------------------------------------------

ALGORITHM = "HS256"


def create_access_token(user_id: int) -> str:
    """Create a short-lived access token."""
    now = datetime.now(timezone.utc)
    claims = {
        "sub": str(user_id),
        "jti": str(uuid.uuid4()),
        "token_type": "access",
        "iat": now,
        "exp": now + timedelta(minutes=settings.ACCESS_TOKEN_EXPIRES_MINUTES),
    }
    return jwt.encode(claims, settings.SECRET_KEY, algorithm=ALGORITHM)


def create_refresh_token(user_id: int) -> str:
    """Create a long-lived refresh token."""
    now = datetime.now(timezone.utc)
    claims = {
        "sub": str(user_id),
        "jti": str(uuid.uuid4()),
        "token_type": "refresh",
        "iat": now,
        "exp": now + timedelta(days=settings.REFRESH_TOKEN_EXPIRES_DAYS),
    }
    return jwt.encode(claims, settings.SECRET_KEY, algorithm=ALGORITHM)


def _decode_token(token: str) -> dict:
    """Decode and validate a JWT, returning its claims."""
    try:
        payload = jwt.decode(token, settings.SECRET_KEY, algorithms=[ALGORITHM])
    except JWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired token",
            headers={"WWW-Authenticate": "Bearer"},
        )
    # Check blocklist
    jti = payload.get("jti")
    if jti and jti in token_blocklist:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token has been revoked",
            headers={"WWW-Authenticate": "Bearer"},
        )
    return payload


# ---------------------------------------------------------------------------
# FastAPI dependencies
# ---------------------------------------------------------------------------

oauth2_scheme = OAuth2PasswordBearer(tokenUrl="/user/login", auto_error=True)


async def get_current_user(
    token: str = Depends(oauth2_scheme),
    db: AsyncSession = Depends(get_db),
) -> User:
    """Validate an **access** token and return the corresponding User.

    Replaces Flask-JWT-Extended's ``@jwt_required()`` + ``get_jwt_identity()``.
    """
    payload = _decode_token(token)

    if payload.get("token_type") != "access":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Access token required",
            headers={"WWW-Authenticate": "Bearer"},
        )

    user_id = payload.get("sub")
    if user_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token payload",
            headers={"WWW-Authenticate": "Bearer"},
        )

    result = await db.execute(select(User).where(User.id == int(user_id)))
    user = result.scalar_one_or_none()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User not found",
            headers={"WWW-Authenticate": "Bearer"},
        )
    return user


async def get_current_user_from_refresh(
    token: str = Depends(oauth2_scheme),
    db: AsyncSession = Depends(get_db),
) -> dict:
    """Validate a **refresh** token and return identity info.

    Replaces Flask-JWT-Extended's ``@jwt_required(refresh=True)``.
    Returns a dict with ``user_id`` (str) so the router can issue a new access token.
    """
    payload = _decode_token(token)

    if payload.get("token_type") != "refresh":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Refresh token required",
            headers={"WWW-Authenticate": "Bearer"},
        )

    user_id = payload.get("sub")
    if user_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token payload",
            headers={"WWW-Authenticate": "Bearer"},
        )

    return {"user_id": user_id}


def get_jti_from_token(token: str) -> str:
    """Extract the JTI claim from a token (for logout / revocation)."""
    payload = _decode_token(token)
    jti = payload.get("jti")
    if jti is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token missing jti claim",
            headers={"WWW-Authenticate": "Bearer"},
        )
    return jti
