"""Test configuration and fixtures."""

import os

import pytest


@pytest.fixture(autouse=True)
def set_test_env(monkeypatch):
    """Ensure test environment variables are set for all tests."""
    monkeypatch.setenv("SECRET_KEY", os.environ.get("SECRET_KEY", "test-secret-key"))
    monkeypatch.setenv("DATABASE_URL", os.environ.get("DATABASE_URL", "sqlite:///./test.db"))

    from app.config import get_settings
    get_settings.cache_clear()

    from app import database
    database._engine = None
