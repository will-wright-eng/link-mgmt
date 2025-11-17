class MetadataService:
    """Placeholder service for extracting link metadata."""

    async def extract(self, url: str) -> dict[str, str | None]:
        # In a full implementation this would fetch and parse Open Graph tags.
        return {"title": None, "description": None, "url": url}
