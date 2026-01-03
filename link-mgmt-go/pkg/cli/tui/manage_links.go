package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/cli/logger"
	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/scraper"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// manageLinksModel is a combined Bubble Tea model that allows listing, viewing,
// deleting, and scraping links in a single unified flow.
type manageLinksModel struct {
	client         *client.Client
	scraperService *scraper.ScraperService

	links    []models.Link
	selected int
	step     int // 0=list, 1=action menu, 2=view details, 3=delete confirm, 4=scraping, 5=scrape saving, 6=scrape done, 7=done
	err      error
	ready    bool

	// For delete confirmation
	confirm textinput.Model

	// Scraping state
	scraping       bool
	scrapeResult   *scraper.ScrapeResponse
	scrapeError    error
	scrapeStage    scraper.ScrapeStage
	scrapeMessage  string
	scrapeCtx      context.Context
	scrapeCancel   context.CancelFunc
	progressChan   chan manageScrapeProgressMsg
	timeoutSeconds int
	updated        *models.Link

	// Viewport dimensions for proper rendering
	width int
}

const (
	manageStepListLinks = iota
	manageStepActionMenu
	manageStepViewDetails
	manageStepDeleteConfirm
	manageStepScraping
	manageStepScrapeSaving
	manageStepScrapeDone
	manageStepDone
)

const defaultWidth = 80 // Default terminal width fallback

// manageLinksLoadedMsg is emitted when links have been fetched.
type manageLinksLoadedMsg struct {
	links []models.Link
	err   error
}

type manageDeleteErrorMsg struct {
	err error
}

type manageDeleteSuccessMsg struct{}

type manageScrapeDoneMsg struct {
	result *scraper.ScrapeResponse
	err    error
}

type manageEnrichSavedMsg struct {
	link *models.Link
	err  error
}

type manageScrapeProgressMsg struct {
	stage   scraper.ScrapeStage
	message string
}

type manageProgressTickMsg struct {
	done bool
}

// NewManageLinksModel creates a new combined manage links flow.
func NewManageLinksModel(
	c *client.Client,
	scraperService *scraper.ScraperService,
	timeoutSeconds int,
) tea.Model {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	confirm := textinput.New()
	confirm.Placeholder = "y/N"
	confirm.CharLimit = 1
	confirm.Width = 10

	model := &manageLinksModel{
		client:         c,
		scraperService: scraperService,
		timeoutSeconds: timeoutSeconds,
		step:           manageStepListLinks,
		confirm:        confirm,
	}

	// Wrap with viewport (enable scrolling for long lists)
	return NewViewportWrapper(model, ViewportConfig{
		Title:       "Manage Links",
		ShowHeader:  true,
		ShowFooter:  true,
		UseViewport: true, // Enable scrolling for long link lists
		EnableHelp:  true,
		EnableMenu:  true,
		HelpContent: ManageLinksHelpContent,
		OnMenu: func() tea.Cmd {
			// Return command that sends MenuNavigationMsg to return to root menu
			return func() tea.Msg {
				return MenuNavigationMsg{}
			}
		},
		MinWidth:  60,
		MinHeight: 10,
	})
}
func (m *manageLinksModel) Init() tea.Cmd {
	return func() tea.Msg {
		links, err := m.client.ListLinks()
		return manageLinksLoadedMsg{links: links, err: err}
	}
}

func (m *manageLinksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	logger.Log("manageLinksModel.Update() called: msg_type=%T, step=%d, ready=%v", msg, m.step, m.ready)

	// Forward MenuNavigationMsg unchanged (let it bubble up to root)
	switch msg.(type) {
	case MenuNavigationMsg:
		logger.Log("manageLinksModel.Update: forwarding MenuNavigationMsg")
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Store width for rendering - this allows us to render content that fits the viewport
		m.width = msg.Width
		if m.width == 0 {
			m.width = defaultWidth
		}
		logger.Log("manageLinksModel.Update: received WindowSizeMsg, width=%d", m.width)
		return m, nil

	case manageLinksLoadedMsg:
		logger.Log("manageLinksModel.Update: received manageLinksLoadedMsg, links_count=%d, err=%v", len(msg.links), msg.err != nil)
		if msg.err != nil {
			m.err = msg.err
			m.ready = true
			return m, nil
		}
		m.links = msg.links
		m.ready = true
		return m, nil

	case manageDeleteErrorMsg:
		m.err = msg.err
		return m, tea.Quit

	case manageDeleteSuccessMsg:
		m.step = manageStepDone
		// Reload links after deletion
		return m, func() tea.Msg {
			links, err := m.client.ListLinks()
			return manageLinksLoadedMsg{links: links, err: err}
		}

	case manageScrapeDoneMsg:
		m.scraping = false
		if msg.err != nil {
			m.scrapeError = userFacingError(msg.err)
			m.step = manageStepScrapeDone
			return m, nil
		}
		m.scrapeResult = msg.result
		m.scrapeError = nil
		m.step = manageStepScrapeSaving
		return m, m.saveEnrichedLink()

	case manageEnrichSavedMsg:
		if msg.err != nil {
			m.err = userFacingError(msg.err)
			m.step = manageStepScrapeDone
			return m, nil
		}
		m.updated = msg.link
		m.step = manageStepScrapeDone
		// Reload links after enrichment
		return m, func() tea.Msg {
			links, err := m.client.ListLinks()
			return manageLinksLoadedMsg{links: links, err: err}
		}

	case manageScrapeProgressMsg:
		m.scrapeStage = msg.stage
		m.scrapeMessage = msg.message
		// Continue watching progress if still scraping
		if m.scraping {
			return m, m.watchProgress()
		}
		return m, nil

	case manageProgressTickMsg:
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

	case tea.KeyMsg:
		logger.Log("manageLinksModel.Update: received KeyMsg, key=%q, step=%d", msg.String(), m.step)
		switch m.step {
		case manageStepListLinks:
			logger.Log("manageLinksModel.Update: handling list keys")
			return m.handleListKeys(msg)
		case manageStepActionMenu:
			return m.handleActionMenuKeys(msg)
		case manageStepViewDetails:
			return m.handleViewDetailsKeys(msg)
		case manageStepDeleteConfirm:
			return m.handleDeleteConfirmKeys(msg)
		case manageStepScraping:
			switch msg.String() {
			case "ctrl+c", "esc":
				if m.scrapeCancel != nil {
					m.scrapeCancel()
				}
				m.step = manageStepActionMenu
				return m, nil
			}
		case manageStepScrapeDone:
			// Any key goes back to action menu after scraping
			m.step = manageStepActionMenu
			return m, nil
		case manageStepDone:
			// Any key exits after deletion success
			return m, tea.Quit
		}
	}

	// Handle text input updates for delete confirmation
	if m.step == manageStepDeleteConfirm {
		var cmd tea.Cmd
		m.confirm, cmd = m.confirm.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *manageLinksModel) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handleQuitKeys(msg.String()) {
		return m, tea.Quit
	}
	if newSelected, handled := handleListNavigation(msg.String(), m.selected, len(m.links)); handled {
		m.selected = newSelected
		return m, nil
	}
	if msg.String() == "enter" {
		if len(m.links) == 0 {
			return m, nil
		}
		if m.selected < len(m.links) {
			m.step = manageStepActionMenu
			return m, nil
		}
		return m, nil
	}
	return m, nil
}

func (m *manageLinksModel) handleActionMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handleQuitKeys(msg.String()) {
		return m, tea.Quit
	}
	switch msg.String() {
	case "esc", "b":
		m.step = manageStepListLinks
		return m, nil
	case "1", "v":
		m.step = manageStepViewDetails
		return m, nil
	case "2", "d":
		m.step = manageStepDeleteConfirm
		m.confirm.Focus()
		return m, textinput.Blink
	case "3", "s":
		// Start scraping
		if m.selected < 0 || m.selected >= len(m.links) {
			return m, nil
		}
		return m.startScraping()
	}
	return m, nil
}

func (m *manageLinksModel) handleViewDetailsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handleQuitKeys(msg.String()) {
		return m, tea.Quit
	}
	switch msg.String() {
	case "esc", "b", "enter":
		m.step = manageStepActionMenu
		return m, nil
	}
	return m, nil
}

func (m *manageLinksModel) handleDeleteConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.step = manageStepActionMenu
		return m, nil
	case "enter":
		answer := strings.ToLower(strings.TrimSpace(m.confirm.Value()))
		if answer == "y" || answer == "yes" {
			return m, m.deleteLink()
		}
		// Cancelled - go back to action menu
		m.step = manageStepActionMenu
		return m, nil
	default:
		var cmd tea.Cmd
		m.confirm, cmd = m.confirm.Update(msg)
		return m, cmd
	}
}

func (m *manageLinksModel) View() string {
	logger.Log("manageLinksModel.View() called: ready=%v, step=%d, err=%v, links_count=%d, selected=%d",
		m.ready, m.step, m.err != nil, len(m.links), m.selected)

	if !m.ready {
		logger.Log("View: returning loading state")
		return renderLoadingState("Loading links...")
	}

	if m.err != nil && m.step != manageStepDone {
		logger.LogError(m.err, "View: returning error view, step=%d", m.step)
		return renderErrorView(m.err)
	}

	var result string
	switch m.step {
	case manageStepListLinks:
		logger.Log("View: rendering list view")
		result = m.renderList()
	case manageStepActionMenu:
		logger.Log("View: rendering action menu, selected=%d", m.selected)
		result = m.renderActionMenu()
	case manageStepViewDetails:
		logger.Log("View: rendering view details, selected=%d", m.selected)
		result = m.renderViewDetails()
	case manageStepDeleteConfirm:
		logger.Log("View: rendering delete confirm, selected=%d", m.selected)
		result = m.renderDeleteConfirm()
	case manageStepScraping:
		logger.Log("View: rendering scraping, stage=%s", m.scrapeStage)
		result = m.renderScraping()
	case manageStepScrapeSaving:
		logger.Log("View: rendering scrape saving")
		result = "\n" + infoStyle.Render("Saving enriched content...") + "\n"
	case manageStepScrapeDone:
		logger.Log("View: rendering scrape done, error=%v, updated=%v", m.scrapeError != nil, m.updated != nil)
		result = m.renderScrapeDone()
	case manageStepDone:
		logger.Log("View: rendering done (deletion success)")
		result = renderSuccessView("Link deleted successfully!")
	default:
		logger.Log("View: unknown step=%d, returning empty string", m.step)
		return ""
	}

	logger.Log("View: result length=%d bytes", len(result))
	return result
}

// getMaxWidth returns the maximum width for rendering, using defaultWidth as fallback
func (m *manageLinksModel) getMaxWidth() int {
	if m.width > 0 {
		return m.width
	}
	return defaultWidth
}

// GetSelectedIndex implements SelectableModel interface for automatic viewport scrolling
func (m *manageLinksModel) GetSelectedIndex() int {
	// Only return selection when in list view step
	if m.step == manageStepListLinks {
		return m.selected
	}
	return -1 // No selection in other steps
}

// GetItemHeight implements SelectableModel interface
// Each link renders as 2 lines: title + URL
func (m *manageLinksModel) GetItemHeight() int {
	return 2
}

// GetListHeaderHeight implements SelectableModel interface
// The list has a subtitle "Select a link:" (1 line) + blank line (1 line) = 2 lines
// Plus the help text at the bottom (1 line) = 3 lines total before items
// Actually, looking at renderLinkList: subtitle + blank line = 2 lines before items
// Help text is after items, so we only count the subtitle + blank = 2 lines
func (m *manageLinksModel) GetListHeaderHeight() int {
	if m.step == manageStepListLinks {
		// Subtitle "Select a link:" (1 line) + blank line (1 line) = 2 lines
		return 2
	}
	return 0
}

func (m *manageLinksModel) renderList() string {
	logger.Log("renderList: called with %d links, selected=%d, width=%d", len(m.links), m.selected, m.width)

	if len(m.links) == 0 {
		logger.Log("renderList: no links, returning empty state")
		return renderEmptyState("No links found.")
	}

	// Use stored width for rendering, with fallback
	maxWidth := m.getMaxWidth()

	// Title is rendered by the viewport wrapper header
	s := renderLinkList(m.links, m.selected, "", "Select a link:", maxWidth)
	s += helpStyle.Render("(Use ↑/↓ or j/k to navigate, Enter to select, Esc to quit)") + "\n"

	logger.Log("renderList: generated content, length=%d bytes", len(s))
	return s
}

func (m *manageLinksModel) renderActionMenu() string {
	if m.selected >= len(m.links) {
		return renderErrorView(fmt.Errorf("invalid selection"))
	}

	// Use stored width for rendering, with fallback
	maxWidth := m.getMaxWidth()

	link := m.links[m.selected]
	title := formatLinkTitle(link)
	// Use maxWidth for URL truncation, but leave some margin for formatting
	urlTruncateWidth := maxWidth - 10
	if urlTruncateWidth < 40 {
		urlTruncateWidth = 40 // Minimum
	}
	url := truncateURL(link.URL, urlTruncateWidth)

	var b strings.Builder
	b.WriteString(renderTitle("Link Actions"))
	b.WriteString(renderDivider(maxWidth))
	b.WriteString("\n\n")

	b.WriteString(boldStyle.Render("Selected Link:") + "\n")
	b.WriteString(fmt.Sprintf("  %s\n", linkTitleStyle.Render(title)))
	b.WriteString(fmt.Sprintf("  %s\n\n", linkURLStyle.Render(url)))

	b.WriteString(boldStyle.Render("Choose an action:") + "\n\n")
	b.WriteString("  " + selectedMarkerStyle.Render("1)") + " View details\n")
	b.WriteString("  " + selectedMarkerStyle.Render("2)") + " Delete link\n")
	b.WriteString("  " + selectedMarkerStyle.Render("3)") + " Scrape & enrich\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("(Press 1/v to view, 2/d to delete, 3/s to scrape, Esc/b to go back, q to quit)") + "\n")

	return b.String()
}

func (m *manageLinksModel) renderViewDetails() string {
	if m.selected >= len(m.links) {
		return renderErrorView(fmt.Errorf("invalid selection"))
	}

	// Use stored width for rendering, with fallback
	maxWidth := m.getMaxWidth()

	link := m.links[m.selected]
	var b strings.Builder

	b.WriteString(renderTitle("Link Details"))
	b.WriteString(renderDivider(maxWidth))
	b.WriteString("\n\n")

	b.WriteString(renderLinkDetailsFull(&link, maxWidth))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("(Press Enter, 'b', Esc, or 'q' to go back)") + "\n")

	return b.String()
}

func (m *manageLinksModel) renderDeleteConfirm() string {
	if m.selected >= len(m.links) {
		return renderErrorView(fmt.Errorf("invalid selection"))
	}

	// Use stored width for rendering, with fallback
	maxWidth := m.getMaxWidth()

	link := m.links[m.selected]
	title := formatLinkTitle(link)
	// Truncate URL if needed
	urlTruncateWidth := maxWidth - 10
	if urlTruncateWidth < 40 {
		urlTruncateWidth = 40
	}
	url := truncateURL(link.URL, urlTruncateWidth)

	var b strings.Builder
	b.WriteString(renderTitle("Delete Link"))
	b.WriteString(warningStyle.Render("⚠️  Confirm Deletion") + "\n\n")

	b.WriteString(boldStyle.Render("Are you sure you want to delete:") + "\n")
	b.WriteString(fmt.Sprintf("  %s\n", linkTitleStyle.Render(title)))
	b.WriteString(fieldLabelStyle.Render("URL:"))
	b.WriteString(fmt.Sprintf(" %s\n\n", url))

	b.WriteString(boldStyle.Render("Confirm (y/N):"))
	b.WriteString(" ")
	b.WriteString(m.confirm.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("(Press Enter to confirm, Esc to cancel)") + "\n")

	return b.String()
}

func (m *manageLinksModel) deleteLink() tea.Cmd {
	return func() tea.Msg {
		if m.selected >= len(m.links) {
			return manageDeleteErrorMsg{err: fmt.Errorf("invalid selection")}
		}

		link := m.links[m.selected]
		err := m.client.DeleteLink(link.ID)
		if err != nil {
			return manageDeleteErrorMsg{err: err}
		}

		return manageDeleteSuccessMsg{}
	}
}

func (m *manageLinksModel) startScraping() (tea.Model, tea.Cmd) {
	m.step = manageStepScraping
	m.scraping = true
	m.scrapeResult = nil
	m.scrapeError = nil
	m.scrapeStage = scraper.StageHealthCheck
	m.scrapeMessage = "Starting scrape..."
	m.progressChan = make(chan manageScrapeProgressMsg, 10)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.timeoutSeconds)*time.Second)
	m.scrapeCtx = ctx
	m.scrapeCancel = cancel

	link := m.links[m.selected]
	url := link.URL

	return m, tea.Batch(
		m.runScrapeCommand(ctx, url),
		m.watchProgress(),
	)
}

func (m *manageLinksModel) runScrapeCommand(ctx context.Context, url string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			if m.scrapeCancel != nil {
				m.scrapeCancel()
			}
		}()

		// Progress callback that writes to the model's progress channel
		cb := func(stage scraper.ScrapeStage, message string) {
			select {
			case m.progressChan <- manageScrapeProgressMsg{
				stage:   stage,
				message: message,
			}:
			case <-ctx.Done():
				return
			}
		}

		// Run scrape (this blocks until complete)
		result, err := m.scraperService.ScrapeWithProgress(ctx, url, m.timeoutSeconds, cb)

		// Close progress channel to signal completion
		if m.progressChan != nil {
			close(m.progressChan)
		}

		if err != nil {
			return manageScrapeDoneMsg{err: err}
		}
		return manageScrapeDoneMsg{result: result}
	}
}

// watchProgress periodically reads from the progress channel and sends updates
func (m *manageLinksModel) watchProgress() tea.Cmd {
	return func() tea.Msg {
		// Read from progress channel if available
		if m.progressChan == nil {
			return manageProgressTickMsg{done: true}
		}
		select {
		case progress, ok := <-m.progressChan:
			if ok {
				// Send progress message and continue watching
				return progress
			}
			// Channel closed, stop watching
			return manageProgressTickMsg{done: true}
		default:
			// No progress yet, check again soon
			return manageProgressTickMsg{done: false}
		}
	}
}

func (m *manageLinksModel) saveEnrichedLink() tea.Cmd {
	return func() tea.Msg {
		if m.scrapeResult == nil {
			return manageEnrichSavedMsg{err: fmt.Errorf("no scrape result to apply")}
		}

		orig := m.links[m.selected]
		update := models.LinkUpdate{}
		changed := false

		// Only fill fields that are currently empty.
		if (orig.Title == nil || strings.TrimSpace(*orig.Title) == "") && m.scrapeResult.Title != "" {
			title := m.scrapeResult.Title
			update.Title = &title
			changed = true
		}

		if (orig.Text == nil || strings.TrimSpace(*orig.Text) == "") && m.scrapeResult.Text != "" {
			text := m.scrapeResult.Text
			update.Text = &text
			changed = true
		}

		if !changed {
			// Nothing to update; return original as "updated" for display.
			return manageEnrichSavedMsg{link: &orig, err: nil}
		}

		updated, err := m.client.UpdateLink(orig.ID, update)
		if err != nil {
			return manageEnrichSavedMsg{err: err}
		}

		return manageEnrichSavedMsg{link: updated}
	}
}

func (m *manageLinksModel) renderScraping() string {
	return renderScrapingProgress(string(m.scrapeStage), m.scrapeMessage)
}

func (m *manageLinksModel) renderScrapeDone() string {
	if m.err != nil {
		return renderErrorView(m.err)
	}
	if m.scrapeError != nil {
		return renderWarningView(fmt.Sprintf("Scraping failed: %v", m.scrapeError))
	}

	if m.updated == nil {
		return renderEmptyState("Done. (No changes applied.)")
	}

	return renderSuccessWithDetails("Link enriched successfully!", m.updated, false)
}
