package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Kupfy/feeds-crawler/internal/data/config"
	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/crawlstatus"
	"github.com/Kupfy/feeds-crawler/internal/messaging"
	"github.com/Kupfy/feeds-crawler/internal/repository"
	"github.com/Kupfy/feeds-crawler/internal/util"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"github.com/google/uuid"
)

type crawlJob struct {
	entity.Crawl
	SeedPath string
	Sitemap  map[string][]string
	Visited  *sync.Map
	Mu       sync.Mutex
}

type CrawlerService interface {
	StartCrawl(ctx context.Context, req dto.StartCrawlRequest) (uuid.UUID, error)
	GetJobStatus(ctx context.Context, jobID uuid.UUID) (entity.Crawl, error)
}

type crawlerService struct {
	cfg       config.ServiceConfig
	crawlRepo repository.CrawlsRepo
	siteRepo  repository.SiteRepo
	pageRepo  repository.PagesRepo
	linksRepo repository.LinksRepo
	queue     messaging.RedisQueue
	//requestTimes *sync.Map
}

func NewCrawlerService(
	cfg config.ServiceConfig,
	crawlRepo repository.CrawlsRepo,
	siteRepo repository.SiteRepo,
	pageRepo repository.PagesRepo,
	linksRepo repository.LinksRepo,
	queue messaging.RedisQueue,
) CrawlerService {
	//requestTimes := sync.Map{}
	return &crawlerService{
		cfg:       cfg,
		crawlRepo: crawlRepo,
		siteRepo:  siteRepo,
		pageRepo:  pageRepo,
		linksRepo: linksRepo,
		queue:     queue,
		//requestTimes: &requestTimes,
	}
}

func (s *crawlerService) StartCrawl(ctx context.Context, req dto.StartCrawlRequest) (uuid.UUID, error) {
	if req.SeedURL == "" {
		return uuid.Nil, errors.New("seed_url required")
	}

	parsed, err := url.Parse(req.SeedURL)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid seed URL: %w", err)
	}

	jobID := uuid.New()
	siteName := req.SiteName
	if siteName == "" {
		siteName = parsed.Hostname()
	}

	site := entity.Site{
		Domain:          parsed.Hostname(),
		Name:            siteName,
		InsertedAt:      time.Now(),
		LastCrawledAt:   time.Now(),
		LastCrawlStatus: crawlstatus.Created,
	}

	err = s.siteRepo.SaveSite(ctx, site)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to save site: %w", err)
	}

	cj := entity.Crawl{
		ID:         jobID,
		SiteDomain: site.Domain,
		SeedURL:    req.SeedURL,
		Status:     crawlstatus.Created,
		StartedAt:  time.Now(),
	}
	err = s.crawlRepo.CreateCrawl(ctx, cj)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create crawl job: %w", err)
	}
	visited := sync.Map{}
	job := crawlJob{Crawl: cj, Sitemap: map[string][]string{}, Visited: &visited, Mu: sync.Mutex{}}

	go func() {
		// Use context.Background() to ensure the job completes
		// even if the HTTP request that started it gets cancelled
		jobCtx := context.Background()

		if err := s.runJob(jobCtx, &job, req); err != nil {
			log.Printf("[job %s] finished with error: %v\n", jobID, err)
		} else {
			log.Printf("[job %s] finished successfully\n", jobID)
		}
	}()

	return jobID, nil
}

func (s *crawlerService) GetJobStatus(ctx context.Context, jobID uuid.UUID) (entity.Crawl, error) {
	cj, err := s.crawlRepo.GetCrawlByID(ctx, jobID)
	if err != nil {
		return entity.Crawl{}, fmt.Errorf("failed to get crawl job: %w", err)
	}

	return cj, nil
}

func (s *crawlerService) runJob(ctx context.Context, job *crawlJob, req dto.StartCrawlRequest) error {
	maxDepth := req.MaxDepth
	if maxDepth == 0 {
		maxDepth = s.cfg.DefaultMaxDepth
	}
	concurrency := req.Concurrency
	if concurrency == 0 {
		concurrency = s.cfg.DefaultConcurrency
	}

	log.Printf("[job %s] starting crawl from %s (maxDepth=%d, concurrency=%d)\n",
		job.ID, job.SeedURL, maxDepth, concurrency)

	// Create ONE cookie jar that will be shared
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}

	parsedSeed, _ := url.Parse(job.SeedURL)
	host := parsedSeed.Hostname()
	job.SeedPath = parsedSeed.Path

	// Perform login FIRST with the shared jar
	if err := s.performLogin(ctx, jar, host, job, req.Login); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// NOW create the main collector with the jar that has cookies
	c := colly.NewCollector(
		colly.MaxDepth(maxDepth),
		colly.Async(true),
	)
	c.SetCookieJar(jar) // This jar now has the login cookies!
	c.AllowedDomains = []string{host}

	// Configure rate limiting
	if err := s.configureRateLimiting(c, req.RateLimit, concurrency); err != nil {
		log.Printf("[job %s] warning: rate limit config error: %v", job.ID, err)
	}

	// Setup callbacks
	s.setupCollectorCallbacks(ctx, c, job, req.ExcludedPathSegments)

	// Create queue
	q, err := queue.New(concurrency, &util.InMemoryQueueBackend{})
	if err != nil {
		return fmt.Errorf("failed to create queue: %w", err)
	}

	// Start crawl from seed
	if err := q.AddURL(job.SeedURL); err != nil {
		return fmt.Errorf("failed to add seed URL: %w", err)
	}

	// Run queue - this blocks until all URLs are processed
	if err := q.Run(c); err != nil {
		log.Printf("[job %s] queue run error: %v\n", job.ID, err)
	}

	// Wait for all async requests to complete
	c.Wait()

	log.Printf("[job %s] crawl complete\n", job.ID)

	return s.saveSitemap(ctx, job)
}

func (s *crawlerService) performLogin(ctx context.Context,
	jar *cookiejar.Jar, host string, job *crawlJob, login *dto.LoginCredentials,
) error {
	if login == nil || login.LoginURL == "" {
		return nil
	}

	if (login.Username == "" && login.Email == "") || login.Password == "" {
		return errors.New("username and password required for login")
	}

	log.Printf("[job %s] attempting login to %s\n", job.ID, login.LoginURL)

	// Use Colly to extract tokens from login page
	loginCollector := colly.NewCollector()
	loginCollector.SetCookieJar(jar)
	loginCollector.AllowedDomains = []string{host}

	extractedTokens := make(map[string]string)

	tokenSelectors := []string{
		"input[name='csrf_token']",
		"input[name='_csrf']",
		"input[name='_wpnonce']",
		"input[name='authenticity_token']",
		"input[name='_token']",
		"meta[name='csrf-token']",
	}

	if login.TokenSelector != "" {
		tokenSelectors = []string{login.TokenSelector}
	}

	for _, selector := range tokenSelectors {
		loginCollector.OnHTML(selector, func(e *colly.HTMLElement) {
			fieldName := e.Attr("name")
			var fieldValue string

			if e.Name == "input" {
				fieldValue = e.Attr("value")
			} else if e.Name == "meta" {
				fieldValue = e.Attr("content")
				if fieldName == "" {
					fieldName = "csrf_token"
				}
			}

			if fieldValue != "" && fieldName != "" {
				extractedTokens[fieldName] = fieldValue
			}
		})
	}

	loginCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("[job %s] error visiting login page: %v (status=%d)\n", job.ID, err, r.StatusCode)
	})

	if err := loginCollector.Visit(login.LoginURL); err != nil {
		log.Printf("failed to visit login page: %w\n", err)
	}
	loginCollector.Wait()

	// Prepare form data
	formData := map[string]string{
		"password": login.Password,
	}

	if login.Username != "" {
		formData["username"] = login.Username
	}

	if login.Email != "" {
		formData["email"] = login.Email
	}

	for fieldName, fieldValue := range extractedTokens {
		formData[fieldName] = fieldValue
	}

	if login.AdditionalFields != nil {
		for k, v := range login.AdditionalFields {
			formData[k] = v
		}
	}

	// POST login
	if err := s.postLogin(ctx, login.LoginURL, formData, jar); err != nil {
		return fmt.Errorf("login POST failed: %w", err)
	}

	// Verify cookies
	parsedLoginURL, _ := url.Parse(login.LoginURL)
	cookies := jar.Cookies(parsedLoginURL)

	if len(cookies) == 0 {
		return errors.New("login did not set any session cookies")
	}

	log.Printf("[job %s] login successful (%d cookies)\n", job.ID, len(cookies))
	return nil
}

func (s *crawlerService) configureRateLimiting(c *colly.Collector, rate float64, concurrency int) error {
	rule := &colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: concurrency,
	}

	if rate > 0 {
		rule.Delay = time.Duration(float64(time.Second) / rate)
	}

	return c.Limit(rule)
}

func (s *crawlerService) setupCollectorCallbacks(ctx context.Context,
	c *colly.Collector, job *crawlJob, excludedSegments []string,
) {
	c.OnRequestHeaders(func(r *colly.Request) {
		//if r.URL != nil {
		//	s.requestTimes.Store(r.URL.String(), time.Now())
		//}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("[job %s] error crawling %s: %v (status=%d)\n",
			job.ID, r.Request.URL, err, r.StatusCode)
		_ = s.pageRepo.MarkPageStatus(ctx, r.Request.URL.String(), crawlstatus.PageErrored)
	})

	c.OnResponse(func(r *colly.Response) {
		u := r.Request.URL.String()
		//if startTime, ok := s.requestTimes.LoadAndDelete(u); ok {
		//	duration := time.Since(startTime.(time.Time))
		//	log.Printf("[url %s] response took %v", u, duration)
		//}

		//doc, err := parseHTML(r.Body)
		//var textContent string
		//if err != nil {
		//	log.Printf("[job %s] HTML parse error for %s: %v\n", job.ID, u, err)
		//} else {
		//	textContent = doc.Text()
		//}

		parentURL := r.Request.Headers.Get("Referer")

		page := &entity.Page{
			URL:        u,
			Path:       r.Request.URL.Path,
			SiteDomain: r.Request.Host,
			ParentURL:  parentURL,
			Depth:      r.Request.Depth,
			HTML:       string(r.Body),
			//Text:       textContent,
			Status: crawlstatus.Running,
			Meta: map[string]any{
				"content_type": r.Headers.Get("Content-Type"),
				"status_code":  r.StatusCode,
			},
			InsertedAt: time.Now().UTC(),
		}

		if err := s.pageRepo.SavePage(ctx, page); err != nil {
			log.Printf("[job %s] storage save error for %s: %v\n", job.ID, u, err)
		}
		err := s.queue.Enqueue(ctx, u)
		if err != nil {
			log.Printf("[job %s] failed to enqueue %s: %v\n", job.ID, u, err)
		}
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))

		if link == "" || !job.shouldFollowLink(link, excludedSegments) {
			return
		}

		lnk := entity.Link{
			FromURL: e.Response.Request.URL.String(),
			ToURL:   link,
			CrawlID: job.ID,
		}

		parent := e.Request.URL.String()
		job.Mu.Lock()
		job.Sitemap[parent] = appendIfMissing(job.Sitemap[parent], link)
		err := s.linksRepo.SaveLink(ctx, lnk)
		if err != nil {
			log.Printf("[job %s] storage save error for link %s\n", job.ID, link)
		}
		job.Mu.Unlock()

		// Dedupe check
		if _, loaded := job.Visited.LoadOrStore(link, true); loaded {
			return
		}
		if isVisited, _ := s.pageRepo.IsVisited(ctx, link); isVisited {
			return
		}

		// Visit the link - this adds it to Colly's queue automatically
		if err := e.Request.Visit(link); err != nil {
			log.Printf("[job %s] error queueing link %s: %v\n", job.ID, link, err)
		}
	})
}

func (j *crawlJob) shouldFollowLink(link string, excludedSegments []string) bool {
	if strings.HasPrefix(link, "mailto:") ||
		strings.HasPrefix(link, "javascript:") ||
		strings.HasPrefix(link, "tel:") {
		return false
	}

	parsedURL, _ := url.Parse(link)
	if parsedURL.Host != j.SiteDomain {
		return false
	}

	if !isWithinSeedPath(parsedURL.Path, j.SeedPath) {
		return false
	}

	// Check for excluded path segments
	if len(excludedSegments) > 0 {
		linkLower := strings.ToLower(link)
		for _, segment := range excludedSegments {
			segmentLower := strings.ToLower(segment)
			// Check if segment appears in path (not just query params)
			if strings.Contains(linkLower, "/"+segmentLower+"/") ||
				strings.Contains(linkLower, "/"+segmentLower+"?") ||
				strings.HasSuffix(linkLower, "/"+segmentLower) {
				return false
			}
		}
	}

	if strings.Contains(link, "?") {
		lower := strings.ToLower(link)
		if strings.Contains(lower, "q=") ||
			strings.Contains(lower, "search") ||
			strings.Contains(lower, "page=") {
			return false
		}
	}

	for _, ext := range []string{".pdf", ".zip", ".exe", ".dmg", ".jpg", ".png", ".gif"} {
		if strings.HasSuffix(strings.ToLower(link), ext) {
			return false
		}
	}

	return true
}

func (s *crawlerService) saveSitemap(ctx context.Context, job *crawlJob) error {
	if err := s.crawlRepo.UpdateCrawlStatus(ctx, job.ID, crawlstatus.Finished); err != nil {
		return fmt.Errorf("failed to save Sitemap: %w", err)
	}

	log.Printf("[job %s] saved Sitemap: %d pages, %d links\n",
		job.ID, len(job.Sitemap), countTotalLinks(job.Sitemap))
	return nil
}

func parseHTML(body []byte) (*goquery.Document, error) {
	reader := bytes.NewReader(body)
	return goquery.NewDocumentFromReader(reader)
}

func appendIfMissing(slice []string, s string) []string {
	for _, x := range slice {
		if x == s {
			return slice
		}
	}
	return append(slice, s)
}

func countTotalLinks(sitemap map[string][]string) int {
	count := 0
	for _, links := range sitemap {
		count += len(links)
	}
	return count
}

// postLogin makes a form POST request for authentication
func (s *crawlerService) postLogin(ctx context.Context, targetURL string, formData map[string]string, jar *cookiejar.Jar) error {
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Follow redirects
		},
	}

	parsedURL, _ := url.Parse(targetURL)

	formValues := url.Values{}
	for key, value := range formData {
		formValues.Set(key, value)
	}
	encodedBody := formValues.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(encodedBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", targetURL)
	req.Header.Set("Origin", parsedURL.Scheme+"://"+parsedURL.Host)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// isWithinSeedPath checks if a URL path is within the seed path scope
func isWithinSeedPath(linkPath, seedPath string) bool {
	// Clean paths
	linkPath = strings.TrimSuffix(linkPath, "/")
	seedPath = strings.TrimSuffix(seedPath, "/")

	// If seed is root, allow everything
	if seedPath == "" || seedPath == "/" {
		return true
	}

	// Check if link path starts with seed path
	return strings.HasPrefix(linkPath, seedPath)
}
