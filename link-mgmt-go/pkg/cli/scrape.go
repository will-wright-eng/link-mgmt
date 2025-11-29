package cli

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/utils"
)

// HandleScrapeCommand handles the --scrape command to extract content from a URL
func (a *App) HandleScrapeCommand(urlStr string) error {
	// Validate URL format
	var err error
	urlStr, err = utils.ValidateURL(urlStr)
	if err != nil {
		return err
	}

	// Get scraper service
	scraperService, err := a.getScraperService()
	if err != nil {
		return fmt.Errorf("failed to initialize scraper service: %w", err)
	}

	// Check health first
	fmt.Print("⏳ Checking scraper service... ")
	if err := scraperService.CheckHealth(); err != nil {
		fmt.Println("✗")

		// Provide helpful guidance for connection errors
		errStr := err.Error()
		if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "dial tcp") {
			return fmt.Errorf("scraper service unavailable: %w\n\n"+
				"💡 The services are not running. To start them:\n"+
				"   From project root: make dev-upd\n"+
				"   Or: docker compose --profile dev up -d --build\n\n"+
				"This will start:\n"+
				"  - Nginx reverse proxy (port 80)\n"+
				"  - API service (api-dev)\n"+
				"  - Scraper service (scraper-dev)\n"+
				"  - PostgreSQL database", err)
		}

		return fmt.Errorf("scraper service unavailable: %w\n\nPlease check if the service is running", err)
	}
	fmt.Println("✓")

	// Scrape the URL
	fmt.Printf("⏳ Scraping URL... (this may take a few seconds)\n")
	timeout := a.cfg.CLI.ScrapeTimeout
	result, err := scraperService.Scrape(urlStr, timeout*1000) // timeout in ms
	if err != nil {
		return fmt.Errorf("scraping failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("scraping failed: %s", result.Error)
	}

	// Display results
	fmt.Println("\n✓ Scraping successful!")
	fmt.Printf("\nURL: %s\n", result.URL)
	if result.Title != "" {
		fmt.Printf("Title: %s\n", result.Title)
	} else {
		fmt.Println("Title: (no title)")
	}
	if result.Text != "" {
		truncated := truncateText(result.Text, 500)
		fmt.Printf("Text: %s\n", truncated)
		if len(result.Text) > 500 {
			fmt.Printf("\n(Text truncated, full length: %d characters)\n", len(result.Text))
		}
	} else {
		fmt.Println("Text: (no text content)")
	}

	return nil
}

// truncateText truncates text to a maximum length, adding ellipsis if truncated
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
