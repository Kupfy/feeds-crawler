package entity

import (
	"time"

	"github.com/Kupfy/feeds-crawler/internal/data/enum/crawlstatus"
)

type Site struct {
	Domain          string                  `db:"domain"`
	Name            string                  `db:"name"`
	InsertedAt      time.Time               `db:"inserted_at"`
	LastCrawledAt   time.Time               `db:"last_crawled_at"`
	LastCrawlStatus crawlstatus.CrawlStatus `db:"last_crawl_status"`
}
