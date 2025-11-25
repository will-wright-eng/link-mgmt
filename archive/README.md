# link-mgmt

Prototype link management stack with three services:

- `link-api/`: FastAPI-based backend for managing links, users, and authentication
- `scraper/`: Bun-based web scraper that extracts main content from URLs using Playwright and Mozilla Readability
- `lnk-cli/`: Rust CLI tool for interacting with the link API
- `docs/`: Design documents and planning notes

## Prerequisites

- Python 3.11+
- [uv](https://github.com/astral-sh/uv) for Python dependency management
- [Bun](https://bun.sh) for the scraper service
- [Rust](https://www.rust-lang.org/) and Cargo for the CLI
- Docker & Docker Compose for running the stack locally

## Services

### link-api

FastAPI-based backend that provides a REST API for managing links and users.

#### Setup

```bash
cd link-api
uv sync --extra dev
```

#### Local Development

```bash
# Run the API server
cd link-api
uv run uvicorn app.main:app --reload

# Or use the Makefile
make api
```

#### Database Migrations

```bash
cd link-api

# Run pending migrations
make migrate

# Create a new migration
make migration MESSAGE="your migration message"

# Rollback one migration
make migrate-down

# View migration history
make migrate-history
```

### scraper

Bun-based web scraper that extracts main content from URLs. Supports multiple input sources (command line, files, stdin) and multiple output formats.

#### Setup

```bash
cd scraper
bun install
```

#### Usage

```bash
cd scraper

# Scrape a single URL
make scraper URL="https://example.com"

# Scrape from command line
bun run src/cli.ts https://example.com

# Scrape from file
bun run src/cli.ts --input urls.txt --output results.jsonl

# Scrape from stdin
echo "https://example.com" | bun run src/cli.ts

# Available options:
#   --format: jsonl (default), json, or text
#   --timeout: Request timeout in ms (default: 10000)
#   --headless: Run browser in headless mode (default: true)
```

#### Development

```bash
cd scraper

# Type check
make check

# Format code
make fmt

# Check formatting
make fmt-check
```

### lnk-cli

Rust command-line interface for interacting with the link management API.

#### Setup

```bash
cd lnk-cli
cargo build --release
```

The binary will be available at `target/release/lnk`.

#### Installation

```bash
cd lnk-cli
make install  # Installs to ~/.cargo/bin
```

#### Configuration

Set the API URL (defaults to `http://localhost:8000`):

```bash
lnk config set api.url http://localhost:8000
```

Or use the `--api-url` flag or `LNK_API_URL` environment variable:

```bash
lnk --api-url http://localhost:8000 save https://example.com
```

#### Usage

```bash
# Save a link
lnk save https://example.com
lnk save https://example.com --title "Example Site"
lnk save https://example.com --title "Example" --description "An example website"

# List links
lnk list
lnk list --limit 10

# Get a specific link
lnk get <link-id>
```

#### Development

```bash
cd lnk-cli

# Build release binary
make build

# Run in development mode
cargo run -- save https://example.com

# Type check
make check

# Run tests
make test

# Format code
make fmt

# Run clippy
make clippy
```

## Docker Compose

```bash
docker compose up --build
```

This starts PostgreSQL and the API container, which uses `uv sync --frozen` inside the image to install dependencies.
