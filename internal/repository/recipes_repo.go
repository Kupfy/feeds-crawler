package repository

import (
	"context"

	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/jmoiron/sqlx"
)

type RecipeRepo interface {
}

type recipeRepo struct {
	db *sqlx.DB
}

func NewRecipeRepo(db *sqlx.DB) RecipeRepo {
	return &recipeRepo{db: db}
}

func (u recipeRepo) CreateRecipe(ctx context.Context, recipe entity.Recipe) error {
	query := `
		INSERT INTO recipes () VALUES ()
	`
	_, err := u.db.NamedExecContext(ctx, query, recipe)
	if err != nil {
		return err
	}

	return nil
}
