package entity

import (
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/google/uuid"
)

type Recipe struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	Title       string           `db:"title" json:"title"`
	Author      string           `db:"author" json:"author"`
	Publication string           `db:"publication" json:"publication"`
	Blurb       string           `db:"blurb" json:"blurb"`
	Ingredients dto.Ingredients  `db:"ingredients" json:"ingredients"`
	Method      []dto.MethodItem `db:"method" json:"method"`
	Serving     int              `db:"serving" json:"serving" default:"4"`
	CookingTime int              `db:"cooking_time" json:"cookingTime"`
	PrepTime    *int             `db:"prep_time" json:"prepTime"`
}
