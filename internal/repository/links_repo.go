package repository

import (
	"context"
	"log"

	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/jmoiron/sqlx"
)

type LinksRepo interface {
	SaveLink(ctx context.Context, link entity.Link) error
}

type linksRepo struct {
	db *sqlx.DB
}

func NewLinksRepo(db *sqlx.DB) LinksRepo {
	return &linksRepo{db: db}
}

func (l linksRepo) SaveLink(ctx context.Context, link entity.Link) error {
	query := `INSERT INTO links (from_url, to_url, crawl_id) 
			      VALUES 
			  (:from_url, :to_url, :crawl_id)
			      ON CONFLICT (from_url, to_url) DO NOTHING;`

	_, err := l.db.NamedExecContext(ctx, query, link)
	if err != nil {
		log.Printf("Failed to save link: %v", err)
	}
	return err
}
