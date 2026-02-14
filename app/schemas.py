from pydantic import BaseModel, ConfigDict


# ── User Schemas ──────────────────────────────────────────────────


class UserCreate(BaseModel):
    username: str
    password: str


class UserResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    username: str


# ── Store Schemas ─────────────────────────────────────────────────


class StoreCreate(BaseModel):
    name: str


class StoreResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    user_id: int


# ── Item Schemas ──────────────────────────────────────────────────


class ItemCreate(BaseModel):
    name: str
    price: float
    store_id: int


class ItemUpdate(BaseModel):
    name: str | None = None
    price: float | None = None
    store_id: int | None = None


class ItemResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    price: float
    store_id: int


# ── Tag Schemas ───────────────────────────────────────────────────


class TagCreate(BaseModel):
    name: str


class TagResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    store_id: int
