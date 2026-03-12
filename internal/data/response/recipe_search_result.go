package response

import "github.com/google/uuid"

type RecipeSearchResult struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Slug        string    `json:"slug" db:"slug"`
	Title       string    `json:"title" db:"title"`
	Author      string    `json:"author" db:"author"`
	Publication string    `json:"publication" db:"publication"`
}
