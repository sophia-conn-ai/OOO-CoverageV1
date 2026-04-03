.PHONY: dev build deploy

# ── Local dev ─────────────────────────────────────────────────────────────
dev:
	@echo "Starting Go API and React frontend..."
	@(cd backend && go run .) &
	@(cd frontend && npm run dev)

# ── Production build ──────────────────────────────────────────────────────
build:
	@echo "Building React frontend..."
	cd frontend && npm ci && npm run build
	@echo "Building Go backend..."
	cd backend && go build -o ../server .
	@echo "Build complete → ./server + ./frontend/dist"

# ── Deploy to Apps Platform ───────────────────────────────────────────────
deploy:
	~/.local/bin/apps-platform app deploy
