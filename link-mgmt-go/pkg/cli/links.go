package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"link-mgmt-go/pkg/cli/forms"

	tea "github.com/charmbracelet/bubbletea"
)

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

// AddLink prompts the user for link information and creates a new link
func (a *App) AddLink() {
	apiClient, err := a.getClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create and run the add link form
	form := forms.NewAddLinkForm(apiClient)
	p := tea.NewProgram(form)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running form: %v\n", err)
		os.Exit(1)
	}
}

// DeleteLink prompts the user to select and delete a link
func (a *App) DeleteLink() {
	apiClient, err := a.getClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create and run the delete link selector
	selector := forms.NewDeleteLinkSelector(apiClient)
	p := tea.NewProgram(selector)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running selector: %v\n", err)
		os.Exit(1)
	}
}
