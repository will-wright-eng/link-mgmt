package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/cli/tui"
	"link-mgmt-go/pkg/config"
	"link-mgmt-go/pkg/scraper"
)

type App struct {
	cfg            *config.Config
	client         *client.Client
	scraperService *scraper.ScraperService
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

// getScraperService returns the scraper service, creating it if necessary
func (a *App) getScraperService() (*scraper.ScraperService, error) {
	if a.scraperService != nil {
		return a.scraperService, nil
	}

	if a.cfg.CLI.BaseURL == "" {
		return nil, fmt.Errorf("base URL not configured (set cli.base_url)")
	}

	// Use same base URL as API (nginx routes /scrape to scraper service)
	a.scraperService = scraper.NewScraperService(a.cfg.CLI.BaseURL)
	return a.scraperService, nil
}

func (a *App) Run() error {
	apiClient, err := a.getClient()
	if err != nil {
		return err
	}

	scraperService, err := a.getScraperService()
	if err != nil {
		return err
	}

	model := tui.NewAddLinkForm(apiClient, scraperService, a.cfg.CLI.ScrapeTimeout)
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}
