package cli

import (
	"fmt"

	"link-mgmt-go/pkg/config"
	"link-mgmt-go/pkg/db"
)

type App struct {
	db  *db.DB
	cfg *config.Config
}

func NewApp(db *db.DB, cfg *config.Config) *App {
	return &App{
		db:  db,
		cfg: cfg,
	}
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
