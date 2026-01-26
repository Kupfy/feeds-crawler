package main

import (
	"fmt"
	"log"

	"github.com/Kupfy/feeds-crawler/internal/data/config"
	"github.com/Kupfy/feeds-crawler/internal/handler"
	"github.com/Kupfy/feeds-crawler/internal/middleware"
	"github.com/Kupfy/feeds-crawler/internal/repository"
	"github.com/Kupfy/feeds-crawler/internal/router"
	"github.com/Kupfy/feeds-crawler/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Basic config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode based on environment
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin engine
	engine := gin.New()

	if engine.SetTrustedProxies(nil) != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	// Apply global middleware
	engine.Use(middleware.RecoveryMiddleware())
	engine.Use(middleware.LoggerMiddleware())

	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%v port=%v user=%v password=%v "+
		"dbname=postgres sslmode=disable", cfg.DbHost, cfg.DBPort, cfg.DbUser, cfg.DbPassword))
	if err != nil {
		log.Fatalln(err)
	}

	crawlRepo := repository.NewCrawlsRepo(db)
	siteRepo := repository.NewSiteRepo(db)
	pageRepo := repository.NewPagesRepo(db)

	crawlerService := service.NewCrawlerService(cfg, crawlRepo, siteRepo, pageRepo)
	recipeService := service.NewRecipeService(cfg)

	publicHandler := handler.NewHandler(crawlerService, recipeService)
	publicRouter := router.NewRouter(engine, middleware.NewAuthMiddleware(cfg.JwtSecret), publicHandler)

	// Setup router
	publicRouter.SetupRoutes()

	// Start server
	log.Printf("Server listening on %s", cfg.Address())
	if err := engine.Run(cfg.Address()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
