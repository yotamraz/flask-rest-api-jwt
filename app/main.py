"""FastAPI application factory and entrypoint.

Replaces Flask's create_app() + app.run() with a FastAPI app factory
served by Uvicorn.
"""

from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from .auth import MissingAuthError
from .database import engine
from .models import Base
from .routers import health, users


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup/shutdown lifecycle. Creates all tables on startup."""
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield


def create_app() -> FastAPI:
    """Application factory — creates and configures the FastAPI instance."""
    app = FastAPI(title="Flask-REST-API-JWT (FastAPI)", lifespan=lifespan)

    # Exception handler for missing auth (matches Flask-JWT-Extended {"msg": ...} format)
    @app.exception_handler(MissingAuthError)
    async def missing_auth_handler(request: Request, exc: MissingAuthError):
        return JSONResponse(
            status_code=401,
            content={"msg": exc.msg},
        )

    app.include_router(health.router)
    app.include_router(users.router)

    return app


app = create_app()
