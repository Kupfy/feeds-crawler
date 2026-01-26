package entity

import "github.com/google/uuid"

type RecipeTag struct {
	RecipeID uuid.UUID `db:"recipe_id"`
	Tag      string    `db:"tag"`
}
