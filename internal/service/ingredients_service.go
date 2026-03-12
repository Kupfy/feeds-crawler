package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/clients"
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
	"github.com/Kupfy/feeds-crawler/internal/repository"
	"github.com/Kupfy/feeds-crawler/internal/util"
	pb "github.com/Kupfy/feeds-crawler/pkg/grpc/ingredientparser"
)

var ingredientsDict map[string]*entity.Ingredient

var unicodeFractions = map[rune]float64{
	'¼': 0.25,
	'½': 0.5,
	'¾': 0.75,
	'⅐': 1.0 / 7.0,
	'⅑': 1.0 / 9.0,
	'⅒': 0.1,
	'⅓': 1.0 / 3.0,
	'⅔': 2.0 / 3.0,
	'⅕': 0.2,
	'⅖': 0.4,
	'⅗': 0.6,
	'⅘': 0.8,
	'⅙': 1.0 / 6.0,
	'⅚': 5.0 / 6.0,
	'⅛': 0.125,
	'⅜': 0.375,
	'⅝': 0.625,
	'⅞': 0.875,
}

var (
	// Matches "1 1/2" (mixed number)
	mixedNumberPattern = regexp.MustCompile(`^(\d+)\s+(\d+)/(\d+)$`)

	// Matches "3/4" (simple fraction)
	simpleFractionPattern = regexp.MustCompile(`^(\d+)/(\d+)$`)

	// Matches "1.5" or "1" (decimal or whole number)
	decimalPattern = regexp.MustCompile(`^\d+\.?\d*$`)
)

type IngredientsService interface {
	LoadIngredients(ctx context.Context) error
	ParseIngredientLine(ctx context.Context, rawLine string) (dto.IngredientsItem, error)
}

type ingredientsService struct {
	ingredientsRepo        repository.IngredientsRepo
	ingredientParserClient *clients.IngredientParserClient
}

func NewIngredientsService(
	ingredientsRepo repository.IngredientsRepo, ingredientParserClient *clients.IngredientParserClient,
) IngredientsService {
	return &ingredientsService{
		ingredientsRepo:        ingredientsRepo,
		ingredientParserClient: ingredientParserClient,
	}
}

func (s *ingredientsService) LoadIngredients(ctx context.Context) error {
	ingredientsDict = make(map[string]*entity.Ingredient)
	ingredients, err := s.ingredientsRepo.GetAllIngredients(ctx)
	if err != nil {
		return err
	}

	for _, ingredient := range ingredients {
		ingredientsDict[ingredient.CanonicalName] = &ingredient
		ingredientsDict[ingredient.DisplayName] = &ingredient
		ingredientsDict[ingredient.DisplayNamePlural] = &ingredient
		for _, alias := range ingredient.Aliases {
			ingredientsDict[alias] = &ingredient
		}
	}
	log.Printf("Loaded %d ingredients", len(ingredientsDict))
	return nil
}

func (s *ingredientsService) ParseIngredientLine(ctx context.Context, rawLine string) (dto.IngredientsItem, error) {
	ingRPC, err := s.ingredientParserClient.ParseIngredient(ctx, rawLine)
	if err != nil {
		return dto.IngredientsItem{}, err
	}
	if ingRPC == nil {
		return dto.IngredientsItem{}, errors.New("ingredient parser returned nil")
	}
	var name string
	if len(ingRPC.Name) == 0 {
		log.Printf("Ingredient parser returned empty name for line: %s", rawLine)
	} else if len(ingRPC.Name) > 1 {
		log.Printf("Ingredient parser returned multiple names for line: %s", rawLine)
		comma, last := ingRPC.Name[:len(ingRPC.Name)-1], ingRPC.Name[len(ingRPC.Name)-1]
		var names []string
		for _, n := range comma {
			names = append(names, n.Text)
		}
		name = strings.Join(names, ", ") + " and " + last.Text
	} else {
		name = ingRPC.Name[0].Text
	}

	var ingredient dto.IngredientsItem

	if len(ingRPC.Amount) > 0 {
		if len(ingRPC.Amount) > 1 {
			log.Printf("Ingredient parser returned multiple quantities for line: %s", rawLine)
			sort.Slice(ingRPC.Amount, func(i, j int) bool {
				return ingRPC.Amount[i].Confidence > ingRPC.Amount[j].Confidence
			})
			ingredient = pickBestUnit(name, ingRPC.Amount)
		} else {
			ingredient, err = ingredientAmountRPCtoDTO(name, ingRPC.Amount[0].Quantity, ingRPC.Amount[0].QuantityMax, unit.NewUnit(ingRPC.Amount[0].Unit))
			if err != nil {
				return dto.IngredientsItem{}, err
			}
		}
	} else {
		ingredient, err = ingredientAmountRPCtoDTO(name, "", "", unit.NewUnit(""))
	}

	var instruction *string
	if ingRPC.Preparation != nil {
		instruction = &ingRPC.Preparation.Text
	}
	ingredient.Instruction = instruction

	return ingredient, nil
}

func pickBestUnit(name string, ingredientsRPC []*pb.Amount) dto.IngredientsItem {
	var bestUnit dto.IngredientsItem
	altQuantMap := make(map[string]string)
	var err error
	for _, ingredient := range ingredientsRPC {
		if u := unit.NewUnit(ingredient.Unit); u.Rank() > bestUnit.Unit.Rank() {
			bestUnit, err = ingredientAmountRPCtoDTO(name, ingredient.Quantity, ingredient.QuantityMax, u)
			if err != nil {
				log.Printf("Failed to parse ingredient %s: %v", name, err)
			}
			if _, ok := altQuantMap[ingredient.Text]; ok {
				delete(altQuantMap, ingredient.Text)
			}
		} else {
			altQuantMap[ingredient.Text] = ingredient.Text
		}
	}

	var altQuantities []string
	for k, _ := range altQuantMap {
		altQuantities = append(altQuantities, k)
	}

	bestUnit.AltQuantity = util.ToPtr(strings.Join(altQuantities, ", "))
	return bestUnit
}

func ingredientAmountRPCtoDTO(name string, quantityStr string, quantityMaxStr string, u unit.Unit) (dto.IngredientsItem, error) {
	if quantityStr == "" {
		return dto.IngredientsItem{Name: name, Unit: u}, nil
	}

	quantity, err := parseQuantity(quantityStr)
	if err != nil {
		log.Printf("Failed to parse quantity for ingredient %s: %v", name, err)
		return dto.IngredientsItem{}, err
	}

	var quantityMax *float64
	if quantityMaxStr != "" {
		quantityMaxVal, err := parseQuantity(quantityMaxStr)
		if err != nil {
			log.Printf("Failed to parse quantity max for ingredient %s: %v", name, err)
		} else {
			quantityMax = &quantityMaxVal
		}
	}

	return dto.IngredientsItem{
		Name:        name,
		Quantity:    util.ToPtr(quantity),
		QuantityMax: quantityMax,
		Unit:        u,
	}, nil
}

func parseQuantity(s string) (float64, error) {
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Check for Unicode fraction characters first
	if val, ok := parseUnicodeFraction(s); ok {
		return val, nil
	}

	// Check for mixed numbers: "1 1/2"
	if matches := mixedNumberPattern.FindStringSubmatch(s); matches != nil {
		whole, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid whole number: %w", err)
		}

		numerator, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numerator: %w", err)
		}

		denominator, err := strconv.ParseFloat(matches[3], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid denominator: %w", err)
		}

		if denominator == 0 {
			return 0, fmt.Errorf("division by zero")
		}

		return whole + (numerator / denominator), nil
	}

	// Check for simple fractions: "3/4"
	if matches := simpleFractionPattern.FindStringSubmatch(s); matches != nil {
		numerator, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numerator: %w", err)
		}

		denominator, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid denominator: %w", err)
		}

		if denominator == 0 {
			return 0, fmt.Errorf("division by zero")
		}

		return numerator / denominator, nil
	}

	// Check for decimal or whole number: "1.5" or "1"
	if decimalPattern.MatchString(s) {
		return strconv.ParseFloat(s, 64)
	}

	// Try to handle mixed Unicode fractions like "1½" or "2¾"
	if val, ok := parseMixedUnicodeFraction(s); ok {
		return val, nil
	}

	return 0, fmt.Errorf("unable to parse quantity: %s", s)
}

// parseUnicodeFraction checks if the entire string is a single Unicode fraction
func parseUnicodeFraction(s string) (float64, bool) {
	if len(s) == 0 {
		return 0, false
	}

	// Check if it's a single Unicode fraction character
	runes := []rune(s)
	if len(runes) == 1 {
		if val, ok := unicodeFractions[runes[0]]; ok {
			return val, true
		}
	}

	return 0, false
}

// parseMixedUnicodeFraction handles "1½", "2¾", etc.
func parseMixedUnicodeFraction(s string) (float64, bool) {
	runes := []rune(s)
	if len(runes) < 2 {
		return 0, false
	}

	// Find where the Unicode fraction starts
	var wholeStr string
	var fractionRune rune
	foundFraction := false

	for i, r := range runes {
		if _, isFraction := unicodeFractions[r]; isFraction {
			wholeStr = string(runes[:i])
			fractionRune = r
			foundFraction = true
			break
		}
	}

	if !foundFraction {
		return 0, false
	}

	// Parse the whole number part
	wholeStr = strings.TrimSpace(wholeStr)
	if wholeStr == "" {
		// Just a fraction like "½"
		return unicodeFractions[fractionRune], true
	}

	whole, err := strconv.ParseFloat(wholeStr, 64)
	if err != nil {
		return 0, false
	}

	fraction := unicodeFractions[fractionRune]
	return whole + fraction, true
}
