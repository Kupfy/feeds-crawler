package service

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/Kupfy/feeds-crawler/internal/clients"
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
	"github.com/Kupfy/feeds-crawler/internal/repository"
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
		name = ingRPC.Name[0].Text
	} else {
		name = ingRPC.Name[0].Text
	}

	var quantity float64
	var u unit.Unit

	if len(ingRPC.Amount) > 0 {
		if len(ingRPC.Amount) > 1 {
			log.Printf("Ingredient parser returned multiple quantities for line: %s", rawLine)
		}
		quantity, err = strconv.ParseFloat(ingRPC.Amount[0].Text, 64)
		if err != nil {
			return dto.IngredientsItem{}, err
		}

		u = unit.NewUnit(ingRPC.Amount[0].Unit)
	}

	var instruction *string
	if ingRPC.Preparation != nil {
		instruction = &ingRPC.Preparation.Text
	}

	return dto.IngredientsItem{
		Name:        name,
		Quantity:    quantity,
		Unit:        u,
		Instruction: instruction,
	}, nil
}
