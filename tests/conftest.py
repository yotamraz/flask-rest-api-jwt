"""Shared pytest fixtures for the FastAPI test suite."""
from __future__ import annotations

from collections.abc import AsyncGenerator

import pytest_asyncio
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import (
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)

from app.auth import token_blocklist
from app.models import Base

# ---------------------------------------------------------------------------
# In-memory SQLite engine for tests
# ---------------------------------------------------------------------------
TEST_DATABASE_URL = "sqlite+aiosqlite://"

test_engine = create_async_engine(TEST_DATABASE_URL, echo=False)
TestSessionLocal = async_sessionmaker(
    bind=test_engine,
    expire_on_commit=False,
    class_=AsyncSession,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest_asyncio.fixture()
async def _setup_db():
    """Create tables before each test and drop them afterwards."""
    async with test_engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield
    async with test_engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    # Clear the token blocklist between tests
    token_blocklist.clear()


@pytest_asyncio.fixture()
async def db_session(_setup_db) -> AsyncGenerator[AsyncSession, None]:
    """Yield an async session bound to the test database."""
    async with TestSessionLocal() as session:
        yield session


@pytest_asyncio.fixture()
async def client(_setup_db) -> AsyncGenerator[AsyncClient, None]:
    """Yield an httpx.AsyncClient wired to the FastAPI app with test DB."""
    from app.database import get_db
    from app.main import create_app

    app = create_app()

    async def _override_get_db() -> AsyncGenerator[AsyncSession, None]:
        async with TestSessionLocal() as session:
            try:
                yield session
            except Exception:
                await session.rollback()
                raise

    app.dependency_overrides[get_db] = _override_get_db

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        yield ac

    app.dependency_overrides.clear()


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


async def register_user(
    client: AsyncClient,
    username: str = "testuser",
    password: str = "testpass123",
) -> dict:
    """Helper: register a user and return the response JSON."""
    resp = await client.post(
        "/user/register",
        json={"username": username, "password": password},
    )
    return resp


async def login_user(
    client: AsyncClient,
    username: str = "testuser",
    password: str = "testpass123",
) -> dict:
    """Helper: login and return the response JSON with tokens."""
    resp = await client.post(
        "/user/login",
        json={"username": username, "password": password},
    )
    return resp


def auth_header(token: str) -> dict[str, str]:
    """Return an Authorization header dict."""
    return {"Authorization": f"Bearer {token}"}
