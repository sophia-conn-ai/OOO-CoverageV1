package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	gh    *GreenhouseClient
	store *Store
}

func NewHandlers(gh *GreenhouseClient, store *Store) *Handlers {
	return &Handlers{gh: gh, store: store}
}

// GET /api/candidates — stale-while-revalidate
func (h *Handlers) GetCandidates(c *gin.Context) {
	forceRefresh := c.Query("refresh") == "true"

	if forceRefresh {
		h.gh.Sync()
	} else if h.store.ResultCacheIsStale() && !h.gh.IsSyncing() {
		go h.gh.Sync()
	}

	rc := h.store.GetResultCache()
	if rc == nil {
		if h.gh.IsSyncing() {
			c.JSON(http.StatusAccepted, gin.H{"syncing": true, "message": "Initial sync in progress, please wait..."})
		} else {
			go h.gh.Sync()
			c.JSON(http.StatusAccepted, gin.H{"syncing": true, "message": "Starting sync..."})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"candidates":  rc.Candidates,
		"count":       rc.Count,
		"lastUpdated": rc.LastUpdated,
		"stageOrder":  rc.StageOrder,
		"refreshing":  h.gh.IsSyncing(),
	})
}

// GET /api/assignments
func (h *Handlers) GetAssignments(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.LoadAssignments())
}

// POST /api/assignments
func (h *Handlers) SaveAssignments(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SaveAssignments(m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GET /api/progress — SSE stream of sync progress messages
func (h *Handlers) SSEProgress(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.gh.Subscribe()
	defer h.gh.Unsubscribe(ch)

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", jsonStr(msg))
			c.Writer.Flush()
			if msg == "done" || msg == "error" {
				return
			}
		}
	}
}

func jsonStr(s string) string {
	b, _ := json.Marshal(map[string]string{"msg": s})
	return string(b)
}
