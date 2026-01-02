#* Setup
.PHONY: $(shell sed -n -e '/^$$/ { n ; /^[^ .\#][^ ]*:/ { s/:.*$$// ; p ; } ; }' $(MAKEFILE_LIST))
.DEFAULT_GOAL := help

help: ## list make commands
	@echo "Root commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Environment Commands:"
	@echo "  [dev]  - Development environment (docker-compose.yml)"
	@echo "  [prod] - Production environment (docker-compose.prod.yml)"
	@echo "  [db]   - Database operations"
	@echo ""
	@echo "Scraper commands (scraper/):"
	@$(MAKE) -C scraper help 2>/dev/null | grep -v "^make" || true
	@echo ""
	@echo "Go commands (link-mgmt-go/):"
	@$(MAKE) -C link-mgmt-go help 2>/dev/null | grep -v "^make" || true

# utils
clean: ## Clean up
	find . -type d -name ".venv" -exec rm -rf {} +
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type f -name "*.pyc" -exec rm -f {} +
	find . -type d -name ".ruff_cache" -exec rm -rf {} +
	find . -type d -name "node_modules" -exec rm -rf {} +
	find . -type d -name "bin" -exec rm -rf {} +
	find . -type f -name "*.test" -exec rm -f {} +
	find . -type f -name "*.out" -exec rm -f {} +

# docker commands - development
up: ## [dev] Start development containers (with hot reloading)
	docker compose up --build --remove-orphans

upd: ## [dev] Start development containers in detached mode
	docker compose up -d --build

down: ## [dev] Stop development containers
	docker compose down --remove-orphans

logs: ## [dev] Follow development container logs
	docker compose logs -f

# docker commands - production
prod-up: ## [prod] Start production containers
	docker compose -f docker-compose.prod.yml up --build --remove-orphans

prod-upd: ## [prod] Start production containers in detached mode
	docker compose -f docker-compose.prod.yml up -d --build

prod-down: ## [prod] Stop production containers
	docker compose -f docker-compose.prod.yml down --remove-orphans

prod-logs: ## [prod] Follow production container logs
	docker compose -f docker-compose.prod.yml logs -f

postgres-up: ## [db] Start only PostgreSQL database
	docker compose up -d postgres

postgres-down: ## [db] Stop PostgreSQL database
	docker compose stop postgres

migrate: ## [db] Run database migrations via Docker
	@echo "Running migrations via Docker..."
	@docker compose exec -T postgres psql -U link_mgmt_user -d link_mgmt_db < link-mgmt-go/migrations/001_create_users.sql
	@docker compose exec -T postgres psql -U link_mgmt_user -d link_mgmt_db < link-mgmt-go/migrations/002_create_links.sql
	@echo "✓ Migrations completed"

# Go delegation
go-build-api go-build-cli go-build-all go-run-api go-run-cli go-test go-deps:
	$(MAKE) -C link-mgmt-go $@

# Scraper delegation
scraper scraper-check scraper-fmt scraper-fmt-check:
	$(MAKE) -C scraper $@
