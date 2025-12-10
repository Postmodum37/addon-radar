# REST API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a JSON REST API serving addon data for any frontend to consume.

**Architecture:** Gin web server with versioned endpoints (`/api/v1/*`), CORS enabled, pagination support. Reads from existing PostgreSQL database populated by sync job.

**Tech Stack:** Go 1.25, Gin, sqlc/pgx, PostgreSQL

---

## Task 1: Add Gin Dependency and Create Server Skeleton

**Files:**
- Modify: `go.mod`
- Create: `cmd/web/main.go`
- Create: `internal/api/server.go`

**Step 1: Add Gin dependency**

Run:
```bash
go get github.com/gin-gonic/gin
```

Expected: `go.mod` and `go.sum` updated with gin dependency

**Step 2: Create API server package**

Create `internal/api/server.go`:
```go
package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"addon-radar/internal/database"
)

type Server struct {
	db     *database.Queries
	router *gin.Engine
}

func NewServer(pool *pgxpool.Pool) *Server {
	s := &Server{
		db: database.New(pool),
	}
	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.loggerMiddleware())
	r.Use(s.corsMiddleware())

	api := r.Group("/api/v1")
	{
		api.GET("/health", s.handleHealth)
	}

	s.router = r
}

func (s *Server) Run(addr string) error {
	slog.Info("starting API server", "addr", addr)
	return s.router.Run(addr)
}

func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
```

**Step 3: Create web entry point**

Create `cmd/web/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"addon-radar/internal/api"
	"addon-radar/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("addon-radar API starting...")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	slog.Info("database connected")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := api.NewServer(pool)
	if err := server.Run(fmt.Sprintf(":%s", port)); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
```

**Step 4: Verify it compiles**

Run:
```bash
go build ./cmd/web
```

Expected: No errors

**Step 5: Test health endpoint locally**

Run in terminal 1:
```bash
source .env && go run ./cmd/web
```

Run in terminal 2:
```bash
curl http://localhost:8080/api/v1/health
```

Expected: `{"status":"ok"}`

**Step 6: Commit**

```bash
git add go.mod go.sum cmd/web/ internal/api/
git commit -m "feat: add API server skeleton with health endpoint"
```

---

## Task 2: Add Response Helpers

**Files:**
- Create: `internal/api/response.go`

**Step 1: Create response helpers**

Create `internal/api/response.go`:
```go
package api

import "github.com/gin-gonic/gin"

type PaginatedResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondWithData(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"data": data})
}

func respondWithPagination(c *gin.Context, data interface{}, page, perPage, total int) {
	totalPages := (total + perPage - 1) / perPage
	c.JSON(200, PaginatedResponse{
		Data: data,
		Meta: Meta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func respondWithError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func respondNotFound(c *gin.Context, message string) {
	respondWithError(c, 404, "not_found", message)
}

func respondBadRequest(c *gin.Context, message string) {
	respondWithError(c, 400, "bad_request", message)
}

func respondInternalError(c *gin.Context) {
	respondWithError(c, 500, "internal_error", "An unexpected error occurred")
}
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./cmd/web
```

Expected: No errors

**Step 3: Commit**

```bash
git add internal/api/response.go
git commit -m "feat: add API response helpers"
```

---

## Task 3: Add Database Queries for Listing Addons

**Files:**
- Modify: `sql/queries.sql`
- Regenerate: `internal/database/`

**Step 1: Add listing queries to sql/queries.sql**

Add to `sql/queries.sql`:
```sql
-- name: ListAddons :many
SELECT * FROM addons
WHERE status = 'active'
ORDER BY download_count DESC
LIMIT $1 OFFSET $2;

-- name: CountActiveAddons :one
SELECT COUNT(*) FROM addons WHERE status = 'active';

-- name: ListAddonsByCategory :many
SELECT a.* FROM addons a
WHERE a.status = 'active'
  AND $3 = ANY(a.categories)
ORDER BY a.download_count DESC
LIMIT $1 OFFSET $2;

-- name: CountAddonsByCategory :one
SELECT COUNT(*) FROM addons
WHERE status = 'active'
  AND $1 = ANY(categories);

-- name: SearchAddons :many
SELECT * FROM addons
WHERE status = 'active'
  AND (name ILIKE '%' || $3 || '%' OR summary ILIKE '%' || $3 || '%')
ORDER BY download_count DESC
LIMIT $1 OFFSET $2;

-- name: CountSearchAddons :one
SELECT COUNT(*) FROM addons
WHERE status = 'active'
  AND (name ILIKE '%' || $1 || '%' OR summary ILIKE '%' || $1 || '%');

-- name: GetAddonBySlug :one
SELECT * FROM addons WHERE slug = $1 AND status = 'active';

-- name: GetAddonSnapshots :many
SELECT recorded_at, download_count, thumbs_up_count, popularity_rank
FROM snapshots
WHERE addon_id = $1
ORDER BY recorded_at DESC
LIMIT $2;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY name;

-- name: GetCategoryBySlug :one
SELECT * FROM categories WHERE slug = $1;
```

**Step 2: Regenerate sqlc code**

Run:
```bash
sqlc generate
```

Expected: Files in `internal/database/` updated

**Step 3: Verify it compiles**

Run:
```bash
go build ./...
```

Expected: No errors

**Step 4: Commit**

```bash
git add sql/queries.sql internal/database/
git commit -m "feat: add sqlc queries for API endpoints"
```

---

## Task 4: Implement List Addons Endpoint

**Files:**
- Create: `internal/api/handlers.go`
- Modify: `internal/api/server.go`

**Step 1: Create handlers file**

Create `internal/api/handlers.go`:
```go
package api

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"addon-radar/internal/database"
)

type AddonResponse struct {
	ID              int32    `json:"id"`
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	Summary         string   `json:"summary,omitempty"`
	AuthorName      string   `json:"author_name,omitempty"`
	LogoURL         string   `json:"logo_url,omitempty"`
	DownloadCount   int64    `json:"download_count"`
	ThumbsUpCount   int32    `json:"thumbs_up_count"`
	PopularityRank  int32    `json:"popularity_rank,omitempty"`
	GameVersions    []string `json:"game_versions"`
	LastUpdatedAt   string   `json:"last_updated_at,omitempty"`
}

func addonToResponse(a database.Addon) AddonResponse {
	resp := AddonResponse{
		ID:            a.ID,
		Name:          a.Name,
		Slug:          a.Slug,
		DownloadCount: a.DownloadCount.Int64,
		ThumbsUpCount: a.ThumbsUpCount.Int32,
		GameVersions:  a.GameVersions,
	}

	if a.Summary.Valid {
		resp.Summary = a.Summary.String
	}
	if a.AuthorName.Valid {
		resp.AuthorName = a.AuthorName.String
	}
	if a.LogoUrl.Valid {
		resp.LogoURL = a.LogoUrl.String
	}
	if a.PopularityRank.Valid {
		resp.PopularityRank = a.PopularityRank.Int32
	}
	if a.LastUpdatedAt.Valid {
		resp.LastUpdatedAt = a.LastUpdatedAt.Time.Format("2006-01-02T15:04:05Z")
	}

	return resp
}

func (s *Server) handleListAddons(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	ctx := c.Request.Context()

	var addons []database.Addon
	var total int64
	var err error

	if search != "" {
		addons, err = s.db.SearchAddons(ctx, database.SearchAddonsParams{
			Limit:  int32(perPage),
			Offset: int32(offset),
			Column3: search,
		})
		if err != nil {
			slog.Error("failed to search addons", "error", err)
			respondInternalError(c)
			return
		}
		total, err = s.db.CountSearchAddons(ctx, search)
	} else {
		addons, err = s.db.ListAddons(ctx, database.ListAddonsParams{
			Limit:  int32(perPage),
			Offset: int32(offset),
		})
		if err != nil {
			slog.Error("failed to list addons", "error", err)
			respondInternalError(c)
			return
		}
		total, err = s.db.CountActiveAddons(ctx)
	}

	if err != nil {
		slog.Error("failed to count addons", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]AddonResponse, len(addons))
	for i, a := range addons {
		response[i] = addonToResponse(a)
	}

	respondWithPagination(c, response, page, perPage, int(total))
}
```

**Step 2: Register the route**

Modify `internal/api/server.go`, update `setupRouter`:
```go
func (s *Server) setupRouter() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.loggerMiddleware())
	r.Use(s.corsMiddleware())

	api := r.Group("/api/v1")
	{
		api.GET("/health", s.handleHealth)
		api.GET("/addons", s.handleListAddons)
	}

	s.router = r
}
```

**Step 3: Verify it compiles**

Run:
```bash
go build ./cmd/web
```

Expected: No errors

**Step 4: Test locally**

Run:
```bash
source .env && go run ./cmd/web
```

Test:
```bash
curl "http://localhost:8080/api/v1/addons?per_page=2" | jq
```

Expected: JSON with `data` array and `meta` pagination info

**Step 5: Commit**

```bash
git add internal/api/
git commit -m "feat: implement list addons endpoint with pagination"
```

---

## Task 5: Implement Get Addon by Slug Endpoint

**Files:**
- Modify: `internal/api/handlers.go`
- Modify: `internal/api/server.go`

**Step 1: Add handler**

Add to `internal/api/handlers.go`:
```go
func (s *Server) handleGetAddon(c *gin.Context) {
	slug := c.Param("slug")
	ctx := c.Request.Context()

	addon, err := s.db.GetAddonBySlug(ctx, slug)
	if err != nil {
		respondNotFound(c, "Addon not found")
		return
	}

	respondWithData(c, addonToResponse(addon))
}
```

**Step 2: Register the route**

Add to `setupRouter` in `internal/api/server.go`:
```go
api.GET("/addons/:slug", s.handleGetAddon)
```

**Step 3: Test locally**

```bash
curl "http://localhost:8080/api/v1/addons/details" | jq
```

Expected: Single addon JSON or 404 error

**Step 4: Commit**

```bash
git add internal/api/
git commit -m "feat: implement get addon by slug endpoint"
```

---

## Task 6: Implement Addon History Endpoint

**Files:**
- Modify: `internal/api/handlers.go`
- Modify: `internal/api/server.go`

**Step 1: Add handler**

Add to `internal/api/handlers.go`:
```go
type SnapshotResponse struct {
	RecordedAt     string `json:"recorded_at"`
	DownloadCount  int64  `json:"download_count"`
	ThumbsUpCount  int32  `json:"thumbs_up_count,omitempty"`
	PopularityRank int32  `json:"popularity_rank,omitempty"`
}

func (s *Server) handleGetAddonHistory(c *gin.Context) {
	slug := c.Param("slug")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "168")) // Default 7 days of hourly data
	if limit < 1 || limit > 720 {
		limit = 168
	}

	ctx := c.Request.Context()

	addon, err := s.db.GetAddonBySlug(ctx, slug)
	if err != nil {
		respondNotFound(c, "Addon not found")
		return
	}

	snapshots, err := s.db.GetAddonSnapshots(ctx, database.GetAddonSnapshotsParams{
		AddonID: addon.ID,
		Limit:   int32(limit),
	})
	if err != nil {
		slog.Error("failed to get snapshots", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]SnapshotResponse, len(snapshots))
	for i, snap := range snapshots {
		response[i] = SnapshotResponse{
			RecordedAt:    snap.RecordedAt.Time.Format("2006-01-02T15:04:05Z"),
			DownloadCount: snap.DownloadCount,
		}
		if snap.ThumbsUpCount.Valid {
			response[i].ThumbsUpCount = snap.ThumbsUpCount.Int32
		}
		if snap.PopularityRank.Valid {
			response[i].PopularityRank = snap.PopularityRank.Int32
		}
	}

	respondWithData(c, response)
}
```

**Step 2: Register the route**

Add to `setupRouter`:
```go
api.GET("/addons/:slug/history", s.handleGetAddonHistory)
```

**Step 3: Test locally**

```bash
curl "http://localhost:8080/api/v1/addons/details/history?limit=24" | jq
```

Expected: Array of snapshot data points

**Step 4: Commit**

```bash
git add internal/api/
git commit -m "feat: implement addon history endpoint"
```

---

## Task 7: Implement Categories Endpoint

**Files:**
- Modify: `internal/api/handlers.go`
- Modify: `internal/api/server.go`

**Step 1: Add handler**

Add to `internal/api/handlers.go`:
```go
type CategoryResponse struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ParentID int32  `json:"parent_id,omitempty"`
	IconURL  string `json:"icon_url,omitempty"`
}

func (s *Server) handleListCategories(c *gin.Context) {
	ctx := c.Request.Context()

	categories, err := s.db.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]CategoryResponse, len(categories))
	for i, cat := range categories {
		response[i] = CategoryResponse{
			ID:   cat.ID,
			Name: cat.Name,
			Slug: cat.Slug,
		}
		if cat.ParentID.Valid {
			response[i].ParentID = cat.ParentID.Int32
		}
		if cat.IconUrl.Valid {
			response[i].IconURL = cat.IconUrl.String
		}
	}

	respondWithData(c, response)
}
```

**Step 2: Register the route**

Add to `setupRouter`:
```go
api.GET("/categories", s.handleListCategories)
```

**Step 3: Test locally**

```bash
curl "http://localhost:8080/api/v1/categories" | jq
```

Expected: Array of categories

**Step 4: Commit**

```bash
git add internal/api/
git commit -m "feat: implement categories endpoint"
```

---

## Task 8: Implement Trending Endpoints (Placeholder)

**Files:**
- Modify: `internal/api/handlers.go`
- Modify: `internal/api/server.go`

**Step 1: Add placeholder handlers**

Add to `internal/api/handlers.go`:
```go
func (s *Server) handleTrendingHot(c *gin.Context) {
	ctx := c.Request.Context()

	// Placeholder: return top 20 by downloads until trending algorithm is implemented
	addons, err := s.db.ListAddons(ctx, database.ListAddonsParams{
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		slog.Error("failed to get hot addons", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]AddonResponse, len(addons))
	for i, a := range addons {
		response[i] = addonToResponse(a)
	}

	respondWithData(c, response)
}

func (s *Server) handleTrendingRising(c *gin.Context) {
	ctx := c.Request.Context()

	// Placeholder: return addons with 50-10000 downloads until trending algorithm is implemented
	addons, err := s.db.ListAddons(ctx, database.ListAddonsParams{
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		slog.Error("failed to get rising addons", "error", err)
		respondInternalError(c)
		return
	}

	// Filter for rising stars range (50-10000 downloads)
	var filtered []AddonResponse
	for _, a := range addons {
		if a.DownloadCount.Int64 >= 50 && a.DownloadCount.Int64 <= 10000 {
			filtered = append(filtered, addonToResponse(a))
		}
		if len(filtered) >= 20 {
			break
		}
	}

	respondWithData(c, filtered)
}
```

**Step 2: Register the routes**

Add to `setupRouter`:
```go
api.GET("/trending/hot", s.handleTrendingHot)
api.GET("/trending/rising", s.handleTrendingRising)
```

**Step 3: Test locally**

```bash
curl "http://localhost:8080/api/v1/trending/hot" | jq '.data | length'
curl "http://localhost:8080/api/v1/trending/rising" | jq '.data | length'
```

Expected: 20 (or fewer for rising)

**Step 4: Commit**

```bash
git add internal/api/
git commit -m "feat: add placeholder trending endpoints"
```

---

## Task 9: Create Dockerfile for API Server

**Files:**
- Modify: `Dockerfile`

**Step 1: Update Dockerfile to build both binaries**

Replace `Dockerfile`:
```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sync ./cmd/sync
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web ./cmd/web

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/sync .
COPY --from=builder /build/web .

# Default to web server, override with CMD for sync
CMD ["/app/web"]
```

**Step 2: Test Docker build**

Run:
```bash
docker build -t addon-radar .
```

Expected: Build succeeds

**Step 3: Commit**

```bash
git add Dockerfile
git commit -m "feat: update Dockerfile to build both sync and web binaries"
```

---

## Task 10: Deploy API to Railway

**Step 1: Create railway.toml for web service**

The existing `railway.toml` is for the sync job. You'll need to create a **new service** in Railway for the API.

In Railway dashboard:
1. Click "+ New" in your project
2. Select "Empty Service"
3. Name it "addon-radar-api"
4. Connect to same GitHub repo
5. Set build command: `docker build -t api .`
6. Set start command: `/app/web`

**Step 2: Set environment variables**

In Railway dashboard for the new API service:
- `DATABASE_URL` = `${{Postgres.DATABASE_URL}}`
- `PORT` = `8080` (Railway sets this automatically)
- `ENV` = `production`

**Step 3: Generate domain**

In Railway dashboard:
1. Click on API service
2. Go to "Settings" > "Networking"
3. Click "Generate Domain"

**Step 4: Test deployed API**

```bash
curl "https://your-domain.railway.app/api/v1/health"
curl "https://your-domain.railway.app/api/v1/addons?per_page=2" | jq
```

Expected: JSON responses

**Step 5: Commit any final changes**

```bash
git add .
git commit -m "chore: finalize API deployment configuration"
git push
```

---

## Summary

After completing all tasks you will have:

- ✅ Gin-based REST API server
- ✅ Versioned endpoints (`/api/v1/*`)
- ✅ CORS enabled for frontend flexibility
- ✅ Pagination support
- ✅ Endpoints:
  - `GET /api/v1/health` - Health check
  - `GET /api/v1/addons` - List with pagination & search
  - `GET /api/v1/addons/:slug` - Single addon
  - `GET /api/v1/addons/:slug/history` - Download history
  - `GET /api/v1/categories` - All categories
  - `GET /api/v1/trending/hot` - Placeholder trending
  - `GET /api/v1/trending/rising` - Placeholder trending
- ✅ Deployed to Railway

**Next steps after this plan:**
1. Implement real trending algorithm (replace placeholders)
2. Add category filtering to list endpoint
3. Add sorting options
4. Build frontend to consume the API
