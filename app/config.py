from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application configuration loaded from environment variables and .env file."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    DEBUG: bool = False
    SECRET_KEY: str = "your-secret-key-change-me"

    # Database URI — must use an async driver (aiosqlite for SQLite, asyncpg for Postgres)
    SQLALCHEMY_DATABASE_URI: str = "sqlite+aiosqlite:///./data-dev.sqlite"

    # JWT token expiration settings
    ACCESS_TOKEN_EXPIRES_MINUTES: int = 15
    REFRESH_TOKEN_EXPIRES_DAYS: int = 30


def get_settings() -> Settings:
    """Factory that creates a Settings instance. Useful for dependency injection and testing."""
    return Settings()


settings = get_settings()
