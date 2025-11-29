package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/scraper"

	tea "github.com/charmbracelet/bubbletea"
)

// scrapeExistingLinkForm lets the user select an existing link, scrape its URL,
// and fill in missing title/text fields from the scraped content.
type scrapeExistingLinkForm struct {
	client         *client.Client
	scraperService *scraper.ScraperService

	links    []models.Link
	selected int

	step int
	err  error

	// Scraping state
	scraping       bool
	scrapeResult   *scraper.ScrapeResponse
	scrapeError    error
	scrapeStage    scraper.ScrapeStage
	scrapeMessage  string
	scrapeCtx      context.Context
	scrapeCancel   context.CancelFunc
	timeoutSeconds int

	updated *models.Link
}

const (
	stepScrapeSelect = iota
	stepScrapeRunning
	stepScrapeSaving
	stepScrapeDone
)

// Messages (scoped to this flow; names are prefixed to avoid collisions)
type scrapeLinksLoadedMsg struct {
	links []models.Link
	err   error
}

type scrapeDoneMsg struct {
	result *scraper.ScrapeResponse
	err    error
}

type enrichSavedMsg struct {
	link *models.Link
	err  error
}

// NewScrapeExistingLinkForm constructs the flow model.
func NewScrapeExistingLinkForm(
	apiClient *client.Client,
	scraperService *scraper.ScraperService,
	timeoutSeconds int,
) tea.Model {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	return &scrapeExistingLinkForm{
		client:         apiClient,
		scraperService: scraperService,
		timeoutSeconds: timeoutSeconds,
		step:           stepScrapeSelect,
	}
}

func (m *scrapeExistingLinkForm) Init() tea.Cmd {
	// Load links up-front so user can select which to enrich.
	return func() tea.Msg {
		links, err := m.client.ListLinks()
		return scrapeLinksLoadedMsg{links: links, err: err}
	}
}

func (m *scrapeExistingLinkForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case scrapeLinksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.links = msg.links
		if len(m.links) == 0 {
			m.err = fmt.Errorf("no links available to enrich")
			return m, tea.Quit
		}
		return m, nil

	case scrapeDoneMsg:
		m.scraping = false
		if msg.err != nil {
			m.scrapeError = m.userFacingError(msg.err)
			m.step = stepScrapeDone
			return m, nil
		}
		m.scrapeResult = msg.result
		m.scrapeError = nil
		m.step = stepScrapeSaving
		return m, m.saveEnrichedLink()

	case enrichSavedMsg:
		if msg.err != nil {
			m.err = m.userFacingError(msg.err)
			m.step = stepScrapeDone
			return m, nil
		}
		m.updated = msg.link
		m.step = stepScrapeDone
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case stepScrapeSelect:
			return m.handleSelectKey(msg)
		case stepScrapeRunning:
			switch msg.String() {
			case "ctrl+c", "esc":
				if m.scrapeCancel != nil {
					m.scrapeCancel()
				}
				return m, tea.Quit
			}
		case stepScrapeDone:
			// Any key exits on final screen.
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *scrapeExistingLinkForm) handleSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handleQuitKeys(msg.String()) {
		return m, tea.Quit
	}
	if newSelected, handled := handleListNavigation(msg.String(), m.selected, len(m.links)); handled {
		m.selected = newSelected
		return m, nil
	}
	if msg.String() == "enter" {
		// Start scraping for the selected link.
		if m.selected < 0 || m.selected >= len(m.links) {
			return m, nil
		}
		return m.startScraping()
	}
	return m, nil
}

func (m *scrapeExistingLinkForm) startScraping() (tea.Model, tea.Cmd) {
	m.step = stepScrapeRunning
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

func (m *scrapeExistingLinkForm) runScrapeCommand(ctx context.Context, url string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			if m.scrapeCancel != nil {
				m.scrapeCancel()
			}
		}()

		result, err := m.scraperService.ScrapeWithProgress(ctx, url, m.timeoutSeconds, nil)
		if err != nil {
			return scrapeDoneMsg{err: err}
		}
		return scrapeDoneMsg{result: result}
	}
}

func (m *scrapeExistingLinkForm) saveEnrichedLink() tea.Cmd {
	return func() tea.Msg {
		if m.scrapeResult == nil {
			return enrichSavedMsg{err: fmt.Errorf("no scrape result to apply")}
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
			return enrichSavedMsg{link: &orig, err: nil}
		}

		updated, err := m.client.UpdateLink(orig.ID, update)
		if err != nil {
			return enrichSavedMsg{err: err}
		}

		return enrichSavedMsg{link: updated}
	}
}

func (m *scrapeExistingLinkForm) userFacingError(err error) error {
	if err == nil {
		return nil
	}

	var scraperErr *scraper.ScraperError
	if errors.As(err, &scraperErr) {
		return errors.New(scraperErr.UserMessage())
	}

	return err
}

func (m *scrapeExistingLinkForm) View() string {
	switch m.step {
	case stepScrapeSelect:
		return m.renderSelect()
	case stepScrapeRunning:
		return m.renderScraping()
	case stepScrapeSaving:
		return "\n" + infoStyle.Render("Saving enriched content...") + "\n"
	case stepScrapeDone:
		return m.renderDone()
	default:
		return ""
	}
}

func (m *scrapeExistingLinkForm) renderSelect() string {
	if m.err != nil {
		return renderErrorView(m.err)
	}

	if len(m.links) == 0 {
		return renderEmptyState("No links available to enrich.")
	}

	s := renderLinkList(m.links, m.selected, "Scrape & Enrich Existing Link",
		"Select a link whose title/text you want to fill from scraped content:")
	s += helpStyle.Render("(Use ↑/↓ or j/k to navigate, Enter to scrape, Esc to cancel)") + "\n"
	return s
}

func (m *scrapeExistingLinkForm) renderScraping() string {
	return renderScrapingProgress("Scraping Selected Link", string(m.scrapeStage), m.scrapeMessage)
}

func (m *scrapeExistingLinkForm) renderDone() string {
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
