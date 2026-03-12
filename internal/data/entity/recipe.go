package entity

import (
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/google/uuid"
)

type Recipe struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	Slug        string          `db:"slug" json:"slug"`
	Title       string          `db:"title" json:"title"`
	Author      string          `db:"author" json:"author"`
	Publication string          `db:"publication" json:"publication"`
	Blurb       string          `db:"blurb" json:"blurb"`
	Ingredients dto.Ingredients `db:"ingredients" json:"ingredients"`
	Method      dto.Method      `db:"method" json:"method"`
	Serving     dto.Serving     `db:"serving" json:"serving"`
	CookingTime *int            `db:"cooking_time" json:"cookingTime"`
	PrepTime    *int            `db:"prep_time" json:"prepTime"`
	Note        *string         `db:"note" json:"note"`
}
