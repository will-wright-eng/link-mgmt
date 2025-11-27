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
