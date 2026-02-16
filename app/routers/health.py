from fastapi import APIRouter, Depends
from fastapi.responses import JSONResponse
from sqlalchemy import text
from sqlalchemy.orm import Session

from ..database import get_db

router = APIRouter(prefix="/health", tags=["health"])


@router.get("/")
def health_check(db: Session = Depends(get_db)) -> JSONResponse:
    """Lightweight liveness / readiness probe.

    Returns 200 when the API process is up and the database is reachable.
    Returns 503 if the database connection fails.
    """
    try:
        db.execute(text("SELECT 1"))
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
