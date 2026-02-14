import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from .config import get_settings
from .database import Base, get_engine

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan: create DB tables on startup."""
    engine = get_engine()
    Base.metadata.create_all(bind=engine)
    settings = get_settings()
    logger.info(
        "Application started (environment=%s, database=%s)",
        settings.ENVIRONMENT,
        settings.DATABASE_URL.split("@")[-1] if "@" in settings.DATABASE_URL else settings.DATABASE_URL,
    )
    yield


def create_app() -> FastAPI:
    application = FastAPI(
        title="flask-rest-api-jwt (FastAPI)",
        version="1.0.0",
        lifespan=lifespan,
    )

    # Routers will be registered here as they are implemented in subsequent tasks.
    # Example:
    #   from .routers import health
    #   application.include_router(health.router)

    return application


app = create_app()
