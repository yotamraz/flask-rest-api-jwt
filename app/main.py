"""FastAPI application factory and entrypoint.

Replaces Flask's create_app() + app.run() with a FastAPI app factory
served by Uvicorn.
"""

from contextlib import asynccontextmanager

from fastapi import FastAPI

from .database import engine
from .models import Base
from .routers import health


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup/shutdown lifecycle. Creates all tables on startup."""
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield


def create_app() -> FastAPI:
    """Application factory — creates and configures the FastAPI instance."""
    app = FastAPI(title="Flask-REST-API-JWT (FastAPI)", lifespan=lifespan)

    app.include_router(health.router)

    return app


app = create_app()
