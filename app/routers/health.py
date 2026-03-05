"""Health check router.

Migrated from ``app/resources/health.py``.  Provides a lightweight
liveness / readiness probe that verifies database connectivity.
"""

from fastapi import APIRouter, Depends
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

from ..database import get_db

router = APIRouter(prefix="/health", tags=["health"])


@router.get("/")
async def health_check(db: AsyncSession = Depends(get_db)) -> dict:
    """Lightweight liveness / readiness probe.

    Returns 200 when the API process is up **and** the database is reachable.
    Returns 503 if the database connection fails.
    """
    from fastapi.responses import JSONResponse

    try:
        await db.execute(text("SELECT 1"))
        db_status = "healthy"
        status_code = 200
    except Exception:
        db_status = "unhealthy"
        status_code = 503

    payload = {
        "status": "healthy" if status_code == 200 else "unhealthy",
        "database": db_status,
    }
    return JSONResponse(content=payload, status_code=status_code)
