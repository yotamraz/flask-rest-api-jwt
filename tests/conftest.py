"""Shared pytest fixtures for the FastAPI test suite.

Uses an in-memory SQLite database (aiosqlite) and httpx.AsyncClient for
fully async integration tests.
"""

from __future__ import annotations

from typing import AsyncGenerator

import pytest_asyncio
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from app.auth import token_blocklist
from app.database import get_db
from app.main import create_app
from app.models import Base

# ---------------------------------------------------------------------------
# Test database engine (in-memory SQLite)
# ---------------------------------------------------------------------------

TEST_DATABASE_URI = "sqlite+aiosqlite://"

test_engine = create_async_engine(TEST_DATABASE_URI, echo=False)
TestSessionLocal = async_sessionmaker(
    bind=test_engine,
    expire_on_commit=False,
    class_=AsyncSession,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest_asyncio.fixture
async def app():
    """Create a fresh FastAPI app with a clean in-memory database for each test."""
    # Create tables
    async with test_engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    fastapi_app = create_app()

    # Override the get_db dependency to use the test database
    async def override_get_db() -> AsyncGenerator[AsyncSession, None]:
        async with TestSessionLocal() as session:
            try:
                yield session
                await session.commit()
            except Exception:
                await session.rollback()
                raise

    fastapi_app.dependency_overrides[get_db] = override_get_db

    yield fastapi_app

    # Clean up — drop all tables and clear token blocklist
    async with test_engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    token_blocklist.clear()


@pytest_asyncio.fixture
async def client(app) -> AsyncGenerator[AsyncClient, None]:
    """Provide an httpx.AsyncClient bound to the test app."""
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        yield ac


@pytest_asyncio.fixture
async def registered_user(client: AsyncClient) -> dict:
    """Register a test user and return their info plus tokens.

    Returns a dict with keys: username, password, id, access_token, refresh_token.
    """
    username = "testuser"
    password = "testpass123"

    resp = await client.post(
        "/user/register",
        json={"username": username, "password": password},
    )
    assert resp.status_code == 201
    user_data = resp.json()

    # Login to get tokens
    resp = await client.post(
        "/user/login",
        json={"username": username, "password": password},
    )
    assert resp.status_code == 200
    tokens = resp.json()

    return {
        "username": username,
        "password": password,
        "id": user_data["id"],
        "access_token": tokens["access_token"],
        "refresh_token": tokens["refresh_token"],
    }


def auth_header(token: str) -> dict:
    """Return an Authorization header dict for the given bearer token."""
    return {"Authorization": f"Bearer {token}"}
