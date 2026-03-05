from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    DEBUG: bool = False
    SECRET_KEY: str = "your-secret-key-change-me"

    # Database URI — accepts SQLALCHEMY_DATABASE_URI or DATABASE_URL env var.
    # Must use an async driver (e.g. sqlite+aiosqlite, postgresql+asyncpg).
    SQLALCHEMY_DATABASE_URI: str = "sqlite+aiosqlite:///./data-dev.sqlite"

    # JWT settings (matching Flask-JWT-Extended defaults from the source)
    ACCESS_TOKEN_EXPIRES_MINUTES: int = 15
    REFRESH_TOKEN_EXPIRES_DAYS: int = 30


settings = Settings()
