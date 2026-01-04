package api

import (
	"link-mgmt-go/pkg/api/handlers"
	"link-mgmt-go/pkg/api/middleware"
	"link-mgmt-go/pkg/config"
	"link-mgmt-go/pkg/db"
	"link-mgmt-go/pkg/scraper"
	"link-mgmt-go/pkg/services"

	"github.com/gin-gonic/gin"
)

func NewRouter(db *db.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Initialize services
	// Use Scraper.BaseURL from config (defaults to CLI.BaseURL if not set)
	scraperBaseURL := cfg.Scraper.BaseURL
	if scraperBaseURL == "" {
		scraperBaseURL = cfg.CLI.BaseURL
	}
	scraperService := scraper.NewScraperService(scraperBaseURL)
	linkService := services.NewLinkService(db, scraperService)

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
			links.GET("", handlers.ListLinks(linkService))
			links.POST("", handlers.CreateLink(linkService))
			links.POST("/with-scraping", handlers.CreateLinkWithScraping(linkService))
			links.GET("/:id", handlers.GetLink(linkService))
			links.PUT("/:id", handlers.UpdateLink(linkService))
			links.DELETE("/:id", handlers.DeleteLink(linkService))
			links.POST("/:id/enrich", handlers.EnrichLink(linkService))
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
