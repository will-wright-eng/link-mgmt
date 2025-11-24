from datetime import datetime
from typing import Optional
from uuid import UUID

from pydantic import BaseModel, ConfigDict, HttpUrl


class LinkBase(BaseModel):
    url: HttpUrl
    title: Optional[str] = None
    description: Optional[str] = None
    text: Optional[str] = None


class LinkCreate(LinkBase):
    pass


class LinkUpdate(BaseModel):
    title: Optional[str] = None
    description: Optional[str] = None
    text: Optional[str] = None


class LinkRead(LinkBase):
    id: UUID
    user_id: UUID
    created_at: datetime
    updated_at: datetime

    model_config = ConfigDict(from_attributes=True)
