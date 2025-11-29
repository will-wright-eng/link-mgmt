package cli

import (
	"fmt"
	"os"

	linkformatter "link-mgmt-go/pkg/cli/links"
	"link-mgmt-go/pkg/cli/tui"
	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/utils"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) ListLinks() {
	apiClient, err := a.getClient()
	if err != nil {
		linkformatter.WriteToStderr(linkformatter.FormatErrorMessage(err))
		os.Exit(1)
	}

	links, err := apiClient.ListLinks()
	if err != nil {
		linkformatter.WriteToStderr(linkformatter.FormatErrorMessage(fmt.Errorf("fetching links: %w", err)))
		os.Exit(1)
	}

	// Display links in a polished table format
	output := linkformatter.FormatTableOutput(links)
	linkformatter.WriteToStdout(output)
}

// AddLink creates a new link with the provided URL
// If url is empty, it launches the interactive TUI form
func (a *App) AddLink(url string) error {
	apiClient, err := a.getClient()
	if err != nil {
		return err
	}

	// If no URL provided, launch interactive TUI form
	if url == "" {
		form := tui.NewBasicAddLinkForm(apiClient)
		p := tea.NewProgram(form)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running form: %w", err)
		}
		return nil
	}

	// Validate URL
	validatedURL, err := utils.ValidateURL(url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Create link with just the URL
	linkCreate := models.LinkCreate{
		URL: validatedURL,
	}

	created, err := apiClient.CreateLink(linkCreate)
	if err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	}

	// Print polished success message
	output := linkformatter.FormatSuccessMessage(created)
	fmt.Print(output)

	return nil
}

// DeleteLink prompts the user to select and delete a link
func (a *App) DeleteLink() {
	apiClient, err := a.getClient()
	if err != nil {
		linkformatter.WriteToStderr(linkformatter.FormatErrorMessage(err))
		os.Exit(1)
	}

	// Create and run the delete link TUI form
	selector := tui.NewDeleteLinkForm(apiClient)
	p := tea.NewProgram(selector)
	if _, err := p.Run(); err != nil {
		linkformatter.WriteToStderr(linkformatter.FormatErrorMessage(fmt.Errorf("running selector: %w", err)))
		os.Exit(1)
	}
}

// ViewLinkDetails prompts the user to select a link and view all its fields
func (a *App) ViewLinkDetails() {
	apiClient, err := a.getClient()
	if err != nil {
		linkformatter.WriteToStderr(linkformatter.FormatErrorMessage(err))
		os.Exit(1)
	}

	// Create and run the view link details TUI form
	viewer := tui.NewViewLinkDetailsModel(apiClient)
	p := tea.NewProgram(viewer)
	if _, err := p.Run(); err != nil {
		linkformatter.WriteToStderr(linkformatter.FormatErrorMessage(fmt.Errorf("running viewer: %w", err)))
		os.Exit(1)
	}
}
