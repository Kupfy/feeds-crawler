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
	protected.Use(r.authMiddleware.RequireAuth())
	{
		// User router
		r.setupUserRoutes(protected)
	}
}

// setupUserRoutes configures user-related router
func (r Router) setupUserRoutes(rg *gin.RouterGroup) {
	crawl := rg.Group("/crawl")
	{
		// POST /api/v1/crawl
		// Body: CreateUserRequest (JSON)
		// Creates a new user
		crawl.POST("", r.userHandler.StartCrawl)

		// GET /api/v1/crawl
		// Get crawl by filters
		crawl.GET("", r.userHandler.GetCrawl)
	}
}

// healthCheck is a simple health check endpoint
func (r Router) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "go-http-server-template",
	})
}
