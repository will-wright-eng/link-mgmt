from typing import Annotated
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import get_current_user, get_session
from app.models import Link, User
from app.schemas import LinkCreate, LinkRead, LinkUpdate

router = APIRouter(prefix="/links", tags=["links"])


@router.get("", response_model=list[LinkRead])
async def list_links(
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_session)],
) -> list[LinkRead]:
    """List all links for the current user."""
    result = await db.execute(
        select(Link)
        .where(Link.user_id == current_user.id)
        .order_by(Link.created_at.desc())
    )
    links = result.scalars().all()
    return [LinkRead.model_validate(link) for link in links]


@router.post("", response_model=LinkRead, status_code=status.HTTP_201_CREATED)
async def create_link(
    payload: LinkCreate,
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_session)],
) -> LinkRead:
    """Create a new link for the current user."""
    link = Link(
        user_id=current_user.id,
        url=str(payload.url),
        title=payload.title,
        description=payload.description,
        text=payload.text,
    )
    db.add(link)
    await db.commit()
    await db.refresh(link)
    return LinkRead.model_validate(link)


@router.get("/{link_id}", response_model=LinkRead)
async def get_link(
    link_id: UUID,
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_session)],
) -> LinkRead:
    """Get a specific link by ID (must belong to current user)."""
    result = await db.execute(
        select(Link).where(Link.id == link_id, Link.user_id == current_user.id)
    )
    link = result.scalar_one_or_none()
    if not link:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND, detail="Link not found"
        )
    return LinkRead.model_validate(link)


@router.patch("/{link_id}", response_model=LinkRead)
async def update_link(
    link_id: UUID,
    payload: LinkUpdate,
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_session)],
) -> LinkRead:
    """Update a link's title and/or description (must belong to current user)."""
    result = await db.execute(
        select(Link).where(Link.id == link_id, Link.user_id == current_user.id)
    )
    link = result.scalar_one_or_none()
    if not link:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND, detail="Link not found"
        )

    # Update only provided fields
    if payload.title is not None:
        link.title = payload.title
    if payload.description is not None:
        link.description = payload.description
    if payload.text is not None:
        link.text = payload.text

    await db.commit()
    await db.refresh(link)
    return LinkRead.model_validate(link)
