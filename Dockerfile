# ── Stage 1: Build React frontend ────────────────────────────────────────
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ── Stage 2: Build Go backend ─────────────────────────────────────────────
FROM golang:1.25-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go build -o /app/server .

# ── Stage 3: Minimal runtime image ───────────────────────────────────────
FROM alpine:latest
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app

COPY --from=backend /app/server ./server
COPY --from=frontend /app/frontend/dist ./frontend/dist

# Persistent data volume (candidate cache + assignments)
RUN mkdir -p /data
VOLUME ["/data"]

ENV DATA_DIR=/data
ENV PORT=8080
EXPOSE 8080

CMD ["./server"]
