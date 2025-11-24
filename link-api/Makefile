#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# api commands
api: ## Run the API
	uv run uvicorn app.main:app --reload

# migration commands
migrate: ## Run all pending migrations
	uv sync --extra dev
	uv run alembic upgrade head

migrate-down: ## Rollback one migration
	uv sync --extra dev
	uv run alembic downgrade -1

migration: ## Create a new migration (usage: make migration MESSAGE="description")
	uv sync --extra dev
	uv run alembic revision --autogenerate -m "$(MESSAGE)"

migrate-history: ## Show migration history
	uv sync --extra dev
	uv run alembic history

init-db: ## Initialize database after postgres volume reset (runs all migrations)
	@echo "Initializing database..."
	uv sync --extra dev
	uv run alembic upgrade head
	@echo "✓ Database initialized successfully!"

# typecheck commands
typecheck: ## Run type checks
	uv sync --extra dev
	uv run ty check
