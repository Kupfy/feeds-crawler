package service

import (
	"context"
	"errors"
	"log"
	"sort"
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

	quantity, err := util.ParseQuantity(quantityStr)
	if err != nil {
		log.Printf("Failed to parse quantity for ingredient %s: %v", name, err)
		return dto.IngredientsItem{}, err
	}

	var quantityMax *float64
	if quantityMaxStr != "" {
		quantityMaxVal, err := util.ParseQuantity(quantityMaxStr)
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
