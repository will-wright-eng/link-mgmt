# link-mgmt

Prototype link management stack:

- `link-api/`: FastAPI-based backend managed with [uv](https://github.com/astral-sh/uv)
- `docs/`: Design documents and planning notes

## Prerequisites

- Python 3.11+
- [uv](https://github.com/astral-sh/uv)
- Docker & Docker Compose for running the stack locally

## Local API development

```bash
cd link-api
uv sync --extra dev
uv run uvicorn app.main:app --reload
```

## Docker Compose

```bash
docker compose up --build
```

This starts PostgreSQL and the API container, which uses `uv sync --frozen` inside the image to install dependencies.
