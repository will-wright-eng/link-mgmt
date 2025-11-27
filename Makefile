#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@echo "Root commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo "Scraper commands (scraper/):"
	@$(MAKE) -C scraper help 2>/dev/null | grep -v "^make" || true

# utils
clean: ## Clean up
	find . -type d -name ".venv" -exec rm -rf {} +
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type f -name "*.pyc" -exec rm -f {} +
	find . -type d -name ".ruff_cache" -exec rm -rf {} +
	find . -type d -name "node_modules" -exec rm -rf {} +

# Scraper delegation
scraper scraper-check scraper-fmt scraper-fmt-check:
	$(MAKE) -C scraper $@
