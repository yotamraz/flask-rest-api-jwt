from pydantic import BaseModel, ConfigDict


# ---------------------------------------------------------------------------
# User schemas
# ---------------------------------------------------------------------------

class UserCreate(BaseModel):
    """Payload for POST /user/register."""
    username: str
    password: str


class UserLogin(BaseModel):
    """Payload for POST /user/login."""
    username: str
    password: str


class UserResponse(BaseModel):
    """Response body for user endpoints (excludes password_hash)."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    username: str


# ---------------------------------------------------------------------------
# Store schemas
# ---------------------------------------------------------------------------

class StoreCreate(BaseModel):
    """Payload for POST /store/."""
    name: str


class StoreResponse(BaseModel):
    """Response body for store endpoints."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    user_id: int


# ---------------------------------------------------------------------------
# Tag schemas
# ---------------------------------------------------------------------------

class TagCreate(BaseModel):
    """Payload for POST /tag/store/<store_id>."""
    name: str


class TagResponse(BaseModel):
    """Response body for tag endpoints."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    store_id: int


# ---------------------------------------------------------------------------
# Item schemas
# ---------------------------------------------------------------------------

class ItemCreate(BaseModel):
    """Payload for POST /item/."""
    name: str
    price: float
    store_id: int


class ItemUpdate(BaseModel):
    """Payload for PUT /item/<id>.  All fields optional."""
    name: str | None = None
    price: float | None = None
    store_id: int | None = None


class ItemResponse(BaseModel):
    """Response body for item endpoints."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    price: float
    store_id: int


# ---------------------------------------------------------------------------
# Token schemas
# ---------------------------------------------------------------------------

class TokenPair(BaseModel):
    """Response body for POST /user/login."""
    access_token: str
    refresh_token: str


class TokenRefresh(BaseModel):
    """Response body for POST /user/refresh."""
    access_token: str


class MessageResponse(BaseModel):
    """Generic message response."""
    message: str
