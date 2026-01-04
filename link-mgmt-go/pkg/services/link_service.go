package services

import (
	"context"
	"fmt"
	"strings"

	"link-mgmt-go/pkg/db"
	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/scraper"

	"github.com/google/uuid"
)

// LinkService handles business logic for link operations
type LinkService struct {
	db      *db.DB
	scraper *scraper.ScraperService
}

// NewLinkService creates a new link service
func NewLinkService(db *db.DB, scraperService *scraper.ScraperService) *LinkService {
	return &LinkService{
		db:      db,
		scraper: scraperService,
	}
}

// ListLinks retrieves all links for a user
func (s *LinkService) ListLinks(ctx context.Context, userID uuid.UUID) ([]models.Link, error) {
	return s.db.GetLinksByUserID(ctx, userID)
}

// GetLink retrieves a single link by ID
func (s *LinkService) GetLink(ctx context.Context, linkID, userID uuid.UUID) (*models.Link, error) {
	return s.db.GetLinkByID(ctx, linkID, userID)
}

// CreateLink creates a new link
func (s *LinkService) CreateLink(ctx context.Context, userID uuid.UUID, linkCreate models.LinkCreate) (*models.Link, error) {
	// Validation could go here
	if strings.TrimSpace(linkCreate.URL) == "" {
		return nil, fmt.Errorf("URL is required")
	}

	return s.db.CreateLink(ctx, userID, linkCreate)
}

// UpdateLink updates an existing link
func (s *LinkService) UpdateLink(ctx context.Context, linkID, userID uuid.UUID, update models.LinkUpdate) (*models.Link, error) {
	return s.db.UpdateLink(ctx, linkID, userID, update)
}

// DeleteLink deletes a link
func (s *LinkService) DeleteLink(ctx context.Context, linkID, userID uuid.UUID) error {
	return s.db.DeleteLink(ctx, linkID, userID)
}

// CreateLinkWithScraping creates a link and enriches it with scraped content
// This is the key method that moves orchestration from CLI to API
func (s *LinkService) CreateLinkWithScraping(
	ctx context.Context,
	userID uuid.UUID,
	linkCreate models.LinkCreate,
	scrapeOptions ScrapeOptions,
) (*models.Link, error) {
	// Step 1: Create the link first (even if scraping fails, we have the link)
	link, err := s.CreateLink(ctx, userID, linkCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	// Step 2: Scrape if requested
	if !scrapeOptions.Enabled {
		return link, nil
	}

	scrapeResult, err := s.scraper.ScrapeWithContext(ctx, linkCreate.URL, scrapeOptions.TimeoutSeconds)
	if err != nil {
		// Log error but don't fail - return link without enrichment
		// In production, you might want to queue this for retry
		return link, nil // or return error if you want to fail fast
	}

	// Step 3: Merge scraped content (only fill empty fields if OnlyFillEmpty is true)
	update := models.LinkUpdate{}
	changed := false

	if scrapeOptions.OnlyFillEmpty {
		// Only update fields that are currently empty
		if (link.Title == nil || strings.TrimSpace(*link.Title) == "") && scrapeResult.Title != "" {
			title := scrapeResult.Title
			update.Title = &title
			changed = true
		}

		if (link.Text == nil || strings.TrimSpace(*link.Text) == "") && scrapeResult.Text != "" {
			text := scrapeResult.Text
			update.Text = &text
			changed = true
		}
	} else {
		// Overwrite with scraped content
		if scrapeResult.Title != "" {
			title := scrapeResult.Title
			update.Title = &title
			changed = true
		}
		if scrapeResult.Text != "" {
			text := scrapeResult.Text
			update.Text = &text
			changed = true
		}
	}

	// Step 4: Update link with enriched content
	if changed {
		updated, err := s.UpdateLink(ctx, link.ID, userID, update)
		if err != nil {
			// Log error but return original link
			return link, nil
		}
		return updated, nil
	}

	return link, nil
}

// EnrichLink enriches an existing link with scraped content
func (s *LinkService) EnrichLink(
	ctx context.Context,
	linkID, userID uuid.UUID,
	scrapeOptions ScrapeOptions,
) (*models.Link, error) {
	// Get existing link
	link, err := s.GetLink(ctx, linkID, userID)
	if err != nil {
		return nil, err
	}

	// Scrape the URL
	scrapeResult, err := s.scraper.ScrapeWithContext(ctx, link.URL, scrapeOptions.TimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape URL: %w", err)
	}

	// Merge scraped content
	update := models.LinkUpdate{}
	changed := false

	if scrapeOptions.OnlyFillEmpty {
		if (link.Title == nil || strings.TrimSpace(*link.Title) == "") && scrapeResult.Title != "" {
			title := scrapeResult.Title
			update.Title = &title
			changed = true
		}
		if (link.Text == nil || strings.TrimSpace(*link.Text) == "") && scrapeResult.Text != "" {
			text := scrapeResult.Text
			update.Text = &text
			changed = true
		}
	} else {
		if scrapeResult.Title != "" {
			title := scrapeResult.Title
			update.Title = &title
			changed = true
		}
		if scrapeResult.Text != "" {
			text := scrapeResult.Text
			update.Text = &text
			changed = true
		}
	}

	if !changed {
		return link, nil
	}

	return s.UpdateLink(ctx, linkID, userID, update)
}

// ScrapeOptions configures scraping behavior
type ScrapeOptions struct {
	Enabled        bool // Whether to scrape
	TimeoutSeconds int  // Scraping timeout in seconds
	OnlyFillEmpty  bool // Only fill fields that are currently empty
}
