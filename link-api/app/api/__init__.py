from fastapi import APIRouter

from app.api.v1 import links

api_router = APIRouter(prefix="/api")
api_router.include_router(links.router)

__all__ = ["api_router"]
