#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# cli commands
build: ## Build the CLI (release)
	cargo build --release

check: ## Check the CLI code
	cargo check

test: ## Run CLI tests
	cargo test

add: ## Add new link
	cargo run save https://example.com

links: ## Get list of all links
	cargo run list

seed: ## Seed the database with sample links
	./seed-db.sh

install: ## Install the CLI to ~/.cargo/bin
	cargo install --path .

fmt: ## Format Rust code
	cargo fmt --all

fmt-check: ## Check Rust code formatting
	cargo fmt --all -- --check

clippy: ## Run clippy linter
	cargo clippy --all-targets --all-features -- -D warnings

clean: ## Clean build artifacts
	cargo clean
