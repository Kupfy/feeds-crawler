package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/Kupfy/feeds-crawler/internal/data/domain"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/request"
	"github.com/Kupfy/feeds-crawler/internal/data/response"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RecipesRepo interface {
	SaveRecipe(recipe entity.Recipe, ctx context.Context) (*entity.Recipe, error)
	GetRecipesBySearchQuery(ctx context.Context, searchRequest request.SearchRequest) (response.PagedResponse[response.RecipeSearchResult], error)
	GetTopRecipes(ctx context.Context, limit int) ([]response.RecipeSearchResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.Recipe, error)
	GetBySlug(ctx context.Context, slug string) (entity.Recipe, error)
}

type recipesRepo struct {
	db *sqlx.DB
}

func NewRecipesRepo(db *sqlx.DB) RecipesRepo {
	return &recipesRepo{db: db}
}

func (r recipesRepo) SaveRecipe(recipe entity.Recipe, ctx context.Context) (*entity.Recipe, error) {
	var inserted entity.Recipe
	query := `
		INSERT INTO recipes (
			slug,
		    title,
			author,
			publication,
			blurb,
			ingredients,
			method,
			serving,
			cooking_time,
			prep_time,
		    note
		) VALUES (
		    :slug, :title, :author, :publication, :blurb, :ingredients, :method, 
		    :serving, :cooking_time, :prep_time, :note
		) RETURNING (
			id,
			slug,
		    title,
			author,
			publication,
			blurb,
			ingredients,
			method,
			serving,
			cooking_time,
			prep_time,
		    note
		);
	`
	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		log.Printf("Failed to prepare statement: %v", err)
		return nil, err
	}

	err = stmt.GetContext(ctx, &inserted, recipe)
	return &inserted, err
}

func (r recipesRepo) GetRecipesBySearchQuery(ctx context.Context, searchRequest request.SearchRequest) (response.PagedResponse[response.RecipeSearchResult], error) {
	var resultItems []response.RecipeSearchResult
	query := `SELECT id, slug, title, author, publication 
			      FROM recipes
			  WHERE search_vector @@ to_prefix_tsquery($1)
			  ORDER BY ts_rank(search_vector, to_prefix_tsquery($1)) DESC LIMIT $2 OFFSET $3;
	`
	err := r.db.SelectContext(ctx, &resultItems, query, searchRequest.Query, searchRequest.Size,
		searchRequest.Page*searchRequest.Size)
	if err != nil {
		log.Printf("Failed to get recipes by search query: %v", err)
		return response.PagedResponse[response.RecipeSearchResult]{}, err
	}
	var count int
	countQuery := `SELECT COUNT(*) FROM recipes WHERE search_vector @@ to_prefix_tsquery($1);`
	err = r.db.GetContext(ctx, &count, countQuery, searchRequest.Query)
	if err != nil {
		log.Printf("Failed to get recipes count by search query: %v", err)
	}

	results := response.PagedResponse[response.RecipeSearchResult]{
		Items: resultItems,
		Page:  searchRequest.Page,
		Total: count,
	}

	return results, err
}

func (r recipesRepo) GetTopRecipes(ctx context.Context, limit int) ([]response.RecipeSearchResult, error) {
	var results []response.RecipeSearchResult
	query := `SELECT id, slug, title, author, publication FROM recipes LIMIT $1;`
	err := r.db.SelectContext(ctx, &results, query, limit)
	if err != nil {
		log.Printf("Failed to get top recipes: %v", err)
	}

	return results, err
}

func (r recipesRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.Recipe, error) {
	var recipe entity.Recipe
	query := `
		SELECT id,
			   slug,
		       title,
			   author,
			   publication,
			   blurb,
			   ingredients,
			   method,
			   serving,
			   cooking_time,
			   prep_time,
		       note
		FROM recipes WHERE id = $1;`
	err := r.db.GetContext(ctx, &recipe, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return recipe, domain.Wrap(domain.ErrNotFound, err)
	}
	return recipe, err
}

func (r recipesRepo) GetBySlug(ctx context.Context, slug string) (entity.Recipe, error) {
	var recipe entity.Recipe
	query := `
		SELECT id,
			   slug,
			   title,
			   author,
			   publication,
			   blurb,
			   ingredients,
			   method,
			   serving,
			   cooking_time,
			   prep_time,
			   note
    	FROM recipes WHERE slug = $1;`
	err := r.db.GetContext(ctx, &recipe, query, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return recipe, domain.Wrap(domain.ErrNotFound, err)
	}
	return recipe, err

}
