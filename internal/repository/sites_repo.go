package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/jmoiron/sqlx"
)

type SiteRepo interface {
	SaveSite(ctx context.Context, site entity.Site) error
}

func NewSiteRepo(db *sqlx.DB) SiteRepo {
	return &siteRepo{db: db}
}

type siteRepo struct {
	db *sqlx.DB
}

func (r siteRepo) SaveSite(ctx context.Context, site entity.Site) error {
	query := `INSERT INTO sites (domain, name, inserted_at, last_crawled_at, last_crawl_status) VALUES (
					:domain, :name, :inserted_at, :last_crawled_at, :last_crawl_status
			) ON CONFLICT (domain) DO UPDATE SET last_crawled_at = now(), last_crawl_status = :last_crawl_status;`

	_, err := r.db.NamedExecContext(ctx, query, site)
	return err
}

func (r siteRepo) GetSiteByDomain(ctx context.Context, domain string) (*entity.Site, error) {
	query := "SELECT * FROM sites WHERE domain = $1;"
	var site entity.Site
	err := r.db.GetContext(ctx, &site, query, domain)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &site, err
}
