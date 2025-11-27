package middleware

import (
	"net/http"
	"strings"

	"link-mgmt-go/pkg/db"

	"github.com/gin-gonic/gin"
)

func RequireAuth(db *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract API key from "Bearer <key>" or just "<key>"
		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		apiKey = strings.TrimSpace(apiKey)

		user, err := db.GetUserByAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		c.Set("userID", user.ID)
		c.Set("user", user)
		c.Next()
	}
}
