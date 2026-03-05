"""Pydantic v2 request/response schemas.

Replaces Marshmallow-SQLAlchemy ``SQLAlchemyAutoSchema`` classes with
explicit Pydantic models.  ``model_config = ConfigDict(from_attributes=True)``
enables direct construction from ORM model instances.
"""

from pydantic import BaseModel, ConfigDict


# ---------------------------------------------------------------------------
# User schemas
# ---------------------------------------------------------------------------

class UserCreate(BaseModel):
    """Request body for user registration."""
    username: str
    password: str


class UserLogin(BaseModel):
    """Request body for user login."""
    username: str
    password: str


class UserResponse(BaseModel):
    """Public user representation (excludes password_hash)."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    username: str


# ---------------------------------------------------------------------------
# Store schemas
# ---------------------------------------------------------------------------

class StoreCreate(BaseModel):
    """Request body for creating a store."""
    name: str


class StoreResponse(BaseModel):
    """Store representation including foreign key."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    user_id: int


# ---------------------------------------------------------------------------
# Tag schemas
# ---------------------------------------------------------------------------

class TagCreate(BaseModel):
    """Request body for creating a tag."""
    name: str


class TagResponse(BaseModel):
    """Tag representation including foreign key."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    store_id: int


# ---------------------------------------------------------------------------
# Item schemas
# ---------------------------------------------------------------------------

class ItemCreate(BaseModel):
    """Request body for creating an item."""
    name: str
    price: float
    store_id: int


class ItemUpdate(BaseModel):
    """Request body for updating an item (all fields optional)."""
    name: str | None = None
    price: float | None = None
    store_id: int | None = None


class ItemResponse(BaseModel):
    """Item representation including foreign key."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    price: float
    store_id: int


# ---------------------------------------------------------------------------
# Auth / Token schemas
# ---------------------------------------------------------------------------

class TokenResponse(BaseModel):
    """Response for login (access + refresh tokens)."""
    access_token: str
    refresh_token: str


class AccessTokenResponse(BaseModel):
    """Response for token refresh (access token only)."""
    access_token: str


class MessageResponse(BaseModel):
    """Generic message response."""
    message: str
