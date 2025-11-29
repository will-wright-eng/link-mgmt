package tui

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listLinksModel is a simple Bubble Tea model that loads and displays links.
// It follows the same dependency-injection pattern as other flows.
type listLinksModel struct {
	client *client.Client

	links []models.Link
	err   error
	ready bool
}

// NewListLinksModel creates a new list-links flow.
func NewListLinksModel(c *client.Client) tea.Model {
	return &listLinksModel{
		client: c,
	}
}

// listLoadedMsg is emitted when links have been fetched.
type listLoadedMsg struct {
	links []models.Link
	err   error
}

func (m *listLinksModel) Init() tea.Cmd {
	return func() tea.Msg {
		links, err := m.client.ListLinks()
		return listLoadedMsg{links: links, err: err}
	}
}

func (m *listLinksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case listLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.links = msg.links
		}
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc", "enter":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *listLinksModel) View() string {
	if !m.ready {
		return "\nLoading links...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("\n❌ Error loading links: %v\n\nPress any key to exit...", m.err)
	}

	if len(m.links) == 0 {
		return "\nNo links found.\n\nPress any key to exit..."
	}

	var b strings.Builder
	b.WriteString("\nYour Links\n")
	b.WriteString(strings.Repeat("─", 10))
	b.WriteString("\n\n")

	for _, link := range m.links {
		title := "(no title)"
		if link.Title != nil && *link.Title != "" {
			title = *link.Title
		}

		url := link.URL
		if len(url) > 60 {
			url = url[:57] + "..."
		}

		idShort := link.ID.String()[:8] + "..."
		line := fmt.Sprintf("%s  %s\n    %s\n",
			lipgloss.NewStyle().Bold(true).Render(idShort),
			title,
			url,
		)
		b.WriteString(line)
	}

	b.WriteString("\nPress Enter, Esc, or q to exit.\n")
	return b.String()
}
