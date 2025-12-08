# Tech Stack Design

## Overview

This document describes the technology choices for Addon Radar, optimized for learning Go while maintaining development velocity.

## Goals

- Learn Go through a real project
- Keep frontend complexity minimal (focus on backend)
- Ship MVP quickly, polish later
- Separate concerns: sync job vs. web server

## Stack Summary

| Layer | Choice | Rationale |
|-------|--------|-----------|
| **Language** | Go | Learning goal, excellent for APIs and background jobs |
| **Web framework** | Gin | Popular, extensive documentation, built-in validation |
| **Database** | PostgreSQL | Time-series queries, concurrent writes, array columns |
| **DB library** | sqlc | Type-safe generated code from SQL, idiomatic Go |
| **Templates** | html/template (stdlib) | Server-rendered HTML, secure by default |
| **Interactivity** | HTMX | Dynamic updates without JS framework complexity |
| **CSS** | Tailwind CSS (standalone CLI) | No Node.js required, fast iteration |
| **Config** | envconfig | Simple environment variable parsing |
| **Logging** | slog (stdlib) | Structured logging, built into Go 1.21+ |
| **HTTP client** | net/http (stdlib) | For CurseForge API calls |

## Architecture

### Two Binary Approach

The application consists of two separate binaries sharing a common codebase:

| Binary | Purpose | Schedule |
|--------|---------|----------|
| `addon-radar-sync` | Fetches data from CurseForge, writes to DB | Cron (hourly/daily) |
| `addon-radar-web` | Serves website, reads from DB | Always running |

**Benefits:**
- Sync job can run independently without affecting web performance
- Web server has no background job complexity
- Can scale each component independently
- Sync job failures don't crash the website

### Local Development Workflow

1. Production sync job populates PostgreSQL
2. Export script creates database snapshot (pg_dump)
3. Download snapshot for local development
4. Develop web UI against real production data
5. No need to hit CurseForge API during development

## Project Structure

```
addon-radar/
├── cmd/
│   ├── sync/
│   │   └── main.go           # Sync job entry point
│   └── web/
│       └── main.go           # Web server entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Shared configuration
│   ├── database/
│   │   ├── db.go             # Connection setup
│   │   ├── queries.sql.go    # sqlc generated
│   │   └── models.go         # sqlc generated
│   ├── curseforge/
│   │   ├── client.go         # API client
│   │   └── types.go          # API response types
│   ├── trending/
│   │   ├── algorithm.go      # Score calculations
│   │   ├── hot.go            # Hot Right Now logic
│   │   └── rising.go         # Rising Stars logic
│   └── models/
│       └── addon.go          # Shared domain types
├── web/
│   ├── handlers/
│   │   ├── home.go           # Homepage handler
│   │   ├── addon.go          # Addon detail handler
│   │   └── api.go            # HTMX endpoints
│   ├── templates/
│   │   ├── layout.html       # Base template
│   │   ├── home.html         # Homepage
│   │   ├── addon.html        # Addon detail page
│   │   └── partials/         # HTMX fragments
│   └── static/
│       ├── css/
│       │   └── styles.css    # Tailwind output
│       └── images/
├── sql/
│   ├── schema.sql            # Database schema
│   └── queries.sql           # sqlc query definitions
├── scripts/
│   ├── export-db.sh          # Export production snapshot
│   └── import-db.sh          # Import snapshot locally
├── sqlc.yaml                 # sqlc configuration
├── tailwind.config.js        # Tailwind configuration
├── input.css                 # Tailwind input
├── go.mod
├── go.sum
└── Makefile                  # Build commands
```

## Key Libraries

### Gin (Web Framework)

```go
import "github.com/gin-gonic/gin"

func main() {
    r := gin.Default()
    r.GET("/", handlers.Home)
    r.GET("/addon/:slug", handlers.AddonDetail)
    r.Run(":8080")
}
```

**Why Gin over Chi:**
- More tutorials and Stack Overflow answers available
- Built-in JSON binding and validation
- Larger community for troubleshooting
- Good choice when learning Go

### sqlc (Database)

Write SQL, generate type-safe Go:

```sql
-- sql/queries.sql

-- name: GetAddonByID :one
SELECT * FROM addons WHERE id = $1;

-- name: GetHotAddons :many
SELECT * FROM addons
WHERE total_downloads >= 500
ORDER BY hot_score DESC
LIMIT $1;

-- name: CreateSnapshot :exec
INSERT INTO snapshots (addon_id, recorded_at, download_count, thumbs_up_count)
VALUES ($1, $2, $3, $4);
```

Generates:
```go
func (q *Queries) GetAddonByID(ctx context.Context, id int32) (Addon, error)
func (q *Queries) GetHotAddons(ctx context.Context, limit int32) ([]Addon, error)
func (q *Queries) CreateSnapshot(ctx context.Context, arg CreateSnapshotParams) error
```

**Why sqlc over GORM:**
- Learn SQL, not ORM abstractions
- Type-safe without reflection magic
- Better performance (no runtime query building)
- Very Go-idiomatic approach

### HTMX (Interactivity)

```html
<!-- Filter addons without page reload -->
<select hx-get="/partials/addons"
        hx-target="#addon-list"
        hx-trigger="change"
        name="category">
    <option value="">All Categories</option>
    <option value="ui">User Interface</option>
</select>

<div id="addon-list">
    <!-- Server returns HTML fragment, swapped in here -->
</div>
```

**Why HTMX:**
- No JavaScript framework to learn
- Server-rendered HTML (good for SEO)
- Perfect for filtering, pagination, sorting
- Keeps focus on Go backend

### Tailwind CSS (Standalone)

No Node.js required:

```bash
# Download standalone binary
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64
chmod +x tailwindcss-macos-arm64
mv tailwindcss-macos-arm64 tailwindcss

# Watch and compile
./tailwindcss -i input.css -o web/static/css/styles.css --watch
```

## Database Setup

### Local Development (Docker)

```bash
docker run -d --name addon-radar-db \
  -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=addon_radar \
  -p 5432:5432 \
  postgres:16
```

### Production Options

| Service | Free Tier | Notes |
|---------|-----------|-------|
| **Neon** | 0.5 GB | Serverless, branching for dev |
| **Supabase** | 500 MB | Good dashboard, auth if needed later |
| **Railway** | $5/month credit | Simple deployment |

## Build & Run

### Makefile

```makefile
.PHONY: build run-web run-sync db-up db-down sqlc tailwind

# Build both binaries
build:
	go build -o bin/sync ./cmd/sync
	go build -o bin/web ./cmd/web

# Run web server
run-web:
	go run ./cmd/web

# Run sync job once
run-sync:
	go run ./cmd/sync

# Database
db-up:
	docker start addon-radar-db || docker run -d --name addon-radar-db \
		-e POSTGRES_PASSWORD=dev \
		-e POSTGRES_DB=addon_radar \
		-p 5432:5432 postgres:16

db-down:
	docker stop addon-radar-db

# Generate sqlc code
sqlc:
	sqlc generate

# Watch Tailwind
tailwind:
	./tailwindcss -i input.css -o web/static/css/styles.css --watch
```

## Environment Variables

```bash
# .env (local development)
DATABASE_URL=postgres://postgres:dev@localhost:5432/addon_radar?sslmode=disable
CURSEFORGE_API_KEY=your-api-key
PORT=8080
ENV=development

# Production adds:
# DATABASE_URL=postgres://user:pass@host:5432/addon_radar?sslmode=require
# ENV=production
```

## Future Considerations

### Potential Svelte Migration

If richer UI is needed later:
1. Keep Go API (Gin serves JSON)
2. Add SvelteKit frontend
3. Deploy as two services
4. Gradual migration possible

### Scaling

Current stack handles the projected load easily:
- ~7,000 addons, ~5M snapshots/year
- Single PostgreSQL instance sufficient
- Can add read replicas if needed
- Horizontal scaling via multiple web instances

## Summary

| Decision | Choice |
|----------|--------|
| Language | Go (learning goal) |
| Framework | Gin (docs + community) |
| Database | PostgreSQL (time-series, concurrent writes) |
| DB access | sqlc (type-safe, idiomatic) |
| Frontend | Server-rendered HTML + HTMX |
| CSS | Tailwind standalone |
| Architecture | Two binaries (sync + web) |
| Local dev | Download production snapshots |
