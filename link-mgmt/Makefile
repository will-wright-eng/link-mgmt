#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

bin: ## Ensure bin directory exists
	mkdir -p bin

build-api: bin ## Build the API server
	go build -o bin/api ./cmd/api

build-cli: bin ## Build the CLI
	go build -o bin/cli ./cmd/cli

build-all: build-api build-cli ## Build both API and CLI

run-api: ## Run the API server
	go run ./cmd/api

run-cli: ## Run the CLI
	go run ./cmd/cli

deps: ## Install dependencies
	go mod download
	go mod tidy

seed: ## Seed the database with links
	bash add-links.sh links.txt
