package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/config"
	"github.com/PuerkitoBio/goquery"
)

// Recipe represents structured recipe data
type Recipe struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Ingredients  []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
	PrepTime     string   `json:"prepTime,omitempty"`
	CookTime     string   `json:"cookTime,omitempty"`
	TotalTime    string   `json:"totalTime,omitempty"`
	Servings     string   `json:"servings,omitempty"`
	Cuisine      string   `json:"cuisine,omitempty"`
	Category     string   `json:"category,omitempty"`
	Author       string   `json:"author,omitempty"`
	ImageURL     string   `json:"imageUrl,omitempty"`
}

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
}

type recipeService struct {
	patterns map[string]DomainPattern
	config   config.ServiceConfig
}

func NewRecipeService(config config.ServiceConfig) RecipeService {
	return &recipeService{
		patterns: initializeDomainPatterns(),
		config:   config,
	}
}

// ExtractRecipe uses hybrid approach: Schema.org → Selectors → LLM fallback
func (s *recipeService) ExtractRecipe(html string, pageURL string) (*Recipe, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	domain := extractDomain(pageURL)
	log.Printf("Extracting recipe from domain: %s", domain)

	// Step 1: Try schema.org structured data (free, fast)
	if recipe, err := s.extractFromSchema(doc); err == nil && recipe != nil {
		log.Printf("✓ Extracted recipe from schema.org: %s", recipe.Title)
		return recipe, nil
	}

	// Step 2: Try domain-specific CSS selectors (free, fast)
	if pattern, exists := s.patterns[domain]; exists {
		if recipe, err := s.extractWithPattern(doc, pattern); err == nil && recipe != nil {
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

	return nil, errors.New("no extraction method succeeded")
}

// extractFromSchema extracts recipe from JSON-LD schema.org markup
func (s *recipeService) extractFromSchema(doc *goquery.Document) (*Recipe, error) {
	var recipe *Recipe

	doc.Find("script[type='application/ld+json']").Each(func(i int, script *goquery.Selection) {
		if recipe != nil {
			return // Already found one
		}

		jsonText := script.Text()

		// Try to parse as single object
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonText), &data); err == nil {
			if r := parseSchemaRecipe(data); r != nil {
				recipe = r
				return
			}
		}

		// Try to parse as array (sometimes multiple schemas)
		var dataArray []map[string]interface{}
		if err := json.Unmarshal([]byte(jsonText), &dataArray); err == nil {
			for _, item := range dataArray {
				if r := parseSchemaRecipe(item); r != nil {
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
func parseSchemaRecipe(data map[string]interface{}) *Recipe {
	schemaType, ok := data["@type"].(string)
	if !ok || schemaType != "Recipe" {
		return nil
	}

	recipe := &Recipe{}

	if title, ok := data["name"].(string); ok {
		recipe.Title = title
	}

	if desc, ok := data["description"].(string); ok {
		recipe.Description = desc
	}

	// Extract ingredients (can be array of strings or objects)
	if ingredients, ok := data["recipeIngredient"].([]interface{}); ok {
		for _, ing := range ingredients {
			if str, ok := ing.(string); ok {
				recipe.Ingredients = append(recipe.Ingredients, str)
			}
		}
	}

	// Extract instructions (can be string, array, or HowToStep objects)
	if instructions, ok := data["recipeInstructions"].([]interface{}); ok {
		for _, inst := range instructions {
			if str, ok := inst.(string); ok {
				recipe.Instructions = append(recipe.Instructions, str)
			} else if obj, ok := inst.(map[string]interface{}); ok {
				if text, ok := obj["text"].(string); ok {
					recipe.Instructions = append(recipe.Instructions, text)
				}
			}
		}
	} else if inst, ok := data["recipeInstructions"].(string); ok {
		recipe.Instructions = append(recipe.Instructions, inst)
	}

	// Extract times
	if prepTime, ok := data["prepTime"].(string); ok {
		recipe.PrepTime = prepTime
	}
	if cookTime, ok := data["cookTime"].(string); ok {
		recipe.CookTime = cookTime
	}
	if totalTime, ok := data["totalTime"].(string); ok {
		recipe.TotalTime = totalTime
	}

	// Extract servings
	if yield, ok := data["recipeYield"].(string); ok {
		recipe.Servings = yield
	} else if yieldArray, ok := data["recipeYield"].([]interface{}); ok && len(yieldArray) > 0 {
		if yieldStr, ok := yieldArray[0].(string); ok {
			recipe.Servings = yieldStr
		}
	}

	// Extract category/cuisine
	if category, ok := data["recipeCategory"].(string); ok {
		recipe.Category = category
	}
	if cuisine, ok := data["recipeCuisine"].(string); ok {
		recipe.Cuisine = cuisine
	}

	// Extract author
	if author, ok := data["author"].(map[string]interface{}); ok {
		if name, ok := author["name"].(string); ok {
			recipe.Author = name
		}
	} else if authorStr, ok := data["author"].(string); ok {
		recipe.Author = authorStr
	}

	// Extract image
	if image, ok := data["image"].(string); ok {
		recipe.ImageURL = image
	} else if imageObj, ok := data["image"].(map[string]interface{}); ok {
		if url, ok := imageObj["url"].(string); ok {
			recipe.ImageURL = url
		}
	} else if imageArray, ok := data["image"].([]interface{}); ok && len(imageArray) > 0 {
		if imageStr, ok := imageArray[0].(string); ok {
			recipe.ImageURL = imageStr
		}
	}

	// Validate we got at least title and ingredients
	if recipe.Title == "" || len(recipe.Ingredients) == 0 {
		return nil
	}

	return recipe
}

// extractWithPattern uses domain-specific CSS selectors
func (s *recipeService) extractWithPattern(doc *goquery.Document, pattern DomainPattern) (*Recipe, error) {
	recipe := &Recipe{}

	// Extract title
	if pattern.Title != "" {
		recipe.Title = strings.TrimSpace(doc.Find(pattern.Title).First().Text())
	}

	// Extract description
	if pattern.Description != "" {
		recipe.Description = strings.TrimSpace(doc.Find(pattern.Description).First().Text())
	}

	// Extract ingredients
	if pattern.Ingredients != "" {
		doc.Find(pattern.Ingredients).Each(func(i int, s *goquery.Selection) {
			ingredient := strings.TrimSpace(s.Text())
			if ingredient != "" {
				recipe.Ingredients = append(recipe.Ingredients, ingredient)
			}
		})
	}

	// Extract instructions
	if pattern.Instructions != "" {
		doc.Find(pattern.Instructions).Each(func(i int, s *goquery.Selection) {
			instruction := strings.TrimSpace(s.Text())
			if instruction != "" {
				recipe.Instructions = append(recipe.Instructions, instruction)
			}
		})
	}

	// Extract metadata
	if pattern.PrepTime != "" {
		recipe.PrepTime = strings.TrimSpace(doc.Find(pattern.PrepTime).First().Text())
	}
	if pattern.CookTime != "" {
		recipe.CookTime = strings.TrimSpace(doc.Find(pattern.CookTime).First().Text())
	}
	if pattern.TotalTime != "" {
		recipe.TotalTime = strings.TrimSpace(doc.Find(pattern.TotalTime).First().Text())
	}
	if pattern.Servings != "" {
		recipe.Servings = strings.TrimSpace(doc.Find(pattern.Servings).First().Text())
	}
	if pattern.ImageURL != "" {
		if img, exists := doc.Find(pattern.ImageURL).First().Attr("src"); exists {
			recipe.ImageURL = img
		}
	}

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
	doc.Find("ul li, .ingredient, [class*='ingredient'], [class*='Ingredient']").Each(func(i int, s *goquery.Selection) {
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
func (s *recipeService) extractWithLLMCleaned(cleanedData map[string]string) (*Recipe, error) {
	log.Printf("LLM (cleaned) called with data: %v", cleanedData)
	panic("LLM cleaned extraction not implemented - feature flag: llm_cleaned_html")
}

// extractWithLLMFull uses LLM with full HTML
func (s *recipeService) extractWithLLMFull(html string) (*Recipe, error) {
	log.Printf("LLM (full) called with HTML length: %d", len(html))
	panic("LLM full extraction not implemented - feature flag: llm_full_html")
}

// initializeDomainPatterns creates patterns for known recipe sites
func initializeDomainPatterns() map[string]DomainPattern {
	return map[string]DomainPattern{
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
			Ingredients:  "ul.o-Ingredients__m-List li.o-Ingredients__a-Ingredient",
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
