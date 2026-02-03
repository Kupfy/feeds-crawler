package repository

import (
	"context"
	"log"

	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/jmoiron/sqlx"
)

type IngredientsRepo interface {
	SaveIngredient(ctx context.Context, ingredient entity.Ingredient) (entity.Ingredient, error)
	GetAllIngredients(ctx context.Context) ([]entity.Ingredient, error)
}

type ingredientsRepo struct {
	db *sqlx.DB
}

func NewIngredientsRepo(db *sqlx.DB) IngredientsRepo {
	return &ingredientsRepo{db: db}
}

func (r ingredientsRepo) SaveIngredient(ctx context.Context, ingredient entity.Ingredient) (entity.Ingredient, error) {
	var insertedIngredient entity.Ingredient
	query := `INSERT INTO ingredients (
                  canonical_name, display_name, display_name_plural, aliases
			  ) VALUES (
			      :name, :display_name, :display_name_plural, :aliases
			  ) RETURNING *;`
	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		log.Printf("Failed to prepare statement: %v", err)
		return ingredient, err
	}
	err = stmt.GetContext(ctx, &insertedIngredient, ingredient)
	return insertedIngredient, err
}

func (r ingredientsRepo) GetAllIngredients(ctx context.Context) ([]entity.Ingredient, error) {
	var ingredients []entity.Ingredient
	query := "SELECT * FROM ingredients;"
	err := r.db.SelectContext(ctx, &ingredients, query)
	return ingredients, err
}
