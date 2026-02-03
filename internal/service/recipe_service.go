package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/config"
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/messaging"
	"github.com/Kupfy/feeds-crawler/internal/repository"
	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

// DomainPattern defines CSS selectors for a specific domain
type DomainPattern struct {
	Domain       string
	Title        string
	Description  string
	Ingredients  string
	Instructions string
	PrepTime     string
	CookTime     string
	TotalTime    string
	Servings     string
	ImageURL     string
}

type RecipeService interface {
	ProcessRecipeByUrl(ctx context.Context, url string) error
	ProcessRecipeMessage(ctx context.Context)
}

type recipeService struct {
	patterns           map[string]DomainPattern
	config             config.ServiceConfig
	recipesRepo        repository.RecipesRepo
	pagesRepo          repository.PagesRepo
	ingredientsService IngredientsService
	queue              messaging.RedisQueue
}

func NewRecipeService(
	config config.ServiceConfig, recipesRepo repository.RecipesRepo,
	pagesRepo repository.PagesRepo, ingredientsService IngredientsService,
	queue messaging.RedisQueue,
) RecipeService {
	return &recipeService{
		patterns:           initializeDomainPatterns(),
		config:             config,
		recipesRepo:        recipesRepo,
		pagesRepo:          pagesRepo,
		ingredientsService: ingredientsService,
		queue:              queue,
	}
}

func (s *recipeService) ProcessRecipeMessage(ctx context.Context) {
	recipeJob, err := s.queue.Dequeue(ctx, s.config.QueueTimeout)
	if err != nil {
		log.Printf("Failed to dequeue message: %v", err)
		return
	}

	if recipeJob == nil {
		return
	}

	_ = s.ProcessRecipeByUrl(ctx, recipeJob.URL)
}

func (s *recipeService) ProcessRecipeByUrl(ctx context.Context, url string) error {
	page, err := s.pagesRepo.GetPageByUrl(ctx, url)
	if err != nil {
		log.Printf("Failed to get page by URL: %v", err)
		return err
	}

	_, err = s.extractRecipe(ctx, page.HTML, page.URL)
	if err != nil {
		log.Printf("Failed to extract recipe: %v", err)
		return err
	}
	return nil
}

// ExtractRecipe uses hybrid approach: Schema.org → Selectors → LLM fallback
func (s *recipeService) extractRecipe(ctx context.Context, html string, pageURL string) (*entity.Recipe, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	domain := extractDomain(pageURL)
	log.Printf("Extracting recipe from domain: %s", domain)
	var recipe *entity.Recipe

	// Step 1: Try schema.org structured data (free, fast)
	if recipe, err = s.extractFromSchema(ctx, doc); err == nil && recipe != nil {
		log.Printf("✓ Extracted recipe from schema.org: %s", recipe.Title)
		return recipe, nil
	}

	// Step 2: Try domain-specific CSS selectors (free, fast)
	if pattern, exists := s.patterns[domain]; exists {
		if recipe, err := s.extractWithPattern(ctx, doc, pattern); err == nil && recipe != nil {
			log.Printf("✓ Extracted recipe with pattern for %s: %s", domain, recipe.Title)
			return recipe, nil
		}
	}

	// Step 3: Try LLM with cleaned HTML (if enabled)
	if s.config.FeatureFlags.IsEnabled("llm_cleaned_html") {
		cleanedData := s.extractCleanedData(doc)
		return s.extractWithLLMCleaned(cleanedData)
	}

	// Step 4: Try LLM with full HTML (if enabled)
	if s.config.FeatureFlags.IsEnabled("llm_full_html") {
		return s.extractWithLLMFull(html)
	}

	if recipe != nil {
		err = s.recipesRepo.SaveRecipe(ctx, *recipe)
		if err != nil {
			log.Printf("failed to save recipe: %w", err)
		}
		return recipe, err
	}

	return nil, errors.New("no extraction method succeeded")
}

// extractFromSchema extracts recipe from JSON-LD schema.org markup
func (s *recipeService) extractFromSchema(ctx context.Context, doc *goquery.Document) (*entity.Recipe, error) {
	var recipe *entity.Recipe

	doc.Find("script[type='application/ld+json']").Each(func(i int, script *goquery.Selection) {
		if recipe != nil {
			return // Already found one
		}

		jsonText := script.Text()

		// Try to parse as single object
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonText), &data); err == nil {
			if r := s.parseSchemaRecipe(ctx, data); r != nil {
				recipe = r
				return
			}
		}

		// Try to parse as array (sometimes multiple schemas)
		var dataArray []map[string]interface{}
		if err := json.Unmarshal([]byte(jsonText), &dataArray); err == nil {
			for _, item := range dataArray {
				if r := s.parseSchemaRecipe(ctx, item); r != nil {
					recipe = r
					return
				}
			}
		}
	})

	if recipe == nil {
		return nil, errors.New("no schema.org recipe found")
	}

	return recipe, nil
}

// parseSchemaRecipe parses a schema.org Recipe object
func (s *recipeService) parseSchemaRecipe(ctx context.Context, data map[string]interface{}) *entity.Recipe {
	schemaType, ok := data["@type"].(string)
	if !ok || schemaType != "Recipe" {
		return nil
	}

	recipe := &entity.Recipe{}

	if title, ok := data["name"].(string); ok {
		recipe.Title = title
	}

	if desc, ok := data["description"].(string); ok {
		recipe.Blurb = desc
	}

	// Extract ingredients (can be array of strings or objects)
	if ingredients, ok := data["recipeIngredient"].([]interface{}); ok {
		for _, ing := range ingredients {
			if str, ok := ing.(string); ok {
				ingredient, err := s.ingredientsService.ParseIngredientLine(context.Background(), str)
				if err != nil {
					log.Printf("failed to parse ingredient: %v", err)
					continue
				}
				recipe.Ingredients = append(recipe.Ingredients, ingredient)
			}
		}
	}

	// Extract instructions (can be string, array, or HowToStep objects)
	if instructions, ok := data["recipeInstructions"].([]interface{}); ok {
		for _, inst := range instructions {
			var methodStr string
			var method dto.MethodItem
			if str, ok := inst.(string); ok {
				methodStr = str
			} else if obj, ok := inst.(map[string]interface{}); ok {
				if text, ok := obj["text"].(string); ok {
					methodStr = text
				}
			}
			if methodStr != "" {
				_ = method.UnmarshalText([]byte(methodStr))
				recipe.Method = append(recipe.Method, method)
			}
		}
	} else if inst, ok := data["recipeInstructions"].(string); ok {
		var method dto.MethodItem
		_ = method.UnmarshalText([]byte(inst))
		recipe.Method = append(recipe.Method, method)
	}

	// Extract times
	if prepTime, ok := data["prepTime"].(string); ok {
		ptime, err := strconv.Atoi(prepTime)
		if err == nil {
			recipe.PrepTime = &ptime
		}
	}
	if cookTime, ok := data["cookTime"].(string); ok {
		ctime, err := strconv.Atoi(cookTime)
		if err == nil {
			recipe.CookingTime = ctime
		}
	}
	//if totalTime, ok := data["totalTime"].(string); ok {
	//	recipe.TotalTime = totalTime
	//}

	// Extract servings
	if yield, ok := data["recipeYield"].(string); ok {
		y, err := strconv.Atoi(yield)
		if err == nil {
			recipe.Serving = y
		}
	} else if yieldArray, ok := data["recipeYield"].([]interface{}); ok && len(yieldArray) > 0 {
		if yieldStr, ok := yieldArray[0].(string); ok {
			y, err := strconv.Atoi(yieldStr)
			if err == nil {
				recipe.Serving = y
			}
		}
	}

	//// Extract category/cuisine
	//if category, ok := data["recipeCategory"].(string); ok {
	//	recipe.Category = category
	//}
	//if cuisine, ok := data["recipeCuisine"].(string); ok {
	//	recipe.Cuisine = cuisine
	//}

	// Extract author
	if author, ok := data["author"].(map[string]interface{}); ok {
		if name, ok := author["name"].(string); ok {
			recipe.Author = name
		}
	} else if authorStr, ok := data["author"].(string); ok {
		recipe.Author = authorStr
	}

	//// Extract image
	//if image, ok := data["image"].(string); ok {
	//	recipe.ImageURL = image
	//} else if imageObj, ok := data["image"].(map[string]interface{}); ok {
	//	if url, ok := imageObj["url"].(string); ok {
	//		recipe.ImageURL = url
	//	}
	//} else if imageArray, ok := data["image"].([]interface{}); ok && len(imageArray) > 0 {
	//	if imageStr, ok := imageArray[0].(string); ok {
	//		recipe.ImageURL = imageStr
	//	}
	//}

	// Validate we got at least title and ingredients
	if recipe.Title == "" || len(recipe.Ingredients) == 0 {
		return nil
	}

	return recipe
}

// extractWithPattern uses domain-specific CSS selectors
func (s *recipeService) extractWithPattern(ctx context.Context, doc *goquery.Document, pattern DomainPattern) (*entity.Recipe, error) {
	recipe := &entity.Recipe{}

	// Extract title
	if pattern.Title != "" {
		recipe.Title = strings.TrimSpace(doc.Find(pattern.Title).First().Text())
	}

	// Extract description
	if pattern.Description != "" {
		recipe.Blurb = strings.TrimSpace(doc.Find(pattern.Description).First().Text())
	}

	// Extract ingredients
	if pattern.Ingredients != "" {
		doc.Find(pattern.Ingredients).Each(func(i int, gs *goquery.Selection) {
			ingredient := strings.TrimSpace(gs.Text())
			if ingredient != "" {
				ing, err := s.ingredientsService.ParseIngredientLine(ctx, ingredient)
				if err != nil {
					log.Printf("failed to parse ingredient: %v", err)
					return
				}
				recipe.Ingredients = append(recipe.Ingredients, ing)
			}
		})
	}

	// Extract instructions
	if pattern.Instructions != "" {
		doc.Find(pattern.Instructions).Each(func(i int, s *goquery.Selection) {
			instruction := strings.TrimSpace(s.Text())
			if instruction != "" {
				methodItm := dto.MethodItem{Content: instruction}
				recipe.Method = append(recipe.Method, methodItm)
			}
		})
	}

	// Extract metadata
	if pattern.PrepTime != "" {
		prepStr := strings.TrimSpace(doc.Find(pattern.PrepTime).First().Text())
		if prepStr != "" {
			ptime, err := strconv.Atoi(prepStr)
			if err == nil {
				recipe.PrepTime = &ptime
			}
		}
	}
	if pattern.CookTime != "" {
		cookStr := strings.TrimSpace(doc.Find(pattern.CookTime).First().Text())
		if cookStr != "" {
			ctime, err := strconv.Atoi(cookStr)
			if err == nil {
				recipe.CookingTime = ctime
			}
		}
	}

	if pattern.Servings != "" {
		servingStr := strings.TrimSpace(doc.Find(pattern.Servings).First().Text())
		if servingStr != "" {
			serving, err := strconv.Atoi(servingStr)
			if err == nil {
				recipe.Serving = serving
			}
		}
	}
	//if pattern.ImageURL != "" {
	//	if img, exists := doc.Find(pattern.ImageURL).First().Attr("src"); exists {
	//		recipe.ImageURL = img
	//	}
	//}

	// Validate we got essential data
	if recipe.Title == "" || len(recipe.Ingredients) == 0 {
		return nil, errors.New("pattern extraction failed: missing essential data")
	}

	return recipe, nil
}

// extractCleanedData extracts clean text data for LLM processing
func (s *recipeService) extractCleanedData(doc *goquery.Document) map[string]string {
	cleaned := make(map[string]string)

	// Extract potential title
	if title := doc.Find("h1").First().Text(); title != "" {
		cleaned["title"] = strings.TrimSpace(title)
	}

	// Extract all text that might be ingredients (common patterns)
	var ingredientText []string
	doc.Find("ul li, .ingredient, [class*='ingredient'], [class*='IngredientsItem']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" && len(text) < 200 { // Ingredients are usually short
			ingredientText = append(ingredientText, text)
		}
	})
	if len(ingredientText) > 0 {
		cleaned["ingredients"] = strings.Join(ingredientText, "\n")
	}

	// Extract all text that might be instructions
	var instructionText []string
	doc.Find("ol li, .instruction, .step, [class*='instruction'], [class*='step'], [class*='direction']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			instructionText = append(instructionText, text)
		}
	})
	if len(instructionText) > 0 {
		cleaned["instructions"] = strings.Join(instructionText, "\n")
	}

	// Extract potential metadata
	doc.Find("[class*='time'], [class*='Time'], [class*='serving'], [class*='Serving']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			cleaned["metadata"] = cleaned["metadata"] + " " + text
		}
	})

	return cleaned
}

// extractWithLLMCleaned uses LLM with cleaned/extracted data
func (s *recipeService) extractWithLLMCleaned(cleanedData map[string]string) (*entity.Recipe, error) {
	log.Printf("LLM (cleaned) called with data: %v", cleanedData)
	panic("LLM cleaned extraction not implemented - feature flag: llm_cleaned_html")
}

// extractWithLLMFull uses LLM with full HTML
func (s *recipeService) extractWithLLMFull(html string) (*entity.Recipe, error) {
	log.Printf("LLM (full) called with HTML length: %d", len(html))
	panic("LLM full extraction not implemented - feature flag: llm_full_html")
}

func (s *recipeService) SearchRecipes(ctx context.Context, searchTerm string) ([]dto.RecipeSearchResult, error) {
	return s.recipesRepo.GetRecipesBySearchQuery(ctx, searchTerm)
}

func (s *recipeService) GetRecipeByID(ctx context.Context, id uuid.UUID) (entity.Recipe, error) {
	return s.recipesRepo.GetByID(ctx, id)
}

// initializeDomainPatterns creates patterns for known recipe sites
func initializeDomainPatterns() map[string]DomainPattern {
	return map[string]DomainPattern{
		"books.ottolenghi.co.uk": {
			Domain:       "books.ottolenghi.co.uk",
			Title:        ".print-recipe-heading",
			Description:  ".accordion__content .jsAccordionContentInner p",
			Ingredients:  ".recipe__aside ul li",
			Instructions: ".recipe-list ol li",
			Servings:     ".recipe-meta--yield",
			ImageURL:     ".recipe__aside img.wp-post-image",
		},
		"allrecipes.com": {
			Domain:       "allrecipes.com",
			Title:        "h1.article-heading",
			Ingredients:  "ul.mntl-structured-ingredients__list li",
			Instructions: "#mntl-sc-page_1-0 ol li p",
			PrepTime:     ".mntl-recipe-details__label:contains('Prep Time') + .mntl-recipe-details__value",
			CookTime:     ".mntl-recipe-details__label:contains('Cook Time') + .mntl-recipe-details__value",
			TotalTime:    ".mntl-recipe-details__label:contains('Total Time') + .mntl-recipe-details__value",
			Servings:     "#mntl-recipe-details_1-0 .mntl-recipe-details__value",
			ImageURL:     "img.primary-image__image",
		},
		"foodnetwork.com": {
			Domain:       "foodnetwork.com",
			Title:        "h1.o-AssetTitle__a-Headline",
			Ingredients:  "ul.o-Ingredients__m-List li.o-Ingredients__a-IngredientsItem",
			Instructions: "ol.o-Method__m-Step li.o-Method__m-Step",
			TotalTime:    "span.o-RecipeInfo__a-Description",
			Servings:     "span.o-RecipeInfo__a-Description",
		},
		"simplyrecipes.com": {
			Domain:       "simplyrecipes.com",
			Title:        "h1.heading-title",
			Description:  ".entry-details__description",
			Ingredients:  "ul.structured-ingredients li",
			Instructions: "ol.structured-project-steps li",
			PrepTime:     ".meta-text__data:contains('Prep:') time",
			CookTime:     ".meta-text__data:contains('Cook:') time",
			Servings:     ".meta-text__data:contains('Serves:') span",
		},
		"tasty.co": {
			Domain:       "tasty.co",
			Title:        "h1.recipe-name",
			Description:  ".recipe-description",
			Ingredients:  "ul.ingredients li",
			Instructions: "ol.prep-steps li",
			Servings:     ".servings-display",
		},
		"bonappetit.com": {
			Domain:       "bonappetit.com",
			Title:        "h1[data-testid='ContentHeaderHed']",
			Ingredients:  "div[data-testid='IngredientList'] p",
			Instructions: "div[data-testid='InstructionsWrapper'] li",
			Servings:     "div[data-testid='servings-value']",
		},
	}
}

// extractDomain gets the domain from a URL
func extractDomain(pageURL string) string {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}

	// Remove www. prefix if present
	domain := parsed.Hostname()
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	return domain
}
