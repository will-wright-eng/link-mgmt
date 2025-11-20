from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import get_session
from app.models import Link
from app.schemas import LinkCreate, LinkRead

router = APIRouter(prefix="/links", tags=["links"])


@router.get("", response_model=list[LinkRead])
async def list_links(
    db: Annotated[AsyncSession, Depends(get_session)],
) -> list[LinkRead]:
    result = await db.execute(select(Link).order_by(Link.created_at.desc()))
    links = result.scalars().all()
    return [LinkRead.model_validate(link) for link in links]


@router.post("", response_model=LinkRead, status_code=status.HTTP_201_CREATED)
async def create_link(
    payload: LinkCreate,
    db: Annotated[AsyncSession, Depends(get_session)],
) -> LinkRead:
    link = Link(
        url=str(payload.url), title=payload.title, description=payload.description
    )
    db.add(link)
    await db.commit()
    await db.refresh(link)
    return LinkRead.model_validate(link)


@router.get("/{link_id}", response_model=LinkRead)
async def get_link(
    link_id: int, db: Annotated[AsyncSession, Depends(get_session)]
) -> LinkRead:
    link = await db.get(Link, link_id)
    if not link:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND, detail="Link not found"
        )
    return LinkRead.model_validate(link)
