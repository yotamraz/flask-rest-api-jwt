"""Test configuration and fixtures."""

import os

# Set environment variables BEFORE any app imports, since app/main.py
# calls create_app() at module level which requires these.
os.environ.setdefault("SECRET_KEY", "test-secret-key")
os.environ.setdefault("DATABASE_URL", "sqlite:///./test.db")

import pytest
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from app.config import Settings, get_settings
from app.database import Base, get_db
from app.main import create_app


TEST_DATABASE_URL = "sqlite:///./test.db"


def get_test_settings() -> Settings:
    return Settings(
        SECRET_KEY=os.environ.get("SECRET_KEY", "test-secret-key"),
        DATABASE_URL=TEST_DATABASE_URL,
        ENVIRONMENT="test",
    )


@pytest.fixture(autouse=True)
def set_test_env(monkeypatch):
    """Ensure test environment variables are set for all tests."""
    monkeypatch.setenv("SECRET_KEY", os.environ.get("SECRET_KEY", "test-secret-key"))
    monkeypatch.setenv("DATABASE_URL", TEST_DATABASE_URL)

    get_settings.cache_clear()

    from app import database
    database._engine = None


@pytest.fixture()
def db_session():
    """Provide a clean database session for each test.

    Creates all tables before the test and drops them after.
    """
    engine = create_engine(TEST_DATABASE_URL, future=True)
    Base.metadata.create_all(bind=engine)
    TestingSessionLocal = sessionmaker(
        autocommit=False, autoflush=False, bind=engine
    )
    session = TestingSessionLocal()
    try:
        yield session
    finally:
        session.close()
        Base.metadata.drop_all(bind=engine)


@pytest.fixture()
def app(db_session):
    """Create a FastAPI app with test dependency overrides."""
    get_settings.cache_clear()

    from app import database
    database._engine = None

    application = create_app()

    application.dependency_overrides[get_settings] = get_test_settings
    application.dependency_overrides[get_db] = lambda: db_session

    yield application

    application.dependency_overrides.clear()


@pytest.fixture()
def client(app):
    """Provide a TestClient for HTTP-level tests."""
    with TestClient(app) as c:
        yield c
