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

# ── Deploy to Railway ─────────────────────────────────────────────────────
deploy:
	@which railway > /dev/null 2>&1 || (echo "Installing Railway CLI..." && brew install railway)
	railway up --detach
	@echo ""
	@echo "Deployed! Run 'railway open' to view your live URL."
	@echo "Make sure GREENHOUSE_API_KEY is set in Railway dashboard → Variables."
