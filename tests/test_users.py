"""Tests for the users router endpoints."""

import pytest
from httpx import AsyncClient

from tests.conftest import auth_header


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_register_success(client: AsyncClient):
    """POST /user/register with a new username returns 201 and user data."""
    resp = await client.post(
        "/user/register",
        json={"username": "newuser", "password": "secret123"},
    )
    assert resp.status_code == 201

    data = resp.json()
    assert "id" in data
    assert data["username"] == "newuser"
    # password_hash must NOT be exposed
    assert "password_hash" not in data
    assert "password" not in data


@pytest.mark.asyncio
async def test_register_duplicate_username(client: AsyncClient):
    """POST /user/register with an existing username returns 400."""
    await client.post(
        "/user/register",
        json={"username": "dupuser", "password": "pass1"},
    )
    resp = await client.post(
        "/user/register",
        json={"username": "dupuser", "password": "pass2"},
    )
    assert resp.status_code == 400
    assert "User exists" in resp.json()["detail"]


# ---------------------------------------------------------------------------
# Login
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_login_success(client: AsyncClient):
    """POST /user/login with valid credentials returns access + refresh tokens."""
    await client.post(
        "/user/register",
        json={"username": "loginuser", "password": "mypass"},
    )
    resp = await client.post(
        "/user/login",
        json={"username": "loginuser", "password": "mypass"},
    )
    assert resp.status_code == 200

    data = resp.json()
    assert "access_token" in data
    assert "refresh_token" in data
    assert len(data["access_token"]) > 0
    assert len(data["refresh_token"]) > 0


@pytest.mark.asyncio
async def test_login_invalid_password(client: AsyncClient):
    """POST /user/login with wrong password returns 401."""
    await client.post(
        "/user/register",
        json={"username": "wrongpw", "password": "correct"},
    )
    resp = await client.post(
        "/user/login",
        json={"username": "wrongpw", "password": "incorrect"},
    )
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_login_nonexistent_user(client: AsyncClient):
    """POST /user/login for a user that doesn't exist returns 401."""
    resp = await client.post(
        "/user/login",
        json={"username": "ghostuser", "password": "whatever"},
    )
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Logout
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_logout_success(registered_user, client: AsyncClient):
    """POST /user/logout revokes the access token."""
    token = registered_user["access_token"]

    resp = await client.post("/user/logout", headers=auth_header(token))
    assert resp.status_code == 200
    assert resp.json()["message"] == "Logged out"

    # After logout, the token should be revoked
    resp = await client.get(
        f"/user/{registered_user['id']}",
        headers=auth_header(token),
    )
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_logout_without_token(client: AsyncClient):
    """POST /user/logout without a token returns 403."""
    resp = await client.post("/user/logout")
    assert resp.status_code == 403


# ---------------------------------------------------------------------------
# Refresh
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_refresh_success(registered_user, client: AsyncClient):
    """POST /user/refresh with a valid refresh token returns a new access token."""
    resp = await client.post(
        "/user/refresh",
        headers=auth_header(registered_user["refresh_token"]),
    )
    assert resp.status_code == 200

    data = resp.json()
    assert "access_token" in data
    assert len(data["access_token"]) > 0


@pytest.mark.asyncio
async def test_refresh_with_access_token_fails(registered_user, client: AsyncClient):
    """POST /user/refresh with an access token (not refresh) returns 401."""
    resp = await client.post(
        "/user/refresh",
        headers=auth_header(registered_user["access_token"]),
    )
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Get user
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_get_user_success(registered_user, client: AsyncClient):
    """GET /user/{id} for the authenticated user returns their profile."""
    user_id = registered_user["id"]
    token = registered_user["access_token"]

    resp = await client.get(f"/user/{user_id}", headers=auth_header(token))
    assert resp.status_code == 200

    data = resp.json()
    assert data["id"] == user_id
    assert data["username"] == registered_user["username"]
    assert "password_hash" not in data


@pytest.mark.asyncio
async def test_get_other_user_unauthorized(registered_user, client: AsyncClient):
    """GET /user/{id} for a different user returns 401."""
    token = registered_user["access_token"]
    other_id = registered_user["id"] + 999  # non-existent/other user

    resp = await client.get(f"/user/{other_id}", headers=auth_header(token))
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_get_user_without_token(client: AsyncClient):
    """GET /user/{id} without authentication returns 403."""
    resp = await client.get("/user/1")
    assert resp.status_code == 403


# ---------------------------------------------------------------------------
# Delete user
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_delete_user_success(registered_user, client: AsyncClient):
    """DELETE /user/{id} for the authenticated user deletes the account."""
    user_id = registered_user["id"]
    token = registered_user["access_token"]

    resp = await client.delete(f"/user/{user_id}", headers=auth_header(token))
    assert resp.status_code == 200
    assert resp.json()["message"] == "Deleted"


@pytest.mark.asyncio
async def test_delete_other_user_unauthorized(registered_user, client: AsyncClient):
    """DELETE /user/{id} for a different user returns 401."""
    token = registered_user["access_token"]
    other_id = registered_user["id"] + 999

    resp = await client.delete(f"/user/{other_id}", headers=auth_header(token))
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_delete_user_without_token(client: AsyncClient):
    """DELETE /user/{id} without authentication returns 403."""
    resp = await client.delete("/user/1")
    assert resp.status_code == 403
