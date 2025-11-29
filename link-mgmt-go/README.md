# Link Management System - Go Implementation

A link management system with REST API and CLI, built with Go.

## Project Structure

```
link-mgmt-go/
├── cmd/
│   ├── api/          # API server entry point
│   └── cli/          # CLI entry point
├── pkg/
│   ├── models/       # Shared data models
│   ├── db/           # Database layer (pgx)
│   ├── config/       # Configuration management (TOML)
│   ├── api/          # API handlers and routing
│   └── cli/          # CLI application
├── migrations/       # Database migrations
└── internal/         # Internal packages
```

## Setup

1. **Install dependencies:**

   ```bash
   go mod download
   ```

2. **Start PostgreSQL (using docker-compose):**

   ```bash
   # From the project root
   docker compose up -d
   ```

   This starts PostgreSQL with:
   - Database: `link_mgmt`
   - User: `link_mgmt`
   - Password: `link_mgmt`
   - Port: `5432`

3. **Configure database connection:**

   The config file is auto-created at `~/.config/link-mgmt/config.toml` with defaults matching docker-compose.

   You can view/update config using the CLI:

   ```bash
   # View current config
   ./bin/cli --config-show

   # Update database URL
   ./bin/cli --config-set 'database.url=postgres://link_mgmt:link_mgmt@localhost:5432/link_mgmt?sslmode=disable'
   ```

4. **Run migrations:**

   ```bash
   # Using psql with docker-compose credentials
   PGPASSWORD=link_mgmt psql -h localhost -U link_mgmt -d link_mgmt -f migrations/001_create_users.sql
   PGPASSWORD=link_mgmt psql -h localhost -U link_mgmt -d link_mgmt -f migrations/002_create_links.sql
   ```

## Running

### API Server

**Local development:**

```bash
go run ./cmd/api
```

**Docker with hot reloading:**

```bash
# From the project root
docker compose --profile dev up api-dev
```

The API will start on `http://localhost:8080` (configurable in config.toml). With Docker, changes to Go files will automatically trigger a rebuild and restart.

### CLI

```bash
go run ./cmd/cli
```

**CLI Commands:**

- `--config-show` - Show current configuration (no database connection required)
- `--config-set <section.key=value>` - Set a config value (no database connection required)
- `--list` - List all links (requires database)
- `--add` - Add a new link (requires database)
- `--delete` - Delete a link (requires database)

## API Endpoints

- `GET /health` - Health check
- `POST /api/v1/users` - Create user
- `GET /api/v1/users/me` - Get current user (requires auth)
- `GET /api/v1/links` - List links (requires auth)
- `POST /api/v1/links` - Create link (requires auth)
- `GET /api/v1/links/:id` - Get link (requires auth)
- `DELETE /api/v1/links/:id` - Delete link (requires auth)

## Authentication

Use API key authentication via `Authorization` header:

```
Authorization: Bearer <api_key>
```

## Configuration

Configuration is stored in `~/.config/link-mgmt/config.toml`:

```toml
[database]
url = "postgres://link_mgmt:link_mgmt@localhost:5432/link_mgmt?sslmode=disable"

[api]
host = "0.0.0.0"
port = 8080

[cli]
base_url = "http://localhost"
api_key = ""
scrape_timeout = 30
```

**Note:** The default config matches the docker-compose.yml PostgreSQL settings. You can manage the config file using the CLI commands above without requiring a database connection.
