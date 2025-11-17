# syntax=docker/dockerfile:1.7-labs
# FROM ghcr.io/astral-sh/uv:python3.14-alpine
FROM python:3.14

COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/
# ENV UV_PROJECT_ENVIRONMENT=/app/.venv \
#     PATH="/app/.venv/bin:$PATH"

WORKDIR /app

# RUN pip install --no-cache-dir uv

COPY pyproject.toml uv.lock README.md ./
RUN uv sync --frozen --no-dev --no-install-project
# RUN uv sync --extra dev

COPY app ./app

EXPOSE 8000
CMD ["uv", "run", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
