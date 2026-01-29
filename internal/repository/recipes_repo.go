package repository

import (
	"context"
	"log"

	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RecipesRepo interface {
	SaveRecipe(ctx context.Context, recipe entity.Recipe) error
	GetRecipesBySearchQuery(ctx context.Context, search string) ([]dto.RecipeSearchResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.Recipe, error)
}

type recipesRepo struct {
	db *sqlx.DB
}

func NewRecipesRepo(db *sqlx.DB) RecipesRepo {
	return &recipesRepo{db: db}
}

func (r recipesRepo) SaveRecipe(ctx context.Context, recipe entity.Recipe) error {
	query := `
		INSERT INTO recipes (
		    id,
			title,
			author,
			publication,
			blurb,
			ingredients,
			method,
			serving,
			cooking_time,
			prep_time
		) VALUES (
		    :id, :title, :author, :publication, :blurb, :ingredients, :method, 
		    :serving, :cooking_time, :prep_time
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, recipe)
	if err != nil {
		return err
	}

	return nil
}

func (r recipesRepo) GetRecipesBySearchQuery(ctx context.Context, search string) ([]dto.RecipeSearchResult, error) {
	var results []dto.RecipeSearchResult
	query := `SELECT id, title, author, publication FROM recipes WHERE title LIKE $1 LIMIT 10;`
	err := r.db.SelectContext(ctx, &results, query, "%"+search+"%")
	if err != nil {
		log.Printf("Failed to get recipes by search query: %v", err)
	}
	return results, err
}

func (r recipesRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.Recipe, error) {
	var recipe entity.Recipe
	query := "SELECT * FROM recipes WHERE id = $1;"
	err := r.db.GetContext(ctx, &recipe, query, id)
	return recipe, err
}
