package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env in dev; in production env vars are set by the platform
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	if os.Getenv("GREENHOUSE_API_KEY") == "" {
		log.Fatal("GREENHOUSE_API_KEY is not set")
	}

	// DATA_DIR defaults to ../data in dev, /data in production containers
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		if _, err := os.Stat("../data"); err == nil {
			dataDir = "../data"
		} else {
			dataDir = "/data"
		}
	}

	store := NewStore(dataDir)
	gh := NewGreenhouseClient(os.Getenv("GREENHOUSE_API_KEY"), store)
	h := NewHandlers(gh, store)

	go func() {
		if store.ResultCacheIsStale() {
			log.Println("No fresh cache — warming in background...")
			gh.Sync()
		} else {
			log.Println("Cache loaded from disk. Ready.")
		}
		ScheduleSyncs(gh)
	}()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// API routes
	api := r.Group("/api")
	api.GET("/candidates", h.GetCandidates)
	api.GET("/assignments", h.GetAssignments)
	api.POST("/assignments", h.SaveAssignments)
	api.GET("/progress", h.SSEProgress)

	// Serve React static build — look next to the binary or at ../frontend/dist
	staticDir := findStaticDir()
	if staticDir != "" {
		log.Printf("Serving frontend from %s", staticDir)
		r.Static("/assets", filepath.Join(staticDir, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))
		// SPA fallback — all non-API routes serve index.html
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(staticDir, "index.html"))
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("OOO Coverage → http://localhost:%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// findStaticDir looks for the React build output in common locations.
func findStaticDir() string {
	candidates := []string{
		"./frontend/dist",   // production: dist copied next to binary
		"../frontend/dist",  // dev: running from backend/
	}
	for _, dir := range candidates {
		if info, err := os.Stat(filepath.Join(dir, "index.html")); err == nil && !info.IsDir() {
			return dir
		}
	}
	return ""
}

// Keep CORS for local dev (when Vite proxy isn't in play)
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
