package tui

import (
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/scraper"

	tea "github.com/charmbracelet/bubbletea"
)

// rootModel is the Bubble Tea model that acts as an app shell for multiple flows.
// It presents a simple menu and then hands control to a specific flow model.
type rootModel struct {
	// Shared dependencies
	client         *client.Client
	scraperService *scraper.ScraperService
	scrapeTimeout  int

	// Current active flow (when nil, we are in the main menu)
	current tea.Model
}

// NewRootModel constructs the root app-shell model that can launch multiple flows.
func NewRootModel(
	apiClient *client.Client,
	scraperService *scraper.ScraperService,
	scrapeTimeoutSeconds int,
) tea.Model {
	if scrapeTimeoutSeconds <= 0 {
		scrapeTimeoutSeconds = 30
	}

	return &rootModel{
		client:         apiClient,
		scraperService: scraperService,
		scrapeTimeout:  scrapeTimeoutSeconds,
	}
}

func (m *rootModel) Init() tea.Cmd {
	// No async work on start; just render the menu.
	return nil
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If we have an active flow, delegate all messages to it.
	if m.current != nil {
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "1":
			// Basic add link flow (no scraping).
			m.current = NewBasicAddLinkForm(m.client)
			if initer, ok := m.current.(interface{ Init() tea.Cmd }); ok {
				return m, initer.Init()
			}
			return m, nil

		case "2":
			// Enhanced add link flow with scraping from pkg/cli/tui.
			m.current = NewAddLinkForm(m.client, m.scraperService, m.scrapeTimeout)
			if initer, ok := m.current.(interface{ Init() tea.Cmd }); ok {
				return m, initer.Init()
			}
			return m, nil

		case "3":
			// Delete link flow.
			m.current = NewDeleteLinkForm(m.client)
			if initer, ok := m.current.(interface{ Init() tea.Cmd }); ok {
				return m, initer.Init()
			}
			return m, nil

		case "4":
			// Combined manage links flow (list, view, delete, scrape).
			m.current = NewManageLinksModel(m.client, m.scraperService, m.scrapeTimeout)
			if initer, ok := m.current.(interface{ Init() tea.Cmd }); ok {
				return m, initer.Init()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *rootModel) View() string {
	// When a flow is active, defer to its view.
	if m.current != nil {
		return m.current.View()
	}

	var b strings.Builder

	b.WriteString(renderTitle("Link Management"))
	b.WriteString(renderDivider(60))
	b.WriteString("\n\n")
	b.WriteString(boldStyle.Render("Select an action:") + "\n\n")
	b.WriteString("  " + selectedMarkerStyle.Render("1)") + " Add link (basic)\n")
	b.WriteString("  " + selectedMarkerStyle.Render("2)") + " Add link (with scraping)\n")
	b.WriteString("  " + selectedMarkerStyle.Render("3)") + " Delete link\n")
	b.WriteString("  " + selectedMarkerStyle.Render("4)") + " Manage links (list, view, delete, scrape)\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press the number of an option, or 'q' / Esc to quit.") + "\n")

	return b.String()
}
