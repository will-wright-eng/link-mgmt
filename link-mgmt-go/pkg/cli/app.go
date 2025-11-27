package cli

import (
	"context"
	"fmt"
	"strings"

	"link-mgmt-go/pkg/config"
	"link-mgmt-go/pkg/db"

	"github.com/pelletier/go-toml/v2"
)

type App struct {
	db  *db.DB
	cfg *config.Config
}

func NewApp(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}

// ConnectDB establishes a database connection (lazy connection)
func (a *App) ConnectDB(ctx context.Context) (*db.DB, error) {
	if a.db != nil {
		return a.db, nil
	}

	database, err := db.New(ctx, a.cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	a.db = database
	return database, nil
}

// ShowConfig displays the current configuration
func (a *App) ShowConfig() {
	data, err := toml.Marshal(a.cfg)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// SetConfig sets a configuration value
// Format: section.key=value (e.g., "database.url=postgres://...")
func (a *App) SetConfig(setStr string) error {
	parts := strings.SplitN(setStr, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected 'section.key=value'")
	}

	keyPath := strings.Split(parts[0], ".")
	value := parts[1]

	if len(keyPath) != 2 {
		return fmt.Errorf("invalid key format: expected 'section.key'")
	}

	section := keyPath[0]
	key := keyPath[1]

	switch section {
	case "database":
		switch key {
		case "url":
			a.cfg.Database.URL = value
		default:
			return fmt.Errorf("unknown database key: %s", key)
		}
	case "api":
		switch key {
		case "host":
			a.cfg.API.Host = value
		case "port":
			var port int
			if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
				return fmt.Errorf("invalid port value: %s", value)
			}
			a.cfg.API.Port = port
		default:
			return fmt.Errorf("unknown api key: %s", key)
		}
	case "cli":
		switch key {
		case "api_base_url":
			a.cfg.CLI.APIBaseURL = value
		case "api_key":
			a.cfg.CLI.APIKey = value
		default:
			return fmt.Errorf("unknown cli key: %s", key)
		}
	default:
		return fmt.Errorf("unknown section: %s", section)
	}

	return config.Save(a.cfg)
}

func (a *App) Run() error {
	fmt.Println("Interactive TUI mode - coming soon!")
	fmt.Println("Use --list, --add, or --delete for now")
	return nil
}

func (a *App) ListLinks() {
	fmt.Println("List links - coming soon!")
}

func (a *App) AddLink() {
	fmt.Println("Add link - coming soon!")
}

func (a *App) DeleteLink() {
	fmt.Println("Delete link - coming soon!")
}
