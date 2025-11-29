package tui

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
		return m, nil
	case "down", "j":
		if m.selected < len(m.links)-1 {
			m.selected++
		}
		return m, nil
	case "enter":
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
		return fmt.Sprintf("\n❌ Error: %v\n\nPress any key to exit...", m.err)
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
		return "\nNo links found.\n\nPress any key to exit..."
	}

	var b strings.Builder
	b.WriteString("\nSelect a link to view details\n")
	b.WriteString(strings.Repeat("─", 35))
	b.WriteString("\n\n")

	for i, link := range m.links {
		marker := " "
		if i == m.selected {
			marker = "→"
		}

		title := "(no title)"
		if link.Title != nil && *link.Title != "" {
			title = *link.Title
		}

		url := link.URL
		if len(url) > 60 {
			url = url[:57] + "..."
		}

		style := lipgloss.NewStyle()
		if i == m.selected {
			style = style.Bold(true)
		}

		line := fmt.Sprintf("%s %s\n    %s\n",
			marker,
			style.Render(title),
			url,
		)
		b.WriteString(line)
	}

	b.WriteString("\n")
	b.WriteString("(Use ↑/↓ or j/k to navigate, Enter to view details, Esc to quit)\n")

	return b.String()
}

func (m *viewLinkDetailsModel) renderDetails() string {
	if m.selected >= len(m.links) {
		return "\n❌ Invalid selection\n\nPress any key to exit..."
	}

	link := m.links[m.selected]
	var b strings.Builder

	b.WriteString("\nLink Details\n")
	b.WriteString(strings.Repeat("─", 12))
	b.WriteString("\n\n")

	// ID
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("ID:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.ID.String()))

	// User ID
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("User ID:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.UserID.String()))

	// URL
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("URL:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.URL))

	// Title
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Title:"))
	if link.Title != nil && *link.Title != "" {
		b.WriteString(fmt.Sprintf(" %s\n", *link.Title))
	} else {
		b.WriteString(" (not set)\n")
	}

	// Description
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Description:"))
	if link.Description != nil && *link.Description != "" {
		desc := *link.Description
		// Wrap long descriptions
		if len(desc) > 80 {
			words := strings.Fields(desc)
			line := ""
			for _, word := range words {
				if len(line)+len(word)+1 > 80 {
					b.WriteString(fmt.Sprintf(" %s\n", line))
					line = word
				} else {
					if line != "" {
						line += " "
					}
					line += word
				}
			}
			if line != "" {
				b.WriteString(fmt.Sprintf(" %s\n", line))
			}
		} else {
			b.WriteString(fmt.Sprintf(" %s\n", desc))
		}
	} else {
		b.WriteString(" (not set)\n")
	}

	// Text
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Text:"))
	if link.Text != nil && *link.Text != "" {
		text := *link.Text
		// For long text, show first portion and indicate length
		if len(text) > 500 {
			preview := text[:500]
			// Try to break at word boundary
			if lastSpace := strings.LastIndex(preview, " "); lastSpace > 400 {
				preview = preview[:lastSpace]
			}
			b.WriteString(fmt.Sprintf(" %s...\n", preview))
			b.WriteString(fmt.Sprintf("  (truncated, full length: %d characters)\n", len(text)))
		} else {
			// Wrap text content
			words := strings.Fields(text)
			line := ""
			for _, word := range words {
				if len(line)+len(word)+1 > 80 {
					b.WriteString(fmt.Sprintf(" %s\n", line))
					line = word
				} else {
					if line != "" {
						line += " "
					}
					line += word
				}
			}
			if line != "" {
				b.WriteString(fmt.Sprintf(" %s\n", line))
			}
		}
	} else {
		b.WriteString(" (not set)\n")
	}

	// Created At
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Created At:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.CreatedAt.Format("2006-01-02 15:04:05")))

	// Updated At
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Updated At:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.UpdatedAt.Format("2006-01-02 15:04:05")))

	b.WriteString("\n")
	b.WriteString("(Press Enter, 'b', Esc, or 'q' to go back)\n")

	return b.String()
}
