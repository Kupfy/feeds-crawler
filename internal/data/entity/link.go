package entity

import "github.com/google/uuid"

type Link struct {
	FromURL string    `db:"from_url" json:"fromURL"`
	ToURL   string    `db:"to_url" json:"toURL"`
	CrawlID uuid.UUID `db:"crawl_id" json:"crawlID"`
}
