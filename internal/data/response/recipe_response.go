package response

import (
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/google/uuid"
)

type IngredientsResponse struct {
	Item        string  `json:"item"`
	Component   *string `json:"component"`
	Instruction *string `json:"instruction"`
}

type RecipeResponse struct {
	ID          uuid.UUID             `json:"id"`
	Slug        string                `json:"slug"`
	Title       string                `json:"title"`
	Author      string                `json:"author"`
	Publication string                `json:"publication"`
	Blurb       string                `json:"blurb"`
	Ingredients []IngredientsResponse `json:"ingredients"`
	Method      []string              `json:"method"`
	Serving     string                `json:"serving"`
	CookingTime *int                  `json:"cookingTime"`
	PrepTime    *int                  `json:"prepTime"`
	Note        *string               `json:"note"`
}

func NewRecipeResponse(recipe entity.Recipe) RecipeResponse {
	var ingredients []IngredientsResponse
	var method []string

	for _, ingredient := range recipe.Ingredients {
		ingredients = append(ingredients, IngredientsResponse{
			Item:        ingredient.String(),
			Component:   ingredient.Component,
			Instruction: ingredient.Instruction,
		})
	}

	for _, step := range recipe.Method {
		method = append(method, step.String())
	}

	return RecipeResponse{
		ID:          recipe.ID,
		Slug:        recipe.Slug,
		Title:       recipe.Title,
		Author:      recipe.Author,
		Publication: recipe.Publication,
		Blurb:       recipe.Blurb,
		Ingredients: ingredients,
		Method:      method,
		Serving:     recipe.Serving.String(),
		CookingTime: recipe.CookingTime,
		PrepTime:    recipe.PrepTime,
		Note:        recipe.Note,
	}
}
