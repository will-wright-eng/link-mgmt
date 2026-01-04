package handlers

import (
	"net/http"

	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListLinks lists all links for the authenticated user
func ListLinks(service *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		links, err := service.ListLinks(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, links)
	}
}

// CreateLink creates a new link
func CreateLink(service *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		var linkCreate models.LinkCreate
		if err := c.ShouldBindJSON(&linkCreate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		link, err := service.CreateLink(c.Request.Context(), userID, linkCreate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, link)
	}
}

// CreateLinkWithScraping creates a link and enriches it with scraped content
func CreateLinkWithScraping(service *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		var req struct {
			models.LinkCreate
			Scrape *struct {
				Enabled       bool `json:"enabled"`
				Timeout       int  `json:"timeout"` // seconds
				OnlyFillEmpty bool `json:"only_fill_empty"`
			} `json:"scrape,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Default scrape options
		scrapeOpts := services.ScrapeOptions{
			Enabled:        false,
			TimeoutSeconds: 30,
			OnlyFillEmpty:  true,
		}

		// Override with request options if provided
		if req.Scrape != nil {
			scrapeOpts.Enabled = req.Scrape.Enabled
			if req.Scrape.Timeout > 0 {
				scrapeOpts.TimeoutSeconds = req.Scrape.Timeout
			}
			scrapeOpts.OnlyFillEmpty = req.Scrape.OnlyFillEmpty
		}

		link, err := service.CreateLinkWithScraping(
			c.Request.Context(),
			userID,
			req.LinkCreate,
			scrapeOpts,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, link)
	}
}

// GetLink retrieves a single link
func GetLink(service *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		linkID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
			return
		}

		link, err := service.GetLink(c.Request.Context(), linkID, userID)
		if err != nil {
			if err.Error() == "link not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, link)
	}
}

// UpdateLink updates an existing link
func UpdateLink(service *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		linkID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
			return
		}

		var linkUpdate models.LinkUpdate
		if err := c.ShouldBindJSON(&linkUpdate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		link, err := service.UpdateLink(c.Request.Context(), linkID, userID, linkUpdate)
		if err != nil {
			if err.Error() == "link not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, link)
	}
}

// DeleteLink deletes a link
func DeleteLink(service *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		linkID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
			return
		}

		if err := service.DeleteLink(c.Request.Context(), linkID, userID); err != nil {
			if err.Error() == "link not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "link deleted"})
	}
}

// EnrichLink enriches an existing link with scraped content
func EnrichLink(service *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		linkID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
			return
		}

		var req struct {
			Timeout       int  `json:"timeout"` // seconds
			OnlyFillEmpty bool `json:"only_fill_empty"`
		}

		// Parse optional request body (defaults if not provided)
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		scrapeOpts := services.ScrapeOptions{
			Enabled:        true,
			TimeoutSeconds: 30,
			OnlyFillEmpty:  true,
		}

		if req.Timeout > 0 {
			scrapeOpts.TimeoutSeconds = req.Timeout
		}
		if c.Request.ContentLength > 0 {
			scrapeOpts.OnlyFillEmpty = req.OnlyFillEmpty
		}

		link, err := service.EnrichLink(c.Request.Context(), linkID, userID, scrapeOpts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, link)
	}
}
