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
	uv run alembic upgrade head

migrate-down: ## Rollback one migration
	uv run alembic downgrade -1

migration: ## Create a new migration (usage: make migration MESSAGE="description")
	uv run alembic revision --autogenerate -m "$(MESSAGE)"

migrate-history: ## Show migration history
	uv run alembic history

# typecheck commands
typecheck: ## Run type checks
	uv run ty check
