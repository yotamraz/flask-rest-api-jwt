"""Tests for the /health/ endpoint."""


def test_health_returns_200_when_db_reachable(client):
    """GET /health/ returns 200 with healthy status when the database is reachable."""
    response = client.get("/health/")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "healthy"
    assert data["database"] == "healthy"


def test_health_response_has_expected_keys(client):
    """GET /health/ response contains both 'status' and 'database' keys."""
    response = client.get("/health/")
    data = response.json()
    assert "status" in data
    assert "database" in data
