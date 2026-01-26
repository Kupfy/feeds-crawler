package dto

type StartCrawlRequest struct {
	SeedURL              string            `json:"seedUrl" binding:"required"`
	SiteName             string            `json:"siteName,omitempty"`
	MaxDepth             int               `json:"maxDepth,omitempty"`
	Concurrency          int               `json:"concurrency,omitempty"`
	RateLimit            float64           `json:"rateLimitPerSec,omitempty"`
	ExcludedPathSegments []string          `json:"excludedPathSegments,omitempty"`
	Login                *LoginCredentials `json:"login,omitempty"`
}
