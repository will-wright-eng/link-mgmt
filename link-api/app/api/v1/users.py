from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import get_current_user, get_session
from app.models.user import User
from app.schemas.user import UserCreate, UserRead, UserWithApiKey
from app.utils.security import generate_api_key

router = APIRouter(prefix="/users", tags=["users"])


@router.post("", response_model=UserWithApiKey, status_code=status.HTTP_201_CREATED)
async def create_user(
    payload: UserCreate,
    db: AsyncSession = Depends(get_session),
) -> UserWithApiKey:
    """Create a new user and return with API key."""
    # Check if email already exists
    result = await db.execute(select(User).where(User.email == payload.email))
    existing_user = result.scalar_one_or_none()
    if existing_user:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Email already registered",
        )

    # Generate API key
    api_key = generate_api_key()

    # Create user
    user = User(email=payload.email, api_key=api_key)
    db.add(user)
    await db.commit()
    await db.refresh(user)

    return UserWithApiKey.model_validate(user)


@router.get("/me", response_model=UserRead)
async def get_me(
    current_user: User = Depends(get_current_user),
) -> UserRead:
    """Get the current authenticated user."""
    return UserRead.model_validate(current_user)
