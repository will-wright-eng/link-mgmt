package tui

import (
	"strings"

	"link-mgmt-go/pkg/cli/client"

	tea "github.com/charmbracelet/bubbletea"
)

// MenuNavigationMsg signals that we should return to the root menu
type MenuNavigationMsg struct{}

// rootModel is the Bubble Tea model that acts as an app shell for multiple flows.
// It presents a simple menu and then hands control to a specific flow model.
type rootModel struct {
	// Shared dependencies
	client        *client.Client
	scrapeTimeout int

	// Current active flow (when nil, we are in the main menu)
	current tea.Model
}

// IsDelegating returns true if the rootModel is currently delegating to a child flow
func (m *rootModel) IsDelegating() bool {
	return m.current != nil
}

// NewRootModel constructs the root app-shell model that can launch multiple flows.
func NewRootModel(
	apiClient *client.Client,
	scrapeTimeoutSeconds int,
) tea.Model {
	if scrapeTimeoutSeconds <= 0 {
		scrapeTimeoutSeconds = 30
	}

	root := &rootModel{
		client:        apiClient,
		scrapeTimeout: scrapeTimeoutSeconds,
	}

	// Wrap with viewport (simple responsive, no scrolling needed for menu)
	return NewViewportWrapper(root, ViewportConfig{
		Title:       "Link Management",
		ShowHeader:  true,
		ShowFooter:  true,
		UseViewport: false, // Menu is short, no scrolling needed
		EnableHelp:  true,
		EnableMenu:  false, // Already at menu
		HelpContent: RootMenuHelpContent,
		MinWidth:    60,
		MinHeight:   10,
	})
}

func (m *rootModel) Init() tea.Cmd {
	// No async work on start; just render the menu.
	return nil
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle menu navigation message before delegating
	if _, ok := msg.(MenuNavigationMsg); ok {
		// Return to menu (clear current flow)
		m.current = nil
		return m, nil
	}

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
			// Add link flow (scraping handled by API).
			m.current = NewAddLinkForm(m.client, m.scrapeTimeout)
			if initer, ok := m.current.(interface{ Init() tea.Cmd }); ok {
				return m, initer.Init()
			}
			return m, nil

		case "2":
			// Manage links flow (list, view, delete, enrich).
			m.current = NewManageLinksModel(m.client, m.scrapeTimeout)
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

	// Title is rendered by the viewport wrapper header
	b.WriteString(renderDivider(60))
	b.WriteString("\n\n")
	b.WriteString(boldStyle.Render("Select an action:") + "\n\n")
	b.WriteString("  " + selectedMarkerStyle.Render("1)") + " Add link (with scraping)\n")
	b.WriteString("  " + selectedMarkerStyle.Render("2)") + " Manage links (list, view, delete, scrape)\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press the number of an option, or 'q' / Esc to quit.") + "\n")

	return b.String()
}
