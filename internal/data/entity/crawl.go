package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/Kupfy/feeds-crawler/internal/data/enum/crawlstatus"
)

type Crawl struct {
	ID         uuid.UUID               `json:"id" db:"id"`
	SiteDomain string                  `json:"siteDomain" db:"site_domain"`
	SeedURL    string                  `json:"seedURL" db:"seed_url"`
	Status     crawlstatus.CrawlStatus `json:"status" db:"status"`
	StartedAt  time.Time               `json:"startedAt" db:"started_at"`
	EndedAt    *time.Time              `json:"endedAt" db:"ended_at"`
}
