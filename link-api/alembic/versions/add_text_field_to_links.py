"""add text field to links

Revision ID: add_text_to_links
Revises: e7bd135ea8ca
Create Date: 2025-01-20 12:00:00.000000

"""

from typing import Sequence, Union

from alembic import op  # type: ignore[attr-defined]
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = "a1b2c3d4e5f6"
down_revision: Union[str, Sequence[str], None] = "e7bd135ea8ca"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Upgrade schema."""
    op.add_column("links", sa.Column("text", sa.String(), nullable=True))


def downgrade() -> None:
    """Downgrade schema."""
    op.drop_column("links", "text")
