"""SQLAlchemy 2.x ORM models.

Migrated from Flask-SQLAlchemy ``db.Model`` to ``DeclarativeBase`` with modern
``Mapped`` type annotations and ``mapped_column()``.  Password hashing uses
``bcrypt`` directly instead of ``werkzeug.security``.

Relationships use ``lazy="selectin"`` so they load correctly in async
contexts without triggering implicit IO.
"""

from __future__ import annotations

from typing import List

import bcrypt
from sqlalchemy import ForeignKey, String
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .database import Base


class User(Base):
    __tablename__ = "user"

    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)
    password_hash: Mapped[str] = mapped_column(String(256), nullable=False)

    stores: Mapped[List["Store"]] = relationship(
        back_populates="owner",
        lazy="selectin",
        cascade="all, delete-orphan",
    )

    def set_password(self, password: str) -> None:
        self.password_hash = bcrypt.hashpw(
            password.encode("utf-8"), bcrypt.gensalt()
        ).decode("utf-8")

    def check_password(self, password: str) -> bool:
        return bcrypt.checkpw(
            password.encode("utf-8"), self.password_hash.encode("utf-8")
        )


class Store(Base):
    __tablename__ = "store"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    user_id: Mapped[int] = mapped_column(ForeignKey("user.id"), nullable=False)

    owner: Mapped["User"] = relationship(back_populates="stores", lazy="selectin")
    tags: Mapped[List["Tag"]] = relationship(
        back_populates="store",
        lazy="selectin",
        cascade="all, delete-orphan",
    )
    items: Mapped[List["Item"]] = relationship(
        back_populates="store",
        lazy="selectin",
        cascade="all, delete-orphan",
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
