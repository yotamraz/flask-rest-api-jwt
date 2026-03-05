from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables and .env file."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
    )

    DEBUG: bool = False
    SECRET_KEY: str = "your-secret-key-change-me"

    # Database URI (async drivers: sqlite+aiosqlite or postgresql+asyncpg)
    SQLALCHEMY_DATABASE_URI: str = "sqlite+aiosqlite:///./data-dev.sqlite"

    # JWT settings
    ACCESS_TOKEN_EXPIRES_MINUTES: int = 15
    REFRESH_TOKEN_EXPIRES_DAYS: int = 30


settings = Settings()
