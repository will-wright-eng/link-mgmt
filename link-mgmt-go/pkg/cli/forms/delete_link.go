package forms

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// deleteLinkSelector is a Bubble Tea model for selecting a link to delete
type deleteLinkSelector struct {
	client   *client.Client
	links    []models.Link
	selected int
	step     int // 0=selecting, 1=confirming, 2=done
	err      error
	confirm  textinput.Model
}

// NewDeleteLinkSelector creates a new delete link selector
func NewDeleteLinkSelector(client *client.Client) *deleteLinkSelector {
	confirm := textinput.New()
	confirm.Placeholder = "y/N"
	confirm.CharLimit = 1
	confirm.Width = 10

	return &deleteLinkSelector{
		client:  client,
		links:   []models.Link{},
		step:    0,
		confirm: confirm,
	}
}

func (m *deleteLinkSelector) Init() tea.Cmd {
	return m.loadLinks
}

func (m *deleteLinkSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case linksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.links = msg.links
		if len(m.links) == 0 {
			m.err = fmt.Errorf("no links available to delete")
			return m, tea.Quit
		}
		return m, nil

	case deleteErrorMsg:
		m.err = msg.err
		return m, tea.Quit
	case deleteSuccessMsg:
		m.step = 2
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case 0:
			// Selection step
			switch msg.String() {
			case "ctrl+c", "esc":
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
					m.step = 1
					m.confirm.Focus()
					return m, textinput.Blink
				}
				return m, nil
			}
		case 1:
			// Confirmation step
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter":
				answer := strings.ToLower(strings.TrimSpace(m.confirm.Value()))
				if answer == "y" || answer == "yes" {
					return m, m.deleteLink()
				}
				// Cancelled
				return m, tea.Quit
			default:
				var cmd tea.Cmd
				m.confirm, cmd = m.confirm.Update(msg)
				return m, cmd
			}
		case 2:
			// Done step - any key exits
			return m, tea.Quit
		}
	}

	switch m.step {
	case 1:
		var cmd tea.Cmd
		m.confirm, cmd = m.confirm.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *deleteLinkSelector) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n❌ Error: %v\n\nPress any key to exit...", m.err)
	}

	switch m.step {
	case 2:
		return "\n✓ Link deleted successfully!\n\nPress any key to exit..."
	case 0:
		// Selection view
		var s strings.Builder
		s.WriteString("\nSelect a link to delete:\n\n")
		for i, link := range m.links {
			marker := " "
			if i == m.selected {
				marker = "→"
			}

			title := ""
			if link.Title != nil && *link.Title != "" {
				title = *link.Title
			} else {
				title = "(no title)"
			}

			url := link.URL
			if len(url) > 50 {
				url = url[:47] + "..."
			}

			style := lipgloss.NewStyle()
			if i == m.selected {
				style = style.Bold(true)
			}

			s.WriteString(fmt.Sprintf("%s %s - %s\n", marker, style.Render(title), url))
		}
		s.WriteString("\n(Use ↑/↓ or j/k to navigate, Enter to select, Esc to cancel)")
		return s.String()
	case 1:
		// Confirmation view
		var s strings.Builder
		s.WriteString("\nSelect a link to delete:\n\n")
		link := m.links[m.selected]
		title := ""
		if link.Title != nil && *link.Title != "" {
			title = *link.Title
		} else {
			title = "(no title)"
		}

		s.WriteString(fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", title))
		s.WriteString(fmt.Sprintf("URL: %s\n\n", link.URL))
		s.WriteString("Confirm (y/N): ")
		s.WriteString(m.confirm.View())
		s.WriteString("\n\n(Press Enter to confirm, Esc to cancel)")
		return s.String()
	}

	return ""
}

func (m *deleteLinkSelector) loadLinks() tea.Msg {
	links, err := m.client.ListLinks()
	if err != nil {
		return linksLoadedMsg{err: err}
	}
	return linksLoadedMsg{links: links}
}

func (m *deleteLinkSelector) deleteLink() tea.Cmd {
	return func() tea.Msg {
		if m.selected >= len(m.links) {
			return deleteErrorMsg{err: fmt.Errorf("invalid selection")}
		}

		link := m.links[m.selected]
		err := m.client.DeleteLink(link.ID)
		if err != nil {
			return deleteErrorMsg{err: err}
		}

		return deleteSuccessMsg{}
	}
}
