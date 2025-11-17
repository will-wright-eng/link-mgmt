# Link API

Minimal FastAPI backend for the link organizer system.

## Prerequisites

- [uv](https://github.com/astral-sh/uv) for dependency management
- Python 3.11+

## Setup

```bash
uv sync --extra dev
```

## Local development

```bash
uv run uvicorn app.main:app --reload
```

## Dependency updates

```bash
uv add <package>
uv remove <package>
uv lock
```
