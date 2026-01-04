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

// addLinkForm is the Bubble Tea model for the add-link flow.
type addLinkForm struct {
	// Core dependencies
	client *client.Client

	// Inputs
	urlInput   textinput.Model
	titleInput textinput.Model
	descInput  textinput.Model
	textInput  textarea.Model

	// Flow / state
	step          int
	err           error
	created       *models.Link
	currentField  int
	scrapeEnabled bool

	// Config
	scrapeTimeoutSeconds int
}

const (
	stepURLInput = iota
	stepReview
	stepSaving
	stepSuccess
)

// NewAddLinkForm creates a new add link form model.
func NewAddLinkForm(
	apiClient *client.Client,
	scrapeTimeoutSeconds int,
) tea.Model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com"
	urlInput.Focus()
	urlInput.CharLimit = 2048
	urlInput.Width = 60

	titleInput := textinput.New()
	titleInput.Placeholder = "Title (optional)"
	titleInput.CharLimit = 255
	titleInput.Width = 60

	descInput := textinput.New()
	descInput.Placeholder = "Optional description"
	descInput.CharLimit = 1000
	descInput.Width = 60

	txt := textarea.New()
	txt.Placeholder = "Optional text content (multi-line)"
	txt.SetWidth(60)
	txt.SetHeight(5)
	txt.CharLimit = 10000

	if scrapeTimeoutSeconds <= 0 {
		scrapeTimeoutSeconds = 30
	}

	form := &addLinkForm{
		client:               apiClient,
		urlInput:             urlInput,
		titleInput:           titleInput,
		descInput:            descInput,
		textInput:            txt,
		step:                 stepURLInput,
		currentField:         0,
		scrapeTimeoutSeconds: scrapeTimeoutSeconds,
		scrapeEnabled:        true, // Enable scraping by default
	}

	// Wrap with viewport
	return NewViewportWrapper(form, ViewportConfig{
		Title:       "Add New Link",
		ShowHeader:  true,
		ShowFooter:  true,
		UseViewport: false, // Forms are typically short
		EnableHelp:  true,
		EnableMenu:  true,
		HelpContent: AddLinkFormHelpContent,
		OnMenu: func() tea.Cmd {
			// Return command that sends MenuNavigationMsg to return to root menu
			return func() tea.Msg {
				return MenuNavigationMsg{}
			}
		},
		MinWidth:  60,
		MinHeight: 15,
	})
}

// Init implements tea.Model.
func (m *addLinkForm) Init() tea.Cmd {
	return textinput.Blink
}

// Messages for submission results.
type submitErrorMsg struct {
	err error
}

type submitSuccessMsg struct {
	link *models.Link
}

// Update implements tea.Model.
func (m *addLinkForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Forward MenuNavigationMsg unchanged (let it bubble up to root)
	switch msg.(type) {
	case MenuNavigationMsg:
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}

		switch m.step {
		case stepURLInput:
			return m.handleURLInputKey(msg)
		case stepReview:
			return m.handleReviewStep(msg)
		}

	case submitErrorMsg:
		m.err = userFacingError(msg.err)
		m.step = stepReview
		return m, nil

	case submitSuccessMsg:
		m.created = msg.link
		m.step = stepSuccess
		return m, nil
	}

	// Route updates to active input based on step.
	var cmd tea.Cmd
	switch m.step {
	case stepURLInput:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case stepReview:
		switch m.currentField {
		case 0:
			m.urlInput, cmd = m.urlInput.Update(msg)
		case 1:
			m.titleInput, cmd = m.titleInput.Update(msg)
		case 2:
			m.descInput, cmd = m.descInput.Update(msg)
		case 3:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	case stepSaving, stepSuccess:
		// No interactive inputs during these steps besides global keys handled above.
	}

	return m, cmd
}

func (m *addLinkForm) handleURLInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate URL then move to review.
		_, err := utils.ValidateURL(m.urlInput.Value())
		if err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.step = stepReview
		m.currentField = 1
		m.focusCurrentField()
		return m, textinput.Blink

	case "s":
		// Toggle scraping
		m.scrapeEnabled = !m.scrapeEnabled
		return m, nil
	}

	// Default: let urlInput handle typing/navigation.
	var cmd tea.Cmd
	m.urlInput, cmd = m.urlInput.Update(msg)
	return m, cmd
}

// handleReviewStep manages multi-field navigation and submit from the review step.
func (m *addLinkForm) handleReviewStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.currentField = (m.currentField + 1) % 4
		m.focusCurrentField()
		return m, textinput.Blink
	case "shift+tab":
		m.currentField = (m.currentField - 1 + 4) % 4
		m.focusCurrentField()
		return m, textinput.Blink
	case "enter":
		// Save the link.
		m.step = stepSaving
		return m, m.submit()
	case "esc":
		return m, tea.Quit
	}

	// Route input to current field.
	var cmd tea.Cmd
	switch m.currentField {
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

func (m *addLinkForm) focusCurrentField() {
	m.urlInput.Blur()
	m.titleInput.Blur()
	m.descInput.Blur()
	m.textInput.Blur()

	switch m.currentField {
	case 0:
		m.urlInput.Focus()
	case 1:
		m.titleInput.Focus()
	case 2:
		m.descInput.Focus()
	case 3:
		m.textInput.Focus()
	}
}

// submit builds the API payload and submits the link creation request.
func (m *addLinkForm) submit() tea.Cmd {
	return func() tea.Msg {
		urlStr, err := utils.ValidateURL(m.urlInput.Value())
		if err != nil {
			return submitErrorMsg{err: err}
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

		// Use new API endpoint - API handles scraping
		created, err := m.client.CreateLinkWithScraping(
			linkCreate,
			m.scrapeEnabled,
			m.scrapeTimeoutSeconds,
			true, // only fill empty
		)
		if err != nil {
			return submitErrorMsg{err: err}
		}

		return submitSuccessMsg{link: created}
	}
}

// View implements tea.Model.
func (m *addLinkForm) View() string {
	switch m.step {
	case stepSuccess:
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

	case stepURLInput:
		return m.renderURLInput()

	case stepReview, stepSaving:
		return m.renderReview()
	}

	return ""
}

func (m *addLinkForm) renderURLInput() string {
	var b strings.Builder
	// Title is rendered by the viewport wrapper header
	b.WriteString(fieldLabelStyle.Render("URL (required):"))
	b.WriteString("\n")
	b.WriteString(m.urlInput.View())

	// Show scraping toggle status
	scrapeStatus := "enabled"
	if !m.scrapeEnabled {
		scrapeStatus = "disabled"
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("Scraping: %s (press 's' to toggle)", scrapeStatus)))

	if m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(renderInlineError(m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press Enter to continue, 's' to toggle scraping, Esc to cancel"))

	return b.String()
}

func (m *addLinkForm) renderReview() string {
	var b strings.Builder
	b.WriteString(renderTitle("Review & Edit Link"))

	// URL (read-only)
	b.WriteString(successStyle.Render("✓"))
	b.WriteString(" ")
	b.WriteString(fieldLabelStyle.Render("URL:"))
	b.WriteString(" " + m.urlInput.Value() + "\n\n")

	// Title field
	b.WriteString(fieldLabelStyle.Render("Title:"))
	b.WriteString("\n")
	if m.currentField == 1 {
		b.WriteString(selectedStyle.Render(m.titleInput.View()))
	} else {
		b.WriteString(m.titleInput.View())
	}
	b.WriteString("\n\n")

	// Description field
	b.WriteString(fieldLabelStyle.Render("Description (optional):"))
	b.WriteString("\n")
	if m.currentField == 2 {
		b.WriteString(selectedStyle.Render(m.descInput.View()))
	} else {
		b.WriteString(m.descInput.View())
	}
	b.WriteString("\n\n")

	// Text field
	b.WriteString(fieldLabelStyle.Render("Text (optional):"))
	b.WriteString("\n")
	if m.currentField == 3 {
		b.WriteString(selectedStyle.Render(m.textInput.View()))
	} else {
		b.WriteString(m.textInput.View())
	}

	if m.step == stepSaving {
		b.WriteString("\n\n")
		b.WriteString(infoStyle.Render("Saving link..."))
	}

	if m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(renderInlineError(m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("[Tab] Navigate  [Enter] Save  [Esc] Cancel"))

	return b.String()
}
