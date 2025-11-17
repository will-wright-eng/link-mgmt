from fastapi import FastAPI

from app.api import api_router
from app.config import settings
from app.database import init_db

app = FastAPI(title=settings.app_name)
app.include_router(api_router)


@app.on_event("startup")
async def on_startup() -> None:
    await init_db()


@app.get("/health", tags=["system"])
async def healthcheck() -> dict[str, str]:
    return {"status": "ok", "app": settings.app_name}
