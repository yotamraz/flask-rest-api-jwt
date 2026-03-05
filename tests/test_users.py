"""Tests for /user endpoints — registration, login, logout, refresh, get, delete."""
import pytest
from httpx import AsyncClient

from tests.conftest import auth_header, login_user, register_user


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_register_success(client: AsyncClient):
    """POST /user/register with a new username returns 201."""
    resp = await register_user(client)
    assert resp.status_code == 201
    data = resp.json()
    assert data["username"] == "testuser"
    assert "id" in data
    assert "password_hash" not in data
    assert "password" not in data


@pytest.mark.asyncio
async def test_register_duplicate(client: AsyncClient):
    """POST /user/register with an existing username returns 400."""
    await register_user(client)
    resp = await register_user(client)
    assert resp.status_code == 400
    assert resp.json()["message"] == "User exists"


# ---------------------------------------------------------------------------
# Login
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_login_success(client: AsyncClient):
    """POST /user/login with valid credentials returns tokens."""
    await register_user(client)
    resp = await login_user(client)
    assert resp.status_code == 200
    data = resp.json()
    assert "access_token" in data
    assert "refresh_token" in data


@pytest.mark.asyncio
async def test_login_invalid_password(client: AsyncClient):
    """POST /user/login with wrong password returns 401."""
    await register_user(client)
    resp = await login_user(client, password="wrongpassword")
    assert resp.status_code == 401
    assert resp.json()["message"] == "Invalid credentials"


@pytest.mark.asyncio
async def test_login_nonexistent_user(client: AsyncClient):
    """POST /user/login with unknown username returns 401."""
    resp = await login_user(client, username="noone")
    assert resp.status_code == 401
    assert resp.json()["message"] == "Invalid credentials"


# ---------------------------------------------------------------------------
# Logout
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_logout_success(client: AsyncClient):
    """POST /user/logout revokes the access token."""
    await register_user(client)
    login_resp = await login_user(client)
    tokens = login_resp.json()

    resp = await client.post("/user/logout", headers=auth_header(tokens["access_token"]))
    assert resp.status_code == 200
    assert resp.json()["message"] == "Logged out"


@pytest.mark.asyncio
async def test_logout_token_revoked(client: AsyncClient):
    """After logout, using the same access token should fail."""
    await register_user(client)
    login_resp = await login_user(client)
    tokens = login_resp.json()
    headers = auth_header(tokens["access_token"])

    # Logout
    await client.post("/user/logout", headers=headers)

    # Try to use the revoked token
    resp = await client.get("/user/1", headers=headers)
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_logout_without_token(client: AsyncClient):
    """POST /user/logout without a token returns 401."""
    resp = await client.post("/user/logout")
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Refresh
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_refresh_success(client: AsyncClient):
    """POST /user/refresh with a valid refresh token returns a new access token."""
    await register_user(client)
    login_resp = await login_user(client)
    tokens = login_resp.json()

    resp = await client.post(
        "/user/refresh", headers=auth_header(tokens["refresh_token"])
    )
    assert resp.status_code == 200
    data = resp.json()
    assert "access_token" in data
    # The new access token should be different from the original
    assert data["access_token"] != tokens["access_token"]


@pytest.mark.asyncio
async def test_refresh_with_access_token_fails(client: AsyncClient):
    """POST /user/refresh with an access token (not refresh) returns 401."""
    await register_user(client)
    login_resp = await login_user(client)
    tokens = login_resp.json()

    resp = await client.post(
        "/user/refresh", headers=auth_header(tokens["access_token"])
    )
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_refresh_without_token(client: AsyncClient):
    """POST /user/refresh without a token returns 401."""
    resp = await client.post("/user/refresh")
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Get User
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_get_user_success(client: AsyncClient):
    """GET /user/{id} returns own user profile."""
    reg_resp = await register_user(client)
    user_id = reg_resp.json()["id"]

    login_resp = await login_user(client)
    tokens = login_resp.json()

    resp = await client.get(
        f"/user/{user_id}", headers=auth_header(tokens["access_token"])
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["id"] == user_id
    assert data["username"] == "testuser"


@pytest.mark.asyncio
async def test_get_user_unauthorized_other_user(client: AsyncClient):
    """GET /user/{id} for another user's ID returns 401."""
    await register_user(client, username="user1", password="pass1")
    await register_user(client, username="user2", password="pass2")

    login_resp = await login_user(client, username="user1", password="pass1")
    tokens = login_resp.json()

    # Try to access user2's profile (ID 2)
    resp = await client.get("/user/2", headers=auth_header(tokens["access_token"]))
    assert resp.status_code == 401
    assert resp.json()["message"] == "Unauthorized"


@pytest.mark.asyncio
async def test_get_user_without_token(client: AsyncClient):
    """GET /user/{id} without auth returns 401."""
    await register_user(client)
    resp = await client.get("/user/1")
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Delete User
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_delete_user_success(client: AsyncClient):
    """DELETE /user/{id} deletes own account."""
    reg_resp = await register_user(client)
    user_id = reg_resp.json()["id"]

    login_resp = await login_user(client)
    tokens = login_resp.json()

    resp = await client.delete(
        f"/user/{user_id}", headers=auth_header(tokens["access_token"])
    )
    assert resp.status_code == 200
    assert resp.json()["message"] == "Deleted"


@pytest.mark.asyncio
async def test_delete_user_unauthorized_other_user(client: AsyncClient):
    """DELETE /user/{id} for another user returns 401."""
    await register_user(client, username="user1", password="pass1")
    await register_user(client, username="user2", password="pass2")

    login_resp = await login_user(client, username="user1", password="pass1")
    tokens = login_resp.json()

    resp = await client.delete("/user/2", headers=auth_header(tokens["access_token"]))
    assert resp.status_code == 401
    assert resp.json()["message"] == "Unauthorized"


@pytest.mark.asyncio
async def test_delete_user_without_token(client: AsyncClient):
    """DELETE /user/{id} without auth returns 401."""
    await register_user(client)
    resp = await client.delete("/user/1")
    assert resp.status_code == 401
