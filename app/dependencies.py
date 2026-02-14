"""Shared FastAPI dependencies.

Re-exports core dependencies for convenient use across routers.
Auth dependencies will be added in Milestone 2.
"""

from .config import get_settings
from .database import get_db

__all__ = ["get_settings", "get_db"]
