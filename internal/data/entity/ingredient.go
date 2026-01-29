package entity

import (
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/google/uuid"
)

type Ingredient struct {
	ID      uuid.UUID      `db:"id"`
	Name    string         `db:"name"`
	Aliases dto.DbStrArray `db:"aliases"`
}
