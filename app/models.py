from __future__ import annotations

from typing import List, Optional

import bcrypt
from sqlalchemy import ForeignKey, String
from sqlalchemy.orm import (
    DeclarativeBase,
    Mapped,
    mapped_column,
    relationship,
)


class Base(DeclarativeBase):
    """Base class for all ORM models."""
    pass


class User(Base):
    __tablename__ = "user"

    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)
    password_hash: Mapped[str] = mapped_column(String(128), nullable=False)

    stores: Mapped[List["Store"]] = relationship(
        back_populates="owner",
        cascade="all, delete-orphan",
        lazy="selectin",
    )

    def set_password(self, password: str) -> None:
        pw_bytes = password.encode("utf-8")
        self.password_hash = bcrypt.hashpw(pw_bytes, bcrypt.gensalt()).decode("utf-8")

    def check_password(self, password: str) -> bool:
        pw_bytes = password.encode("utf-8")
        hash_bytes = self.password_hash.encode("utf-8")
        return bcrypt.checkpw(pw_bytes, hash_bytes)


class Store(Base):
    __tablename__ = "store"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    user_id: Mapped[int] = mapped_column(ForeignKey("user.id"), nullable=False)

    owner: Mapped["User"] = relationship(back_populates="stores", lazy="selectin")
    tags: Mapped[List["Tag"]] = relationship(
        back_populates="store",
        cascade="all, delete-orphan",
        lazy="selectin",
    )
    items: Mapped[List["Item"]] = relationship(
        back_populates="store",
        cascade="all, delete-orphan",
        lazy="selectin",
    )


class Tag(Base):
    __tablename__ = "tag"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    store_id: Mapped[int] = mapped_column(ForeignKey("store.id"), nullable=False)

    store: Mapped["Store"] = relationship(back_populates="tags", lazy="selectin")


class Item(Base):
    __tablename__ = "item"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    price: Mapped[float] = mapped_column(nullable=False)
    store_id: Mapped[int] = mapped_column(ForeignKey("store.id"), nullable=False)

    store: Mapped["Store"] = relationship(back_populates="items", lazy="selectin")
