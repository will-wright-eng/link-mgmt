package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/cli/logger"
	"link-mgmt-go/pkg/cli/tui/managelinks"
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
	scrapeState managelinks.ScrapeState

	// Viewport dimensions for proper rendering
	width int
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
		step:           managelinks.StepListLinks,
		confirm:        confirm,
		scrapeState: managelinks.ScrapeState{
			TimeoutSeconds: timeoutSeconds,
		},
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
		return managelinks.LinksLoadedMsg{Links: links, Err: err}
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
			m.width = managelinks.DefaultWidth
		}
		logger.Log("manageLinksModel.Update: received WindowSizeMsg, width=%d", m.width)
		return m, nil

	case managelinks.LinksLoadedMsg:
		logger.Log("manageLinksModel.Update: received LinksLoadedMsg, links_count=%d, err=%v", len(msg.Links), msg.Err != nil)
		if msg.Err != nil {
			m.err = msg.Err
			m.ready = true
			return m, nil
		}
		m.links = msg.Links
		m.ready = true
		return m, nil

	case managelinks.DeleteErrorMsg:
		m.err = msg.Err
		return m, tea.Quit

	case managelinks.DeleteSuccessMsg:
		m.step = managelinks.StepDone
		// Reload links after deletion
		return m, func() tea.Msg {
			links, err := m.client.ListLinks()
			return managelinks.LinksLoadedMsg{Links: links, Err: err}
		}

	case managelinks.ScrapeDoneMsg:
		m.scrapeState.Scraping = false
		if msg.Err != nil {
			m.scrapeState.Error = userFacingError(msg.Err)
			m.step = managelinks.StepScrapeDone
			return m, nil
		}
		m.scrapeState.Result = msg.Result
		m.scrapeState.Error = nil
		m.step = managelinks.StepScrapeSaving
		return m, m.saveEnrichedLink()

	case managelinks.EnrichSavedMsg:
		if msg.Err != nil {
			m.err = userFacingError(msg.Err)
			m.step = managelinks.StepScrapeDone
			return m, nil
		}
		m.scrapeState.Updated = msg.Link
		m.step = managelinks.StepScrapeDone
		// Reload links after enrichment
		return m, func() tea.Msg {
			links, err := m.client.ListLinks()
			return managelinks.LinksLoadedMsg{Links: links, Err: err}
		}

	case managelinks.ScrapeProgressMsg:
		m.scrapeState.Stage = msg.Stage
		m.scrapeState.Message = msg.Message
		// Continue watching progress if still scraping
		if m.scrapeState.Scraping {
			return m, m.watchProgress()
		}
		return m, nil

	case managelinks.ProgressTickMsg:
		// Continue watching for progress if still scraping
		if m.scrapeState.Scraping && !msg.Done {
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
		case managelinks.StepListLinks:
			logger.Log("manageLinksModel.Update: handling list keys")
			return m.handleListKeys(msg)
		case managelinks.StepActionMenu:
			return m.handleActionMenuKeys(msg)
		case managelinks.StepViewDetails:
			return m.handleViewDetailsKeys(msg)
		case managelinks.StepDeleteConfirm:
			return m.handleDeleteConfirmKeys(msg)
		case managelinks.StepScraping:
			switch msg.String() {
			case "ctrl+c", "esc":
				if m.scrapeState.Cancel != nil {
					m.scrapeState.Cancel()
				}
				m.step = managelinks.StepActionMenu
				return m, nil
			}
		case managelinks.StepScrapeDone:
			// Any key goes back to action menu after scraping
			m.step = managelinks.StepActionMenu
			return m, nil
		case managelinks.StepDone:
			// Any key exits after deletion success
			return m, tea.Quit
		}
	}

	// Handle text input updates for delete confirmation
	if m.step == managelinks.StepDeleteConfirm {
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
			m.step = managelinks.StepActionMenu
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
		m.step = managelinks.StepListLinks
		return m, nil
	case "1", "v":
		m.step = managelinks.StepViewDetails
		return m, nil
	case "2", "d":
		m.step = managelinks.StepDeleteConfirm
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
		m.step = managelinks.StepActionMenu
		return m, nil
	}
	return m, nil
}

func (m *manageLinksModel) handleDeleteConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.step = managelinks.StepActionMenu
		return m, nil
	case "enter":
		answer := strings.ToLower(strings.TrimSpace(m.confirm.Value()))
		if answer == "y" || answer == "yes" {
			return m, m.deleteLink()
		}
		// Cancelled - go back to action menu
		m.step = managelinks.StepActionMenu
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

	if m.err != nil && m.step != managelinks.StepDone {
		logger.LogError(m.err, "View: returning error view, step=%d", m.step)
		return renderErrorView(m.err)
	}

	var result string
	switch m.step {
	case managelinks.StepListLinks:
		logger.Log("View: rendering list view")
		result = m.renderList()
	case managelinks.StepActionMenu:
		logger.Log("View: rendering action menu, selected=%d", m.selected)
		result = m.renderActionMenu()
	case managelinks.StepViewDetails:
		logger.Log("View: rendering view details, selected=%d", m.selected)
		result = m.renderViewDetails()
	case managelinks.StepDeleteConfirm:
		logger.Log("View: rendering delete confirm, selected=%d", m.selected)
		result = m.renderDeleteConfirm()
	case managelinks.StepScraping:
		logger.Log("View: rendering scraping, stage=%s", m.scrapeState.Stage)
		result = m.renderScraping()
	case managelinks.StepScrapeSaving:
		logger.Log("View: rendering scrape saving")
		result = "\n" + infoStyle.Render("Saving enriched content...") + "\n"
	case managelinks.StepScrapeDone:
		logger.Log("View: rendering scrape done, error=%v, updated=%v", m.scrapeState.Error != nil, m.scrapeState.Updated != nil)
		result = m.renderScrapeDone()
	case managelinks.StepDone:
		logger.Log("View: rendering done (deletion success)")
		result = renderSuccessView("Link deleted successfully!")
	default:
		logger.Log("View: unknown step=%d, returning empty string", m.step)
		return ""
	}

	logger.Log("View: result length=%d bytes", len(result))
	return result
}

// getMaxWidth returns the maximum width for rendering, using DefaultWidth as fallback
func (m *manageLinksModel) getMaxWidth() int {
	if m.width > 0 {
		return m.width
	}
	return managelinks.DefaultWidth
}

// GetSelectedIndex implements SelectableModel interface for automatic viewport scrolling
func (m *manageLinksModel) GetSelectedIndex() int {
	// Only return selection when in list view step
	if m.step == managelinks.StepListLinks {
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
	if m.step == managelinks.StepListLinks {
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
			return managelinks.DeleteErrorMsg{Err: fmt.Errorf("invalid selection")}
		}

		link := m.links[m.selected]
		err := m.client.DeleteLink(link.ID)
		if err != nil {
			return managelinks.DeleteErrorMsg{Err: err}
		}

		return managelinks.DeleteSuccessMsg{}
	}
}

func (m *manageLinksModel) startScraping() (tea.Model, tea.Cmd) {
	m.step = managelinks.StepScraping
	m.scrapeState.Scraping = true
	m.scrapeState.Result = nil
	m.scrapeState.Error = nil
	m.scrapeState.Stage = scraper.StageHealthCheck
	m.scrapeState.Message = "Starting scrape..."
	m.scrapeState.ProgressChan = make(chan managelinks.ScrapeProgressMsg, 10)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.scrapeState.TimeoutSeconds)*time.Second)
	m.scrapeState.Ctx = ctx
	m.scrapeState.Cancel = cancel

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
			if m.scrapeState.Cancel != nil {
				m.scrapeState.Cancel()
			}
		}()

		// Progress callback that writes to the model's progress channel
		cb := func(stage scraper.ScrapeStage, message string) {
			select {
			case m.scrapeState.ProgressChan <- managelinks.ScrapeProgressMsg{
				Stage:   stage,
				Message: message,
			}:
			case <-ctx.Done():
				return
			}
		}

		// Run scrape (this blocks until complete)
		result, err := m.scraperService.ScrapeWithProgress(ctx, url, m.scrapeState.TimeoutSeconds, cb)

		// Close progress channel to signal completion
		if m.scrapeState.ProgressChan != nil {
			close(m.scrapeState.ProgressChan)
		}

		if err != nil {
			return managelinks.ScrapeDoneMsg{Err: err}
		}
		return managelinks.ScrapeDoneMsg{Result: result}
	}
}

// watchProgress periodically reads from the progress channel and sends updates
func (m *manageLinksModel) watchProgress() tea.Cmd {
	return func() tea.Msg {
		// Read from progress channel if available
		if m.scrapeState.ProgressChan == nil {
			return managelinks.ProgressTickMsg{Done: true}
		}
		select {
		case progress, ok := <-m.scrapeState.ProgressChan:
			if ok {
				// Send progress message and continue watching
				return progress
			}
			// Channel closed, stop watching
			return managelinks.ProgressTickMsg{Done: true}
		default:
			// No progress yet, check again soon
			return managelinks.ProgressTickMsg{Done: false}
		}
	}
}

func (m *manageLinksModel) saveEnrichedLink() tea.Cmd {
	return func() tea.Msg {
		if m.scrapeState.Result == nil {
			return managelinks.EnrichSavedMsg{Err: fmt.Errorf("no scrape result to apply")}
		}

		orig := m.links[m.selected]
		update := models.LinkUpdate{}
		changed := false

		// Only fill fields that are currently empty.
		if (orig.Title == nil || strings.TrimSpace(*orig.Title) == "") && m.scrapeState.Result.Title != "" {
			title := m.scrapeState.Result.Title
			update.Title = &title
			changed = true
		}

		if (orig.Text == nil || strings.TrimSpace(*orig.Text) == "") && m.scrapeState.Result.Text != "" {
			text := m.scrapeState.Result.Text
			update.Text = &text
			changed = true
		}

		if !changed {
			// Nothing to update; return original as "updated" for display.
			return managelinks.EnrichSavedMsg{Link: &orig, Err: nil}
		}

		updated, err := m.client.UpdateLink(orig.ID, update)
		if err != nil {
			return managelinks.EnrichSavedMsg{Err: err}
		}

		return managelinks.EnrichSavedMsg{Link: updated}
	}
}

func (m *manageLinksModel) renderScraping() string {
	return renderScrapingProgress(string(m.scrapeState.Stage), m.scrapeState.Message)
}

func (m *manageLinksModel) renderScrapeDone() string {
	if m.err != nil {
		return renderErrorView(m.err)
	}
	if m.scrapeState.Error != nil {
		return renderWarningView(fmt.Sprintf("Scraping failed: %v", m.scrapeState.Error))
	}

	if m.scrapeState.Updated == nil {
		return renderEmptyState("Done. (No changes applied.)")
	}

	return renderSuccessWithDetails("Link enriched successfully!", m.scrapeState.Updated, false)
}
