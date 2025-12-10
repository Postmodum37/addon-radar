# Sync Job Implementation Plan

> **Status: ✅ COMPLETE** (December 2025)
>
> This plan has been fully implemented. The sync job is deployed to Railway and running hourly.
> - 12,406 Retail addons synced
> - Multi-query strategy added to achieve 99.8% catalog coverage
> - Two-pass category sync implemented to handle FK constraints

**Goal:** Build and deploy the CurseForge sync job to start accumulating addon data.

**Architecture:** Single Go binary that fetches all WoW addons from CurseForge API, stores metadata in PostgreSQL, and creates download snapshots. Runs via cron on Railway.

**Tech Stack:** Go 1.21+, Gin (not needed for sync, but shared code), sqlc, PostgreSQL, Railway (hosting)

---

## Task 1: Initialize Go Project

**Files:**
- Create: `go.mod`
- Create: `cmd/sync/main.go`
- Create: `.gitignore`

**Step 1: Initialize Go module**

Run:
```bash
go mod init github.com/your-username/addon-radar
```

Expected: Creates `go.mod` file

**Step 2: Create project structure**

Run:
```bash
mkdir -p cmd/sync cmd/web internal/config internal/database internal/curseforge internal/models sql scripts
```

**Step 3: Create minimal sync entry point**

Create `cmd/sync/main.go`:
```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("addon-radar sync starting...")

	apiKey := os.Getenv("CURSEFORGE_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: CURSEFORGE_API_KEY not set")
		os.Exit(1)
	}

	fmt.Println("API key found, sync would run here")
}
```

**Step 4: Update .gitignore**

Create `.gitignore`:
```
# Binaries
bin/
*.exe

# Environment
.env
.env.local

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store

# Build
dist/
```

**Step 5: Verify it compiles**

Run:
```bash
go build -o bin/sync ./cmd/sync
```

Expected: Creates `bin/sync` binary without errors

**Step 6: Test run**

Run:
```bash
CURSEFORGE_API_KEY=test ./bin/sync
```

Expected output:
```
addon-radar sync starting...
API key found, sync would run here
```

**Step 7: Commit**

```bash
git add go.mod cmd/ .gitignore
git commit -m "feat: initialize Go project with sync command skeleton"
```

---

## Task 2: Create Database Schema

**Files:**
- Create: `sql/schema.sql`
- Create: `sql/queries.sql`
- Create: `sqlc.yaml`

**Step 1: Install sqlc**

Run:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**Step 2: Create database schema**

Create `sql/schema.sql`:
```sql
-- Addons table: stores addon metadata
CREATE TABLE addons (
    id INTEGER PRIMARY KEY,  -- CurseForge addon ID
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    summary TEXT,
    author_name TEXT,
    author_id INTEGER,
    logo_url TEXT,
    primary_category_id INTEGER,
    categories INTEGER[] DEFAULT '{}',
    game_versions TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ,
    last_updated_at TIMESTAMPTZ,
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    is_hot BOOLEAN DEFAULT FALSE,
    hot_until TIMESTAMPTZ,
    status TEXT DEFAULT 'active',

    -- Current metrics (updated each sync)
    download_count BIGINT DEFAULT 0,
    thumbs_up_count INTEGER DEFAULT 0,
    popularity_rank INTEGER,
    rating DECIMAL(3,2),
    latest_file_date TIMESTAMPTZ
);

CREATE INDEX idx_addons_slug ON addons(slug);
CREATE INDEX idx_addons_is_hot ON addons(is_hot) WHERE is_hot = TRUE;
CREATE INDEX idx_addons_status ON addons(status);

-- Snapshots table: time-series metrics
CREATE TABLE snapshots (
    id BIGSERIAL PRIMARY KEY,
    addon_id INTEGER NOT NULL REFERENCES addons(id) ON DELETE CASCADE,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    download_count BIGINT NOT NULL,
    thumbs_up_count INTEGER,
    popularity_rank INTEGER,
    rating DECIMAL(3,2),
    latest_file_date TIMESTAMPTZ
);

CREATE INDEX idx_snapshots_addon_time ON snapshots(addon_id, recorded_at DESC);
CREATE INDEX idx_snapshots_recorded_at ON snapshots(recorded_at DESC);

-- Categories table: reference data
CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_id INTEGER REFERENCES categories(id),
    icon_url TEXT
);
```

**Step 3: Create sqlc queries**

Create `sql/queries.sql`:
```sql
-- name: UpsertAddon :exec
INSERT INTO addons (
    id, name, slug, summary, author_name, author_id, logo_url,
    primary_category_id, categories, game_versions,
    created_at, last_updated_at, last_synced_at,
    download_count, thumbs_up_count, popularity_rank, rating, latest_file_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), $13, $14, $15, $16, $17
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    summary = EXCLUDED.summary,
    author_name = EXCLUDED.author_name,
    author_id = EXCLUDED.author_id,
    logo_url = EXCLUDED.logo_url,
    primary_category_id = EXCLUDED.primary_category_id,
    categories = EXCLUDED.categories,
    game_versions = EXCLUDED.game_versions,
    last_updated_at = EXCLUDED.last_updated_at,
    last_synced_at = NOW(),
    download_count = EXCLUDED.download_count,
    thumbs_up_count = EXCLUDED.thumbs_up_count,
    popularity_rank = EXCLUDED.popularity_rank,
    rating = EXCLUDED.rating,
    latest_file_date = EXCLUDED.latest_file_date;

-- name: CreateSnapshot :exec
INSERT INTO snapshots (addon_id, recorded_at, download_count, thumbs_up_count, popularity_rank, rating, latest_file_date)
VALUES ($1, NOW(), $2, $3, $4, $5, $6);

-- name: GetAddonByID :one
SELECT * FROM addons WHERE id = $1;

-- name: GetAddonBySlug :one
SELECT * FROM addons WHERE slug = $1;

-- name: GetAllAddonIDs :many
SELECT id FROM addons WHERE status = 'active';

-- name: GetHotAddonIDs :many
SELECT id FROM addons WHERE is_hot = TRUE AND status = 'active';

-- name: CountAddons :one
SELECT COUNT(*) FROM addons WHERE status = 'active';

-- name: CountSnapshots :one
SELECT COUNT(*) FROM snapshots;

-- name: UpsertCategory :exec
INSERT INTO categories (id, name, slug, parent_id, icon_url)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    parent_id = EXCLUDED.parent_id,
    icon_url = EXCLUDED.icon_url;
```

**Step 4: Create sqlc config**

Create `sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries.sql"
    schema: "sql/schema.sql"
    gen:
      go:
        package: "database"
        out: "internal/database"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
```

**Step 5: Generate Go code from SQL**

Run:
```bash
sqlc generate
```

Expected: Creates files in `internal/database/`:
- `db.go`
- `models.go`
- `queries.sql.go`

**Step 6: Add pgx dependency**

Run:
```bash
go get github.com/jackc/pgx/v5
```

**Step 7: Verify generated code compiles**

Run:
```bash
go build ./...
```

Expected: No errors

**Step 8: Commit**

```bash
git add sql/ sqlc.yaml internal/database/
git commit -m "feat: add database schema and sqlc generated code"
```

---

## Task 3: Implement Configuration

**Files:**
- Create: `internal/config/config.go`

**Step 1: Add envconfig dependency**

Run:
```bash
go get github.com/kelseyhightower/envconfig
```

**Step 2: Create config struct**

Create `internal/config/config.go`:
```go
package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL      string `envconfig:"DATABASE_URL" required:"true"`
	CurseForgeAPIKey string `envconfig:"CURSEFORGE_API_KEY" required:"true"`
	Environment      string `envconfig:"ENV" default:"development"`
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

**Step 3: Update main.go to use config**

Update `cmd/sync/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"addon-radar/internal/config"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("addon-radar sync starting...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("config loaded", "env", cfg.Environment)

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	err = pool.Ping(ctx)
	if err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	slog.Info("database connected successfully")
	slog.Info("sync complete")
}
```

**Step 4: Update go.mod module path**

Edit `go.mod` to ensure the module path matches imports:
```
module addon-radar

go 1.21
```

**Step 5: Verify compilation**

Run:
```bash
go build -o bin/sync ./cmd/sync
```

Expected: Compiles without errors

**Step 6: Commit**

```bash
git add internal/config/ cmd/sync/main.go go.mod go.sum
git commit -m "feat: add configuration loading and database connection"
```

---

## Task 4: Implement CurseForge API Client

**Files:**
- Create: `internal/curseforge/client.go`
- Create: `internal/curseforge/types.go`

**Step 1: Create API types**

Create `internal/curseforge/types.go`:
```go
package curseforge

import "time"

const (
	BaseURL   = "https://api.curseforge.com"
	GameIDWoW = 1
)

// SearchModsResponse is the response from /v1/mods/search
type SearchModsResponse struct {
	Data       []Mod      `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination info from API responses
type Pagination struct {
	Index       int `json:"index"`
	PageSize    int `json:"pageSize"`
	ResultCount int `json:"resultCount"`
	TotalCount  int `json:"totalCount"`
}

// Mod represents a CurseForge addon/mod
type Mod struct {
	ID                   int        `json:"id"`
	GameID               int        `json:"gameId"`
	Name                 string     `json:"name"`
	Slug                 string     `json:"slug"`
	Summary              string     `json:"summary"`
	DownloadCount        int64      `json:"downloadCount"`
	ThumbsUpCount        int        `json:"thumbsUpCount"`
	Rating               float64    `json:"rating"`
	PopularityRank       int        `json:"popularityRank"`
	DateCreated          time.Time  `json:"dateCreated"`
	DateModified         time.Time  `json:"dateModified"`
	DateReleased         time.Time  `json:"dateReleased"`
	Categories           []Category `json:"categories"`
	Authors              []Author   `json:"authors"`
	Logo                 *Logo      `json:"logo"`
	LatestFiles          []File     `json:"latestFiles"`
	LatestFilesIndexes   []FileIndex `json:"latestFilesIndexes"`
}

// Category represents an addon category
type Category struct {
	ID       int    `json:"id"`
	GameID   int    `json:"gameId"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	URL      string `json:"url"`
	IconURL  string `json:"iconUrl"`
	ParentID int    `json:"parentCategoryId"`
}

// Author represents an addon author
type Author struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Logo represents addon logo/thumbnail
type Logo struct {
	ID           int    `json:"id"`
	ModID        int    `json:"modId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnailUrl"`
	URL          string `json:"url"`
}

// File represents an addon file/release
type File struct {
	ID           int       `json:"id"`
	GameID       int       `json:"gameId"`
	ModID        int       `json:"modId"`
	DisplayName  string    `json:"displayName"`
	FileName     string    `json:"fileName"`
	FileDate     time.Time `json:"fileDate"`
	GameVersions []string  `json:"gameVersions"`
}

// FileIndex for quick file lookup
type FileIndex struct {
	GameVersion       string    `json:"gameVersion"`
	FileID            int       `json:"fileId"`
	Filename          string    `json:"filename"`
	ReleaseType       int       `json:"releaseType"`
	GameVersionTypeID int       `json:"gameVersionTypeId"`
	ModLoader         *int      `json:"modLoader"`
}

// GetCategoriesResponse is the response from /v1/categories
type GetCategoriesResponse struct {
	Data []Category `json:"data"`
}
```

**Step 2: Create API client**

Create `internal/curseforge/client.go`:
```go
package curseforge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is a CurseForge API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new CurseForge API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: BaseURL,
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// SearchMods searches for mods with pagination
func (c *Client) SearchMods(ctx context.Context, gameID, index, pageSize int) (*SearchModsResponse, error) {
	query := url.Values{}
	query.Set("gameId", strconv.Itoa(gameID))
	query.Set("index", strconv.Itoa(index))
	query.Set("pageSize", strconv.Itoa(pageSize))
	query.Set("sortField", "2") // 2 = popularity
	query.Set("sortOrder", "desc")

	body, err := c.doRequest(ctx, http.MethodGet, "/v1/mods/search", query)
	if err != nil {
		return nil, fmt.Errorf("search mods: %w", err)
	}

	var result SearchModsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// GetAllWoWAddons fetches all WoW addons with pagination
func (c *Client) GetAllWoWAddons(ctx context.Context) ([]Mod, error) {
	var allMods []Mod
	pageSize := 50
	index := 0

	for {
		slog.Info("fetching addons page", "index", index, "pageSize", pageSize)

		resp, err := c.SearchMods(ctx, GameIDWoW, index, pageSize)
		if err != nil {
			return nil, fmt.Errorf("fetch page at index %d: %w", index, err)
		}

		allMods = append(allMods, resp.Data...)

		slog.Info("fetched page",
			"count", len(resp.Data),
			"total", len(allMods),
			"totalAvailable", resp.Pagination.TotalCount,
		)

		// Check if we've fetched all results
		if len(resp.Data) < pageSize || index+pageSize >= resp.Pagination.TotalCount {
			break
		}

		index += pageSize

		// Small delay to be nice to the API
		time.Sleep(100 * time.Millisecond)
	}

	return allMods, nil
}

// GetCategories fetches all categories for a game
func (c *Client) GetCategories(ctx context.Context, gameID int) ([]Category, error) {
	query := url.Values{}
	query.Set("gameId", strconv.Itoa(gameID))

	body, err := c.doRequest(ctx, http.MethodGet, "/v1/categories", query)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}

	var result GetCategoriesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal categories: %w", err)
	}

	return result.Data, nil
}
```

**Step 3: Verify compilation**

Run:
```bash
go build ./...
```

Expected: No errors

**Step 4: Commit**

```bash
git add internal/curseforge/
git commit -m "feat: implement CurseForge API client"
```

---

## Task 5: Implement Sync Logic

**Files:**
- Create: `internal/sync/sync.go`
- Modify: `cmd/sync/main.go`

**Step 1: Create sync service**

Create `internal/sync/sync.go`:
```go
package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"addon-radar/internal/curseforge"
	"addon-radar/internal/database"
)

// Service handles the sync process
type Service struct {
	db     *database.Queries
	client *curseforge.Client
}

// NewService creates a new sync service
func NewService(pool *pgxpool.Pool, apiKey string) *Service {
	return &Service{
		db:     database.New(pool),
		client: curseforge.NewClient(apiKey),
	}
}

// RunFullSync performs a full sync of all WoW addons
func (s *Service) RunFullSync(ctx context.Context) error {
	startTime := time.Now()
	slog.Info("starting full sync")

	// Fetch all addons from CurseForge
	mods, err := s.client.GetAllWoWAddons(ctx)
	if err != nil {
		return fmt.Errorf("fetch addons: %w", err)
	}

	slog.Info("fetched all addons", "count", len(mods))

	// Sync categories first
	if err := s.syncCategories(ctx); err != nil {
		slog.Warn("failed to sync categories", "error", err)
		// Continue anyway, categories are not critical
	}

	// Upsert each addon and create snapshot
	var successCount, errorCount int
	for _, mod := range mods {
		if err := s.upsertAddon(ctx, mod); err != nil {
			slog.Error("failed to upsert addon", "id", mod.ID, "name", mod.Name, "error", err)
			errorCount++
			continue
		}

		if err := s.createSnapshot(ctx, mod); err != nil {
			slog.Error("failed to create snapshot", "id", mod.ID, "error", err)
			errorCount++
			continue
		}

		successCount++
	}

	duration := time.Since(startTime)
	slog.Info("full sync complete",
		"duration", duration,
		"total", len(mods),
		"success", successCount,
		"errors", errorCount,
	)

	return nil
}

// syncCategories fetches and stores all WoW addon categories
func (s *Service) syncCategories(ctx context.Context) error {
	categories, err := s.client.GetCategories(ctx, curseforge.GameIDWoW)
	if err != nil {
		return fmt.Errorf("fetch categories: %w", err)
	}

	for _, cat := range categories {
		var parentID *int32
		if cat.ParentID > 0 {
			pid := int32(cat.ParentID)
			parentID = &pid
		}

		err := s.db.UpsertCategory(ctx, database.UpsertCategoryParams{
			ID:       int32(cat.ID),
			Name:     cat.Name,
			Slug:     cat.Slug,
			ParentID: parentID,
			IconUrl:  &cat.IconURL,
		})
		if err != nil {
			slog.Warn("failed to upsert category", "id", cat.ID, "error", err)
		}
	}

	slog.Info("synced categories", "count", len(categories))
	return nil
}

// upsertAddon inserts or updates an addon
func (s *Service) upsertAddon(ctx context.Context, mod curseforge.Mod) error {
	// Extract primary author
	var authorName *string
	var authorID *int32
	if len(mod.Authors) > 0 {
		authorName = &mod.Authors[0].Name
		aid := int32(mod.Authors[0].ID)
		authorID = &aid
	}

	// Extract logo URL
	var logoURL *string
	if mod.Logo != nil {
		logoURL = &mod.Logo.ThumbnailURL
	}

	// Extract category IDs
	categoryIDs := make([]int32, len(mod.Categories))
	var primaryCategoryID *int32
	for i, cat := range mod.Categories {
		categoryIDs[i] = int32(cat.ID)
		if i == 0 {
			cid := int32(cat.ID)
			primaryCategoryID = &cid
		}
	}

	// Extract game versions from latest files
	gameVersions := extractGameVersions(mod)

	// Get latest file date
	var latestFileDate *time.Time
	if len(mod.LatestFiles) > 0 {
		latestFileDate = &mod.LatestFiles[0].FileDate
	}

	// Calculate rating as decimal
	var rating *string
	if mod.Rating > 0 {
		r := fmt.Sprintf("%.2f", mod.Rating)
		rating = &r
	}

	return s.db.UpsertAddon(ctx, database.UpsertAddonParams{
		ID:                int32(mod.ID),
		Name:              mod.Name,
		Slug:              mod.Slug,
		Summary:           &mod.Summary,
		AuthorName:        authorName,
		AuthorID:          authorID,
		LogoUrl:           logoURL,
		PrimaryCategoryID: primaryCategoryID,
		Categories:        categoryIDs,
		GameVersions:      gameVersions,
		CreatedAt:         &mod.DateCreated,
		LastUpdatedAt:     &mod.DateModified,
		DownloadCount:     &mod.DownloadCount,
		ThumbsUpCount:     &mod.ThumbsUpCount,
		PopularityRank:    &mod.PopularityRank,
		Rating:            rating,
		LatestFileDate:    latestFileDate,
	})
}

// createSnapshot creates a point-in-time snapshot of addon metrics
func (s *Service) createSnapshot(ctx context.Context, mod curseforge.Mod) error {
	var latestFileDate *time.Time
	if len(mod.LatestFiles) > 0 {
		latestFileDate = &mod.LatestFiles[0].FileDate
	}

	var rating *string
	if mod.Rating > 0 {
		r := fmt.Sprintf("%.2f", mod.Rating)
		rating = &r
	}

	return s.db.CreateSnapshot(ctx, database.CreateSnapshotParams{
		AddonID:        int32(mod.ID),
		DownloadCount:  mod.DownloadCount,
		ThumbsUpCount:  &mod.ThumbsUpCount,
		PopularityRank: &mod.PopularityRank,
		Rating:         rating,
		LatestFileDate: latestFileDate,
	})
}

// extractGameVersions gets unique game versions from mod files
func extractGameVersions(mod curseforge.Mod) []string {
	versionSet := make(map[string]bool)
	for _, file := range mod.LatestFiles {
		for _, v := range file.GameVersions {
			versionSet[v] = true
		}
	}

	versions := make([]string, 0, len(versionSet))
	for v := range versionSet {
		versions = append(versions, v)
	}
	return versions
}
```

**Step 2: Update main.go to run sync**

Update `cmd/sync/main.go`:
```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"addon-radar/internal/config"
	"addon-radar/internal/sync"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("addon-radar sync starting...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("config loaded", "env", cfg.Environment)

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	err = pool.Ping(ctx)
	if err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	slog.Info("database connected successfully")

	// Run sync
	syncService := sync.NewService(pool, cfg.CurseForgeAPIKey)
	if err := syncService.RunFullSync(ctx); err != nil {
		slog.Error("sync failed", "error", err)
		os.Exit(1)
	}

	slog.Info("sync complete")
}
```

**Step 3: Create internal/sync directory and verify**

Run:
```bash
mkdir -p internal/sync
go build ./...
```

Expected: Compiles without errors

**Step 4: Commit**

```bash
git add internal/sync/ cmd/sync/main.go
git commit -m "feat: implement full sync logic for CurseForge addons"
```

---

## Task 6: Local Testing with Docker PostgreSQL

**Files:**
- Create: `scripts/db-setup.sh`
- Create: `.env.example`

**Step 1: Create database setup script**

Create `scripts/db-setup.sh`:
```bash
#!/bin/bash
set -e

# Start PostgreSQL container
docker run -d --name addon-radar-db \
  -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=addon_radar \
  -p 5432:5432 \
  postgres:16 || docker start addon-radar-db

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
sleep 3

# Apply schema
echo "Applying schema..."
docker exec -i addon-radar-db psql -U postgres -d addon_radar < sql/schema.sql

echo "Database ready!"
```

**Step 2: Make script executable**

Run:
```bash
chmod +x scripts/db-setup.sh
```

**Step 3: Create .env.example**

Create `.env.example`:
```bash
# Database connection
DATABASE_URL=postgres://postgres:dev@localhost:5432/addon_radar?sslmode=disable

# CurseForge API key (get from https://console.curseforge.com/)
CURSEFORGE_API_KEY=your-api-key-here

# Environment (development/production)
ENV=development
```

**Step 4: Create local .env file**

Run:
```bash
cp .env.example .env
# Edit .env and add your real CURSEFORGE_API_KEY
```

**Step 5: Setup database**

Run:
```bash
./scripts/db-setup.sh
```

Expected: PostgreSQL starts and schema is applied

**Step 6: Run sync locally**

Run:
```bash
source .env && go run ./cmd/sync
```

Expected: Sync runs, fetches ~7000 addons, creates snapshots

**Step 7: Verify data**

Run:
```bash
docker exec -it addon-radar-db psql -U postgres -d addon_radar -c "SELECT COUNT(*) FROM addons;"
docker exec -it addon-radar-db psql -U postgres -d addon_radar -c "SELECT COUNT(*) FROM snapshots;"
docker exec -it addon-radar-db psql -U postgres -d addon_radar -c "SELECT name, download_count FROM addons ORDER BY download_count DESC LIMIT 5;"
```

Expected: Shows addon counts and top addons by downloads

**Step 8: Commit**

```bash
git add scripts/ .env.example
git commit -m "feat: add local development database setup"
```

---

## Task 7: Deploy to Railway

**Files:**
- Create: `Dockerfile`
- Create: `railway.toml`

**Step 1: Create Dockerfile**

Create `Dockerfile`:
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build sync binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /sync ./cmd/sync

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /sync /app/sync

CMD ["/app/sync"]
```

**Step 2: Create railway.toml**

Create `railway.toml`:
```toml
[build]
builder = "dockerfile"
dockerfilePath = "Dockerfile"

[deploy]
numReplicas = 1
sleepApplication = false
restartPolicyType = "never"
```

**Step 3: Commit deployment files**

```bash
git add Dockerfile railway.toml
git commit -m "feat: add Railway deployment configuration"
```

**Step 4: Push to GitHub**

Run:
```bash
git remote add origin https://github.com/YOUR_USERNAME/addon-radar.git
git push -u origin main
```

**Step 5: Create Railway project**

1. Go to https://railway.app/
2. Click "New Project"
3. Select "Deploy from GitHub repo"
4. Select your addon-radar repository
5. Railway will detect the Dockerfile and start building

**Step 6: Add PostgreSQL**

1. In Railway dashboard, click "+ New"
2. Select "Database" → "Add PostgreSQL"
3. Railway creates a PostgreSQL instance

**Step 7: Connect services**

1. Click on your sync service
2. Go to "Variables"
3. Add variable: `DATABASE_URL` = `${{Postgres.DATABASE_URL}}`
4. Add variable: `CURSEFORGE_API_KEY` = your actual API key
5. Add variable: `ENV` = `production`

**Step 8: Run initial sync**

1. In Railway dashboard, click on sync service
2. Click "Deploy" to trigger a new deployment
3. Check logs to see sync running

**Step 9: Setup cron schedule**

1. In Railway dashboard, click on sync service
2. Go to "Settings"
3. Under "Cron Schedule", add: `0 * * * *` (hourly)
4. This will run the sync every hour

---

## Task 8: Verify Deployment

**Step 1: Check Railway logs**

In Railway dashboard, click on sync service and view logs.

Expected: See sync completing successfully with addon counts.

**Step 2: Verify database has data**

1. Click on PostgreSQL service in Railway
2. Go to "Connect" tab
3. Use the connection string to connect with psql or a GUI
4. Run: `SELECT COUNT(*) FROM addons;`

Expected: ~7000 addons

**Step 3: Wait for second sync**

Wait 1 hour for cron to trigger, then verify snapshots table is growing:
```sql
SELECT COUNT(*) FROM snapshots;
SELECT addon_id, COUNT(*) as snapshot_count
FROM snapshots
GROUP BY addon_id
LIMIT 5;
```

---

## Summary

After completing all tasks you will have:
- ✅ Go project with proper structure
- ✅ PostgreSQL schema with addons, snapshots, categories tables
- ✅ CurseForge API client
- ✅ Full sync job that fetches all WoW addons
- ✅ Local development environment with Docker
- ✅ Production deployment on Railway with hourly cron
- ✅ Data accumulating for trending algorithm

**Next steps after this plan:**
1. Add "hot" addon detection (based on download velocity)
2. Implement hourly hot-only sync
3. Build the web server
4. Implement trending algorithm
