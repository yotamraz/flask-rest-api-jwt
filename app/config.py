from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8")

    ENVIRONMENT: str = "development"  # "development" | "production" | "test"

    SECRET_KEY: str

    # Database
    DATABASE_URL: str = "sqlite:///./dev.db"

    # JWT settings
    ACCESS_TOKEN_EXPIRES_MINUTES: int = 15
    REFRESH_TOKEN_EXPIRES_DAYS: int = 30


@lru_cache
def get_settings() -> Settings:
    return Settings()
