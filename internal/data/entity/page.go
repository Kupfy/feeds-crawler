package entity

import (
	"time"

	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/crawlstatus"
)

type Page struct {
	Path       string                  `db:"path" json:"path"`
	SiteDomain string                  `db:"site_domain" json:"site_domain"`
	URL        string                  `db:"url" json:"url"`
	ParentURL  string                  `db:"parent_url" json:"parentURL"`
	Depth      int                     `db:"depth" json:"depth"`
	HTML       string                  `db:"html" json:"html,omitempty"`
	Text       string                  `db:"text" json:"text,omitempty"`
	Images     dto.DbStrArray          `db:"images" json:"images,omitempty"`
	Status     crawlstatus.CrawlStatus `db:"status" json:"status"`
	Meta       dto.JsonB               `db:"meta" json:"meta,omitempty"`
	InsertedAt time.Time               `db:"inserted_at" json:"inserted_at"`
}
