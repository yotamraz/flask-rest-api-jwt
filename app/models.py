from __future__ import annotations

from sqlalchemy import ForeignKey, String, Float, Integer
from sqlalchemy.orm import (
    DeclarativeBase,
    Mapped,
    mapped_column,
    relationship,
)


class Base(DeclarativeBase):
    """Declarative base for all ORM models."""
    pass


class User(Base):
    __tablename__ = "user"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    username: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)
    password_hash: Mapped[str] = mapped_column(String(256), nullable=False)

    stores: Mapped[list[Store]] = relationship(
        back_populates="owner",
        cascade="all, delete-orphan",
        lazy="selectin",
    )


class Store(Base):
    __tablename__ = "store"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    user_id: Mapped[int] = mapped_column(Integer, ForeignKey("user.id"), nullable=False)

    owner: Mapped[User] = relationship(back_populates="stores", lazy="selectin")
    tags: Mapped[list[Tag]] = relationship(
        back_populates="store",
        cascade="all, delete-orphan",
        lazy="selectin",
    )
    items: Mapped[list[Item]] = relationship(
        back_populates="store",
        cascade="all, delete-orphan",
        lazy="selectin",
    )


class Tag(Base):
    __tablename__ = "tag"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    store_id: Mapped[int] = mapped_column(Integer, ForeignKey("store.id"), nullable=False)

    store: Mapped[Store] = relationship(back_populates="tags", lazy="selectin")


class Item(Base):
    __tablename__ = "item"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    price: Mapped[float] = mapped_column(Float, nullable=False)
    store_id: Mapped[int] = mapped_column(Integer, ForeignKey("store.id"), nullable=False)

    store: Mapped[Store] = relationship(back_populates="items", lazy="selectin")
