package managelinks

import (
	"link-mgmt-go/pkg/models"
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

// EnrichSuccessMsg is emitted when link enrichment succeeds
type EnrichSuccessMsg struct {
	Link *models.Link
}

// EnrichErrorMsg is emitted when link enrichment fails
type EnrichErrorMsg struct {
	Err error
}
