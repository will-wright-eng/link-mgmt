package tui

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// deleteLinkForm is a Bubble Tea model for selecting a link to delete, migrated from the old forms package.
type deleteLinkForm struct {
	client   *client.Client
	links    []models.Link
	selected int
	step     int // 0=selecting, 1=confirming, 2=done
	err      error
	confirm  textinput.Model
}

// NewDeleteLinkForm creates a new delete link form.
func NewDeleteLinkForm(c *client.Client) tea.Model {
	confirm := textinput.New()
	confirm.Placeholder = "y/N"
	confirm.CharLimit = 1
	confirm.Width = 10

	return &deleteLinkForm{
		client:  c,
		links:   []models.Link{},
		step:    0,
		confirm: confirm,
	}
}

// deleteLinksLoadedMsg is emitted when links are fetched for deletion.
type deleteLinksLoadedMsg struct {
	links []models.Link
	err   error
}

type deleteErrorMsg struct {
	err error
}

type deleteSuccessMsg struct{}

func (m *deleteLinkForm) Init() tea.Cmd {
	return func() tea.Msg {
		links, err := m.client.ListLinks()
		return deleteLinksLoadedMsg{links: links, err: err}
	}
}

func (m *deleteLinkForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deleteLinksLoadedMsg:
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
			if handleQuitKeys(msg.String()) {
				return m, tea.Quit
			}
			if newSelected, handled := handleListNavigation(msg.String(), m.selected, len(m.links)); handled {
				m.selected = newSelected
				return m, nil
			}
			if msg.String() == "enter" {
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

func (m *deleteLinkForm) View() string {
	if m.err != nil {
		return renderErrorView(m.err)
	}

	switch m.step {
	case 2:
		return renderSuccessView("Link deleted successfully!")
	case 0:
		// Selection view
		s := renderLinkList(m.links, m.selected, "Delete Link", "Select a link to delete:")
		s += helpStyle.Render("(Use ↑/↓ or j/k to navigate, Enter to select, Esc to cancel)")
		return s
	case 1:
		// Confirmation view
		var s strings.Builder
		s.WriteString(renderTitle("Delete Link"))
		s.WriteString(warningStyle.Render("⚠️  Confirm Deletion") + "\n\n")

		link := m.links[m.selected]
		title := formatLinkTitle(link)

		s.WriteString(boldStyle.Render("Are you sure you want to delete:"))
		s.WriteString("\n")
		s.WriteString(fmt.Sprintf("  %s\n", linkTitleStyle.Render(title)))
		s.WriteString(fieldLabelStyle.Render("URL:"))
		s.WriteString(fmt.Sprintf(" %s\n\n", link.URL))

		s.WriteString(boldStyle.Render("Confirm (y/N):"))
		s.WriteString(" ")
		s.WriteString(m.confirm.View())
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("(Press Enter to confirm, Esc to cancel)"))
		return s.String()
	}

	return ""
}

func (m *deleteLinkForm) deleteLink() tea.Cmd {
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
