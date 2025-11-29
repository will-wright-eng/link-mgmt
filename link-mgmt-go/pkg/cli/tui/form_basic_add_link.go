package tui

import (
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
			var b strings.Builder
			b.WriteString("\n")
			b.WriteString(renderSuccess("Link created successfully!"))
			b.WriteString("\n\n")
			b.WriteString(renderLinkDetails(m.created, false))
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("Press any key to exit...") + "\n")
			return b.String()
		}
		return renderSuccessView("Link created successfully!")
	default:
		var s strings.Builder
		s.WriteString(renderTitle("Add New Link"))

		switch m.step {
		case 0:
			s.WriteString(fieldLabelStyle.Render("URL (required):"))
			s.WriteString("\n")
			s.WriteString(m.urlInput.View())
			if m.err != nil {
				s.WriteString("\n\n")
				s.WriteString(renderInlineError(m.err))
			}
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("(Press Enter to continue, Esc to cancel)"))
		case 1:
			s.WriteString(successStyle.Render("✓"))
			s.WriteString(" ")
			s.WriteString(fieldLabelStyle.Render("URL:"))
			s.WriteString(" " + m.urlInput.Value() + "\n\n")
			s.WriteString(fieldLabelStyle.Render("Title (optional, press Enter to skip):"))
			s.WriteString("\n")
			s.WriteString(m.titleInput.View())
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("(Press Enter to continue, Esc to cancel)"))
		case 2:
			s.WriteString(successStyle.Render("✓"))
			s.WriteString(" ")
			s.WriteString(fieldLabelStyle.Render("URL:"))
			s.WriteString(" " + m.urlInput.Value() + "\n")
			titleVal := m.titleInput.Value()
			if titleVal != "" {
				s.WriteString(successStyle.Render("✓"))
				s.WriteString(" ")
				s.WriteString(fieldLabelStyle.Render("Title:"))
				s.WriteString(" " + titleVal + "\n")
			}
			s.WriteString("\n")
			s.WriteString(fieldLabelStyle.Render("Description (optional, press Enter to skip):"))
			s.WriteString("\n")
			s.WriteString(m.descInput.View())
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("(Press Enter to continue, Esc to cancel)"))
		case 3:
			s.WriteString(successStyle.Render("✓"))
			s.WriteString(" ")
			s.WriteString(fieldLabelStyle.Render("URL:"))
			s.WriteString(" " + m.urlInput.Value() + "\n")
			titleVal := m.titleInput.Value()
			if titleVal != "" {
				s.WriteString(successStyle.Render("✓"))
				s.WriteString(" ")
				s.WriteString(fieldLabelStyle.Render("Title:"))
				s.WriteString(" " + titleVal + "\n")
			}
			descVal := m.descInput.Value()
			if descVal != "" {
				s.WriteString(successStyle.Render("✓"))
				s.WriteString(" ")
				s.WriteString(fieldLabelStyle.Render("Description:"))
				s.WriteString(" " + descVal + "\n")
			}
			s.WriteString("\n")
			s.WriteString(fieldLabelStyle.Render("Text (optional, press Enter to submit, Esc to cancel):"))
			s.WriteString("\n")
			s.WriteString(m.textInput.View())
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("(Press Enter to submit, Esc to cancel)"))
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
