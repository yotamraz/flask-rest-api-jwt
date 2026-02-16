"""Smoke tests for core infrastructure: configuration, database, and package structure."""

from sqlalchemy.orm import DeclarativeBase, Session


def test_settings_reads_env_vars():
    """Settings class reads required values from environment variables."""
    from app.config import Settings

    settings = Settings(
        SECRET_KEY="test-secret",
        DATABASE_URL="sqlite:///./test_infra.db",
    )
    assert settings.SECRET_KEY == "test-secret"
    assert settings.DATABASE_URL == "sqlite:///./test_infra.db"


def test_settings_defaults():
    """Settings class provides sensible defaults for optional fields."""
    from app.config import Settings

    settings = Settings(
        SECRET_KEY="test-secret",
        DATABASE_URL="sqlite:///./test_infra.db",
    )
    assert settings.ENVIRONMENT == "development"
    assert settings.ACCESS_TOKEN_EXPIRES_MINUTES == 15
    assert settings.REFRESH_TOKEN_EXPIRES_DAYS == 30


def test_get_settings_returns_settings_instance():
    """get_settings() returns a Settings instance with values from env."""
    from app.config import get_settings

    get_settings.cache_clear()
    settings = get_settings()
    assert settings.SECRET_KEY is not None
    assert settings.DATABASE_URL is not None


def test_get_settings_is_cached():
    """get_settings() returns the same cached instance on repeated calls."""
    from app.config import get_settings

    get_settings.cache_clear()
    s1 = get_settings()
    s2 = get_settings()
    assert s1 is s2


def test_base_is_declarative_base():
    """Base is a SQLAlchemy 2.x DeclarativeBase subclass."""
    from app.database import Base

    assert issubclass(Base, DeclarativeBase)


def test_get_engine_creates_engine():
    """get_engine() returns a SQLAlchemy engine with the configured URL."""
    from app.database import get_engine

    engine = get_engine()
    assert engine is not None
    assert str(engine.url) != ""


def test_session_local_creates_sessions():
    """SessionLocal produces SQLAlchemy Session instances."""
    from app.database import SessionLocal

    session = SessionLocal()
    try:
        assert isinstance(session, Session)
    finally:
        session.close()


def test_get_db_yields_session():
    """get_db() generator yields a session and closes it."""
    from app.database import get_db

    gen = get_db()
    session = next(gen)
    assert isinstance(session, Session)
    try:
        next(gen)
    except StopIteration:
        pass


def test_app_package_importable():
    """app package can be imported without error."""
    import app
    assert app is not None


def test_routers_package_importable():
    """app.routers package can be imported without error."""
    import app.routers
    assert app.routers is not None
