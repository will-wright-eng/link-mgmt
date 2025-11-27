# Go API + CLI Design Document

> **Implementation Status**: This document has been updated with current implementation status as of the latest review. Status markers:
>
> - ✅ **Implemented** - Feature is complete and working
> - ⚠️ **Partially Implemented** - Feature exists but incomplete or missing some functionality
> - ❌ **Not Implemented** - Feature is planned but not yet built

## Quick Status Summary

**Overall Progress**: ~70% Complete

- ✅ **Core Infrastructure**: Models, database layer, configuration, API server structure
- ✅ **API Endpoints**: GET/POST/DELETE for links, user management (missing PUT for updates)
- ✅ **Middleware**: Authentication, error handling, logging
- ⚠️ **CLI**: Basic structure and config commands work, but link management commands are stubs
- ❌ **TUI**: Bubble Tea interactive interface not implemented
- ❌ **Missing Features**: UpdateLink functionality, Dockerfile, cross-compilation targets

**Additional Features Found** (not in original design):

- ✅ `GetLinkByID` query and handler (GET `/api/v1/links/:id`)
- ✅ CLI config management commands (`--config-show`, `--config-set`)
- ✅ Health check endpoint (`/health`)

## Overview

This document outlines the architecture and design for a Go-based link management system consisting of:

- **REST API** - HTTP server for programmatic access
- **CLI** - Terminal user interface for interactive use
- **Shared Core Library** - Common code, models, and database logic

Based on the language comparison analysis, Go was selected for its:

- ⭐⭐⭐⭐⭐ Native shared library pattern (`cmd/`, `pkg/`)
- ⭐⭐⭐⭐⭐ Single binary distribution with easy cross-compilation
- ⭐⭐⭐⭐⭐ Excellent TUI libraries (Bubble Tea)
- ⭐⭐⭐⭐⭐ Mature API frameworks (stdlib + gin/echo/fiber)
- ⭐⭐⭐⭐⭐ Excellent PostgreSQL integration (pgx)
- ⭐⭐⭐⭐ Fast development iteration
- ⭐⭐⭐⭐ Strong type safety

## Project Structure

Following Go conventions for multi-binary projects:

```
link-mgmt-go/
├── cmd/
│   ├── api/
│   │   └── main.go              # ✅ API server entry point
│   └── cli/
│       └── main.go              # ✅ CLI entry point
├── pkg/
│   ├── models/
│   │   ├── user.go              # ✅ User model
│   │   └── link.go              # ✅ Link model (includes LinkCreate, LinkUpdate)
│   ├── db/
│   │   ├── postgres.go          # ✅ Connection setup & pool
│   │   └── queries.go           # ⚠️ Database queries (missing UpdateLink)
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── users.go         # ✅ User API handlers
│   │   │   ├── links.go         # ⚠️ Link API handlers (missing UpdateLink)
│   │   │   └── health.go        # ✅ Health check handler
│   │   ├── middleware/
│   │   │   ├── auth.go          # ✅ API key authentication
│   │   │   ├── error.go         # ✅ Error handling middleware
│   │   │   └── logging.go       # ✅ Request logging
│   │   └── router.go            # ⚠️ HTTP router setup (missing PUT /links/:id)
│   ├── cli/
│   │   ├── app.go               # ⚠️ Basic CLI app (TUI not implemented)
│   │   └── models/              # ❌ TUI models directory exists but empty
│   └── config/
│       └── config.go            # ✅ Configuration management
├── migrations/
│   ├── 001_create_users.sql     # ✅ Users table migration
│   └── 002_create_links.sql     # ✅ Links table migration
├── internal/
│   └── auth/                    # ❌ Directory exists but empty (api_key.go not implemented)
├── go.mod                       # ✅ Go module file
├── go.sum                       # ✅ Go checksums
├── Makefile                     # ⚠️ Basic build commands (missing cross-compilation targets)
├── Dockerfile                   # ❌ Not implemented
└── README.md                    # ✅ README with setup instructions
```

## Architecture Principles

### 1. Shared Core Library Pattern

All shared code lives in `pkg/`:

- **Models** (`pkg/models/`) - Data structures used by both API and CLI
- **Database** (`pkg/db/`) - PostgreSQL connection and queries
- **Config** (`pkg/config/`) - Configuration loading and validation

Both `cmd/api` and `cmd/cli` import from `pkg/`:

```go
// cmd/api/main.go
import "link-mgmt-go/pkg/db"
import "link-mgmt-go/pkg/models"

// cmd/cli/main.go
import "link-mgmt-go/pkg/db"
import "link-mgmt-go/pkg/models"
```

### 2. Separation of Concerns

- **API** (`pkg/api/`) - HTTP handlers, middleware, routing
- **CLI** (`pkg/cli/`) - TUI models, commands, user interaction
- **Core** (`pkg/models/`, `pkg/db/`) - Business logic, data access

### 3. Database-First Design

- Use `pgx` for direct PostgreSQL access
- Migrations managed with `golang-migrate` or `goose`
- Consider `sqlc` for type-safe query generation

## Shared Core Library (`pkg/`)

### Models (`pkg/models/`) ✅ **Implemented**

#### User Model ✅

```go
// pkg/models/user.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID        uuid.UUID `db:"id" json:"id"`
    Email     string    `db:"email" json:"email"`
    APIKey    string    `db:"api_key" json:"api_key"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
    UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
```

#### Link Model ✅

**Status**: Fully implemented. Includes `Link`, `LinkCreate`, and `LinkUpdate` types.

### Database Layer (`pkg/db/`) ⚠️ **Partially Implemented**

**Status**: Connection setup and most queries implemented. Missing `UpdateLink` function.

#### Connection Setup ✅

#### Query Functions ⚠️

**Status**:

- ✅ `GetUserByAPIKey` - Implemented
- ✅ `CreateUser` - Implemented
- ✅ `GetLinksByUserID` - Implemented
- ✅ `CreateLink` - Implemented
- ✅ `GetLinkByID` - Implemented (not in design doc, but added)
- ✅ `DeleteLink` - Implemented
- ❌ `UpdateLink` - Not implemented

```go
// pkg/db/queries.go
package db

import (
    "context"
    "fmt"
    "link-mgmt-go/pkg/models"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

// UpdateLink updates an existing link
// ❌ NOT IMPLEMENTED - This function is missing from queries.go
func (db *DB) UpdateLink(ctx context.Context, linkID, userID uuid.UUID, update models.LinkUpdate) (*models.Link, error) {
    // Build dynamic update query
    // Implementation details...
}

// DeleteLink deletes a link
func (db *DB) DeleteLink(ctx context.Context, linkID, userID uuid.UUID) error {
    result, err := db.Pool.Exec(ctx,
        `DELETE FROM links WHERE id = $1 AND user_id = $2`,
        linkID, userID,
    )
    if err != nil {
        return fmt.Errorf("failed to delete link: %w", err)
    }

    if result.RowsAffected() == 0 {
        return fmt.Errorf("link not found")
    }

    return nil
}
```

### Configuration (`pkg/config/`) ✅ **Implemented**

**Status**: Fully implemented with auto-creation of config file, defaults, and save functionality.

**Note:** The config file will be automatically created with defaults on first run if it doesn't exist.

## API Server (`cmd/api/`) ✅ **Implemented**

**Status**: Entry point, router, handlers, and middleware are implemented. Missing `UpdateLink` handler and route.

### Entry Point ✅

### Router Setup ⚠️

**Status**: Router implemented with Gin. Missing `PUT /api/v1/links/:id` route for updating links.

```go
// pkg/api/router.go
package api

import (
    "link-mgmt-go/pkg/api/handlers"
    "link-mgmt-go/pkg/api/middleware"
    "link-mgmt-go/pkg/db"

    "github.com/gin-gonic/gin"
)

func NewRouter(db *db.DB) *gin.Engine {
    router := gin.Default()

    // Middleware
    router.Use(middleware.RequestLogger())
    router.Use(middleware.ErrorHandler())

    // Health check
    router.GET("/health", handlers.HealthCheck)

    // API routes
    v1 := router.Group("/api/v1")
    {
        // Links
        links := v1.Group("/links")
        links.Use(middleware.RequireAuth(db))
        {
            links.GET("", handlers.ListLinks(db))
            links.POST("", handlers.CreateLink(db))
            links.GET("/:id", handlers.GetLink(db))
            // ❌ Missing: links.PUT("/:id", handlers.UpdateLink(db))
            links.DELETE("/:id", handlers.DeleteLink(db))
        }

        // Users
        users := v1.Group("/users")
        {
            users.POST("", handlers.CreateUser(db))
            users.GET("/me", middleware.RequireAuth(db), handlers.GetCurrentUser(db))
        }
    }

    return router
}
```

### Handlers ⚠️

**Status**:

- ✅ `ListLinks` - Implemented
- ✅ `CreateLink` - Implemented
- ✅ `GetLink` - Implemented (not in design doc, but added)
- ❌ `UpdateLink` - Not implemented
- ✅ `DeleteLink` - Implemented
- ✅ `CreateUser` - Implemented
- ✅ `GetCurrentUser` - Implemented

### Middleware ✅ **Implemented**

**Status**: All middleware components are implemented:

- ✅ `RequireAuth` - API key authentication
- ✅ `ErrorHandler` - Error recovery middleware
- ✅ `RequestLogger` - Request logging

## CLI Application (`cmd/cli/`) ⚠️ **Partially Implemented**

**Status**: Entry point and basic app structure exist. TUI (Bubble Tea) not implemented. CLI commands (`--list`, `--add`, `--delete`) are stubs. Config commands (`--config-show`, `--config-set`) are implemented.

### Entry Point ✅

**Note**: Entry point includes additional config management commands not in design doc:

- `--config-show` - Show current configuration
- `--config-set` - Set config values

### Bubble Tea TUI Application ❌ **Not Implemented**

**Status**: Basic app structure exists, but TUI functionality is not implemented. The `Run()` method just prints a "coming soon" message. TUI models directory exists but is empty.

```go
// pkg/cli/app.go
package cli

import (
    "link-mgmt-go/pkg/cli/models"
    "link-mgmt-go/pkg/config"
    "link-mgmt-go/pkg/db"

    tea "github.com/charmbracelet/bubbletea"
)

type App struct {
    db   *db.DB
    cfg  *config.Config
    program *tea.Program
}

func NewApp(db *db.DB, cfg *config.Config) *App {
    return &App{
        db:  db,
        cfg: cfg,
    }
}

func (a *App) Run() error {
    initialModel := models.NewListModel(a.db, a.cfg)
    a.program = tea.NewProgram(initialModel, tea.WithAltScreen())

    _, err := a.program.Run()
    return err
}

func (a *App) ListLinks() {
    // ❌ Stub implementation - just prints "coming soon"
}

func (a *App) AddLink() {
    // ❌ Stub implementation - just prints "coming soon"
}

func (a *App) DeleteLink() {
    // ❌ Stub implementation - just prints "coming soon"
}
```

### TUI Models ❌ **Not Implemented**

**Status**: The `pkg/cli/models/` directory exists but is empty. None of the TUI models (list.go, form.go, help.go) are implemented.

```go
// pkg/cli/models/list.go
package models

import (
    "context"
    "fmt"
    "link-mgmt-go/pkg/config"
    "link-mgmt-go/pkg/db"
    "link-mgmt-go/pkg/models"

    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type listKeyMap struct {
    Quit    key.Binding
    Add     key.Binding
    Delete  key.Binding
    Refresh key.Binding
}

func newListKeyMap() *listKeyMap {
    return &listKeyMap{
        Quit: key.NewBinding(
            key.WithKeys("q", "ctrl+c"),
            key.WithHelp("q", "quit"),
        ),
        Add: key.NewBinding(
            key.WithKeys("a"),
            key.WithHelp("a", "add link"),
        ),
        Delete: key.NewBinding(
            key.WithKeys("d"),
            key.WithHelp("d", "delete"),
        ),
        Refresh: key.NewBinding(
            key.WithKeys("r"),
            key.WithHelp("r", "refresh"),
        ),
    }
}

type listItem struct {
    link models.Link
}

func (i listItem) FilterValue() string {
    if i.link.Title != nil {
        return *i.link.Title
    }
    return i.link.URL
}

func (i listItem) Title() string {
    if i.link.Title != nil {
        return *i.link.Title
    }
    return i.link.URL
}

func (i listItem) Description() string {
    return i.link.URL
}

type ListModel struct {
    db     *db.DB
    cfg    *config.Config
    list   list.Model
    keys   *listKeyMap
    userID string // Would need to get from config/auth
}

func NewListModel(db *db.DB, cfg *config.Config) ListModel {
    items := []list.Item{} // Load from DB

    l := list.New(items, list.NewDefaultDelegate(), 80, 20)
    l.Title = "My Links"
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(true)

    return ListModel{
        db:   db,
        cfg:  cfg,
        list: l,
        keys: newListKeyMap(),
    }
}

func (m ListModel) Init() tea.Cmd {
    return m.loadLinks()
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, m.keys.Quit):
            return m, tea.Quit
        case key.Matches(msg, m.keys.Add):
            // Transition to add form
            return m, nil
        case key.Matches(msg, m.keys.Delete):
            // Delete selected item
            return m, nil
        case key.Matches(msg, m.keys.Refresh):
            return m, m.loadLinks()
        }
    case linksLoadedMsg:
        items := make([]list.Item, len(msg.links))
        for i, link := range msg.links {
            items[i] = listItem{link: link}
        }
        m.list.SetItems(items)
        return m, nil
    }

    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m ListModel) View() string {
    return m.list.View()
}

type linksLoadedMsg struct {
    links []models.Link
}

func (m ListModel) loadLinks() tea.Cmd {
    return func() tea.Msg {
        // In real implementation, would need user context
        ctx := context.Background()
        links, err := m.db.GetLinksByUserID(ctx, /* userID */)
        if err != nil {
            return linksLoadedMsg{links: []models.Link{}}
        }
        return linksLoadedMsg{links: links}
    }
}
```

## Database Migrations ✅ **Implemented**

**Status**: Both migrations are implemented. Note: The design doc mentions `003_add_indexes.sql`, but indexes are included in the first two migration files.

## Build & Distribution

### Makefile ⚠️ **Partially Implemented**

**Status**: Basic build commands are implemented. Missing cross-compilation targets (`build-release`) and migration commands.

```makefile
# Makefile
.PHONY: build-api build-cli build-all test migrate-up migrate-down run-api run-cli

# Build API
build-api:
	go build -o bin/api ./cmd/api

# Build CLI
build-cli:
	go build -o bin/cli ./cmd/cli

# Build both
build-all: build-api build-cli

# ❌ Cross-compile for multiple platforms - NOT IMPLEMENTED
# build-release:
#	GOOS=linux GOARCH=amd64 go build -o bin/api-linux-amd64 ./cmd/api
#	...

# ✅ Run tests - IMPLEMENTED
test:
	go test ./...

# ❌ Database migrations - NOT IMPLEMENTED (migrations run manually via psql)
# migrate-up:
#	migrate -path migrations -database "$(DATABASE_URL)" up

# Run API
run-api:
	go run ./cmd/api

# Run CLI
run-cli:
	go run ./cmd/cli

# Install dependencies
deps:
	go mod download
	go mod tidy
```

### Dockerfile ❌ **Not Implemented**

**Status**: Dockerfile is not present in the repository.

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build API
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Final stage
FROM scratch
COPY --from=builder /app/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
```

## Development Workflow

### 1. Setup

```bash
# Initialize Go module
go mod init link-mgmt-go

# Install dependencies
go get github.com/gin-gonic/gin
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/charmbracelet/bubbletea
go get github.com/google/uuid
go get github.com/golang-migrate/migrate/v4
go get github.com/pelletier/go-toml/v2

# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 2. Development

```bash
# Run migrations
make migrate-up

# Run API server (with hot reload using air)
air -c .air.toml

# Run CLI
make run-cli

# Run tests
make test
```

### 3. Code Generation (Optional)

Consider using `sqlc` for type-safe queries:

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "pkg/db/queries.sql"
    schema: "migrations"
    gen:
      go:
        package: "db"
        out: "pkg/db"
        sql_package: "pgx/v5"
```

## Key Design Decisions

### 1. Framework Choice: Gin vs stdlib ✅ **Gin Selected**

**Status**: Gin was chosen and implemented.

- **stdlib**: Zero dependencies, full control, good for learning
- **Gin**: Faster development, middleware ecosystem, good for production

**Implementation**: Gin is used throughout the API:

- ✅ Middleware support (auth, logging, error handling)
- ✅ JSON binding/validation
- ✅ Route grouping
- ✅ Lightweight (~2MB binary)

### 2. Database Access: pgx vs ORM ✅ **pgx Selected**

**Status**: pgx is implemented. sqlc not used (queries written manually).

- **pgx**: ✅ Direct PostgreSQL access, excellent performance, native types
- **sqlc**: ❌ Not used (queries written manually in queries.go)
- **Avoid GORM**: Not used

### 3. CLI: Bubble Tea vs Cobra ⚠️ **Basic Implementation**

**Status**: Basic flag-based CLI implemented. Bubble Tea TUI not yet implemented.

- **Bubble Tea**: ❌ Not implemented (dependency present but TUI not built)
- **Cobra**: ❌ Not used
- **Current approach**: ⚠️ Standard library `flag` package for CLI commands
- **Future**: Bubble Tea planned for interactive mode

### 4. Configuration Management ✅ **TOML Implemented**

**Status**: Fully implemented with TOML config file.

- ✅ Single config file at `~/.config/link-mgmt/config.toml` for all settings
- ✅ Auto-created with sensible defaults on first run
- ✅ TOML format is human-readable and easy to edit
- ✅ Uses `github.com/pelletier/go-toml/v2` for parsing
- ✅ Both API and CLI read from the same config file
- ✅ CLI includes `--config-show` and `--config-set` commands for managing config
- ⚠️ API key stored in config file (keychain integration not implemented)

### 5. Authentication ⚠️ **API Key Implemented (Unhashed)**

**Status**: API key authentication implemented, but keys are stored unhashed.

- ✅ Simple, stateless, works for both API and CLI
- ⚠️ API keys stored **unhashed** in database (security concern for production)
- ✅ CLI stores API key in config file
- ❌ Keychain integration not implemented
- ⚠️ API key generation in handlers/users.go (should be in internal/auth/)

## Performance Considerations

1. **Connection Pooling**: pgxpool handles this automatically
2. **Query Optimization**: Use indexes, prepared statements
3. **Caching**: Consider Redis for frequently accessed data
4. **Rate Limiting**: Add middleware for API endpoints

## Security Considerations

1. **API Keys**: Hash in database, use secure random generation
2. **Input Validation**: Use `validator` package
3. **SQL Injection**: pgx uses parameterized queries (safe)
4. **CORS**: Configure appropriately for web clients
5. **HTTPS**: Use in production

## Deployment

### API Server

- Single binary deployment
- Container-based (Docker)
- Can use `scratch` base image (~10MB)
- Health check endpoint for orchestration

### CLI

- Single binary distribution
- Cross-compile for target platforms
- Can be distributed via:
    - GitHub Releases
    - Homebrew (macOS)
    - Package managers (Linux)
    - Direct download

## Implementation Summary

### ✅ Completed (Phase 1 - Core Library + API)

- ✅ Models (User, Link, LinkCreate, LinkUpdate)
- ✅ Database layer (connection, most queries)
- ✅ Configuration management
- ✅ API server entry point
- ✅ Router setup (missing UpdateLink route)
- ✅ Handlers (ListLinks, CreateLink, GetLink, DeleteLink, CreateUser, GetCurrentUser)
- ✅ Middleware (auth, error handling, logging)
- ✅ Database migrations
- ✅ CLI entry point with config commands

### ⚠️ Partially Complete

- ⚠️ Database queries (missing `UpdateLink`)
- ⚠️ API router (missing PUT route)
- ⚠️ CLI app (structure exists, but commands are stubs)
- ⚠️ Makefile (basic commands only, missing cross-compilation)

### ❌ Not Implemented

- ❌ `UpdateLink` database query and handler
- ❌ PUT `/api/v1/links/:id` route
- ❌ CLI commands (`--list`, `--add`, `--delete` implementations)
- ❌ Bubble Tea TUI (interactive mode)
- ❌ TUI models (list.go, form.go, help.go)
- ❌ Internal auth package (api_key.go)
- ❌ Dockerfile
- ❌ Cross-compilation build targets
- ❌ Migration tooling (golang-migrate integration)

## Next Steps

1. **Phase 1 Remaining**:
   - Implement `UpdateLink` query and handler
   - Add PUT route to router
2. **Phase 2**: CLI basic commands (list, add, delete) - currently stubs
3. **Phase 3**: Interactive TUI with Bubble Tea - not started
4. **Phase 4**: Advanced features (search, tags, export)

## References

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [pgx Documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [Bubble Tea Guide](https://github.com/charmbracelet/bubbletea)
- [Gin Framework](https://gin-gonic.com/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
