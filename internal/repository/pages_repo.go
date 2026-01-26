package repository

import (
	"context"
	"log"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/crawlstatus"
	"github.com/jmoiron/sqlx"
)

type PagesRepo interface {
	SavePage(ctx context.Context, page *entity.Page) error
	IsVisited(ctx context.Context, url string) (bool, error)
	MarkPageStatus(ctx context.Context, url string, status crawlstatus.CrawlStatus) error
}

type pagesRepo struct {
	db *sqlx.DB
}

func NewPagesRepo(db *sqlx.DB) PagesRepo {
	return &pagesRepo{db: db}
}

func (p pagesRepo) SavePage(ctx context.Context, page *entity.Page) error {
	query := `INSERT INTO pages (
                   path, site_domain, url, parent_url, depth, html, text, images, status, meta, fetched_at
			  ) VALUES (
				   $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
			  ) ON CONFLICT (url) DO NOTHING;`

	imagesParsed := "{\"" + strings.Join(page.Images, "\", \"") + "\"}"

	result, err := p.db.ExecContext(ctx, query, page.Path, page.SiteDomain, page.URL, page.ParentURL, page.Depth, page.HTML,
		page.Text, imagesParsed, page.Status, page.Meta, page.InsertedAt)
	if err != nil {
		log.Printf("Error inserting page: %v", err)
		return err
	}
	if result == nil {
		log.Printf("No result for insertion for: %v", page.URL)
		return nil
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		log.Printf("No pages inserted for: %v", page.URL)
	}
	return nil
}

func (p pagesRepo) IsVisited(ctx context.Context, url string) (bool, error) {
	query := "SELECT COUNT(*) FROM pages WHERE url = $1;"
	var results int

	err := p.db.GetContext(ctx, &results, query, url)
	if err != nil {
		return false, err
	}
	if results > 0 {
		log.Printf("Page already visited: %v", url)
		return true, nil
	}

	return false, nil
}

func (p pagesRepo) MarkPageStatus(ctx context.Context, url string, status crawlstatus.CrawlStatus) error {
	return nil
}
