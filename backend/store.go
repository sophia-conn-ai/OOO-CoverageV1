package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Sync times: 00:00, 08:00, 16:00 PST (UTC-8)
var syncHoursPST = []int{0, 8, 16}

const pstOffsetHours = -8

// Candidate is a single pipeline candidate returned to the frontend.
type Candidate struct {
	ID          string `json:"id"`
	CandidateID int64  `json:"candidateId"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Link        string `json:"link"`
	Stage       string `json:"stage"`
}

// ResultCache is the full processed response saved to disk.
type ResultCache struct {
	Candidates  map[string][]Candidate `json:"candidates"`
	Count       int                    `json:"count"`
	LastUpdated time.Time              `json:"lastUpdated"`
	StageOrder  []string               `json:"stageOrder"`
}

// Store handles all disk persistence.
type Store struct {
	dir         string
	mu          sync.RWMutex
	resultCache *ResultCache
}

func NewStore(dir string) *Store {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	s := &Store{dir: dir}
	s.loadResultCache()
	return s
}

func (s *Store) namesFile() string   { return filepath.Join(s.dir, "candidate-names.json") }
func (s *Store) resultFile() string  { return filepath.Join(s.dir, "result-cache.json") }
func (s *Store) assignFile() string  { return filepath.Join(s.dir, "assignments.json") }

// ── Name cache ────────────────────────────────────────────────────────────

func (s *Store) LoadNames() map[string]string {
	data, err := os.ReadFile(s.namesFile())
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]string{}
	}
	return m
}

func (s *Store) SaveNames(m map[string]string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(s.namesFile(), data, 0o644)
}

// ── Result cache ──────────────────────────────────────────────────────────

func (s *Store) loadResultCache() {
	data, err := os.ReadFile(s.resultFile())
	if err != nil {
		return
	}
	var rc ResultCache
	if err := json.Unmarshal(data, &rc); err != nil {
		return
	}
	s.mu.Lock()
	s.resultCache = &rc
	s.mu.Unlock()
}

func (s *Store) GetResultCache() *ResultCache {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resultCache
}

func (s *Store) SetResultCache(rc *ResultCache) error {
	data, err := json.Marshal(rc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.resultFile(), data, 0o644); err != nil {
		return err
	}
	s.mu.Lock()
	s.resultCache = rc
	s.mu.Unlock()
	return nil
}

// ResultCacheIsStale returns true if the cache pre-dates the last scheduled sync.
func (s *Store) ResultCacheIsStale() bool {
	rc := s.GetResultCache()
	if rc == nil {
		return true
	}
	return rc.LastUpdated.Before(lastScheduledSync())
}

func lastScheduledSync() time.Time {
	now := time.Now().UTC()
	pst := now.Add(time.Duration(pstOffsetHours) * time.Hour)

	h := pst.Hour()*60 + pst.Minute()
	syncHour := -1
	for i := len(syncHoursPST) - 1; i >= 0; i-- {
		if syncHoursPST[i]*60 <= h {
			syncHour = syncHoursPST[i]
			break
		}
	}

	base := time.Date(pst.Year(), pst.Month(), pst.Day(), 0, 0, 0, 0, time.UTC)
	if syncHour >= 0 {
		base = base.Add(time.Duration(syncHour) * time.Hour)
	} else {
		// Before midnight PST — last sync was yesterday at 16:00 PST
		base = base.AddDate(0, 0, -1).Add(16 * time.Hour)
	}
	// Convert PST base back to UTC
	return base.Add(-time.Duration(pstOffsetHours) * time.Hour)
}

// ── Assignments ───────────────────────────────────────────────────────────

func (s *Store) LoadAssignments() map[string]interface{} {
	data, err := os.ReadFile(s.assignFile())
	if err != nil {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

func (s *Store) SaveAssignments(m map[string]interface{}) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.assignFile(), data, 0o644)
}
