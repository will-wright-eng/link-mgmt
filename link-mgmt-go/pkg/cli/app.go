package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/cli/tui"
	"link-mgmt-go/pkg/config"
	"link-mgmt-go/pkg/models"
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

	if a.cfg.CLI.BaseURL == "" {
		return nil, fmt.Errorf("base URL not configured (set cli.base_url)")
	}
	if a.cfg.CLI.APIKey == "" {
		return nil, fmt.Errorf("API key not configured")
	}

	a.client = client.NewClient(a.cfg.CLI.BaseURL, a.cfg.CLI.APIKey)
	return a.client, nil
}

// getClientForRegistration returns an HTTP client without API key (for registration)
func (a *App) getClientForRegistration() (*client.Client, error) {
	if a.cfg.CLI.BaseURL == "" {
		return nil, fmt.Errorf("base URL not configured (set cli.base_url)")
	}
	// Use empty API key for registration endpoint (doesn't require auth)
	return client.NewClient(a.cfg.CLI.BaseURL, ""), nil
}

// SaveLink saves a link to the API
func (a *App) SaveLink(url string) error {
	apiClient, err := a.getClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	linkCreate := models.LinkCreate{
		URL: url,
	}

	created, err := apiClient.CreateLink(linkCreate)
	if err != nil {
		return fmt.Errorf("failed to save link: %w", err)
	}

	fmt.Println("✓ Link saved successfully!")
	fmt.Printf("  URL: %s\n", created.URL)
	if created.Title != nil && *created.Title != "" {
		fmt.Printf("  Title: %s\n", *created.Title)
	}
	fmt.Printf("  ID: %s\n", created.ID.String())

	return nil
}

func (a *App) Run() error {
	apiClient, err := a.getClient()
	if err != nil {
		return err
	}

	// No scraper service needed - API handles it
	model := tui.NewRootModel(apiClient, a.cfg.CLI.ScrapeTimeout)
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}
