"""FastAPI application factory.

Replaces ``app/__init__.py`` (Flask create_app) with a FastAPI equivalent.
The ``create_app()`` function wires routers, startup events, and middleware.
The module-level ``app`` instance is the Uvicorn entrypoint
(``uvicorn app.main:app``).
"""

from contextlib import asynccontextmanager

from fastapi import FastAPI

from .database import Base, engine
from .routers import health


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Create database tables on startup."""
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield


def create_app() -> FastAPI:
    """Build and return the FastAPI application instance."""
    application = FastAPI(
        title="Flask REST API JWT (FastAPI)",
        description="Modernized REST API with JWT authentication",
        version="1.0.0",
        lifespan=lifespan,
    )

    # Register routers (health only in milestone 1; users, stores, items,
    # tags will be added in subsequent milestones)
    application.include_router(health.router)

    return application


app = create_app()
