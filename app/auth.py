"""JWT authentication and password hashing utilities.

Replaces Flask-JWT-Extended with python-jose and passlib[bcrypt].
"""

from __future__ import annotations

import uuid
from datetime import datetime, timedelta, timezone

from fastapi import Depends
from fastapi.security import OAuth2PasswordBearer
from jose import JWTError, jwt
from passlib.context import CryptContext
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from .config import settings
from .database import get_db
from .models import User


class JWTAuthError(Exception):
    """Raised for JWT/auth-related errors.

    Flask-JWT-Extended returns ``{"msg": "..."}`` for all auth errors.
    This custom exception lets us replicate that contract.
    """

    def __init__(self, msg: str, status_code: int = 401):
        self.msg = msg
        self.status_code = status_code

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
        raise JWTAuthError("Invalid or expired token")
    # Check blocklist
    jti = payload.get("jti")
    if jti and jti in token_blocklist:
        raise JWTAuthError("Token has been revoked")
    return payload


# ---------------------------------------------------------------------------
# FastAPI dependencies
# ---------------------------------------------------------------------------

oauth2_scheme = OAuth2PasswordBearer(tokenUrl="/user/login", auto_error=False)


def _require_token(token: str | None) -> str:
    """Raise ``JWTAuthError`` when the Authorization header is absent."""
    if token is None:
        raise JWTAuthError("Missing Authorization Header")
    return token


async def get_current_user(
    token: str | None = Depends(oauth2_scheme),
    db: AsyncSession = Depends(get_db),
) -> User:
    """Validate an **access** token and return the corresponding User.

    Replaces Flask-JWT-Extended's ``@jwt_required()`` + ``get_jwt_identity()``.
    """
    tok = _require_token(token)
    payload = _decode_token(tok)

    if payload.get("token_type") != "access":
        raise JWTAuthError("Access token required")

    user_id = payload.get("sub")
    if user_id is None:
        raise JWTAuthError("Invalid token payload")

    result = await db.execute(select(User).where(User.id == int(user_id)))
    user = result.scalar_one_or_none()
    if user is None:
        raise JWTAuthError("User not found")
    return user


async def get_validated_access_token(
    token: str | None = Depends(oauth2_scheme),
) -> dict:
    """Validate an **access** token and return its claims (no DB lookup).

    Used by logout where the user may no longer exist in the DB but the
    token itself is still valid. Mirrors Flask-JWT-Extended ``@jwt_required()``
    which only validates the token, not the user.
    """
    tok = _require_token(token)
    payload = _decode_token(tok)

    if payload.get("token_type") != "access":
        raise JWTAuthError("Access token required")

    return payload


async def get_current_user_from_refresh(
    token: str | None = Depends(oauth2_scheme),
    db: AsyncSession = Depends(get_db),
) -> dict:
    """Validate a **refresh** token and return identity info.

    Replaces Flask-JWT-Extended's ``@jwt_required(refresh=True)``.
    Returns a dict with ``user_id`` (str) so the router can issue a new access token.
    """
    tok = _require_token(token)
    payload = _decode_token(tok)

    if payload.get("token_type") != "refresh":
        raise JWTAuthError("Refresh token required")

    user_id = payload.get("sub")
    if user_id is None:
        raise JWTAuthError("Invalid token payload")

    return {"user_id": user_id}


def get_jti_from_token(token: str) -> str:
    """Extract the JTI claim from a token (for logout / revocation)."""
    payload = _decode_token(token)
    jti = payload.get("jti")
    if jti is None:
        raise JWTAuthError("Token missing jti claim")
    return jti
