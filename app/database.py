"""Async SQLAlchemy engine, session factory, and FastAPI dependency.

Replaces Flask-SQLAlchemy's ``db = SQLAlchemy()`` pattern with a standalone
async engine and ``async_sessionmaker``.  The ``get_db`` dependency is
injected into route handlers via ``Depends(get_db)``.

Design decision: ``get_db()`` yields a session and rolls back on exception,
but does **not** auto-commit.  Route handlers call ``await session.commit()``
explicitly, mirroring the source Flask code which calls
``db.session.commit()`` in mutation endpoints.
"""

from collections.abc import AsyncGenerator

from sqlalchemy.ext.asyncio import (
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from sqlalchemy.orm import DeclarativeBase

from .config import settings

engine = create_async_engine(
    settings.SQLALCHEMY_DATABASE_URI,
    echo=settings.DEBUG,
)

SessionLocal = async_sessionmaker(
    bind=engine,
    expire_on_commit=False,
    class_=AsyncSession,
)


class Base(DeclarativeBase):
    """Shared declarative base for all ORM models."""


async def get_db() -> AsyncGenerator[AsyncSession, None]:
    """FastAPI dependency that yields an async database session.

    Rolls back the transaction on unhandled exceptions; otherwise the caller
    (route handler) is responsible for committing.
    """
    async with SessionLocal() as session:
        try:
            yield session
        except Exception:
            await session.rollback()
            raise
