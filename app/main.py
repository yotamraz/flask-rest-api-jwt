"""FastAPI application factory — replaces Flask's create_app()."""

from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.responses import JSONResponse
from starlette.requests import Request

from .auth import JWTAuthError
from .database import init_db
from .routers import health, users


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup/shutdown lifecycle handler."""
    # On startup: create database tables
    await init_db()
    yield
    # On shutdown: nothing needed for now


def create_app() -> FastAPI:
    """Build and configure the FastAPI application."""
    application = FastAPI(
        title="Flask REST API JWT (FastAPI)",
        description="Modernized REST API with JWT authentication",
        version="1.0.0",
        lifespan=lifespan,
    )

    # Register routers (mirrors Flask blueprint registration)
    application.include_router(health.router)
    application.include_router(users.router)

    # --- Exception handlers ------------------------------------------------
    # Flask-JWT-Extended returns ``{"msg": "..."}`` for all JWT/auth errors.
    # Replicate that contract with a handler for our custom JWTAuthError.
    @application.exception_handler(JWTAuthError)
    async def jwt_auth_error_handler(_request: Request, exc: JWTAuthError):
        return JSONResponse(
            status_code=exc.status_code,
            content={"msg": exc.msg},
        )

    return application


app = create_app()
