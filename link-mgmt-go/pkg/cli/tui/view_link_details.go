package tui

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
)

// viewLinkDetailsModel is a Bubble Tea model that lists links and allows viewing
// full details of a selected link.
type viewLinkDetailsModel struct {
	client *client.Client

	links    []models.Link
	selected int
	step     int // 0=selecting, 1=viewing details
	err      error
}

const (
	stepSelectLink = iota
	stepViewDetails
)

// viewLinksLoadedMsg is emitted when links have been fetched.
type viewLinksLoadedMsg struct {
	links []models.Link
	err   error
}

// NewViewLinkDetailsModel creates a new view-link-details flow.
func NewViewLinkDetailsModel(c *client.Client) tea.Model {
	return &viewLinkDetailsModel{
		client: c,
		step:   stepSelectLink,
	}
}

func (m *viewLinkDetailsModel) Init() tea.Cmd {
	return func() tea.Msg {
		links, err := m.client.ListLinks()
		return viewLinksLoadedMsg{links: links, err: err}
	}
}

func (m *viewLinkDetailsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case viewLinksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.links = msg.links
		if len(m.links) == 0 {
			m.err = fmt.Errorf("no links available to view")
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case stepSelectLink:
			return m.handleSelectKey(msg)
		case stepViewDetails:
			switch msg.String() {
			case "ctrl+c", "q", "esc", "b":
				// Go back to selection or quit
				m.step = stepSelectLink
				return m, nil
			case "enter":
				// Also allow Enter to go back
				m.step = stepSelectLink
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *viewLinkDetailsModel) handleSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handleQuitKeys(msg.String()) {
		return m, tea.Quit
	}
	if newSelected, handled := handleListNavigation(msg.String(), m.selected, len(m.links)); handled {
		m.selected = newSelected
		return m, nil
	}
	if msg.String() == "enter" {
		if m.selected < len(m.links) {
			m.step = stepViewDetails
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m *viewLinkDetailsModel) View() string {
	if m.err != nil {
		return renderErrorView(m.err)
	}

	switch m.step {
	case stepSelectLink:
		return m.renderSelect()
	case stepViewDetails:
		return m.renderDetails()
	}

	return ""
}

func (m *viewLinkDetailsModel) renderSelect() string {
	if len(m.links) == 0 {
		return renderEmptyState("No links found.")
	}

	s := renderLinkList(m.links, m.selected, "View Link Details", "Select a link:")
	s += helpStyle.Render("(Use ↑/↓ or j/k to navigate, Enter to view details, Esc to quit)") + "\n"
	return s
}

func (m *viewLinkDetailsModel) renderDetails() string {
	if m.selected >= len(m.links) {
		return renderErrorView(fmt.Errorf("invalid selection"))
	}

	link := m.links[m.selected]
	var b strings.Builder

	b.WriteString(renderTitle("Link Details"))
	b.WriteString(renderDivider(60))
	b.WriteString("\n\n")

	// Use helper for full details
	b.WriteString(renderLinkDetailsFull(&link))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("(Press Enter, 'b', Esc, or 'q' to go back)") + "\n")

	return b.String()
}
