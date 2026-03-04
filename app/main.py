"""FastAPI application factory and entrypoint.

Replaces Flask's create_app() + app.run() with a FastAPI app factory
served by Uvicorn.
"""

import os
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from sqlalchemy import select

from .auth import MissingAuthError
from .database import SessionLocal, engine
from .models import Base, User
from .routers import health, users


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup/shutdown lifecycle. Recreates all tables and seeds test user."""
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
        await conn.run_sync(Base.metadata.create_all)

    # Seed auth_test_user if TEST_USER_PASSWORD is provided
    test_pw = os.environ.get("TEST_USER_PASSWORD")
    if test_pw:
        async with SessionLocal() as session:
            result = await session.execute(
                select(User).where(User.username == "auth_test_user")
            )
            if result.scalar_one_or_none() is None:
                user = User(username="auth_test_user")
                user.set_password(test_pw)
                session.add(user)
                await session.commit()

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
