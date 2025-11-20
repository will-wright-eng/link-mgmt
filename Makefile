#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@echo ${MAKEFILE_LIST}
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# api commands
api: ## Run the API
	cd link-api && uv run uvicorn app.main:app --reload

# cli commands
build: ## [cli] Build the CLI (release)
	cd lnk-cli && cargo build --release

check: ## [cli] Check the CLI code
	cd lnk-cli && cargo check

test: ## [cli] Run CLI tests
	cd lnk-cli && cargo test

add: ## [cli] Add new link
	cd lnk-cli && cargo run save https://example.com

links: ## [cli] Get list of all links
	cd lnk-cli && cargo run list

install: ## [cli] Install the CLI to ~/.cargo/bin
	cd lnk-cli && cargo install --path .

fmt: ## [cli] Format Rust code
	cd lnk-cli && cargo fmt --all

fmt-check: ## [cli] Check Rust code formatting
	cd lnk-cli && cargo fmt --all -- --check

clippy: ## [cli] Run clippy linter
	cd lnk-cli && cargo clippy --all-targets --all-features -- -D warnings

# docker commands
up: ## [api] Start the containers
	docker compose up --build --remove-orphans

down: ## [api] Stop the containers
	docker compose down

purge: ## [api] Purge the containers
	docker compose down -v

logs: ## [api] Follow the logs
	docker compose logs -f

# typecheck commands
typecheck: ## [api] Run type checks
	cd link-api && uv run ty check

typecheck-precommit: ## [api] Run type checks
	cd link-api && uv run ty check

# utils
clean: ## Clean up
	find . -type d -name ".venv" -exec rm -rf {} +
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type f -name "*.pyc" -exec rm -f {} +
	rm -rf .ruff_cache
	cd lnk-cli && cargo clean

open: ## open api swagger ui
	open http://localhost:8000
