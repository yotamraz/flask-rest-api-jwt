"""Tests for the health check endpoint."""

import pytest
from httpx import AsyncClient


@pytest.mark.asyncio(loop_scope="session")
async def test_health_returns_200(client: AsyncClient):
    """GET /health/ should return 200 with healthy status."""
    response = await client.get("/health/")
    assert response.status_code == 200


@pytest.mark.asyncio(loop_scope="session")
async def test_health_response_structure(client: AsyncClient):
    """GET /health/ response should contain 'status' and 'database' keys."""
    response = await client.get("/health/")
    data = response.json()
    assert "status" in data
    assert "database" in data
    assert data["status"] == "healthy"
    assert data["database"] == "healthy"
