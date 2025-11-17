from functools import lru_cache
from pydantic import BaseSettings, AnyHttpUrl


class Settings(BaseSettings):
    api_base_url: AnyHttpUrl = "https://api.links.yourdomain.com"
    database_url: str = (
        "postgresql+asyncpg://link_mgmt:link_mgmt@postgres:5432/link_mgmt"
    )
    app_name: str = "Link API"

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


@lru_cache()
def get_settings() -> Settings:
    return Settings()


settings = get_settings()
