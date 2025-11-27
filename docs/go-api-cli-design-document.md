# Go API + CLI Design Document

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
│   │   └── main.go              # API server entry point
│   └── cli/
│       └── main.go              # CLI entry point
├── pkg/
│   ├── models/
│   │   ├── user.go              # User model
│   │   └── link.go              # Link model
│   ├── db/
│   │   ├── postgres.go          # Connection setup & pool
│   │   └── queries.go           # Database queries (or sqlc generated)
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── users.go         # User API handlers
│   │   │   └── links.go         # Link API handlers
│   │   ├── middleware/
│   │   │   ├── auth.go          # API key authentication
│   │   │   └── logging.go       # Request logging
│   │   └── router.go            # HTTP router setup
│   ├── cli/
│   │   ├── app.go               # Bubble Tea TUI application
│   │   ├── models/
│   │   │   ├── list.go          # Link list view model
│   │   │   ├── form.go          # Add/edit form model
│   │   │   └── help.go          # Help view model
│   │   └── commands/
│   │       ├── add.go           # Add link command
│   │       ├── list.go          # List links command
│   │       └── delete.go        # Delete link command
│   └── config/
│       └── config.go            # Configuration management
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_links.sql
│   └── 003_add_indexes.sql
├── internal/
│   └── auth/
│       └── api_key.go           # API key generation/validation
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
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

### Models (`pkg/models/`)

#### User Model

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

#### Link Model

```go
// pkg/models/link.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type Link struct {
    ID          uuid.UUID  `db:"id" json:"id"`
    UserID      uuid.UUID  `db:"user_id" json:"user_id"`
    URL         string     `db:"url" json:"url"`
    Title       *string    `db:"title" json:"title,omitempty"`
    Description *string    `db:"description" json:"description,omitempty"`
    Text        *string    `db:"text" json:"text,omitempty"`
    CreatedAt   time.Time  `db:"created_at" json:"created_at"`
    UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// LinkCreate represents data for creating a new link
type LinkCreate struct {
    URL         string  `json:"url" validate:"required,url"`
    Title       *string `json:"title,omitempty"`
    Description *string `json:"description,omitempty"`
    Text        *string `json:"text,omitempty"`
}

// LinkUpdate represents data for updating a link
type LinkUpdate struct {
    URL         *string `json:"url,omitempty" validate:"omitempty,url"`
    Title       *string `json:"title,omitempty"`
    Description *string `json:"description,omitempty"`
    Text        *string `json:"text,omitempty"`
}
```

### Database Layer (`pkg/db/`)

#### Connection Setup

```go
// pkg/db/postgres.go
package db

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
    Pool *pgxpool.Pool
}

func New(ctx context.Context, connString string) (*DB, error) {
    pool, err := pgxpool.New(ctx, connString)
    if err != nil {
        return nil, fmt.Errorf("failed to create connection pool: %w", err)
    }

    // Test connection
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
    db.Pool.Close()
}
```

#### Query Functions

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

// GetUserByAPIKey retrieves a user by their API key
func (db *DB) GetUserByAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
    var user models.User
    err := db.Pool.QueryRow(ctx,
        `SELECT id, email, api_key, created_at, updated_at
         FROM users WHERE api_key = $1`,
        apiKey,
    ).Scan(
        &user.ID,
        &user.Email,
        &user.APIKey,
        &user.CreatedAt,
        &user.UpdatedAt,
    )

    if err == pgx.ErrNoRows {
        return nil, fmt.Errorf("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return &user, nil
}

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, email, apiKey string) (*models.User, error) {
    var user models.User
    err := db.Pool.QueryRow(ctx,
        `INSERT INTO users (email, api_key)
         VALUES ($1, $2)
         RETURNING id, email, api_key, created_at, updated_at`,
        email, apiKey,
    ).Scan(
        &user.ID,
        &user.Email,
        &user.APIKey,
        &user.CreatedAt,
        &user.UpdatedAt,
    )

    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    return &user, nil
}

// GetLinksByUserID retrieves all links for a user
func (db *DB) GetLinksByUserID(ctx context.Context, userID uuid.UUID) ([]models.Link, error) {
    rows, err := db.Pool.Query(ctx,
        `SELECT id, user_id, url, title, description, text, created_at, updated_at
         FROM links
         WHERE user_id = $1
         ORDER BY created_at DESC`,
        userID,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to query links: %w", err)
    }
    defer rows.Close()

    var links []models.Link
    for rows.Next() {
        var link models.Link
        err := rows.Scan(
            &link.ID,
            &link.UserID,
            &link.URL,
            &link.Title,
            &link.Description,
            &link.Text,
            &link.CreatedAt,
            &link.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan link: %w", err)
        }
        links = append(links, link)
    }

    return links, rows.Err()
}

// CreateLink creates a new link
func (db *DB) CreateLink(ctx context.Context, userID uuid.UUID, link models.LinkCreate) (*models.Link, error) {
    var created models.Link
    err := db.Pool.QueryRow(ctx,
        `INSERT INTO links (user_id, url, title, description, text)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, user_id, url, title, description, text, created_at, updated_at`,
        userID, link.URL, link.Title, link.Description, link.Text,
    ).Scan(
        &created.ID,
        &created.UserID,
        &created.URL,
        &created.Title,
        &created.Description,
        &created.Text,
        &created.CreatedAt,
        &created.UpdatedAt,
    )

    if err != nil {
        return nil, fmt.Errorf("failed to create link: %w", err)
    }

    return &created, nil
}

// UpdateLink updates an existing link
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

### Configuration (`pkg/config/`)

```go
// pkg/config/config.go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/pelletier/go-toml/v2"
)

type Config struct {
    // Database
    Database struct {
        URL string `toml:"url"`
    } `toml:"database"`

    // API
    API struct {
        Port int    `toml:"port"`
        Host string `toml:"host"`
    } `toml:"api"`

    // CLI
    CLI struct {
        APIBaseURL string `toml:"api_base_url"`
        APIKey     string `toml:"api_key"`
    } `toml:"cli"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
    cfg := &Config{}
    cfg.Database.URL = "postgres://localhost/linkmgmt?sslmode=disable"
    cfg.API.Port = 8080
    cfg.API.Host = "0.0.0.0"
    cfg.CLI.APIBaseURL = "http://localhost:8080"
    cfg.CLI.APIKey = ""
    return cfg
}

// ConfigPath returns the path to the config file
func ConfigPath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("failed to get home directory: %w", err)
    }
    configDir := filepath.Join(homeDir, ".config", "link-mgmt")
    return filepath.Join(configDir, "config.toml"), nil
}

// Load reads configuration from ~/.config/link-mgmt/config.toml
// Creates the file with defaults if it doesn't exist
func Load() (*Config, error) {
    configPath, err := ConfigPath()
    if err != nil {
        return nil, err
    }

    // Expand ~ in path if needed
    if strings.HasPrefix(configPath, "~") {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return nil, fmt.Errorf("failed to get home directory: %w", err)
        }
        configPath = strings.Replace(configPath, "~", homeDir, 1)
    }

    // Check if config file exists
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        // Create directory if it doesn't exist
        configDir := filepath.Dir(configPath)
        if err := os.MkdirAll(configDir, 0755); err != nil {
            return nil, fmt.Errorf("failed to create config directory: %w", err)
        }

        // Create default config file
        cfg := DefaultConfig()
        if err := Save(cfg); err != nil {
            return nil, fmt.Errorf("failed to create default config: %w", err)
        }
        return cfg, nil
    }

    // Read existing config file
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var cfg Config
    if err := toml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }

    // Merge with defaults for any missing values
    defaultCfg := DefaultConfig()
    if cfg.Database.URL == "" {
        cfg.Database.URL = defaultCfg.Database.URL
    }
    if cfg.API.Port == 0 {
        cfg.API.Port = defaultCfg.API.Port
    }
    if cfg.API.Host == "" {
        cfg.API.Host = defaultCfg.API.Host
    }
    if cfg.CLI.APIBaseURL == "" {
        cfg.CLI.APIBaseURL = defaultCfg.CLI.APIBaseURL
    }

    return &cfg, nil
}

// Save writes the configuration to the config file
func Save(cfg *Config) error {
    configPath, err := ConfigPath()
    if err != nil {
        return err
    }

    // Expand ~ in path if needed
    if strings.HasPrefix(configPath, "~") {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return fmt.Errorf("failed to get home directory: %w", err)
        }
        configPath = strings.Replace(configPath, "~", homeDir, 1)
    }

    // Create directory if it doesn't exist
    configDir := filepath.Dir(configPath)
    if err := os.MkdirAll(configDir, 0755); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

    // Marshal to TOML
    data, err := toml.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    // Write to file
    if err := os.WriteFile(configPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    return nil
}
```

Example `~/.config/link-mgmt/config.toml`:

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

**Note:** The config file will be automatically created with defaults on first run if it doesn't exist.

## API Server (`cmd/api/`)

### Entry Point

```go
// cmd/api/main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "link-mgmt-go/pkg/api"
    "link-mgmt-go/pkg/config"
    "link-mgmt-go/pkg/db"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    ctx := context.Background()

    // Initialize database
    database, err := db.New(ctx, cfg.Database.URL)
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }
    defer database.Close()

    // Initialize router
    router := api.NewRouter(database)

    // Create server
    srv := &http.Server{
        Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server in goroutine
    go func() {
        log.Printf("API server starting on %s", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server failed: %v", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("server forced to shutdown: %v", err)
    }

    log.Println("server exited")
}
```

### Router Setup

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
            links.PUT("/:id", handlers.UpdateLink(db))
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

### Handlers

```go
// pkg/api/handlers/links.go
package handlers

import (
    "net/http"
    "link-mgmt-go/pkg/db"
    "link-mgmt-go/pkg/models"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func ListLinks(db *db.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.MustGet("userID").(uuid.UUID)

        links, err := db.GetLinksByUserID(c.Request.Context(), userID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusOK, links)
    }
}

func CreateLink(db *db.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.MustGet("userID").(uuid.UUID)

        var linkCreate models.LinkCreate
        if err := c.ShouldBindJSON(&linkCreate); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        link, err := db.CreateLink(c.Request.Context(), userID, linkCreate)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusCreated, link)
    }
}

// Additional handlers: GetLink, UpdateLink, DeleteLink
```

### Middleware

```go
// pkg/api/middleware/auth.go
package middleware

import (
    "net/http"
    "strings"
    "link-mgmt-go/pkg/db"

    "github.com/gin-gonic/gin"
)

func RequireAuth(db *db.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }

        // Extract API key from "Bearer <key>" or just "<key>"
        apiKey := strings.TrimPrefix(authHeader, "Bearer ")
        apiKey = strings.TrimSpace(apiKey)

        user, err := db.GetUserByAPIKey(c.Request.Context(), apiKey)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
            c.Abort()
            return
        }

        c.Set("userID", user.ID)
        c.Set("user", user)
        c.Next()
    }
}
```

## CLI Application (`cmd/cli/`)

### Entry Point

```go
// cmd/cli/main.go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"

    "link-mgmt-go/pkg/cli"
    "link-mgmt-go/pkg/config"
    "link-mgmt-go/pkg/db"
)

func main() {
    var (
        listMode   = flag.Bool("list", false, "List all links")
        addMode    = flag.Bool("add", false, "Add a new link")
        deleteMode = flag.Bool("delete", false, "Delete a link")
        interactive = flag.Bool("i", true, "Interactive mode (default)")
    )
    flag.Parse()

    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    // For CLI, we might connect to API instead of DB directly
    // Or connect to DB for direct access
    ctx := context.Background()
    database, err := db.New(ctx, cfg.Database.URL)
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }
    defer database.Close()

    app := cli.NewApp(database, cfg)

    if *listMode {
        app.ListLinks()
    } else if *addMode {
        app.AddLink()
    } else if *deleteMode {
        app.DeleteLink()
    } else {
        // Interactive TUI mode
        if err := app.Run(); err != nil {
            fmt.Fprintf(os.Stderr, "error: %v\n", err)
            os.Exit(1)
        }
    }
}
```

### Bubble Tea TUI Application

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
    // Non-interactive list mode
}

func (a *App) AddLink() {
    // Non-interactive add mode
}

func (a *App) DeleteLink() {
    // Non-interactive delete mode
}
```

### TUI Models

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

## Database Migrations

Using `golang-migrate`:

```sql
-- migrations/001_create_users.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_api_key ON users(api_key);
```

```sql
-- migrations/002_create_links.sql
CREATE TABLE links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title TEXT,
    description TEXT,
    text TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, url)
);

CREATE INDEX idx_links_user_id ON links(user_id);
CREATE INDEX idx_links_created_at ON links(created_at DESC);
```

## Build & Distribution

### Makefile

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

# Cross-compile for multiple platforms
build-release:
	GOOS=linux GOARCH=amd64 go build -o bin/api-linux-amd64 ./cmd/api
	GOOS=linux GOARCH=amd64 go build -o bin/cli-linux-amd64 ./cmd/cli
	GOOS=darwin GOARCH=amd64 go build -o bin/api-darwin-amd64 ./cmd/api
	GOOS=darwin GOARCH=amd64 go build -o bin/cli-darwin-amd64 ./cmd/cli
	GOOS=darwin GOARCH=arm64 go build -o bin/api-darwin-arm64 ./cmd/api
	GOOS=darwin GOARCH=arm64 go build -o bin/cli-darwin-arm64 ./cmd/cli
	GOOS=windows GOARCH=amd64 go build -o bin/api-windows-amd64.exe ./cmd/api
	GOOS=windows GOARCH=amd64 go build -o bin/cli-windows-amd64.exe ./cmd/cli

# Run tests
test:
	go test ./...

# Database migrations
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

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

### Dockerfile

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

### 1. Framework Choice: Gin vs stdlib

**Recommendation: Start with stdlib, add Gin if needed**

- **stdlib**: Zero dependencies, full control, good for learning
- **Gin**: Faster development, middleware ecosystem, good for production

For this project, **Gin is recommended** for:

- Better middleware support (auth, logging, CORS)
- JSON binding/validation
- Route grouping
- Still lightweight (~2MB binary)

### 2. Database Access: pgx vs ORM

**Recommendation: pgx with optional sqlc**

- **pgx**: Direct PostgreSQL access, excellent performance, native types
- **sqlc**: Generate type-safe code from SQL (optional enhancement)
- **Avoid GORM**: Too much magic, performance overhead

### 3. CLI: Bubble Tea vs Cobra

**Recommendation: Bubble Tea for TUI, Cobra for CLI commands**

- **Bubble Tea**: Excellent for interactive TUI (list, forms, navigation)
- **Cobra**: Better for traditional CLI commands (`link add`, `link list`)
- **Hybrid approach**: Use both - Cobra for commands, Bubble Tea for interactive mode

### 4. Configuration Management

**Recommendation: TOML config file**

- Single config file at `~/.config/link-mgmt/config.toml` for all settings
- Auto-created with sensible defaults on first run
- TOML format is human-readable and easy to edit
- Use `github.com/pelletier/go-toml/v2` for parsing
- Both API and CLI read from the same config file
- API key stored in config file (consider keychain integration for production)

### 5. Authentication

**Recommendation: API key authentication**

- Simple, stateless, works for both API and CLI
- Store hashed API keys in database
- CLI can store API key in config file or keychain

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

## Next Steps

1. **Phase 1**: Core library + API (models, DB, handlers)
2. **Phase 2**: CLI basic commands (list, add, delete)
3. **Phase 3**: Interactive TUI with Bubble Tea
4. **Phase 4**: Advanced features (search, tags, export)

## References

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [pgx Documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [Bubble Tea Guide](https://github.com/charmbracelet/bubbletea)
- [Gin Framework](https://gin-gonic.com/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
