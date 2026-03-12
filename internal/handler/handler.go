package handler

import (
	"net/http"

	"github.com/Kupfy/feeds-crawler/internal/data/domain"
	"github.com/Kupfy/feeds-crawler/internal/data/request"
	"github.com/Kupfy/feeds-crawler/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler interface {
	StartCrawl(c *gin.Context)
	GetCrawl(c *gin.Context)
	ProcessRecipeFromPage(c *gin.Context)
	SearchRecipes(c *gin.Context)
	GetTopRecipes(c *gin.Context)
	GetRecipeByID(c *gin.Context)
	GetRecipeBySlug(c *gin.Context)
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
	var req request.StartCrawlRequest
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
	var req request.RecipeExtractionRequest
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
	var body request.SearchRequest
	if err := c.BindQuery(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	recipes, err := h.recipeService.SearchRecipes(c.Request.Context(), body)
	if err != nil {
		domain.ToHTTPError(c.Writer, err)
		return
	}
	c.JSON(http.StatusOK, recipes)
}

func (h *handler) GetTopRecipes(c *gin.Context) {
	recipes, err := h.recipeService.GetTopRecipes(c.Request.Context(), 10)
	if err != nil {
		domain.ToHTTPError(c.Writer, err)
		return
	}
	c.JSON(http.StatusOK, recipes)
}

func (h *handler) GetRecipeByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	recipe, err := h.recipeService.GetRecipeByID(c.Request.Context(), id)
	if err != nil {
		domain.ToHTTPError(c.Writer, err)
		return
	}
	c.JSON(http.StatusOK, recipe)
}

func (h *handler) GetRecipeBySlug(c *gin.Context) {
	slug := c.Param("slug")
	recipe, err := h.recipeService.GetRecipeBySlug(c.Request.Context(), slug)
	if err != nil {
		domain.ToHTTPError(c.Writer, err)
		return
	}
	c.JSON(http.StatusOK, recipe)
}
