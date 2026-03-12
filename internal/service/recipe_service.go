package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/config"
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/request"
	"github.com/Kupfy/feeds-crawler/internal/data/response"
	"github.com/Kupfy/feeds-crawler/internal/messaging"
	"github.com/Kupfy/feeds-crawler/internal/repository"
	"github.com/Kupfy/feeds-crawler/internal/util"
	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// DomainPattern defines CSS selectors for a specific domain
type HeadingPattern struct {
	Element   string
	Component string
	Text      string
}

type DomainPattern struct {
	Domain       string
	Title        string
	Author       string
	Publication  string
	Description  string
	Ingredients  HeadingPattern
	Instructions HeadingPattern
	PrepTime     string
	CookTime     string
	TotalTime    string
	Servings     string
	Notes        string
	ImageURL     string
}

type RecipeService interface {
	ProcessRecipeByUrl(ctx context.Context, url string) (*entity.Recipe, error)
	ProcessRecipeMessage(ctx context.Context, recipeJob *messaging.RecipeJob) error
	SearchRecipes(ctx context.Context, searchRequest request.SearchRequest) (response.PagedResponse[response.RecipeSearchResult], error)
	GetTopRecipes(ctx context.Context, limit int) ([]response.RecipeSearchResult, error)
	GetRecipeByID(ctx context.Context, id uuid.UUID) (entity.Recipe, error)
	GetRecipeBySlug(ctx context.Context, slug string) (response.RecipeResponse, error)
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

func (s *recipeService) ProcessRecipeMessage(ctx context.Context, recipeJob *messaging.RecipeJob) error {
	if recipeJob == nil {
		return nil
	}

	_, err := s.ProcessRecipeByUrl(ctx, recipeJob.URL)
	return err
}

func (s *recipeService) ProcessRecipeByUrl(ctx context.Context, url string) (*entity.Recipe, error) {
	page, err := s.pagesRepo.GetPageByUrl(ctx, url)
	if err != nil {
		log.Printf("Failed to get page by URL: %v", err)
		return nil, err
	}

	recipe, err := s.extractRecipe(ctx, page.HTML, page.URL)
	if err != nil {
		log.Printf("Failed to extract recipe: %v", err)
		return nil, err
	}

	// add slug
	addSlug(recipe, false)

	if recipe != nil {
		recipe, err = s.recipesRepo.SaveRecipe(*recipe, ctx)
		if util.IsUniqueViolation(err) {
			addSlug(recipe, true)
			recipe, err = s.recipesRepo.SaveRecipe(*recipe, ctx)
		}
		if err != nil {
			log.Printf("failed to save recipe: %w", err)
		}
		return recipe, err
	}
	return nil, nil
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

	if recipe, err = s.extractFromSchema(ctx, doc); err == nil && recipe != nil {
		log.Printf("✓ Extracted recipe from schema.org: %s", recipe.Title)
		return recipe, nil
	}

	if pattern, exists := s.patterns[domain]; exists {
		if recipe, err := s.extractWithPattern(ctx, doc, pattern); err == nil && recipe != nil {
			log.Printf("✓ Extracted recipe with pattern for %s: %s", domain, recipe.Title)
			return recipe, nil
		}
	}

	if s.config.FeatureFlags.IsEnabled("llm_cleaned_html") {
		cleanedData := s.extractCleanedData(doc)
		return s.extractWithLLMCleaned(cleanedData)
	}

	if s.config.FeatureFlags.IsEnabled("llm_full_html") {
		return s.extractWithLLMFull(html)
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

	// Extract ingredients (can be an array of strings or objects)
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
			if str, ok := inst.(string); ok {
				methodStr = str
			} else if obj, ok := inst.(map[string]interface{}); ok {
				if text, ok := obj["text"].(string); ok {
					methodStr = text
				}
			}
			if methodStr != "" {
				method := dto.NewMethodItem(methodStr, nil)
				recipe.Method = append(recipe.Method, method)
			}
		}
	} else if inst, ok := data["recipeInstructions"].(string); ok {
		method := dto.NewMethodItem(inst, nil)
		recipe.Method = append(recipe.Method, method)
	}

	// Extract times
	if prepTime, ok := data["prepTime"].(string); ok {
		ptime, err := util.ParseDuration(prepTime)
		if err == nil {
			recipe.PrepTime = util.ToPtr(int(ptime.Minutes()))
		}
	}
	if cookTime, ok := data["cookTime"].(string); ok {
		ctime, err := util.ParseDuration(cookTime)
		if err == nil {
			recipe.CookingTime = util.ToPtr(int(ctime.Minutes()))
		}
	}

	// Extract servings
	var serving *dto.Serving
	if yield, ok := data["recipeYield"].(string); ok {
		serving = util.ParseServing(yield)

	} else if yieldArray, ok := data["recipeYield"].([]interface{}); ok && len(yieldArray) > 0 {
		if yieldStr, ok := yieldArray[0].(string); ok {
			serving = util.ParseServing(yieldStr)
		}
	}
	if serving != nil {
		recipe.Serving = *serving
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

	if pattern.Author != "" {
		recipe.Author = strings.TrimSpace(doc.Find(pattern.Author).First().Text())
	}

	if pattern.Publication != "" {
		recipe.Publication = strings.TrimSpace(doc.Find(pattern.Publication).First().Text())
	}

	// Extract description
	if pattern.Description != "" {
		recipe.Blurb = strings.TrimSpace(doc.Find(pattern.Description).First().Text())
	}

	// Extract ingredients
	if pattern.Ingredients.Element != "" {
		recipe.Ingredients = s.extractItemsWithHeadings(ctx, doc.Find(pattern.Ingredients.Element), pattern)
	} else if pattern.Ingredients.Text != "" {
		doc.Find(pattern.Ingredients.Text).Each(func(i int, gs *goquery.Selection) {
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
	if pattern.Instructions.Text != "" {
		doc.Find(pattern.Instructions.Text).Each(func(i int, s *goquery.Selection) {
			instruction := strings.TrimSpace(s.Text())
			if instruction != "" {
				methodItm := dto.MethodItem{Content: instruction}
				recipe.Method = append(recipe.Method, methodItm)
			}
		})
	}

	if pattern.Notes != "" {
		notes := strings.TrimSpace(doc.Find(pattern.Notes).First().Text())
		if notes != "" {
			recipe.Note = &notes
		}
	}

	// Extract metadata
	if pattern.PrepTime != "" {
		prepStr := strings.TrimSpace(doc.Find(pattern.PrepTime).First().Text())
		if prepStr != "" {
			ptime, err := util.ParseDuration(prepStr)
			if err == nil {
				recipe.PrepTime = util.ToPtr(int(ptime.Minutes()))
			} else {
				log.Printf("failed to parse prep time: %v", prepStr)
			}
		}
	}
	if pattern.CookTime != "" {
		cookStr := strings.TrimSpace(doc.Find(pattern.CookTime).First().Text())
		if cookStr != "" {
			ctime, err := util.ParseDuration(cookStr)
			if err == nil {
				recipe.CookingTime = util.ToPtr(int(ctime.Minutes()))
			} else {
				log.Printf("failed to parse cook time: %v", cookStr)
			}
		}
	}

	if pattern.Servings != "" {
		servingStr := strings.TrimSpace(doc.Find(pattern.Servings).First().Text())
		if servingStr != "" {
			serving := util.ParseServing(servingStr)
			if serving != nil {
				recipe.Serving = *serving
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

func (s *recipeService) extractIngredientsWithHeadings(ctx context.Context, selection *goquery.Selection, pattern DomainPattern) dto.Ingredients {
	var ingredients dto.Ingredients
	var currentHeading *string
	//strings.Split(pattern.Ingredients.Text, " ")

	selection.Children().Each(func(i int, sl *goquery.Selection) {
		node := sl.Get(0)

		switch node.Data {
		case pattern.Ingredients.Component:
			currentHeading = util.ToPtr(strings.TrimSpace(sl.Text()))

		case pattern.Ingredients.Text:
			sl.Find(pattern.Ingredients.Text).Each(func(j int, li *goquery.Selection) {
				ing, err := s.ingredientsService.ParseIngredientLine(ctx, li.Text())
				if err != nil {
					log.Printf("failed to parse ingredient: %v", err)
					return
				}
				ing.Component = currentHeading
			})
		}
	})

	return ingredients
}

func (s *recipeService) extractItemsWithHeadings(ctx context.Context, container *goquery.Selection, pattern DomainPattern) dto.Ingredients {
	var ingredients dto.Ingredients
	var currentHeading *string

	// Find all headings and items, then sort by DOM order
	type element struct {
		isHeading bool
		text      string
		node      *html.Node
	}

	var elements []element

	// Collect headings
	container.Find(pattern.Ingredients.Component).Each(func(i int, sl *goquery.Selection) {
		elements = append(elements, element{
			isHeading: true,
			text:      strings.TrimSpace(sl.Text()),
			node:      sl.Get(0),
		})
	})

	// Collect items
	container.Find(pattern.Ingredients.Text).Each(func(i int, sl *goquery.Selection) {
		elements = append(elements, element{
			isHeading: false,
			text:      strings.TrimSpace(sl.Text()),
			node:      sl.Get(0),
		})
	})

	// Sort by DOM order
	sort.Slice(elements, func(i, j int) bool {
		return compareNodePosition(elements[i].node, elements[j].node) < 0
	})

	// Process in order
	for _, elem := range elements {
		if elem.isHeading {
			currentHeading = util.ToPtr(elem.text)
		} else {
			ing, err := s.ingredientsService.ParseIngredientLine(ctx, elem.text)
			if err != nil {
				log.Printf("failed to parse ingredient: %v", err)
				continue
			}
			ing.Component = currentHeading
			ingredients = append(ingredients, ing)
		}
	}

	return ingredients
}

// compareNodePosition returns -1 if a comes before b, 1 if b comes before a, 0 if equal
func compareNodePosition(a, b *html.Node) int {
	if a == b {
		return 0
	}

	// Build path to root for both nodes
	pathA := getPathToRoot(a)
	pathB := getPathToRoot(b)

	// Find common ancestor
	minLen := len(pathA)
	if len(pathB) < minLen {
		minLen = len(pathB)
	}

	// Walk from root until paths diverge
	for i := 0; i < minLen; i++ {
		if pathA[i] != pathB[i] {
			// Find position among siblings
			return compareSiblings(pathA[i], pathB[i])
		}
	}

	// One is ancestor of the other - ancestor comes first
	if len(pathA) < len(pathB) {
		return -1
	}
	return 1
}

func getPathToRoot(node *html.Node) []*html.Node {
	var path []*html.Node
	for n := node; n != nil; n = n.Parent {
		path = append([]*html.Node{n}, path...)
	}
	return path
}

func compareSiblings(a, b *html.Node) int {
	if a.Parent == nil || b.Parent == nil {
		return 0
	}

	for c := a.Parent.FirstChild; c != nil; c = c.NextSibling {
		if c == a {
			return -1
		}
		if c == b {
			return 1
		}
	}
	return 0
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

func (s *recipeService) SearchRecipes(ctx context.Context, searchRequest request.SearchRequest) (response.PagedResponse[response.RecipeSearchResult], error) {
	return s.recipesRepo.GetRecipesBySearchQuery(ctx, searchRequest)
}

func (s *recipeService) GetTopRecipes(ctx context.Context, limit int) ([]response.RecipeSearchResult, error) {
	return s.recipesRepo.GetTopRecipes(ctx, limit)
}

func (s *recipeService) GetRecipeByID(ctx context.Context, id uuid.UUID) (entity.Recipe, error) {
	return s.recipesRepo.GetByID(ctx, id)
}

func (s *recipeService) GetRecipeBySlug(ctx context.Context, slug string) (response.RecipeResponse, error) {
	recipe, err := s.recipesRepo.GetBySlug(ctx, slug)
	if err != nil {
		return response.RecipeResponse{}, err
	}

	return response.NewRecipeResponse(recipe), nil
}

func addSlug(recipe *entity.Recipe, generateNewID bool) {
	slug := util.ToSlug(recipe.Title) + "-" + util.ToSlug(recipe.Author) + "-" + util.ToSlug(recipe.Publication)
	if generateNewID {
		slug += "-" + util.NewShortID()
	}
	recipe.Slug = slug
}

// initializeDomainPatterns creates patterns for known recipe sites
func initializeDomainPatterns() map[string]DomainPattern {
	return map[string]DomainPattern{
		"books.ottolenghi.co.uk": {
			Domain:      "books.ottolenghi.co.uk",
			Title:       ".print-recipe-heading",
			Publication: "a.current",
			Description: ".accordion__content .jsAccordionContentInner p",
			Ingredients: HeadingPattern{
				Element:   "div#print_me.recipe__aside",
				Component: "h3",
				Text:      "li",
			},
			Instructions: HeadingPattern{Text: ".recipe-list ol li"},
			Servings:     ".recipe-meta--yield",
			Notes:        ".recipe__aside.pt-0 p",
			ImageURL:     ".recipe__aside img.wp-post-image",
		},
		"greatbritishchefs.com": {
			Domain: "greatbritishchefs.com",
			Title:  "h1.font-stack-title",
			Ingredients: HeadingPattern{
				Element:   "div.px-4",
				Component: "h4",
				Text:      "ul li"},
			Instructions: HeadingPattern{Text: "ol.instructions li"},
			Servings:     ".recipe-meta__item--servings",
			ImageURL:     "img.recipe-image__image",
		},
		"allrecipes.com": {
			Domain:       "allrecipes.com",
			Title:        "h1.article-heading",
			Ingredients:  HeadingPattern{Text: "ul.mntl-structured-ingredients__list li"},
			Instructions: HeadingPattern{Text: "#mntl-sc-page_1-0 ol li p"},
			PrepTime:     ".mntl-recipe-details__label:contains('Prep Time') + .mntl-recipe-details__value",
			CookTime:     ".mntl-recipe-details__label:contains('Cook Time') + .mntl-recipe-details__value",
			TotalTime:    ".mntl-recipe-details__label:contains('Total Time') + .mntl-recipe-details__value",
			Servings:     "#mntl-recipe-details_1-0 .mntl-recipe-details__value",
			ImageURL:     "img.primary-image__image",
		},
		"foodnetwork.com": {
			Domain:       "foodnetwork.com",
			Title:        "h1.o-AssetTitle__a-Headline",
			Ingredients:  HeadingPattern{Text: "ul.o-Ingredients__m-List li.o-Ingredients__a-IngredientsItem"},
			Instructions: HeadingPattern{Text: "ol.o-Method__m-Step li.o-Method__m-Step"},
			TotalTime:    "span.o-RecipeInfo__a-Description",
			Servings:     "span.o-RecipeInfo__a-Description",
		},
		"simplyrecipes.com": {
			Domain:       "simplyrecipes.com",
			Title:        "h1.heading-title",
			Description:  ".entry-details__description",
			Ingredients:  HeadingPattern{Text: "ul.structured-ingredients li"},
			Instructions: HeadingPattern{Text: "ol.structured-project-steps li"},
			PrepTime:     ".meta-text__data:contains('Prep:') time",
			CookTime:     ".meta-text__data:contains('Cook:') time",
			Servings:     ".meta-text__data:contains('Serves:') span",
		},
		"tasty.co": {
			Domain:       "tasty.co",
			Title:        "h1.recipe-name",
			Description:  ".recipe-description",
			Ingredients:  HeadingPattern{Text: "ul.ingredients li"},
			Instructions: HeadingPattern{Text: "ol.prep-steps li"},
			Servings:     ".servings-display",
		},
		"bonappetit.com": {
			Domain:       "bonappetit.com",
			Title:        "h1[data-testid='ContentHeaderHed']",
			Ingredients:  HeadingPattern{Text: "div[data-testid='IngredientList'] p"},
			Instructions: HeadingPattern{Text: "div[data-testid='InstructionsWrapper'] li"},
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
