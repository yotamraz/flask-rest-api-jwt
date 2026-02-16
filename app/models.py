from sqlalchemy import Column, Float, ForeignKey, Integer, String
from sqlalchemy.orm import relationship

from .database import Base


class User(Base):
    __tablename__ = "user"

    id = Column(Integer, primary_key=True)
    username = Column(String(64), unique=True, nullable=False)
    password_hash = Column(String(128), nullable=False)

    stores = relationship(
        "Store", backref="owner", lazy=True, cascade="all, delete-orphan"
    )


class Store(Base):
    __tablename__ = "store"

    id = Column(Integer, primary_key=True)
    name = Column(String(64), nullable=False)
    user_id = Column(Integer, ForeignKey("user.id"), nullable=False)

    tags = relationship(
        "Tag", backref="store", lazy=True, cascade="all, delete-orphan"
    )
    items = relationship(
        "Item", backref="store", lazy=True, cascade="all, delete-orphan"
    )


class Tag(Base):
    __tablename__ = "tag"

    id = Column(Integer, primary_key=True)
    name = Column(String(64), nullable=False)
    store_id = Column(Integer, ForeignKey("store.id"), nullable=False)


class Item(Base):
    __tablename__ = "item"

    id = Column(Integer, primary_key=True)
    name = Column(String(64), nullable=False)
    price = Column(Float, nullable=False)
    store_id = Column(Integer, ForeignKey("store.id"), nullable=False)
