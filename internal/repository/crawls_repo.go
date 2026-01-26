package repository

import (
	"context"

	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/crawlstatus"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CrawlsRepo interface {
	CreateCrawl(ctx context.Context, crawl entity.Crawl) error
	GetCrawlByID(ctx context.Context, id uuid.UUID) (entity.Crawl, error)
	GetLatestCrawlBySite(ctx context.Context, siteDomain string) (entity.Crawl, error)
	UpdateCrawlStatus(ctx context.Context, crawlID uuid.UUID, status crawlstatus.CrawlStatus) error
}

func NewCrawlsRepo(db *sqlx.DB) CrawlsRepo {
	return &crawlsRepo{db: db}
}

type crawlsRepo struct {
	db *sqlx.DB
}

func (r crawlsRepo) UpdateCrawlStatus(ctx context.Context, crawlID uuid.UUID, status crawlstatus.CrawlStatus) error {
	query := `UPDATE crawls SET status = $1, ended_at = now() WHERE id = $2;`
	_, err := r.db.ExecContext(ctx, query, status, crawlID)
	return err
}

func (r crawlsRepo) CreateCrawl(ctx context.Context, crawl entity.Crawl) error {
	query := `INSERT INTO crawls (id, site_domain, seed_url, status, started_at, ended_at) VALUES (
				:id, :site_domain, :seed_url, :status, :started_at, :ended_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, crawl)
	return err
}

func (r crawlsRepo) GetCrawlByID(ctx context.Context, id uuid.UUID) (entity.Crawl, error) {
	var crawl entity.Crawl
	query := "SELECT * FROM crawls WHERE id = $1;"

	err := r.db.GetContext(ctx, crawl, query, id)
	return crawl, err
}

func (r crawlsRepo) GetLatestCrawlBySite(ctx context.Context, siteDomain string) (entity.Crawl, error) {
	var crawl entity.Crawl
	query := "SELECT * FROM crawls WHERE site_domain = $1 ORDER BY started_at DESC LIMIT 1;"

	err := r.db.GetContext(ctx, crawl, query, siteDomain)
	return crawl, err
}
