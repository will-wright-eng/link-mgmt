from datetime import datetime
from typing import Optional

from pydantic import BaseModel, HttpUrl


class LinkBase(BaseModel):
    url: HttpUrl
    title: Optional[str] = None
    description: Optional[str] = None


class LinkCreate(LinkBase):
    pass


class LinkRead(LinkBase):
    id: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True
