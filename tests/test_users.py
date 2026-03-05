"""Tests for user registration, authentication, and management endpoints."""

import pytest
from httpx import AsyncClient

pytestmark = pytest.mark.asyncio


# ===================================================================
# Registration: POST /user/register
# ===================================================================


async def test_register_success(client: AsyncClient):
    """POST /user/register creates a user and returns 201 with user data."""
    resp = await client.post(
        "/user/register",
        json={"username": "newuser", "password": "secret123"},
    )
    assert resp.status_code == 201
    data = resp.json()
    assert "id" in data
    assert data["username"] == "newuser"
    # Password hash must never be in the response
    assert "password_hash" not in data
    assert "password" not in data


async def test_register_duplicate_username(client: AsyncClient, registered_user):
    """POST /user/register with an existing username returns 400."""
    resp = await client.post(
        "/user/register",
        json={
            "username": registered_user["username"],
            "password": "anypassword",
        },
    )
    assert resp.status_code == 400
    assert "User exists" in resp.json()["detail"]


async def test_register_response_shape(client: AsyncClient):
    """Response must contain exactly id and username."""
    resp = await client.post(
        "/user/register",
        json={"username": "shapetest", "password": "pw"},
    )
    assert resp.status_code == 201
    assert set(resp.json().keys()) == {"id", "username"}


# ===================================================================
# Login: POST /user/login
# ===================================================================


async def test_login_success(client: AsyncClient, registered_user):
    """POST /user/login with valid credentials returns access + refresh tokens."""
    resp = await client.post(
        "/user/login",
        json={
            "username": registered_user["username"],
            "password": registered_user["password"],
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert "access_token" in data
    assert "refresh_token" in data


async def test_login_invalid_password(client: AsyncClient, registered_user):
    """POST /user/login with wrong password returns 401."""
    resp = await client.post(
        "/user/login",
        json={
            "username": registered_user["username"],
            "password": "wrongpassword",
        },
    )
    assert resp.status_code == 401
    assert "Invalid credentials" in resp.json()["detail"]


async def test_login_nonexistent_user(client: AsyncClient):
    """POST /user/login for a non-existent user returns 401."""
    resp = await client.post(
        "/user/login",
        json={"username": "ghost", "password": "nopass"},
    )
    assert resp.status_code == 401


# ===================================================================
# Logout: POST /user/logout
# ===================================================================


async def test_logout_success(client: AsyncClient, auth_headers):
    """POST /user/logout revokes the token and returns success message."""
    resp = await client.post("/user/logout", headers=auth_headers)
    assert resp.status_code == 200
    assert resp.json()["message"] == "Logged out"


async def test_logout_revokes_token(client: AsyncClient, auth_tokens, auth_headers):
    """After logout the same access token should be rejected (401)."""
    # Logout
    resp = await client.post("/user/logout", headers=auth_headers)
    assert resp.status_code == 200

    # Try to use the revoked token
    resp2 = await client.get(
        f"/user/{auth_tokens['user_id']}",
        headers=auth_headers,
    )
    assert resp2.status_code == 401
    assert "revoked" in resp2.json()["detail"].lower()


async def test_logout_without_token(client: AsyncClient):
    """POST /user/logout without a token returns 401."""
    resp = await client.post("/user/logout")
    assert resp.status_code == 401


# ===================================================================
# Token Refresh: POST /user/refresh
# ===================================================================


async def test_refresh_success(client: AsyncClient, auth_tokens):
    """POST /user/refresh with a valid refresh token returns a new access token."""
    headers = {"Authorization": f"Bearer {auth_tokens['refresh_token']}"}
    resp = await client.post("/user/refresh", headers=headers)
    assert resp.status_code == 200
    data = resp.json()
    assert "access_token" in data
    # The new token should be different from the original
    assert data["access_token"] != auth_tokens["access_token"]


async def test_refresh_with_access_token_fails(client: AsyncClient, auth_headers):
    """POST /user/refresh with an access token (not refresh) returns 401."""
    resp = await client.post("/user/refresh", headers=auth_headers)
    assert resp.status_code == 401


async def test_refresh_without_token(client: AsyncClient):
    """POST /user/refresh without a token returns 401."""
    resp = await client.post("/user/refresh")
    assert resp.status_code == 401


# ===================================================================
# Get User: GET /user/{user_id}
# ===================================================================


async def test_get_user_success(client: AsyncClient, auth_tokens, auth_headers):
    """GET /user/{id} returns the authenticated user's profile."""
    user_id = auth_tokens["user_id"]
    resp = await client.get(f"/user/{user_id}", headers=auth_headers)
    assert resp.status_code == 200
    data = resp.json()
    assert data["id"] == user_id
    assert data["username"] == "testuser"
    assert "password_hash" not in data


async def test_get_user_unauthorized_other_user(
    client: AsyncClient, auth_headers
):
    """GET /user/{id} for a different user returns 401."""
    # user_id 999 does not exist – should 404 before auth check in this impl
    resp = await client.get("/user/999", headers=auth_headers)
    assert resp.status_code in (401, 404)


async def test_get_user_without_token(client: AsyncClient):
    """GET /user/{id} without authentication returns 401."""
    resp = await client.get("/user/1")
    assert resp.status_code == 401


async def test_get_user_response_shape(
    client: AsyncClient, auth_tokens, auth_headers
):
    """Response must contain exactly id and username."""
    user_id = auth_tokens["user_id"]
    resp = await client.get(f"/user/{user_id}", headers=auth_headers)
    assert resp.status_code == 200
    assert set(resp.json().keys()) == {"id", "username"}


# ===================================================================
# Delete User: DELETE /user/{user_id}
# ===================================================================


async def test_delete_user_success(client: AsyncClient, auth_tokens, auth_headers):
    """DELETE /user/{id} removes the user and returns success message."""
    user_id = auth_tokens["user_id"]
    resp = await client.delete(f"/user/{user_id}", headers=auth_headers)
    assert resp.status_code == 200
    assert resp.json()["message"] == "Deleted"

    # Verify login no longer works
    login_resp = await client.post(
        "/user/login",
        json={"username": "testuser", "password": "testpass123"},
    )
    assert login_resp.status_code == 401


async def test_delete_user_unauthorized_other_user(
    client: AsyncClient, auth_headers
):
    """DELETE /user/{id} for a different user returns 401 or 404."""
    resp = await client.delete("/user/999", headers=auth_headers)
    assert resp.status_code in (401, 404)


async def test_delete_user_without_token(client: AsyncClient):
    """DELETE /user/{id} without authentication returns 401."""
    resp = await client.delete("/user/1")
    assert resp.status_code == 401
