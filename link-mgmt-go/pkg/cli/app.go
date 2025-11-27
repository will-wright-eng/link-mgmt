package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/config"

	"github.com/pelletier/go-toml/v2"
)

type App struct {
	cfg    *config.Config
	client *client.Client
}

func NewApp(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}

// getClient returns the HTTP client, creating it if necessary
func (a *App) getClient() (*client.Client, error) {
	if a.client != nil {
		return a.client, nil
	}

	if a.cfg.CLI.APIBaseURL == "" {
		return nil, fmt.Errorf("API base URL not configured")
	}
	if a.cfg.CLI.APIKey == "" {
		return nil, fmt.Errorf("API key not configured")
	}

	a.client = client.NewClient(a.cfg.CLI.APIBaseURL, a.cfg.CLI.APIKey)
	return a.client, nil
}

// getClientForRegistration returns an HTTP client without API key (for registration)
func (a *App) getClientForRegistration() (*client.Client, error) {
	if a.cfg.CLI.APIBaseURL == "" {
		return nil, fmt.Errorf("API base URL not configured")
	}
	// Use empty API key for registration endpoint (doesn't require auth)
	return client.NewClient(a.cfg.CLI.APIBaseURL, ""), nil
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
	apiClient, err := a.getClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	links, err := apiClient.ListLinks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching links: %v\n", err)
		os.Exit(1)
	}

	if len(links) == 0 {
		fmt.Println("No links found.")
		return
	}

	// Display links in a table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tURL\tTitle\tCreated")
	fmt.Fprintln(w, "───\t───\t───\t───")

	for _, link := range links {
		title := ""
		if link.Title != nil && *link.Title != "" {
			title = *link.Title
		} else {
			title = "(no title)"
		}

		// Truncate URL if too long
		url := link.URL
		if len(url) > 50 {
			url = url[:47] + "..."
		}

		// Format date
		created := link.CreatedAt.Format("2006-01-02 15:04")

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			link.ID.String()[:8]+"...",
			url,
			title,
			created,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d link(s)\n", len(links))
}

func (a *App) AddLink() {
	fmt.Println("Add link - coming soon!")
}

func (a *App) DeleteLink() {
	fmt.Println("Delete link - coming soon!")
}

// RegisterUser creates a new user account and saves the API key
func (a *App) RegisterUser(email string) error {
	apiClient, err := a.getClientForRegistration()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	user, err := apiClient.CreateUser(email)
	if err != nil {
		// Check for common errors and provide helpful messages
		errStr := err.Error()
		if strings.Contains(errStr, "relation \"users\" does not exist") || strings.Contains(errStr, "does not exist") {
			return fmt.Errorf(`database table 'users' does not exist. Please run migrations first.

To run migrations:
  From project root (Docker):  make migrate`)
		}
		// Don't wrap the error again since it already contains "failed to register user"
		return err
	}

	// Save API key to config
	a.cfg.CLI.APIKey = user.APIKey
	if err := config.Save(a.cfg); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}

	// Update the client with the new API key
	a.client = client.NewClient(a.cfg.CLI.APIBaseURL, user.APIKey)

	fmt.Println("✓ User registered successfully!")
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  User ID: %s\n", user.ID.String())
	fmt.Printf("  API key saved to config automatically\n")
	fmt.Println("\n⚠️  Save this API key securely (it won't be shown again):")
	fmt.Printf("  %s\n", user.APIKey)

	return nil
}
