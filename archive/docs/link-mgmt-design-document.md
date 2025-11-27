# Link Organizer System - Design Document

## Executive Summary

A distributed link organization system that allows users to save, tag, and search links from multiple sources. The system consists of a FastAPI backend for receiving and storing links, and a Rust CLI for quick link submission from the command line.

## Architecture Overview

```
┌─────────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│                     │     │                  │     │                 │
│   Rust CLI Client   │────▶│   FastAPI        │────▶│   PostgreSQL    │
│                     │     │   Backend        │     │   Database      │
└─────────────────────┘     │                  │     └─────────────────┘
                            │                  │
┌─────────────────────┐     │   /api/links     │
│  Browser Extension  │────▶│   /api/tags      │
└─────────────────────┘     │   /api/search    │
                            │                  │
┌─────────────────────┐     │                  │
│  Mobile Share       │────▶│                  │
│  Extension (Future) │     │                  │
└─────────────────────┘     └──────────────────┘
```

## Core Components

### 1. FastAPI Backend

**Base URL**: `https://api.links.yourdomain.com`

**Technology Stack**:

- FastAPI 0.104+
- SQLAlchemy 2.0 (async)
- PostgreSQL 15+
- Pydantic for validation
- Alembic for migrations

### 2. Rust CLI Client

**Technology Stack**:

- Clap for argument parsing
- Reqwest for HTTP requests
- Tokio for async runtime
- Serde for JSON serialization
- Keyring for credential storage
- Colored for terminal output

## Database Schema

```sql
-- Core tables
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    api_key VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title TEXT,
    description TEXT,
    domain VARCHAR(255),
    favicon_url TEXT,
    screenshot_url TEXT,
    content_hash VARCHAR(64), -- SHA256 of page content
    is_archived BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Full text search
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(url, '')), 'C')
    ) STORED,

    -- Constraints
    CONSTRAINT unique_user_url UNIQUE(user_id, url)
);

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(7), -- hex color
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_user_tag UNIQUE(user_id, name)
);

CREATE TABLE link_tags (
    link_id UUID REFERENCES links(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),

    PRIMARY KEY (link_id, tag_id)
);

CREATE TABLE link_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID REFERENCES links(id) ON DELETE CASCADE,
    og_title TEXT,
    og_description TEXT,
    og_image TEXT,
    og_type VARCHAR(50),
    twitter_card VARCHAR(50),
    author VARCHAR(255),
    published_date TIMESTAMP,
    word_count INTEGER,
    reading_time_minutes INTEGER,
    extracted_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_links_user_id ON links(user_id);
CREATE INDEX idx_links_created_at ON links(created_at DESC);
CREATE INDEX idx_links_domain ON links(domain);
CREATE INDEX idx_links_search ON links USING GIN(search_vector);
CREATE INDEX idx_link_tags_tag_id ON link_tags(tag_id);
```

## API Endpoints

### Authentication

All endpoints require API key authentication via header:

```
X-API-Key: <user_api_key>
```

### Link Management

#### POST /api/links

Create a new link

```json
Request:
{
    "url": "https://example.com/article",
    "title": "Optional custom title",
    "description": "Optional description",
    "tags": ["rust", "programming"],
    "extract_metadata": true
}

Response:
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "url": "https://example.com/article",
    "title": "Article Title",
    "description": "Article description",
    "domain": "example.com",
    "tags": [
        {"id": "...", "name": "rust", "color": "#CE422B"},
        {"id": "...", "name": "programming", "color": null}
    ],
    "created_at": "2025-01-15T10:00:00Z"
}
```

#### GET /api/links

List links with filtering and pagination

```json
Parameters:
- page (int): Page number, default 1
- per_page (int): Items per page, default 20, max 100
- search (str): Full-text search query
- tags (str): Comma-separated tag names
- domain (str): Filter by domain
- archived (bool): Include archived links
- sort (str): created_at|updated_at|title, default -created_at

Response:
{
    "items": [...],
    "total": 145,
    "page": 1,
    "per_page": 20,
    "pages": 8
}
```

#### GET /api/links/{id}

Get single link with full metadata

#### PUT /api/links/{id}

Update link properties

#### DELETE /api/links/{id}

Delete a link

#### POST /api/links/{id}/archive

Archive/unarchive a link

### Tag Management

#### GET /api/tags

List all user tags with usage counts

```json
Response:
{
    "tags": [
        {
            "id": "...",
            "name": "rust",
            "color": "#CE422B",
            "count": 23
        }
    ]
}
```

#### POST /api/tags

Create a new tag

#### PUT /api/tags/{id}

Update tag properties

#### DELETE /api/tags/{id}

Delete tag (removes from all links)

### Search & Analytics

#### GET /api/search

Advanced search with multiple filters

```json
Parameters:
- q (str): Query string supporting operators
  Examples:
  - "rust AND async"
  - "title:rust -archived:true"
  - "domain:github.com created:>2024-01-01"
```

#### GET /api/stats

User statistics

```json
Response:
{
    "total_links": 523,
    "archived_links": 45,
    "total_tags": 28,
    "top_domains": [
        {"domain": "github.com", "count": 89},
        {"domain": "news.ycombinator.com", "count": 67}
    ],
    "links_by_month": [
        {"month": "2024-12", "count": 45},
        {"month": "2025-01", "count": 23}
    ]
}
```

### Import/Export

#### POST /api/import

Import links from various formats

```json
Request:
{
    "format": "netscape|json|csv",
    "data": "...",
    "deduplicate": true
}
```

#### GET /api/export

Export all links

```
Parameters:
- format: netscape|json|csv|markdown
```

## Rust CLI Design

### Commands Structure

```bash
# Authentication
lnk auth login --api-key <key>
lnk auth logout
lnk auth status

# Quick save (most common use case)
lnk save <url> [--tags tag1,tag2] [--title "Custom Title"]

# Piping support
echo "https://example.com" | lnk save --tags rust,async
cat urls.txt | lnk save --batch

# Search and browse
lnk search "rust async" [--limit 10] [--open]
lnk list [--tags rust] [--recent 7d] [--format table|json]

# Tag management
lnk tags list
lnk tags create <name> [--color "#CE422B"]
lnk tags rename <old> <new>

# Interactive mode
lnk interactive  # TUI for browsing/searching

# Configuration
lnk config set api_url https://api.links.yourdomain.com
lnk config get api_url
```

### CLI Configuration File

Location: `~/.config/lnk/config.toml`

```toml
[api]
url = "https://api.links.yourdomain.com"
timeout = 30

[auth]
# Stored in system keyring, not in file
# api_key = "..."

[defaults]
extract_metadata = true
open_after_save = false
default_tags = []

[display]
format = "table"  # table, json, yaml
color = true
max_width = 120

[cache]
enabled = true
ttl = 300  # seconds
path = "~/.cache/lnk"
```

## Rust CLI Implementation Details

### Project Structure

```
lnk-cli/
├── Cargo.toml
├── src/
│   ├── main.rs           # Entry point, CLI setup
│   ├── cli.rs            # Clap command definitions
│   ├── client/
│   │   ├── mod.rs        # API client module
│   │   ├── auth.rs       # Authentication handling
│   │   ├── links.rs      # Link operations
│   │   └── tags.rs       # Tag operations
│   ├── config/
│   │   ├── mod.rs        # Configuration management
│   │   └── keyring.rs    # Secure credential storage
│   ├── display/
│   │   ├── mod.rs        # Output formatting
│   │   ├── table.rs      # Table display
│   │   └── json.rs       # JSON output
│   ├── interactive/
│   │   └── mod.rs        # TUI implementation (using ratatui)
│   └── utils/
│       ├── mod.rs        # Utility functions
│       └── url.rs        # URL validation and parsing
```

### Key Dependencies (Cargo.toml)

```toml
[dependencies]
clap = { version = "4", features = ["derive", "env"] }
tokio = { version = "1", features = ["full"] }
reqwest = { version = "0.11", features = ["json", "rustls-tls"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
toml = "0.8"
keyring = "2"
colored = "2"
indicatif = "0.17"  # Progress bars
ratatui = "0.25"    # TUI
crossterm = "0.27"  # Terminal control
anyhow = "1"        # Error handling
thiserror = "1"     # Error types
url = "2"
chrono = { version = "0.4", features = ["serde"] }
dirs = "5"          # Platform-specific directories
tabled = "0.15"     # Table formatting
```

## FastAPI Implementation Details

### Project Structure

```
link-api/
├── pyproject.toml
├── alembic.ini
├── .env.example
├── app/
│   ├── __init__.py
│   ├── main.py           # FastAPI app initialization
│   ├── config.py         # Settings management
│   ├── database.py       # Database connection
│   ├── models/
│   │   ├── __init__.py
│   │   ├── user.py
│   │   ├── link.py
│   │   └── tag.py
│   ├── schemas/
│   │   ├── __init__.py
│   │   ├── link.py       # Pydantic models
│   │   ├── tag.py
│   │   └── auth.py
│   ├── api/
│   │   ├── __init__.py
│   │   ├── deps.py       # Dependencies (auth, db)
│   │   └── v1/
│   │       ├── __init__.py
│   │       ├── links.py
│   │       ├── tags.py
│   │       ├── search.py
│   │       └── stats.py
│   ├── services/
│   │   ├── __init__.py
│   │   ├── metadata.py   # Link metadata extraction
│   │   ├── screenshot.py # Screenshot generation
│   │   └── search.py     # Search implementation
│   ├── background/
│   │   ├── __init__.py
│   │   └── tasks.py      # Celery tasks
│   └── utils/
│       ├── __init__.py
│       └── security.py   # API key generation
├── alembic/
│   └── versions/
└── tests/
```

### Background Tasks

Using Celery for async processing:

1. **Metadata Extraction**: Fetch title, description, Open Graph tags
2. **Screenshot Generation**: Using Playwright or Selenium
3. **Content Indexing**: Extract and index full text for search
4. **Favicon Fetching**: Get and cache site favicons
5. **Archive.org Backup**: Optionally trigger wayback machine

## Deployment Strategy

### Development

```bash
# FastAPI
docker-compose up -d postgres
uvicorn app.main:app --reload

# Rust CLI
cargo run -- save https://example.com
```

### Production

**FastAPI Backend**:

```yaml
# docker-compose.yml
services:
  api:
    image: link-api:latest
    environment:
      DATABASE_URL: postgresql://...
    deploy:
      replicas: 2

  postgres:
    image: postgres:15
    volumes:
      - postgres_data:/var/lib/postgresql/data

  nginx:
    image: nginx
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
```

**Rust CLI Distribution**:

```bash
# Build for multiple platforms
cargo build --release --target x86_64-unknown-linux-musl
cargo build --release --target x86_64-apple-darwin
cargo build --release --target aarch64-apple-darwin

# Homebrew tap for macOS
# AUR package for Arch Linux
# Cargo install from crates.io
```

### Phase 4 (Months 7-12)

- Team workspaces
- Advanced analytics
- API for third-party integrations
- Link archival with Wayback Machine
- Reader mode with annotations

## Development Timeline

### Week 1-2: Foundation

- [ ] Setup FastAPI project structure
- [ ] Create database schema and migrations
- [ ] Implement basic CRUD endpoints
- [ ] Setup Docker development environment

### Week 3-4: Rust CLI

- [ ] Create CLI project structure
- [ ] Implement authentication flow
- [ ] Add basic save/list/search commands
- [ ] Setup credential storage

### Week 5-6: Enhanced Features

- [ ] Add metadata extraction service
- [ ] Implement full-text search
- [ ] Add tag management
- [ ] Create import/export functionality

### Week 7-8: Polish & Deploy

- [ ] Add comprehensive tests
- [ ] Setup CI/CD pipeline
- [ ] Deploy to production
- [ ] Write documentation

## Conclusion

This system provides a simple, practical solution for link organization that can start minimal and grow with your needs. The Rust CLI provides fast local interaction, while the FastAPI backend ensures reliable storage and powerful search capabilities. The architecture supports future expansion to mobile apps, browser extensions, and team collaboration features without over-engineering the initial implementation.
