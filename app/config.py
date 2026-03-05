"""Application configuration via pydantic-settings.

Reads from environment variables and an optional ``.env`` file located in the
project root.  All Flask-era configuration keys are preserved so that Docker
and docker-compose deployments continue to work without changes.
"""

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # General
    DEBUG: bool = False
    SECRET_KEY: str = "your-secret-key-change-me"

    # Database – default to async SQLite for local development.
    # Production should set this to a postgresql+asyncpg:// URI.
    SQLALCHEMY_DATABASE_URI: str = "sqlite+aiosqlite:///./data-dev.sqlite"

    # JWT token lifetimes (matching Flask-JWT-Extended defaults from source)
    ACCESS_TOKEN_EXPIRES_MINUTES: int = 15
    REFRESH_TOKEN_EXPIRES_DAYS: int = 30


settings = Settings()
