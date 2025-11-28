#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# scraper commands
server: ## Run scraper service
	bun run src/server.ts

check: ## Type check scraper code
	bun run --bun tsc --noEmit

fmt: ## Format scraper code
	bunx prettier --write "src/**/*.ts"

fmt-check: ## Check scraper code formatting
	bunx prettier --check "src/**/*.ts"
