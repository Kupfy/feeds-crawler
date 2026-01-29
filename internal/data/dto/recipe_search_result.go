package dto

import "github.com/google/uuid"

type RecipeSearchResult struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Author      string    `json:"author" db:"author"`
	Publication string    `json:"publication" db:"publication"`
}
