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

2. **Configure database:**
   - Edit `~/.config/link-mgmt/config.toml` (auto-created on first run)
   - Set your PostgreSQL connection string

3. **Run migrations:**

   ```bash
   # Using psql or your preferred migration tool
   psql -d linkmgmt -f migrations/001_create_users.sql
   psql -d linkmgmt -f migrations/002_create_links.sql
   ```

## Running

### API Server

```bash
go run ./cmd/api
```

The API will start on `http://localhost:8080` (configurable in config.toml)

### CLI

```bash
go run ./cmd/cli
```

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
url = "postgres://localhost/linkmgmt?sslmode=disable"

[api]
host = "0.0.0.0"
port = 8080

[cli]
api_base_url = "http://localhost:8080"
api_key = ""
```
