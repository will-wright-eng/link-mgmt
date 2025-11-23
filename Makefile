#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@echo "Root commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "API commands (link-api/):"
	@$(MAKE) -C link-api help 2>/dev/null | grep -v "^make" || true
	@echo ""
	@echo "CLI commands (lnk-cli/):"
	@$(MAKE) -C lnk-cli help 2>/dev/null | grep -v "^make" || true
	@echo ""
	@echo "Scraper commands (scraper/):"
	@$(MAKE) -C scraper help 2>/dev/null | grep -v "^make" || true

# docker commands
up: ## [api] Start the containers
	docker compose up --build --remove-orphans

down: ## [api] Stop the containers
	docker compose down

purge: ## [api] Purge the containers
	docker compose down -v

logs: ## [api] Follow the logs
	docker compose logs -f

# utils
clean: ## Clean up
	find . -type d -name ".venv" -exec rm -rf {} +
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type f -name "*.pyc" -exec rm -f {} +
	rm -rf .ruff_cache
	$(MAKE) -C lnk-cli clean 2>/dev/null || cd lnk-cli && cargo clean

open: ## open api swagger ui
	open http://localhost:8000

# API delegation
api migrate migrate-down migration migrate-history typecheck typecheck-precommit:
	$(MAKE) -C link-api $@

# CLI delegation
build check test add links install fmt fmt-check clippy:
	$(MAKE) -C lnk-cli $@

# Scraper delegation
scraper scraper-check scraper-fmt scraper-fmt-check:
	$(MAKE) -C scraper $@
