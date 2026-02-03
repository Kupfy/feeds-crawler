package entity

import (
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/google/uuid"
)

type Ingredient struct {
	ID                uuid.UUID      `db:"id"`
	CanonicalName     string         `db:"canonical_name"`
	DisplayName       string         `db:"display_name"`
	DisplayNamePlural string         `db:"display_name_plural"`
	Aliases           dto.DbStrArray `db:"aliases"`
}
