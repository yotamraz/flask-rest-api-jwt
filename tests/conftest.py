"""Shared test fixtures for the FastAPI application.

Uses an in-memory SQLite database (aiosqlite) and httpx.AsyncClient
for fast, isolated integration tests.
"""

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
from app.database import get_db
from app.main import create_app
from app.models import Base

TEST_DATABASE_URL = "sqlite+aiosqlite://"  # in-memory SQLite


# ---------------------------------------------------------------------------
# Database fixtures
# ---------------------------------------------------------------------------

@pytest_asyncio.fixture()
async def test_engine():
    """Create a fresh async engine with in-memory SQLite for each test."""
    engine = create_async_engine(TEST_DATABASE_URL, echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    await engine.dispose()


@pytest_asyncio.fixture()
async def test_session(test_engine) -> AsyncGenerator[AsyncSession, None]:
    """Yield an async session bound to the test engine."""
    session_factory = async_sessionmaker(
        bind=test_engine,
        expire_on_commit=False,
        class_=AsyncSession,
    )
    async with session_factory() as session:
        yield session


# ---------------------------------------------------------------------------
# Application & client fixtures
# ---------------------------------------------------------------------------

@pytest_asyncio.fixture()
async def app(test_engine):
    """FastAPI application wired to the test database."""
    application = create_app()

    # Override the get_db dependency to use the test engine
    session_factory = async_sessionmaker(
        bind=test_engine,
        expire_on_commit=False,
        class_=AsyncSession,
    )

    async def override_get_db() -> AsyncGenerator[AsyncSession, None]:
        async with session_factory() as session:
            try:
                yield session
                await session.commit()
            except Exception:
                await session.rollback()
                raise

    application.dependency_overrides[get_db] = override_get_db

    # Clear the token blocklist before each test
    token_blocklist.clear()

    yield application

    # Clean up overrides
    application.dependency_overrides.clear()


@pytest_asyncio.fixture()
async def client(app) -> AsyncGenerator[AsyncClient, None]:
    """Async HTTP client bound to the test application."""
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        yield ac


# ---------------------------------------------------------------------------
# Helper fixtures for authenticated requests
# ---------------------------------------------------------------------------

@pytest_asyncio.fixture()
async def registered_user(client: AsyncClient) -> dict:
    """Register a test user and return the user data along with credentials.

    Returns dict with keys: id, username, password
    """
    resp = await client.post(
        "/user/register",
        json={"username": "testuser", "password": "testpass123"},
    )
    assert resp.status_code == 201
    data = resp.json()
    return {
        "id": data["id"],
        "username": data["username"],
        "password": "testpass123",
    }


@pytest_asyncio.fixture()
async def auth_tokens(client: AsyncClient, registered_user: dict) -> dict:
    """Log in a registered user and return access + refresh tokens.

    Returns dict with keys: access_token, refresh_token, user_id
    """
    resp = await client.post(
        "/user/login",
        json={
            "username": registered_user["username"],
            "password": registered_user["password"],
        },
    )
    assert resp.status_code == 200
    tokens = resp.json()
    return {
        "access_token": tokens["access_token"],
        "refresh_token": tokens["refresh_token"],
        "user_id": registered_user["id"],
    }


@pytest_asyncio.fixture()
def auth_headers(auth_tokens: dict) -> dict:
    """Authorization headers using the access token."""
    return {"Authorization": f"Bearer {auth_tokens['access_token']}"}
