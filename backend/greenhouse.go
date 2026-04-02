package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)
var linkLastRe = regexp.MustCompile(`page=(\d+)[^>]*>;\s*rel="last"`)

// Excluded pipeline stages — not relevant for OOO coverage
var excludedStages = map[string]bool{
	"application review":  true,
	"document submission": true,
}

// Ordered stage names for sorting
var stageOrder = []string{
	"recruiter screen", "phone screen", "initial phone screen",
	"hiring manager", "technical phone", "coderpad", "take home",
	"assessment", "interview", "on-site", "onsite",
	"pre-interview", "final interview", "onsite and leads",
	"ai hiring", "offer", "background check", "hired",
}

func stageIndex(name string) int {
	l := strings.ToLower(name)
	for i, s := range stageOrder {
		if strings.Contains(l, s) {
			return i
		}
	}
	return 999
}

// ── Greenhouse API types ──────────────────────────────────────────────────

type ghRecruiter struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Name      string `json:"name"`
}

type ghStage struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ghJob struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ghApplication struct {
	ID           int64       `json:"id"`
	CandidateID  int64       `json:"candidate_id"`
	Status       string      `json:"status"`
	CurrentStage *ghStage    `json:"current_stage"`
	Jobs         []ghJob     `json:"jobs"`
	Recruiter    *ghRecruiter `json:"recruiter"`
}

type ghCandidate struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// ── Client ────────────────────────────────────────────────────────────────

type GreenhouseClient struct {
	authHeader string
	store      *Store
	httpClient *http.Client

	syncMu    sync.Mutex
	syncing   bool
	listeners []chan string
	listenerMu sync.Mutex
}

func NewGreenhouseClient(apiKey string, store *Store) *GreenhouseClient {
	encoded := base64.StdEncoding.EncodeToString([]byte(apiKey + ":"))
	return &GreenhouseClient{
		authHeader: "Basic " + encoded,
		store:      store,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GreenhouseClient) IsSyncing() bool {
	c.syncMu.Lock()
	defer c.syncMu.Unlock()
	return c.syncing
}

// Subscribe returns a channel that receives progress messages during a sync.
func (c *GreenhouseClient) Subscribe() chan string {
	ch := make(chan string, 32)
	c.listenerMu.Lock()
	c.listeners = append(c.listeners, ch)
	c.listenerMu.Unlock()
	return ch
}

func (c *GreenhouseClient) Unsubscribe(ch chan string) {
	c.listenerMu.Lock()
	defer c.listenerMu.Unlock()
	for i, l := range c.listeners {
		if l == ch {
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
			return
		}
	}
}

func (c *GreenhouseClient) broadcast(msg string) {
	c.listenerMu.Lock()
	defer c.listenerMu.Unlock()
	for _, ch := range c.listeners {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Sync performs a full fetch and updates the cache. Safe to call concurrently.
func (c *GreenhouseClient) Sync() {
	c.syncMu.Lock()
	if c.syncing {
		c.syncMu.Unlock()
		return
	}
	c.syncing = true
	c.syncMu.Unlock()

	defer func() {
		c.syncMu.Lock()
		c.syncing = false
		c.syncMu.Unlock()
		c.broadcast("done")
	}()

	c.broadcast("Fetching applications from Greenhouse...")

	oneYearAgo := time.Now().AddDate(-1, 0, 0).Format(time.RFC3339)
	apps, err := c.fetchAllApplications(oneYearAgo)
	if err != nil {
		log.Printf("Error fetching applications: %v", err)
		c.broadcast("error")
		return
	}

	c.broadcast(fmt.Sprintf("Scanned %d applications — filtering for Sophia Conn...", len(apps)))

	var sophiaApps []ghApplication
	for _, a := range apps {
		if a.Recruiter == nil {
			continue
		}
		name := a.Recruiter.Name
		if name == "" {
			name = strings.TrimSpace(a.Recruiter.FirstName + " " + a.Recruiter.LastName)
		}
		if strings.EqualFold(name, "sophia conn") {
			sophiaApps = append(sophiaApps, a)
		}
	}

	c.broadcast(fmt.Sprintf("Found %d active applications. Fetching names...", len(sophiaApps)))

	// Collect unique candidate IDs
	seen := map[int64]bool{}
	var candidateIDs []int64
	for _, a := range sophiaApps {
		if !seen[a.CandidateID] {
			seen[a.CandidateID] = true
			candidateIDs = append(candidateIDs, a.CandidateID)
		}
	}

	// Fetch missing names
	nameCache := c.store.LoadNames()
	nameCache = c.fetchMissingNames(candidateIDs, nameCache)

	c.broadcast("Building view...")

	// Group by stage (excluding unwanted ones)
	grouped := map[string][]Candidate{}
	for _, a := range sophiaApps {
		stageName := "Unknown Stage"
		if a.CurrentStage != nil {
			stageName = a.CurrentStage.Name
		}
		if excludedStages[strings.ToLower(stageName)] {
			continue
		}

		idStr := fmt.Sprintf("%d", a.ID)
		name := nameCache[fmt.Sprintf("%d", a.CandidateID)]
		if name == "" {
			name = fmt.Sprintf("Candidate #%d", a.CandidateID)
		}
		role := "Unknown Role"
		if len(a.Jobs) > 0 {
			role = a.Jobs[0].Name
		}
		link := fmt.Sprintf("https://app.greenhouse.io/people/%d?application_id=%d", a.CandidateID, a.ID)

		grouped[stageName] = append(grouped[stageName], Candidate{
			ID:          idStr,
			CandidateID: a.CandidateID,
			Name:        name,
			Role:        role,
			Link:        link,
			Stage:       stageName,
		})
	}

	// Sort candidates within each stage
	for stage := range grouped {
		cands := grouped[stage]
		for i := 1; i < len(cands); i++ {
			for j := i; j > 0 && cands[j].Name < cands[j-1].Name; j-- {
				cands[j], cands[j-1] = cands[j-1], cands[j]
			}
		}
		grouped[stage] = cands
	}

	// Build stage order
	stages := make([]string, 0, len(grouped))
	for s := range grouped {
		stages = append(stages, s)
	}
	for i := 1; i < len(stages); i++ {
		for j := i; j > 0 && stageIndex(stages[j]) < stageIndex(stages[j-1]); j-- {
			stages[j], stages[j-1] = stages[j-1], stages[j]
		}
	}

	total := 0
	for _, cands := range grouped {
		total += len(cands)
	}

	rc := &ResultCache{
		Candidates:  grouped,
		Count:       total,
		LastUpdated: time.Now(),
		StageOrder:  stages,
	}

	if err := c.store.SetResultCache(rc); err != nil {
		log.Printf("Error saving result cache: %v", err)
	}

	log.Printf("Sync complete: %d candidates across %d stages", total, len(stages))
}

// ── HTTP helpers ──────────────────────────────────────────────────────────

func (c *GreenhouseClient) get(url string) (*http.Response, error) {
	for attempt := 0; attempt < 5; attempt++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", c.authHeader)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == 429 {
			wait := 3 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if s, err := strconv.Atoi(ra); err == nil {
					wait = time.Duration(s) * time.Second
				}
			}
			resp.Body.Close()
			log.Printf("  Rate limited, waiting %s...", wait)
			time.Sleep(wait)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("too many rate limit retries")
}

// ── Application pagination ────────────────────────────────────────────────

func (c *GreenhouseClient) fetchAllApplications(since string) ([]ghApplication, error) {
	baseURL := fmt.Sprintf(
		"https://harvest.greenhouse.io/v1/applications?per_page=500&status=active&last_activity_after=%s&page=1",
		since,
	)

	resp, err := c.get(baseURL)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var first []ghApplication
	if err := json.Unmarshal(body, &first); err != nil {
		return nil, err
	}

	totalPages := 1
	if m := linkLastRe.FindStringSubmatch(resp.Header.Get("Link")); len(m) > 1 {
		totalPages, _ = strconv.Atoi(m[1])
	}

	log.Printf("  Fetching %d pages (~%d applications)...", totalPages, totalPages*500)
	c.broadcast(fmt.Sprintf("Fetching %d pages of applications...", totalPages))

	if totalPages == 1 {
		return first, nil
	}

	results := make([]ghApplication, 0, totalPages*500)
	results = append(results, first...)

	const concurrency = 6
	pageNums := make([]int, 0, totalPages-1)
	for p := 2; p <= totalPages; p++ {
		pageNums = append(pageNums, p)
	}

	type pageResult struct {
		apps []ghApplication
		err  error
	}

	for i := 0; i < len(pageNums); i += concurrency {
		batch := pageNums[i:]
		if len(batch) > concurrency {
			batch = batch[:concurrency]
		}

		ch := make(chan pageResult, len(batch))
		for _, page := range batch {
			go func(p int) {
				url := fmt.Sprintf(
					"https://harvest.greenhouse.io/v1/applications?per_page=500&status=active&last_activity_after=%s&page=%d",
					since, p,
				)
				resp, err := c.get(url)
				if err != nil {
					ch <- pageResult{err: err}
					return
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				var apps []ghApplication
				if err := json.Unmarshal(body, &apps); err != nil {
					ch <- pageResult{err: err}
					return
				}
				ch <- pageResult{apps: apps}
			}(page)
		}

		for range batch {
			r := <-ch
			if r.err == nil {
				results = append(results, r.apps...)
			}
		}

		done := i + len(batch) + 1
		if done%20 == 0 || done == totalPages {
			log.Printf("  Pages fetched: %d/%d", done, totalPages)
			c.broadcast(fmt.Sprintf("Pages fetched: %d/%d", done, totalPages))
		}
		time.Sleep(120 * time.Millisecond)
	}

	return results, nil
}

// ── Candidate name fetching ───────────────────────────────────────────────

func (c *GreenhouseClient) fetchMissingNames(ids []int64, cache map[string]string) map[string]string {
	var missing []int64
	for _, id := range ids {
		if _, ok := cache[fmt.Sprintf("%d", id)]; !ok {
			missing = append(missing, id)
		}
	}

	if len(missing) == 0 {
		log.Println("  All candidate names already cached.")
		return cache
	}

	log.Printf("  Fetching %d new candidate names (%d cached)...", len(missing), len(ids)-len(missing))
	c.broadcast(fmt.Sprintf("Fetching %d new candidate names (%d cached)...", len(missing), len(ids)-len(missing)))

	const concurrency = 8
	const batchDelay = 150 * time.Millisecond

	type nameResult struct {
		id   int64
		name string
	}

	for i := 0; i < len(missing); i += concurrency {
		batch := missing[i:]
		if len(batch) > concurrency {
			batch = batch[:concurrency]
		}

		ch := make(chan nameResult, len(batch))
		for _, id := range batch {
			go func(id int64) {
				url := fmt.Sprintf("https://harvest.greenhouse.io/v1/candidates/%d", id)
				resp, err := c.get(url)
				if err != nil {
					ch <- nameResult{id: id}
					return
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				var cand ghCandidate
				if err := json.Unmarshal(body, &cand); err != nil {
					ch <- nameResult{id: id}
					return
				}
				name := strings.TrimSpace(cand.FirstName + " " + cand.LastName)
				if name == "" {
					name = fmt.Sprintf("Candidate #%d", id)
				}
				ch <- nameResult{id: id, name: name}
			}(id)
		}

		for range batch {
			r := <-ch
			cache[fmt.Sprintf("%d", r.id)] = r.name
		}

		done := min(i+concurrency, len(missing))
		if done%40 == 0 || done == len(missing) {
			log.Printf("  Names fetched: %d/%d", done, len(missing))
			c.broadcast(fmt.Sprintf("Names fetched: %d/%d", done, len(missing)))
			_ = c.store.SaveNames(cache)
		}

		time.Sleep(batchDelay)
	}

	_ = c.store.SaveNames(cache)
	return cache
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
