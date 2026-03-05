from pydantic import BaseModel, ConfigDict


# ---------------------------------------------------------------------------
# User schemas
# ---------------------------------------------------------------------------

class UserRegisterRequest(BaseModel):
    username: str
    password: str


class UserLoginRequest(BaseModel):
    username: str
    password: str


class UserResponse(BaseModel):
    """Matches the Marshmallow UserSchema output (excludes password_hash)."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    username: str


# ---------------------------------------------------------------------------
# Store schemas
# ---------------------------------------------------------------------------

class StoreCreateRequest(BaseModel):
    name: str


class StoreResponse(BaseModel):
    """Matches the Marshmallow StoreSchema output (includes FK)."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    user_id: int


# ---------------------------------------------------------------------------
# Item schemas
# ---------------------------------------------------------------------------

class ItemCreateRequest(BaseModel):
    name: str
    price: float
    store_id: int


class ItemUpdateRequest(BaseModel):
    name: str | None = None
    price: float | None = None
    store_id: int | None = None


class ItemResponse(BaseModel):
    """Matches the Marshmallow ItemSchema output (includes FK)."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    price: float
    store_id: int


# ---------------------------------------------------------------------------
# Tag schemas
# ---------------------------------------------------------------------------

class TagCreateRequest(BaseModel):
    name: str


class TagResponse(BaseModel):
    """Matches the Marshmallow TagSchema output (includes FK)."""
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    store_id: int
