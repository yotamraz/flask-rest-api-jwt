from fastapi import FastAPI

from .config import get_settings
from .database import Base, get_engine
from .routers import health


def create_app() -> FastAPI:
    settings = get_settings()

    app = FastAPI(
        title="flask-rest-api-jwt (FastAPI)",
        version="1.0.0",
    )

    @app.on_event("startup")
    def on_startup() -> None:
        engine = get_engine()
        Base.metadata.create_all(bind=engine)

    app.include_router(health.router)

    return app


app = create_app()
