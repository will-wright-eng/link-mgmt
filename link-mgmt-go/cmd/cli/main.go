package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"link-mgmt-go/pkg/cli"
	"link-mgmt-go/pkg/config"
)

func main() {
	var (
		listMode   = flag.Bool("list", false, "List all links")
		addMode    = flag.Bool("add", false, "Add a new link")
		deleteMode = flag.Bool("delete", false, "Delete a link")
		register   = flag.String("register", "", "Register a new user account (provide email)")

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
		if cfg.CLI.APIBaseURL == "" {
			log.Fatalf("API base URL not configured. Set it with: --config-set cli.api_base_url=<url>")
		}
		if err := app.RegisterUser(*register); err != nil {
			log.Fatalf("failed to register user: %v", err)
		}
		return
	}

	// For operations that need API connection
	if *listMode || *addMode || *deleteMode {
		// Validate API configuration
		if cfg.CLI.APIBaseURL == "" {
			log.Fatalf("API base URL not configured. Set it with: --config-set cli.api_base_url=<url>")
		}
		if cfg.CLI.APIKey == "" {
			log.Fatalf("API key not configured.\n\nTo get started:\n  1. Register a new account: --register <email>\n  2. Or set API key manually: --config-set cli.api_key=<key>")
		}

		if *listMode {
			app.ListLinks()
		} else if *addMode {
			app.AddLink()
		} else if *deleteMode {
			app.DeleteLink()
		}
		return
	}

	// Interactive TUI mode (will use API client when implemented)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
