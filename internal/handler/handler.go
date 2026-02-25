package handler

import (
	"net/http"

	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler interface {
	StartCrawl(c *gin.Context)
	GetCrawl(c *gin.Context)
	ProcessRecipeFromPage(c *gin.Context)
	SearchRecipes(c *gin.Context)
	GetRecipeByID(c *gin.Context)
}

type handler struct {
	crawlerService service.CrawlerService
	recipeService  service.RecipeService
}

func NewHandler(
	crawlerService service.CrawlerService, recipeService service.RecipeService,
) Handler {
	return &handler{crawlerService, recipeService}
}

func (h *handler) StartCrawl(c *gin.Context) {
	var req dto.StartCrawlRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	jobID, err := h.crawlerService.StartCrawl(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
}

func (h *handler) GetCrawl(c *gin.Context) {
	//TODO implement me
	panic("implement me")
}

func (h *handler) ProcessRecipeFromPage(c *gin.Context) {
	var req dto.RecipeExtractionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipe, err := h.recipeService.ProcessRecipeByUrl(c.Request.Context(), req.Url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, *recipe)
}

func (h *handler) SearchRecipes(c *gin.Context) {

}

func (h *handler) GetRecipeByID(c *gin.Context) {

}
