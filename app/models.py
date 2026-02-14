from __future__ import annotations

from typing import List

from passlib.context import CryptContext
from sqlalchemy import ForeignKey, String, Float
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .database import Base


pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")


class User(Base):
    __tablename__ = "user"

    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)
    password_hash: Mapped[str] = mapped_column(String(128), nullable=False)

    stores: Mapped[List[Store]] = relationship(
        back_populates="owner",
        cascade="all, delete-orphan",
        lazy="select",
    )

    def set_password(self, password: str) -> None:
        self.password_hash = pwd_context.hash(password)

    def check_password(self, password: str) -> bool:
        return pwd_context.verify(password, self.password_hash)


class Store(Base):
    __tablename__ = "store"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    user_id: Mapped[int] = mapped_column(ForeignKey("user.id"), nullable=False)

    owner: Mapped[User] = relationship(back_populates="stores", lazy="select")
    tags: Mapped[List[Tag]] = relationship(
        back_populates="store",
        cascade="all, delete-orphan",
        lazy="select",
    )
    items: Mapped[List[Item]] = relationship(
        back_populates="store",
        cascade="all, delete-orphan",
        lazy="select",
    )


class Tag(Base):
    __tablename__ = "tag"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    store_id: Mapped[int] = mapped_column(ForeignKey("store.id"), nullable=False)

    store: Mapped[Store] = relationship(back_populates="tags", lazy="select")


class Item(Base):
    __tablename__ = "item"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    price: Mapped[float] = mapped_column(Float, nullable=False)
    store_id: Mapped[int] = mapped_column(ForeignKey("store.id"), nullable=False)

    store: Mapped[Store] = relationship(back_populates="items", lazy="select")
