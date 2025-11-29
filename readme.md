# Link Management System

A full-stack link management system with automatic content extraction, built with Go, TypeScript/Bun, and PostgreSQL.

## Architecture

```mermaid
graph TB
    subgraph "Host Machine"
        CLI[CLI Application<br/>Go CLI]
    end

    subgraph "Docker Network"
        subgraph "Reverse Proxy"
            Nginx[Nginx<br/>Port 80]
        end

        subgraph "API Services"
            API[Go API Server<br/>Port 8080<br/>api-dev]
        end

        subgraph "Scraper Services"
            Scraper[Bun Scraper<br/>Port 3000<br/>scraper-dev]
        end

        subgraph "Database"
            Postgres[(PostgreSQL<br/>Port 5432)]
        end
    end

    CLI -->|HTTP :80<br/>/api/v1/*| Nginx
    CLI -->|HTTP :80<br/>/scrape| Nginx
    CLI -->|HTTP :80<br/>/scraper/health| Nginx

    Nginx -->|/api/*| API
    Nginx -->|/scrape<br/>/scraper/*| Scraper

    API -->|SQL| Postgres
    Scraper -.->|Extract Content| External[External URLs]

    style CLI fill:#4A90E2,stroke:#2E5C8A,stroke-width:2px,color:#fff
    style Nginx fill:#FF9800,stroke:#E65100,stroke-width:2px,color:#fff
    style API fill:#4CAF50,stroke:#2E7D32,stroke-width:2px,color:#fff
    style Scraper fill:#9C27B0,stroke:#6A1B9A,stroke-width:2px,color:#fff
    style Postgres fill:#336791,stroke:#1A3A52,stroke-width:2px,color:#fff
    style External fill:#E91E63,stroke:#AD1457,stroke-width:2px,color:#fff
```

## Components

### CLI Application (`link-mgmt-go/cmd/cli`)

- **Language**: Go
- **Framework**: Bubble Tea (TUI)
- **Purpose**: Interactive command-line interface for managing links
- **Features**:
    - Add links with automatic content scraping
    - List, view, and delete links
    - Configuration management
    - Form-based link creation with auto-filled content

### API Server (`link-mgmt-go/cmd/api`)

- **Language**: Go
- **Framework**: Standard library HTTP
- **Purpose**: REST API for link management
- **Features**:
    - User authentication (API keys)
    - CRUD operations for links
    - Health checks
    - Database persistence

### Scraper Service (`scraper/`)

- **Language**: TypeScript
- **Runtime**: Bun
- **Purpose**: Extract title and content from URLs
- **Features**:
    - Browser-based content extraction
    - Main content extraction
    - Health monitoring
    - Timeout handling

### Nginx Reverse Proxy (`nginx/`)

- **Purpose**: Single entry point for all services
- **Routing**:
    - `/api/*` → Go API server
    - `/scrape` → Scraper service
    - `/scraper/*` → Scraper service
    - `/health` → API health check
    - `/scraper/health` → Scraper health check

### PostgreSQL Database

- **Purpose**: Persistent storage for users and links
- **Port**: 5432 (exposed to host for migrations)

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local CLI development)
- Bun (optional, for local scraper development)

### Start Services

```bash
# Start all services (dev profile)
docker compose --profile dev up -d

# View logs
docker compose --profile dev logs -f

# Stop services
docker compose --profile dev down
```

### Configure CLI

The CLI configuration is stored at `~/.config/link-mgmt/config.toml` and is auto-created with defaults:

```toml
[cli]
base_url = "http://localhost"  # Nginx reverse proxy
api_key = ""                   # Set your API key
scrape_timeout = 30            # Scraping timeout in seconds

[database]
url = "postgres://link_mgmt_user:link_mgmt_pwd@localhost:5432/link_mgmt_db?sslmode=disable"
```

Run the following command to register a CLI user and start saving links:

```bash
cd link-mgmt-go
go run ./cmd/cli --register "me@$(whoami).com"
```

### Run Database Migrations

```bash
# Using psql
PGPASSWORD=link_mgmt_pwd psql -h localhost -U link_mgmt_user -d link_mgmt_db -f link-mgmt-go/migrations/001_create_users.sql
PGPASSWORD=link_mgmt_pwd psql -h localhost -U link_mgmt_user -d link_mgmt_db -f link-mgmt-go/migrations/002_create_links.sql
```

### Use the CLI

```bash
# Build CLI
cd link-mgmt-go
go build -o bin/cli ./cmd/cli

# View configuration
./bin/cli --config-show

# Set API key (after creating a user via API)
./bin/cli --config-set 'cli.api_key=your-api-key-here'

# Add a link (with automatic scraping)
./bin/cli --add

# List links
./bin/cli --list
```

## Service Endpoints

All services are accessed through nginx on port 80:

- **API**: `http://localhost/api/v1/*`
- **Scraper**: `http://localhost/scrape`
- **Health Checks**:
    - `http://localhost/health` (API)
    - `http://localhost/scraper/health` (Scraper)

## Development

### Local Development

**API Server:**

```bash
cd link-mgmt-go
go run ./cmd/api
```

**Scraper Service:**

```bash
cd scraper
bun run dev
```

**CLI:**

```bash
cd link-mgmt-go
go run ./cmd/cli --add
```

### Hot Reloading (Docker)

The dev profile includes hot reloading:

- **API**: Uses Air for automatic rebuilds on Go file changes
- **Scraper**: Uses Bun's watch mode for TypeScript changes

## Project Structure

```bash
link-mgmt/
├── link-mgmt-go/          # Go API and CLI
│   ├── cmd/
│   │   ├── api/          # API server entry point
│   │   └── cli/          # CLI entry point
│   ├── pkg/
│   │   ├── api/          # API handlers and routing
│   │   ├── cli/          # CLI application
│   │   ├── config/       # Configuration management
│   │   ├── db/           # Database layer
│   │   ├── models/       # Data models
│   │   └── scraper/      # Scraper HTTP client
│   └── migrations/       # Database migrations
├── scraper/              # TypeScript scraper service
│   └── src/
│       ├── server.ts     # HTTP server
│       ├── browser.ts    # Browser automation
│       └── extractor.ts  # Content extraction
├── nginx/                # Nginx configuration
│   └── nginx.conf
├── docker-compose.yml    # Service orchestration
└── docs/                 # Documentation
```

## Documentation

- [CLI Scraper Integration Design](./docs/go-app/cli-scraper-integration-design.md) - Detailed design for scraper integration
- [Go API/CLI Design](./docs/go-app/go-api-cli-design-document.md) - Overall system design
- [Scraper Design](./docs/scraper/bun-scraper-design-document.md) - Scraper service design
