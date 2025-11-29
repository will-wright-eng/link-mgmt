package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"link-mgmt-go/pkg/cli"
	"link-mgmt-go/pkg/config"
	"link-mgmt-go/pkg/scraper"
	"link-mgmt-go/pkg/utils"
)

func main() {
	var (
		register  = flag.String("register", "", "Register a new user account (provide email)")
		scrapeURL = flag.String("scrape", "", "Scrape a URL to extract title and text content")

		// Config commands
		configShow = flag.Bool("config-show", false, "Show current configuration")
		configSet  = flag.String("config-set", "", "Set a config value (format: section.key=value)")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app := cli.NewApp(cfg)

	// Handle config commands first (don't need API connection)
	if *configShow {
		app.ShowConfig()
		return
	}
	if *configSet != "" {
		if err := app.SetConfig(*configSet); err != nil {
			log.Fatalf("failed to set config: %v", err)
		}
		fmt.Println("Configuration updated successfully")
		return
	}

	// Handle registration (needs API URL but not API key)
	if *register != "" {
		if cfg.CLI.BaseURL == "" {
			log.Fatalf("Base URL not configured. Set it with: --config-set cli.base_url=<url>")
		}
		if err := app.RegisterUser(*register); err != nil {
			log.Fatalf("failed to register user: %v", err)
		}
		return
	}

	// Handle scrape command (needs base URL but not API key)
	if *scrapeURL != "" {
		if cfg.CLI.BaseURL == "" {
			log.Fatalf("Base URL not configured. Set it with: --config-set cli.base_url=<url>")
		}

		// Validate URL format
		urlStr, err := utils.ValidateURL(*scrapeURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid URL: %v\n", err)
			os.Exit(1)
		}

		// Get scraper service
		scraperService := scraper.NewScraperService(cfg.CLI.BaseURL)

		// Check health first
		fmt.Print("⏳ Checking scraper service... ")
		if err := scraperService.CheckHealth(); err != nil {
			fmt.Println("✗")

			// Provide helpful guidance for connection errors
			errStr := err.Error()
			if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "dial tcp") {
				log.Fatalf("scraper service unavailable: %v\n\n"+
					"💡 The services are not running. To start them:\n"+
					"   From project root: make dev-upd\n"+
					"   Or: docker compose --profile dev up -d --build\n\n"+
					"This will start:\n"+
					"  - Nginx reverse proxy (port 80)\n"+
					"  - API service (api-dev)\n"+
					"  - Scraper service (scraper-dev)\n"+
					"  - PostgreSQL database", err)
			}

			log.Fatalf("scraper service unavailable: %v\n\nPlease check if the service is running", err)
		}
		fmt.Println("✓")

		// Scrape the URL
		fmt.Printf("⏳ Scraping URL... (this may take a few seconds)\n")
		timeout := cfg.CLI.ScrapeTimeout
		if timeout <= 0 {
			timeout = 30
		}
		result, err := scraperService.Scrape(urlStr, timeout*1000) // timeout in ms
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: scraping failed: %v\n", err)
			os.Exit(1)
		}

		if !result.Success {
			fmt.Fprintf(os.Stderr, "Error: scraping failed: %s\n", result.Error)
			os.Exit(1)
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
		return
	}

	// Interactive TUI mode
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// truncateText truncates text to a maximum length, adding ellipsis if truncated
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
