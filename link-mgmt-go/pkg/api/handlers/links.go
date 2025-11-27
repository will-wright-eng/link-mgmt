package handlers

import (
	"net/http"

	"link-mgmt-go/pkg/db"
	"link-mgmt-go/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ListLinks(db *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		links, err := db.GetLinksByUserID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, links)
	}
}

func CreateLink(db *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		var linkCreate models.LinkCreate
		if err := c.ShouldBindJSON(&linkCreate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		link, err := db.CreateLink(c.Request.Context(), userID, linkCreate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, link)
	}
}

func GetLink(db *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		linkID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
			return
		}

		link, err := db.GetLinkByID(c.Request.Context(), linkID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, link)
	}
}

func UpdateLink(db *db.DB) gin.HandlerFunc {
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

		link, err := db.UpdateLink(c.Request.Context(), linkID, userID, linkUpdate)
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

func DeleteLink(db *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		linkID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
			return
		}

		if err := db.DeleteLink(c.Request.Context(), linkID, userID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "link deleted"})
	}
}
