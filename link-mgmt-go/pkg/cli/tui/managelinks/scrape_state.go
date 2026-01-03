package managelinks

import (
	"context"

	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/scraper"
)

// ScrapeState holds all state related to scraping operations
type ScrapeState struct {
	Scraping       bool
	Result         *scraper.ScrapeResponse
	Error          error
	Stage          scraper.ScrapeStage
	Message        string
	Ctx            context.Context
	Cancel         context.CancelFunc
	ProgressChan   chan ScrapeProgressMsg
	TimeoutSeconds int
	Updated        *models.Link
}
