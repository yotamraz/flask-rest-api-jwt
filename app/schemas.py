from typing import Optional

from pydantic import BaseModel, ConfigDict


# ---------------------------------------------------------------------------
# Generic response helpers
# ---------------------------------------------------------------------------

class MessageResponse(BaseModel):
    message: str


# ---------------------------------------------------------------------------
# User schemas
# ---------------------------------------------------------------------------

class UserCreate(BaseModel):
    username: str
    password: str


class UserResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    username: str


class LoginRequest(BaseModel):
    username: str
    password: str


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str


class AccessTokenResponse(BaseModel):
    access_token: str


# ---------------------------------------------------------------------------
# Store schemas
# ---------------------------------------------------------------------------

class StoreCreate(BaseModel):
    name: str


class StoreResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    user_id: int


# ---------------------------------------------------------------------------
# Tag schemas
# ---------------------------------------------------------------------------

class TagCreate(BaseModel):
    name: str


class TagResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    store_id: int


# ---------------------------------------------------------------------------
# Item schemas
# ---------------------------------------------------------------------------

class ItemCreate(BaseModel):
    name: str
    price: float
    store_id: int


class ItemUpdate(BaseModel):
    name: Optional[str] = None
    price: Optional[float] = None
    store_id: Optional[int] = None


class ItemResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    price: float
    store_id: int
