# Tech Stack Design

> **Status**: Reference document. Updated December 2025.
>
> **Implemented**:
> - Go 1.25, sqlc, pgx/v5, envconfig, slog
> - PostgreSQL on Railway
> - Sync job deployed (hourly cron)
>
> **Next**:
> - JSON REST API (cmd/web)
>
> **Deferred**:
> - Frontend (to be decided later: Svelte, React, HTMX, etc.)

## Overview

Addon Radar uses an **API-first architecture**. The backend serves JSON endpoints, and the frontend is a separate concern to be implemented later.

## Goals

- Learn Go through a real project
- Build a clean REST API that any frontend can consume
- Ship working API quickly, add frontend later
- Separate concerns: sync job vs. API server vs. frontend

## Stack Summary

| Layer | Choice | Rationale |
|-------|--------|-----------|
| **Language** | Go 1.25 | Learning goal, excellent for APIs |
| **Web framework** | Gin | Popular, extensive docs, JSON handling |
| **Database** | PostgreSQL | Time-series queries, array columns |
| **DB library** | sqlc + pgx/v5 | Type-safe generated code from SQL |
| **Config** | envconfig | Simple environment variable parsing |
| **Logging** | slog (stdlib) | Structured logging, built-in |
| **HTTP client** | net/http (stdlib) | For CurseForge API calls |

### Removed from Original Plan

| Originally Planned | Status |
|--------------------|--------|
| html/template | Deferred - API-first |
| HTMX | Deferred - API-first |
| Tailwind CSS | Deferred - API-first |

## Architecture

### Three Components

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Sync Job   │────▶│ PostgreSQL  │◀────│  REST API   │
│ (cmd/sync)  │     │  (Railway)  │     │  (cmd/web)  │
└─────────────┘     └─────────────┘     └─────────────┘
     Hourly              Data              Always On
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │  Frontend   │
                                        │  (Future)   │
                                        └─────────────┘
```

| Component | Purpose | Status |
|-----------|---------|--------|
| `cmd/sync` | Fetches data from CurseForge, writes to DB | ✅ Deployed |
| `cmd/web` | JSON REST API, reads from DB | 🔜 Next |
| Frontend | Web UI consuming the API | Deferred |

**Benefits:**
- API can be tested/used before frontend exists
- Frontend technology decision can wait
- Multiple frontends possible (web, mobile, CLI)
- Clear separation of concerns

## API Design

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/addons` | List addons (paginated, filterable) |
| `GET` | `/api/v1/addons/:slug` | Get single addon by slug |
| `GET` | `/api/v1/addons/:slug/history` | Get download history for charts |
| `GET` | `/api/v1/trending/hot` | Hot Right Now list |
| `GET` | `/api/v1/trending/rising` | Rising Stars list |
| `GET` | `/api/v1/categories` | List all categories |
| `GET` | `/api/v1/health` | Health check |

### Query Parameters

**`/api/v1/addons`:**
- `page` - Page number (default: 1)
- `per_page` - Items per page (default: 20, max: 100)
- `category` - Filter by category slug
- `sort` - Sort field: `downloads`, `updated`, `name`, `trending`
- `order` - Sort order: `asc`, `desc`
- `search` - Search by name

### Response Format

```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 12406,
    "total_pages": 621
  }
}
```

### Error Format

```json
{
  "error": {
    "code": "not_found",
    "message": "Addon not found"
  }
}
```

## Project Structure

```
addon-radar/
├── cmd/
│   ├── sync/
│   │   └── main.go           # Sync job entry point ✅
│   └── web/
│       └── main.go           # API server entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Shared configuration ✅
│   ├── database/
│   │   ├── queries.sql.go    # sqlc generated ✅
│   │   └── models.go         # sqlc generated ✅
│   ├── curseforge/
│   │   ├── client.go         # API client ✅
│   │   └── types.go          # API response types ✅
│   ├── sync/
│   │   └── sync.go           # Sync service ✅
│   ├── api/
│   │   ├── server.go         # Gin setup, middleware
│   │   ├── handlers.go       # Route handlers
│   │   └── response.go       # JSON response helpers
│   └── trending/
│       ├── algorithm.go      # Score calculations
│       ├── hot.go            # Hot Right Now logic
│       └── rising.go         # Rising Stars logic
├── sql/
│   ├── schema.sql            # Database schema ✅
│   └── queries.sql           # sqlc query definitions ✅
├── scripts/
│   └── db-setup.sh           # Local Docker setup ✅
├── sqlc.yaml                 # sqlc configuration ✅
├── Dockerfile                # Railway deployment ✅
├── railway.toml              # Railway config ✅
├── go.mod                    # ✅
└── go.sum                    # ✅
```

## Key Libraries

### Gin (Web Framework)

```go
import "github.com/gin-gonic/gin"

func main() {
    r := gin.Default()

    // API routes
    api := r.Group("/api/v1")
    {
        api.GET("/health", handlers.Health)
        api.GET("/addons", handlers.ListAddons)
        api.GET("/addons/:slug", handlers.GetAddon)
        api.GET("/trending/hot", handlers.HotAddons)
        api.GET("/trending/rising", handlers.RisingAddons)
        api.GET("/categories", handlers.ListCategories)
    }

    r.Run(":8080")
}
```

### sqlc (Database)

```sql
-- sql/queries.sql

-- name: ListAddons :many
SELECT * FROM addons
WHERE status = 'active'
ORDER BY download_count DESC
LIMIT $1 OFFSET $2;

-- name: GetAddonBySlug :one
SELECT * FROM addons WHERE slug = $1;

-- name: GetAddonHistory :many
SELECT recorded_at, download_count, thumbs_up_count
FROM snapshots
WHERE addon_id = $1
ORDER BY recorded_at DESC
LIMIT $2;
```

## Database

### Local Development (Docker)

```bash
docker run -d --name addon-radar-db \
  -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=addon_radar \
  -p 5432:5432 \
  postgres:16
```

### Production

PostgreSQL on Railway (already deployed with sync job).

## Environment Variables

```bash
# Required
DATABASE_URL=postgres://...
CURSEFORGE_API_KEY=...  # Only needed for sync job

# Optional
PORT=8080               # API server port
ENV=development         # development/production
```

## CORS Configuration

Since frontend will be separate, API needs CORS headers:

```go
func CORSMiddleware() gin.HandlerFunc {
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
```

## Future Considerations

### Frontend Options (Deferred)

When ready to build frontend:

| Option | Pros | Cons |
|--------|------|------|
| **SvelteKit** | Fast, modern, good DX | New framework to learn |
| **React/Next.js** | Huge ecosystem | Heavy, complex |
| **HTMX + templates** | Simple, Go-native | Less interactive |
| **Static HTML + JS** | Simplest | Limited functionality |

### Rate Limiting (Future)

If needed later:
```go
// Optional rate limiting middleware
api.Use(ratelimit.New(100, time.Minute))
```

### Caching (Future)

Trending scores change hourly, so caching is viable:
- Redis for API response caching
- In-memory cache for trending lists
- CDN for static assets (when frontend exists)

## Summary

| Decision | Choice |
|----------|--------|
| Architecture | API-first (JSON REST) |
| Language | Go 1.25 |
| Framework | Gin |
| Database | PostgreSQL + sqlc |
| Auth | None (public API) |
| Frontend | Deferred |
| Deployment | Railway |
