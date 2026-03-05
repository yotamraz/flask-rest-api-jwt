"""FastAPI application factory — replaces Flask's ``create_app()``."""
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse

from .database import init_db


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup: create DB tables.  Shutdown: nothing special."""
    await init_db()
    yield


def create_app() -> FastAPI:
    """Build and return the FastAPI application."""
    application = FastAPI(
        title="Flask REST API JWT — FastAPI Edition",
        lifespan=lifespan,
    )

    # ------------------------------------------------------------------
    # Custom exception handler so error responses use ``{"message": ...}``
    # instead of FastAPI's default ``{"detail": ...}`` — preserving the
    # Flask API contract.
    # ------------------------------------------------------------------
    @application.exception_handler(HTTPException)
    async def http_exception_handler(request: Request, exc: HTTPException):
        return JSONResponse(
            status_code=exc.status_code,
            content={"message": exc.detail},
        )

    # ------------------------------------------------------------------
    # Register routers (mirrors Flask blueprint registration)
    # ------------------------------------------------------------------
    from .routers import health, users  # noqa: E402

    application.include_router(health.router)
    application.include_router(users.router)

    return application


app = create_app()
