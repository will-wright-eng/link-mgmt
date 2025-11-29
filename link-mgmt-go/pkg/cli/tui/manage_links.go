package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"link-mgmt-go/pkg/cli/client"
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
	timeoutSeconds int
	updated        *models.Link
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

	return &manageLinksModel{
		client:         c,
		scraperService: scraperService,
		timeoutSeconds: timeoutSeconds,
		step:           manageStepListLinks,
		confirm:        confirm,
	}
}

func (m *manageLinksModel) Init() tea.Cmd {
	return func() tea.Msg {
		links, err := m.client.ListLinks()
		return manageLinksLoadedMsg{links: links, err: err}
	}
}

func (m *manageLinksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case manageLinksLoadedMsg:
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

	case tea.KeyMsg:
		switch m.step {
		case manageStepListLinks:
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
	if !m.ready {
		return renderLoadingState("Loading links...")
	}

	if m.err != nil && m.step != manageStepDone {
		return renderErrorView(m.err)
	}

	switch m.step {
	case manageStepListLinks:
		return m.renderList()
	case manageStepActionMenu:
		return m.renderActionMenu()
	case manageStepViewDetails:
		return m.renderViewDetails()
	case manageStepDeleteConfirm:
		return m.renderDeleteConfirm()
	case manageStepScraping:
		return m.renderScraping()
	case manageStepScrapeSaving:
		return "\n" + infoStyle.Render("Saving enriched content...") + "\n"
	case manageStepScrapeDone:
		return m.renderScrapeDone()
	case manageStepDone:
		return renderSuccessView("Link deleted successfully!")
	}

	return ""
}

func (m *manageLinksModel) renderList() string {
	if len(m.links) == 0 {
		return renderEmptyState("No links found.")
	}

	s := renderLinkList(m.links, m.selected, "Manage Links", "Select a link:")
	s += helpStyle.Render("(Use ↑/↓ or j/k to navigate, Enter to select, Esc to quit)") + "\n"
	return s
}

func (m *manageLinksModel) renderActionMenu() string {
	if m.selected >= len(m.links) {
		return renderErrorView(fmt.Errorf("invalid selection"))
	}

	link := m.links[m.selected]
	title := formatLinkTitle(link)
	url := truncateURL(link.URL, 60)

	var b strings.Builder
	b.WriteString(renderTitle("Link Actions"))
	b.WriteString(renderDivider(60))
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

	link := m.links[m.selected]
	var b strings.Builder

	b.WriteString(renderTitle("Link Details"))
	b.WriteString(renderDivider(60))
	b.WriteString("\n\n")

	b.WriteString(renderLinkDetailsFull(&link))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("(Press Enter, 'b', Esc, or 'q' to go back)") + "\n")

	return b.String()
}

func (m *manageLinksModel) renderDeleteConfirm() string {
	if m.selected >= len(m.links) {
		return renderErrorView(fmt.Errorf("invalid selection"))
	}

	link := m.links[m.selected]
	title := formatLinkTitle(link)

	var b strings.Builder
	b.WriteString(renderTitle("Delete Link"))
	b.WriteString(warningStyle.Render("⚠️  Confirm Deletion") + "\n\n")

	b.WriteString(boldStyle.Render("Are you sure you want to delete:") + "\n")
	b.WriteString(fmt.Sprintf("  %s\n", linkTitleStyle.Render(title)))
	b.WriteString(fieldLabelStyle.Render("URL:"))
	b.WriteString(fmt.Sprintf(" %s\n\n", link.URL))

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.timeoutSeconds)*time.Second)
	m.scrapeCtx = ctx
	m.scrapeCancel = cancel

	link := m.links[m.selected]
	url := link.URL

	return m, m.runScrapeCommand(ctx, url)
}

func (m *manageLinksModel) runScrapeCommand(ctx context.Context, url string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			if m.scrapeCancel != nil {
				m.scrapeCancel()
			}
		}()

		result, err := m.scraperService.ScrapeWithProgress(ctx, url, m.timeoutSeconds, nil)
		if err != nil {
			return manageScrapeDoneMsg{err: err}
		}
		return manageScrapeDoneMsg{result: result}
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
	return renderScrapingProgress("Scraping Selected Link", string(m.scrapeStage), m.scrapeMessage)
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
