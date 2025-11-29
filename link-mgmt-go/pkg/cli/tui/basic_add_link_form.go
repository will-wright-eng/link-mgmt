package tui

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/utils"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// basicAddLinkForm is a simpler add-link flow without scraping, migrated from the old forms package.
type basicAddLinkForm struct {
	client     *client.Client
	urlInput   textinput.Model
	titleInput textinput.Model
	descInput  textinput.Model
	textInput  textarea.Model
	step       int // 0=URL, 1=Title, 2=Description, 3=Text, 4=Done
	err        error
	created    *models.Link
}

// NewBasicAddLinkForm creates a new basic add link form.
func NewBasicAddLinkForm(c *client.Client) tea.Model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com"
	urlInput.Focus()
	urlInput.CharLimit = 2048
	urlInput.Width = 60

	titleInput := textinput.New()
	titleInput.Placeholder = "Optional title"
	titleInput.CharLimit = 255
	titleInput.Width = 60

	descInput := textinput.New()
	descInput.Placeholder = "Optional description"
	descInput.CharLimit = 1000
	descInput.Width = 60

	textInput := textarea.New()
	textInput.Placeholder = "Optional text content (multi-line)"
	textInput.SetWidth(60)
	textInput.SetHeight(5)
	textInput.CharLimit = 10000

	return &basicAddLinkForm{
		client:     c,
		urlInput:   urlInput,
		titleInput: titleInput,
		descInput:  descInput,
		textInput:  textInput,
		step:       0,
	}
}

func (m *basicAddLinkForm) Init() tea.Cmd {
	return textinput.Blink
}

func (m *basicAddLinkForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			switch m.step {
			case 0:
				// Validate URL
				_, err := utils.ValidateURL(m.urlInput.Value())
				if err != nil {
					m.err = err
					return m, nil
				}
				m.err = nil
				m.step = 1
				m.titleInput.Focus()
				return m, textinput.Blink
			case 1:
				// Title is optional, move to description
				m.step = 2
				m.descInput.Focus()
				return m, textinput.Blink
			case 2:
				// Description is optional, move to text
				m.step = 3
				m.textInput.Focus()
				return m, textarea.Blink
			case 3:
				// Text is optional, submit the form
				return m, m.submit()
			}
		}

	case basicSubmitErrorMsg:
		m.err = msg.err
		return m, nil
	case basicSubmitSuccessMsg:
		m.created = msg.link
		m.step = 4
		return m, nil
	}

	var cmd tea.Cmd
	switch m.step {
	case 0:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 1:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 2:
		m.descInput, cmd = m.descInput.Update(msg)
	case 3:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m *basicAddLinkForm) View() string {
	switch m.step {
	case 4:
		// Success view
		if m.created != nil {
			title := ""
			if m.created.Title != nil && *m.created.Title != "" {
				title = *m.created.Title
			} else {
				title = "(no title)"
			}
			return fmt.Sprintf(
				"\n✓ Link created successfully!\n\n"+
					"  ID:          %s\n"+
					"  URL:         %s\n"+
					"  Title:       %s\n"+
					"  Created:     %s\n\n"+
					"Press any key to exit...",
				m.created.ID.String()[:8]+"...",
				m.created.URL,
				title,
				m.created.CreatedAt.Format("2006-01-02 15:04"),
			)
		}
		return "\n✓ Link created successfully!\n\nPress any key to exit..."
	default:
		var s strings.Builder
		s.WriteString("\nAdd New Link\n\n")

		switch m.step {
		case 0:
			s.WriteString("URL (required):\n")
			s.WriteString(m.urlInput.View())
			if m.err != nil {
				s.WriteString(fmt.Sprintf("\n\n❌ %s", m.err))
			}
			s.WriteString("\n\n(Press Enter to continue, Esc to cancel)")
		case 1:
			s.WriteString("✓ URL: " + m.urlInput.Value() + "\n\n")
			s.WriteString("Title (optional, press Enter to skip):\n")
			s.WriteString(m.titleInput.View())
			s.WriteString("\n\n(Press Enter to continue, Esc to cancel)")
		case 2:
			s.WriteString("✓ URL: " + m.urlInput.Value() + "\n")
			titleVal := m.titleInput.Value()
			if titleVal != "" {
				s.WriteString("✓ Title: " + titleVal + "\n")
			}
			s.WriteString("\nDescription (optional, press Enter to skip):\n")
			s.WriteString(m.descInput.View())
			s.WriteString("\n\n(Press Enter to continue, Esc to cancel)")
		case 3:
			s.WriteString("✓ URL: " + m.urlInput.Value() + "\n")
			titleVal := m.titleInput.Value()
			if titleVal != "" {
				s.WriteString("✓ Title: " + titleVal + "\n")
			}
			descVal := m.descInput.Value()
			if descVal != "" {
				s.WriteString("✓ Description: " + descVal + "\n")
			}
			s.WriteString("\nText (optional, press Enter to submit, Esc to cancel):\n")
			s.WriteString(m.textInput.View())
			s.WriteString("\n\n(Press Enter to submit, Esc to cancel)")
		}
		return s.String()
	}
}

type basicSubmitErrorMsg struct {
	err error
}

type basicSubmitSuccessMsg struct {
	link *models.Link
}

func (m *basicAddLinkForm) submit() tea.Cmd {
	return func() tea.Msg {
		urlStr, err := utils.ValidateURL(m.urlInput.Value())
		if err != nil {
			return basicSubmitErrorMsg{err: err}
		}

		titleStr := strings.TrimSpace(m.titleInput.Value())
		descStr := strings.TrimSpace(m.descInput.Value())
		textStr := strings.TrimSpace(m.textInput.Value())

		linkCreate := models.LinkCreate{URL: urlStr}

		if titleStr != "" {
			linkCreate.Title = &titleStr
		}
		if descStr != "" {
			linkCreate.Description = &descStr
		}
		if textStr != "" {
			linkCreate.Text = &textStr
		}

		created, err := m.client.CreateLink(linkCreate)
		if err != nil {
			return basicSubmitErrorMsg{err: err}
		}

		return basicSubmitSuccessMsg{link: created}
	}
}
