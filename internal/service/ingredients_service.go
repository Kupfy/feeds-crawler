package service

import (
	"context"

	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/repository"
)

var ingredientsDict map[string]entity.Ingredient

type IngredientsService interface {
	LoadIngredients(ctx context.Context) error
}

type ingredientsService struct {
	ingredientsRepo repository.IngredientsRepo
}

func NewIngredientsService(ingredientsRepo repository.IngredientsRepo) IngredientsService {
	return &ingredientsService{ingredientsRepo}
}

func (s *ingredientsService) LoadIngredients(ctx context.Context) error {
	ingredients, err := s.ingredientsRepo.GetAllIngredients(ctx)
	if err != nil {
		return err
	}

	for _, ingredient := range ingredients {
		ingredientsDict[ingredient.Name] = ingredient
		for _, alias := range ingredient.Aliases {
			ingredientsDict[alias] = ingredient
		}
	}
	return nil
}
