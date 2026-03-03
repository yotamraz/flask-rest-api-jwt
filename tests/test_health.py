"""Tests for the health check endpoint."""

import pytest
from httpx import AsyncClient


@pytest.mark.asyncio
async def test_health_check_returns_200(client: AsyncClient):
    """GET /health/ should return 200 with healthy status when DB is reachable."""
    resp = await client.get("/health/")
    assert resp.status_code == 200

    data = resp.json()
    assert data["status"] == "healthy"
    assert data["database"] == "healthy"


@pytest.mark.asyncio
async def test_health_check_response_shape(client: AsyncClient):
    """GET /health/ response should contain exactly 'status' and 'database' keys."""
    resp = await client.get("/health/")
    data = resp.json()
    assert set(data.keys()) == {"status", "database"}
