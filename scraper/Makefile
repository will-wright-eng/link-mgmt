#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

# Variables
IMAGE_NAME := link-mgmt-scraper-service
CONTAINER_NAME := link-mgmt-scraper
PORT := 3000

help: ## list make commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
server: ## Run scraper service locally
	bun start

dev-server: ## Run scraper service locally with hot reloading
	bun run dev

check: ## Type check scraper code
	bun check

lint: ## Lint scraper code
	bun lint

lint-fix: ## Lint and auto-fix scraper code
	bun lint:fix

fmt: ## Format scraper code
	bun fmt

fmt-check: ## Check code formatting
	bun fmt:check

dev: check lint fmt-check ## Run all development checks (type check, lint, format check)

# Testing commands
test-health: ## Test health endpoint locally
	@curl -f http://localhost:$(PORT)/health | jq . || curl -f http://localhost:$(PORT)/health

test-scrape: ## Test scrape endpoint locally (requires URL env var, e.g., URL=https://example.com)
	@curl -X POST http://localhost:$(PORT)/scrape \
		-H "Content-Type: application/json" \
		-d "{\"url\": \"$${URL:-https://example.com}\"}" | jq . || \
	curl -X POST http://localhost:$(PORT)/scrape \
		-H "Content-Type: application/json" \
		-d "{\"url\": \"$${URL:-https://example.com}\"}"

# Cleanup commands
clean: docker-clean ## Clean up Docker containers and images
	docker rmi $(IMAGE_NAME):latest || true

clean-all: clean ## Clean up everything including Docker images
	docker system prune -f
