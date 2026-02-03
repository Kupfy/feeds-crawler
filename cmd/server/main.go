package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Kupfy/feeds-crawler/internal/clients"
	"github.com/Kupfy/feeds-crawler/internal/data/config"
	"github.com/Kupfy/feeds-crawler/internal/handler"
	"github.com/Kupfy/feeds-crawler/internal/messaging"
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

	queue, err := messaging.NewRedisQueue(messaging.RedisQueueConfig{
		Addr:       fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:   "",
		DB:         0,
		QueueName:  "recipe_jobs",
		MaxRetries: 3,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func(q messaging.RedisQueue) {
		err := q.Close()
		if err != nil {

		}
	}(queue)

	client, err := clients.NewIngredientParserClient(cfg.IngredientServiceAddr)
	if err != nil {
		return
	}
	defer func(client *clients.IngredientParserClient) {
		err := client.Close()
		if err != nil {

		}
	}(client)

	crawlRepo := repository.NewCrawlsRepo(db)
	sitesRepo := repository.NewSiteRepo(db)
	pagesRepo := repository.NewPagesRepo(db)
	linksRepo := repository.NewLinksRepo(db)
	recipesRepo := repository.NewRecipesRepo(db)
	ingredientsRepo := repository.NewIngredientsRepo(db)

	crawlerService := service.NewCrawlerService(cfg, crawlRepo, sitesRepo, pagesRepo, linksRepo, queue)
	ingredientsService := service.NewIngredientsService(ingredientsRepo, client)
	recipeService := service.NewRecipeService(cfg, recipesRepo, pagesRepo, ingredientsService, queue)

	publicHandler := handler.NewHandler(crawlerService, recipeService)
	publicRouter := router.NewRouter(engine, middleware.NewAuthMiddleware(cfg.JwtSecret), publicHandler)

	//if err = ingredientsService.LoadIngredients(context.Background()); err != nil {
	//	log.Fatal("Failed to load ingredients")
	//}
	//
	//

	// Setup router
	publicRouter.SetupRoutes()

	// Spawn multiple workers
	if cfg.FeatureFlags.IsEnabled("recipe_extractor_workers") {
		go func() {
			numWorkers := 5
			for i := 0; i < numWorkers; i++ {
				go func(workerID int) {
					ctx := context.Background()
					log.Printf("Worker %d started", workerID)
					recipeService.ProcessRecipeMessage(ctx)
				}(i)
			}

			monitorQueue(queue)
		}()
	}

	// Start server
	log.Printf("Server listening on %s", cfg.Address())
	if err := engine.Run(cfg.Address()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Periodically check queue status
func monitorQueue(q messaging.RedisQueue) {
	ctx := context.Background()
	ticker := time.NewTicker(30 * time.Second)

	for range ticker.C {
		size, _ := q.Size(ctx)
		log.Printf("Queue size: %d jobs pending", size)

		// Peek at next job
		job, _ := q.Peek(ctx)
		if job != nil {
			log.Printf("Next job: %s (enqueued %v ago)",
				job.URL, time.Since(job.EnqueuedAt))
		}

		// Recover stale jobs (jobs stuck in processing for >5 minutes)
		recovered, _ := q.RecoverStaleJobs(ctx, 5*time.Minute)
		if recovered > 0 {
			log.Printf("Recovered %d stale jobs", recovered)
		}
	}
}
