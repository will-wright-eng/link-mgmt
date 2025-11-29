package tui

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	linkformatter "link-mgmt-go/pkg/cli/links"
	"link-mgmt-go/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
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
		return renderLoadingState("Loading links...")
	}

	if m.err != nil {
		return renderErrorView(fmt.Errorf("Error loading links: %v", m.err))
	}

	if len(m.links) == 0 {
		return renderEmptyState("No links found.")
	}

	var b strings.Builder
	b.WriteString(renderTitle("Your Links"))
	b.WriteString(renderDivider(60))
	b.WriteString("\n\n")

	for i, link := range m.links {
		title := linkformatter.GetTitle(link)
		url := linkformatter.TruncateURL(link.URL, 60)
		idShort := linkformatter.ShortenID(link.ID)

		// Add spacing between items (except after last)
		if i > 0 {
			b.WriteString("\n")
		}

		// ID with styling
		b.WriteString(linkIDStyle.Render(idShort))
		b.WriteString("  ")

		// Title with styling
		b.WriteString(linkTitleStyle.Render(title))
		b.WriteString("\n  ")

		// URL with styling (indented)
		b.WriteString(linkURLStyle.Render(url))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press Enter, Esc, or q to exit."))
	b.WriteString("\n")

	return b.String()
}
