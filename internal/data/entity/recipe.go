package entity

import (
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/google/uuid"
)

type Recipe struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	Title       string           `json:"title" db:"title"`
	Author      string           `json:"author" db:"author"`
	Blurb       string           `json:"blurb" db:"blurb"`
	Ingredients []dto.Ingredient `json:"ingredients" db:"ingredients"`
	Method      []dto.MethodItem `json:"method" db:"method"`
	Serving     int              `json:"serving" db:"serving"`
	CookingTime int              `json:"cookingTime" db:"cooking_time"`
	PrepTime    *int             `json:"prepTime" db:"prep_time"`
}
