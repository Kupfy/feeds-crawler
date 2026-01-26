package dto

import (
	"time"

	"github.com/google/uuid"
)

type CrawlJobStatus struct {
	JobID      uuid.UUID `json:"jobId"`
	Site       string    `json:"site"`
	StartedAt  time.Time `json:"startedAt"`
	PagesFound int       `json:"pagesFound"`
}
