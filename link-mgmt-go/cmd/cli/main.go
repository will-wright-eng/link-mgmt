package main

import (
	"context"
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

	// Handle config commands first (don't need database)
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

	// For operations that need database, connect lazily
	if *listMode || *addMode || *deleteMode {
		ctx := context.Background()
		database, err := app.ConnectDB(ctx)
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer database.Close()

		if *listMode {
			app.ListLinks()
		} else if *addMode {
			app.AddLink()
		} else if *deleteMode {
			app.DeleteLink()
		}
		return
	}

	// Interactive TUI mode (will connect to DB when needed)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
