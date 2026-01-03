package managelinks

import (
	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/scraper"
)

// LinksLoadedMsg is emitted when links have been fetched
type LinksLoadedMsg struct {
	Links []models.Link
	Err   error
}

// DeleteErrorMsg is emitted when link deletion fails
type DeleteErrorMsg struct {
	Err error
}

// DeleteSuccessMsg is emitted when link deletion succeeds
type DeleteSuccessMsg struct{}

// ScrapeDoneMsg is emitted when scraping completes
type ScrapeDoneMsg struct {
	Result *scraper.ScrapeResponse
	Err    error
}

// EnrichSavedMsg is emitted when enriched link data has been saved
type EnrichSavedMsg struct {
	Link *models.Link
	Err  error
}

// ScrapeProgressMsg is emitted to report scraping progress
type ScrapeProgressMsg struct {
	Stage   scraper.ScrapeStage
	Message string
}

// ProgressTickMsg is emitted periodically to check progress
type ProgressTickMsg struct {
	Done bool
}
