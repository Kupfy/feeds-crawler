package router

import (
	"github.com/Kupfy/feeds-crawler/internal/handler"
	"github.com/Kupfy/feeds-crawler/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Router manages all application router
type Router struct {
	engine         *gin.Engine
	authMiddleware middleware.AuthMiddleware
	userHandler    handler.Handler
}

// NewRouter creates a new router with all dependencies
func NewRouter(
	engine *gin.Engine,
	authMiddleware middleware.AuthMiddleware,
	userHandler handler.Handler,
) *Router {
	return &Router{
		engine:         engine,
		authMiddleware: authMiddleware,
		userHandler:    userHandler,
	}
}

// SetupRoutes configures all application router
func (r Router) SetupRoutes() {
	// API v1 group
	v1 := r.engine.Group("/api/v1")

	// Health check (no auth required)
	v1.GET("/health", r.healthCheck)

	// Protected router - require authentication
	protected := v1.Group("")
	protected.Use(r.authMiddleware.OptionalAuth())
	{
		// User router
		r.setupUserRoutes(protected)
	}
}

// setupUserRoutes configures user-related router
func (r Router) setupUserRoutes(rg *gin.RouterGroup) {
	crawl := rg.Group("/crawls")
	crawl.Use(r.authMiddleware.RequireAuth())
	{
		// POST /api/v1/crawl
		// Body: CreateUserRequest (JSON)
		// Creates a new user
		crawl.POST("", r.userHandler.StartCrawl)

		// GET /api/v1/crawl
		// Get crawl by filters
		crawl.GET("/search", r.userHandler.GetCrawl)
	}

	recipe := rg.Group("/recipes")
	{
		recipe.POST("/from-path", r.userHandler.ProcessRecipeFromPage)
		//Use(r.authMiddleware.RequireAuth())

		recipe.GET("/search", r.userHandler.SearchRecipes)

		recipe.GET("/:id", r.userHandler.GetRecipeByID)

		recipe.GET("/slug/:slug", r.userHandler.GetRecipeBySlug)

		recipe.GET("/top", r.userHandler.GetTopRecipes)
	}
}

// healthCheck is a simple health check endpoint
func (r Router) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "go-http-server-template",
	})
}
