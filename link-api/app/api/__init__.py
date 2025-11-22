from fastapi import APIRouter

from app.api.v1 import links, users

api_router = APIRouter(prefix="/api")
api_router.include_router(links.router)
api_router.include_router(users.router)

__all__ = ["api_router"]
