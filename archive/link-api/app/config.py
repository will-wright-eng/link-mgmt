from functools import lru_cache
from typing import cast

from pydantic import AnyHttpUrl, Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables or .env file."""

    api_base_url: AnyHttpUrl = Field(
        default=cast(AnyHttpUrl, "https://api.links.yourdomain.com"),
        description="Base URL for the API",
    )
    database_url: str = Field(
        default="postgresql+asyncpg://link_mgmt:link_mgmt@postgres:5432/link_mgmt",
        description="PostgreSQL database connection URL",
    )
    app_name: str = Field(
        default="Link API",
        description="Application name",
    )

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,  # Allow both DATABASE_URL and database_url
        extra="ignore",  # Ignore extra environment variables
    )


@lru_cache()
def get_settings() -> Settings:
    return Settings()


settings = get_settings()
