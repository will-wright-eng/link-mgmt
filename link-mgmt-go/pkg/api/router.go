package api

import (
	"link-mgmt-go/pkg/api/handlers"
	"link-mgmt-go/pkg/api/middleware"
	"link-mgmt-go/pkg/db"

	"github.com/gin-gonic/gin"
)

func NewRouter(db *db.DB) *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(middleware.RequestLogger())
	router.Use(middleware.ErrorHandler())

	// Health check
	router.GET("/health", handlers.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Links
		links := v1.Group("/links")
		links.Use(middleware.RequireAuth(db))
		{
			links.GET("", handlers.ListLinks(db))
			links.POST("", handlers.CreateLink(db))
			links.GET("/:id", handlers.GetLink(db))
			links.DELETE("/:id", handlers.DeleteLink(db))
		}

		// Users
		users := v1.Group("/users")
		{
			users.POST("", handlers.CreateUser(db))
			users.GET("/me", middleware.RequireAuth(db), handlers.GetCurrentUser(db))
		}
	}

	return router
}
