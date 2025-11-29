package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/scraper"
	"link-mgmt-go/pkg/utils"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// addLinkForm is the Bubble Tea model for the enhanced add-link flow with scraping.
type addLinkForm struct {
	// Core dependencies
	client         *client.Client
	scraperService *scraper.ScraperService

	// Inputs
	urlInput   textinput.Model
	titleInput textinput.Model
	descInput  textinput.Model
	textInput  textarea.Model

	// Flow / state
	step         int
	err          error
	created      *models.Link
	currentField int

	// Scraping state
	scraping          bool
	skipScraping      bool
	scrapeResult      *scraper.ScrapeResponse
	scrapeError       error
	scrapeProgress    scraper.ScrapeStage
	scrapeProgressMsg string
	scrapeCtx         context.Context
	scrapeCancel      context.CancelFunc
	scrapeStartTime   time.Time
	scrapeDuration    time.Duration
	progressChan      chan scrapeProgressMsg

	// Config
	scrapeTimeoutSeconds int
}

const (
	stepURLInput = iota
	stepScraping
	stepReview
	stepSaving
	stepSuccess
)

// NewAddLinkForm creates a new enhanced add link form model.
func NewAddLinkForm(
	apiClient *client.Client,
	scraperService *scraper.ScraperService,
	scrapeTimeoutSeconds int,
) tea.Model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com"
	urlInput.Focus()
	urlInput.CharLimit = 2048
	urlInput.Width = 60

	titleInput := textinput.New()
	titleInput.Placeholder = "Title (optional, will use scraped title if available)"
	titleInput.CharLimit = 255
	titleInput.Width = 60

	descInput := textinput.New()
	descInput.Placeholder = "Optional description"
	descInput.CharLimit = 1000
	descInput.Width = 60

	txt := textarea.New()
	txt.Placeholder = "Optional text content (multi-line, will use scraped text if available)"
	txt.SetWidth(60)
	txt.SetHeight(5)
	txt.CharLimit = 10000

	if scrapeTimeoutSeconds <= 0 {
		scrapeTimeoutSeconds = 30
	}

	return &addLinkForm{
		client:               apiClient,
		scraperService:       scraperService,
		urlInput:             urlInput,
		titleInput:           titleInput,
		descInput:            descInput,
		textInput:            txt,
		step:                 stepURLInput,
		currentField:         0,
		scrapeTimeoutSeconds: scrapeTimeoutSeconds,
	}
}

// Init implements tea.Model.
func (m *addLinkForm) Init() tea.Cmd {
	return textinput.Blink
}

// Messages for scraping and submission results.
type scrapeSuccessMsg struct {
	result *scraper.ScrapeResponse
}

type scrapeErrorMsg struct {
	err error
}

type scrapeProgressMsg struct {
	stage   scraper.ScrapeStage
	message string
}

type submitErrorMsg struct {
	err error
}

type submitSuccessMsg struct {
	link *models.Link
}

// Update implements tea.Model.
func (m *addLinkForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Allow cancelling scraping via context, then quit.
			if m.step == stepScraping && m.scrapeCancel != nil {
				m.scrapeCancel()
			}
			return m, tea.Quit
		}

		switch m.step {
		case stepURLInput:
			return m.handleURLInputKey(msg)
		case stepReview:
			return m.handleReviewStep(msg)
		}

	case scrapeProgressMsg:
		m.scrapeProgress = msg.stage
		m.scrapeProgressMsg = msg.message
		// Continue watching progress if still scraping
		if m.scraping {
			return m, m.watchProgress()
		}
		return m, nil

	case progressTickMsg:
		// Continue watching for progress if still scraping
		if m.scraping && !msg.done {
			// Schedule another check soon
			return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				// Call watchProgress command and return its message
				cmd := m.watchProgress()
				return cmd()
			})
		}
		// If done or not scraping, stop watching
		return m, nil

	case scrapeSuccessMsg:
		m.scraping = false
		m.scrapeResult = msg.result
		m.scrapeError = nil
		m.scrapeDuration = time.Since(m.scrapeStartTime)

		// Pre-fill fields from scraped content if available.
		if msg.result != nil {
			if msg.result.Title != "" && strings.TrimSpace(m.titleInput.Value()) == "" {
				m.titleInput.SetValue(msg.result.Title)
			}
			if msg.result.Text != "" && strings.TrimSpace(m.textInput.Value()) == "" {
				m.textInput.SetValue(msg.result.Text)
			}
		}

		m.step = stepReview
		m.currentField = 1 // Start at title field
		m.focusCurrentField()
		return m, textinput.Blink

	case scrapeErrorMsg:
		m.scraping = false
		m.scrapeError = userFacingError(msg.err)
		m.scrapeDuration = time.Since(m.scrapeStartTime)
		// Move to review step even if scraping failed.
		m.step = stepReview
		m.currentField = 1
		m.focusCurrentField()
		return m, textinput.Blink

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
	case stepScraping, stepSaving, stepSuccess:
		// No interactive inputs during these steps besides global keys handled above.
	}

	return m, cmd
}

func (m *addLinkForm) handleURLInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate URL then start scraping.
		_, err := utils.ValidateURL(m.urlInput.Value())
		if err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.skipScraping = false
		return m.startScraping()

	case "s":
		// Skip scraping, go directly to review/manual entry.
		_, err := utils.ValidateURL(m.urlInput.Value())
		if err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.skipScraping = true
		m.step = stepReview
		m.currentField = 1
		m.focusCurrentField()
		return m, textinput.Blink
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

// startScraping kicks off the asynchronous scraping command.
func (m *addLinkForm) startScraping() (tea.Model, tea.Cmd) {
	m.step = stepScraping
	m.scraping = true
	m.scrapeError = nil
	m.scrapeResult = nil
	m.scrapeProgress = scraper.StageHealthCheck
	m.scrapeProgressMsg = "Starting scrape..."
	m.scrapeStartTime = time.Now()
	m.scrapeDuration = 0
	m.progressChan = make(chan scrapeProgressMsg, 10)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.scrapeTimeoutSeconds)*time.Second)
	m.scrapeCtx = ctx
	m.scrapeCancel = cancel

	urlStr := m.urlInput.Value()

	return m, tea.Batch(
		m.runScrapeCommand(ctx, urlStr),
		m.watchProgress(),
	)
}

// runScrapeCommand performs the scrape using the scraper service and reports progress back to the TUI.
func (m *addLinkForm) runScrapeCommand(ctx context.Context, url string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			if m.scrapeCancel != nil {
				m.scrapeCancel()
			}
		}()

		// Progress callback that writes to the model's progress channel
		cb := func(stage scraper.ScrapeStage, message string) {
			select {
			case m.progressChan <- scrapeProgressMsg{
				stage:   stage,
				message: message,
			}:
			case <-ctx.Done():
				return
			}
		}

		// Run scrape (this blocks until complete)
		result, err := m.scraperService.ScrapeWithProgress(ctx, url, m.scrapeTimeoutSeconds, cb)

		// Close progress channel to signal completion
		close(m.progressChan)

		if err != nil {
			return scrapeErrorMsg{err: err}
		}

		return scrapeSuccessMsg{result: result}
	}
}

// watchProgress periodically reads from the progress channel and sends updates
func (m *addLinkForm) watchProgress() tea.Cmd {
	return func() tea.Msg {
		// Read from progress channel if available
		select {
		case progress, ok := <-m.progressChan:
			if ok {
				// Send progress message and continue watching
				return progress
			}
			// Channel closed, stop watching
			return progressTickMsg{done: true}
		default:
			// No progress yet, check again soon
			return progressTickMsg{done: false}
		}
	}
}

// progressTickMsg is sent to continue watching for progress updates
type progressTickMsg struct {
	done bool
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

		created, err := m.client.CreateLink(linkCreate)
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

			// Add scraping duration if scraping was performed
			if m.scrapeDuration > 0 && m.scrapeResult != nil {
				b.WriteString(fieldLabelStyle.Render("Scraped in:"))
				b.WriteString(fmt.Sprintf(" %s\n", m.scrapeDuration.Round(time.Millisecond)))
			}

			b.WriteString("\n")
			b.WriteString(helpStyle.Render("Press any key to exit...") + "\n")
			return b.String()
		}
		return renderSuccessView("Link created successfully!")

	case stepURLInput:
		return m.renderURLInput()

	case stepScraping:
		return m.renderScraping()

	case stepReview, stepSaving:
		return m.renderReview()
	}

	return ""
}

func (m *addLinkForm) renderURLInput() string {
	var b strings.Builder
	b.WriteString(renderTitle("Add New Link"))
	b.WriteString(fieldLabelStyle.Render("URL (required):"))
	b.WriteString("\n")
	b.WriteString(m.urlInput.View())

	if m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(renderInlineError(m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press Enter to scrape, 's' to skip scraping, Esc to cancel"))

	return b.String()
}

func (m *addLinkForm) renderScraping() string {
	return renderScrapingProgress("Scraping URL", string(m.scrapeProgress), m.scrapeProgressMsg)
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
	if m.scrapeResult != nil && m.scrapeResult.Title != "" {
		b.WriteString(" " + mutedStyle.Render("(scraped)"))
	}
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

	if m.scrapeError != nil {
		b.WriteString("\n\n")
		b.WriteString(renderInlineWarning(fmt.Sprintf("Scraping failed: %v (you can still fill fields manually)", m.scrapeError)))
	}
	if m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(renderInlineError(m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("[Tab] Navigate  [Enter] Save  [Esc] Cancel"))

	return b.String()
}
