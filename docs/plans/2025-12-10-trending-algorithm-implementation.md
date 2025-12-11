# Trending Algorithm Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the trending algorithm to replace placeholder endpoints with real "Hot Right Now" and "Rising Stars" rankings.

**Architecture:** Create a `trending` package with pure calculation functions, add SQL queries for snapshot aggregation, add a `trending_scores` table for caching computed scores, and wire it into the API handlers.

**Tech Stack:** Go 1.25, PostgreSQL, sqlc, pgx/v5

---

## Overview

The trending algorithm calculates scores for two categories:
- **Hot Right Now**: Established addons (500+ downloads) with high download velocity
- **Rising Stars**: Smaller addons (50-10,000 downloads) gaining traction quickly

Key formulas from design doc:
```
Hot Score = (weighted_velocity * size_multiplier * maintenance_multiplier) / (age_hours + 2)^1.5
Rising Score = (weighted_growth_pct * size_multiplier * maintenance_multiplier) / (age_hours + 2)^1.8
```

---

### Task 1: Add Trending Scores Table

**Files:**
- Modify: `sql/schema.sql`

**Step 1: Add schema for trending scores table**

Add to `sql/schema.sql`:

```sql
-- Trending scores table: cached trending calculations
CREATE TABLE trending_scores (
    addon_id INTEGER PRIMARY KEY REFERENCES addons(id) ON DELETE CASCADE,
    hot_score DECIMAL(20,10) DEFAULT 0,
    rising_score DECIMAL(20,10) DEFAULT 0,
    download_velocity DECIMAL(15,5) DEFAULT 0,
    thumbs_velocity DECIMAL(15,5) DEFAULT 0,
    download_growth_pct DECIMAL(10,5) DEFAULT 0,
    thumbs_growth_pct DECIMAL(10,5) DEFAULT 0,
    size_multiplier DECIMAL(5,4) DEFAULT 1.0,
    maintenance_multiplier DECIMAL(5,4) DEFAULT 1.0,
    first_hot_at TIMESTAMPTZ,
    first_rising_at TIMESTAMPTZ,
    calculated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_trending_hot ON trending_scores(hot_score DESC) WHERE hot_score > 0;
CREATE INDEX idx_trending_rising ON trending_scores(rising_score DESC) WHERE rising_score > 0;
```

**Step 2: Apply migration to database**

Run locally:
```bash
psql $DATABASE_URL -f sql/schema.sql
```

Note: For production, apply this migration manually via Railway's psql or a migration tool.

**Step 3: Commit**

```bash
git add sql/schema.sql
git commit -m "feat: add trending_scores table for caching calculations"
```

---

### Task 2: Add SQL Queries for Snapshot Aggregation

**Files:**
- Modify: `sql/queries.sql`

**Step 1: Add queries for velocity calculations**

Add to `sql/queries.sql`:

```sql
-- name: GetSnapshotStats :one
-- Gets download/thumbs changes for velocity calculation
SELECT
    COALESCE(MAX(download_count) - MIN(download_count), 0) AS download_change,
    COALESCE(MAX(thumbs_up_count) - MIN(thumbs_up_count), 0) AS thumbs_change,
    COUNT(*) AS snapshot_count,
    MIN(download_count) AS min_downloads,
    MAX(download_count) AS max_downloads
FROM snapshots
WHERE addon_id = $1
  AND recorded_at >= NOW() - ($2 || ' hours')::INTERVAL;

-- name: GetDownloadPercentile :one
-- Gets the Nth percentile of total downloads for size multiplier calculation
SELECT PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY download_count) AS percentile_95
FROM addons
WHERE status = 'active' AND download_count > 0;

-- name: GetAddonLatestFileDate :one
-- Gets the latest file date for maintenance multiplier
SELECT latest_file_date FROM addons WHERE id = $1;

-- name: CountRecentFileUpdates :one
-- Counts file updates in last N days (approximated by comparing latest_file_date changes in snapshots)
SELECT COUNT(DISTINCT DATE(latest_file_date))
FROM snapshots
WHERE addon_id = $1
  AND recorded_at >= NOW() - ($2 || ' days')::INTERVAL
  AND latest_file_date IS NOT NULL;

-- name: UpsertTrendingScore :exec
INSERT INTO trending_scores (
    addon_id, hot_score, rising_score,
    download_velocity, thumbs_velocity,
    download_growth_pct, thumbs_growth_pct,
    size_multiplier, maintenance_multiplier,
    first_hot_at, first_rising_at, calculated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
ON CONFLICT (addon_id) DO UPDATE SET
    hot_score = EXCLUDED.hot_score,
    rising_score = EXCLUDED.rising_score,
    download_velocity = EXCLUDED.download_velocity,
    thumbs_velocity = EXCLUDED.thumbs_velocity,
    download_growth_pct = EXCLUDED.download_growth_pct,
    thumbs_growth_pct = EXCLUDED.thumbs_growth_pct,
    size_multiplier = EXCLUDED.size_multiplier,
    maintenance_multiplier = EXCLUDED.maintenance_multiplier,
    first_hot_at = COALESCE(EXCLUDED.first_hot_at, trending_scores.first_hot_at),
    first_rising_at = COALESCE(EXCLUDED.first_rising_at, trending_scores.first_rising_at),
    calculated_at = NOW();

-- name: GetTrendingScore :one
SELECT * FROM trending_scores WHERE addon_id = $1;

-- name: ListHotAddons :many
SELECT a.*, t.hot_score
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 500
  AND t.hot_score > 0
ORDER BY t.hot_score DESC
LIMIT $1;

-- name: ListRisingAddons :many
SELECT a.*, t.rising_score
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 50
  AND a.download_count <= 10000
  AND t.rising_score > 0
  AND a.id NOT IN (
      SELECT addon_id FROM trending_scores
      WHERE hot_score > 0
      ORDER BY hot_score DESC
      LIMIT 20
  )
ORDER BY t.rising_score DESC
LIMIT $1;

-- name: ClearTrendingAgeForDroppedAddons :exec
-- Reset first_hot_at for addons that dropped out of hot list
UPDATE trending_scores
SET first_hot_at = NULL
WHERE addon_id NOT IN (
    SELECT addon_id FROM trending_scores
    WHERE hot_score > 0
    ORDER BY hot_score DESC
    LIMIT 20
);

-- name: ClearRisingAgeForDroppedAddons :exec
-- Reset first_rising_at for addons that dropped out of rising list
UPDATE trending_scores
SET first_rising_at = NULL
WHERE addon_id NOT IN (
    SELECT addon_id FROM trending_scores
    WHERE rising_score > 0
    ORDER BY rising_score DESC
    LIMIT 20
);

-- name: ListAddonsForTrendingCalc :many
-- Get addons with basic info needed for trending calculation
SELECT id, download_count, thumbs_up_count, latest_file_date, created_at
FROM addons
WHERE status = 'active';
```

**Step 2: Regenerate sqlc**

```bash
sqlc generate
```

**Step 3: Commit**

```bash
git add sql/queries.sql internal/database/
git commit -m "feat: add SQL queries for trending score calculations"
```

---

### Task 3: Create Trending Package - Core Types and Multipliers

**Files:**
- Create: `internal/trending/trending.go`
- Create: `internal/trending/trending_test.go`

**Step 1: Write the failing test for size multiplier**

Create `internal/trending/trending_test.go`:

```go
package trending

import (
	"math"
	"testing"
)

func TestCalculateSizeMultiplier(t *testing.T) {
	percentile95 := float64(500000)

	tests := []struct {
		downloads float64
		want      float64
	}{
		{10, 0.18},
		{100, 0.35},
		{1000, 0.53},
		{10000, 0.70},
		{100000, 0.88},
		{500000, 1.0},
		{1000000, 1.0}, // Capped at 1.0
		{0, 0.1},       // Minimum 0.1
	}

	for _, tt := range tests {
		got := CalculateSizeMultiplier(tt.downloads, percentile95)
		if math.Abs(got-tt.want) > 0.02 { // Allow 2% tolerance
			t.Errorf("CalculateSizeMultiplier(%v) = %v, want %v", tt.downloads, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/trending/... -v
```

Expected: FAIL with "package trending is not in std"

**Step 3: Write minimal implementation**

Create `internal/trending/trending.go`:

```go
package trending

import "math"

// CalculateSizeMultiplier returns a value between 0.1 and 1.0
// based on logarithmic scaling of downloads against the 95th percentile.
func CalculateSizeMultiplier(downloads, percentile95 float64) float64 {
	if percentile95 <= 0 {
		return 1.0
	}
	if downloads <= 0 {
		return 0.1
	}

	multiplier := math.Log10(downloads+1) / math.Log10(percentile95+1)
	return clamp(multiplier, 0.1, 1.0)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/trending/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/trending/
git commit -m "feat: add size multiplier calculation"
```

---

### Task 4: Add Maintenance Multiplier

**Files:**
- Modify: `internal/trending/trending.go`
- Modify: `internal/trending/trending_test.go`

**Step 1: Write the failing test**

Add to `internal/trending/trending_test.go`:

```go
func TestCalculateMaintenanceMultiplier(t *testing.T) {
	tests := []struct {
		updatesIn90Days int
		want            float64
	}{
		{12, 1.15}, // ~7 days avg = very active
		{6, 1.15},  // 15 days avg = very active
		{4, 1.10},  // ~22 days avg = regular
		{3, 1.10},  // 30 days avg = regular
		{2, 1.05},  // 45 days avg = occasional
		{1, 1.00},  // 90 days avg = baseline
		{0, 0.95},  // No updates = stale
	}

	for _, tt := range tests {
		got := CalculateMaintenanceMultiplier(tt.updatesIn90Days)
		if got != tt.want {
			t.Errorf("CalculateMaintenanceMultiplier(%d) = %v, want %v", tt.updatesIn90Days, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/trending/... -v -run TestCalculateMaintenanceMultiplier
```

Expected: FAIL

**Step 3: Write minimal implementation**

Add to `internal/trending/trending.go`:

```go
// CalculateMaintenanceMultiplier returns a multiplier (0.95-1.15)
// based on update frequency in the last 90 days.
func CalculateMaintenanceMultiplier(updatesIn90Days int) float64 {
	if updatesIn90Days == 0 {
		return 0.95 // Stale/abandoned
	}

	avgDaysBetweenUpdates := 90.0 / float64(updatesIn90Days)

	switch {
	case avgDaysBetweenUpdates <= 14:
		return 1.15 // Very active
	case avgDaysBetweenUpdates <= 30:
		return 1.10 // Regular
	case avgDaysBetweenUpdates <= 60:
		return 1.05 // Occasional
	default:
		return 1.00 // Baseline
	}
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/trending/... -v -run TestCalculateMaintenanceMultiplier
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/trending/
git commit -m "feat: add maintenance multiplier calculation"
```

---

### Task 5: Add Velocity Calculation with Adaptive Windows

**Files:**
- Modify: `internal/trending/trending.go`
- Modify: `internal/trending/trending_test.go`

**Step 1: Write the failing test**

Add to `internal/trending/trending_test.go`:

```go
func TestCalculateVelocity(t *testing.T) {
	tests := []struct {
		name           string
		velocity24h    float64
		velocity7d     float64
		dataPoints24h  int
		change24h      int64
		wantConfident  bool
		wantVelocity   float64
	}{
		{
			name:          "confident 24h - enough data and change",
			velocity24h:   100.0,
			velocity7d:    50.0,
			dataPoints24h: 10,
			change24h:     500,
			wantConfident: true,
			wantVelocity:  90.0, // 0.8 * 100 + 0.2 * 50
		},
		{
			name:          "not confident - few data points",
			velocity24h:   100.0,
			velocity7d:    50.0,
			dataPoints24h: 3,
			change24h:     500,
			wantConfident: false,
			wantVelocity:  65.0, // 0.3 * 100 + 0.7 * 50
		},
		{
			name:          "not confident - small change",
			velocity24h:   100.0,
			velocity7d:    50.0,
			dataPoints24h: 10,
			change24h:     5,
			wantConfident: false,
			wantVelocity:  65.0, // 0.3 * 100 + 0.7 * 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confident, velocity := CalculateVelocity(tt.velocity24h, tt.velocity7d, tt.dataPoints24h, tt.change24h)
			if confident != tt.wantConfident {
				t.Errorf("confident = %v, want %v", confident, tt.wantConfident)
			}
			if math.Abs(velocity-tt.wantVelocity) > 0.01 {
				t.Errorf("velocity = %v, want %v", velocity, tt.wantVelocity)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/trending/... -v -run TestCalculateVelocity
```

Expected: FAIL

**Step 3: Write minimal implementation**

Add to `internal/trending/trending.go`:

```go
// CalculateVelocity uses confidence-based adaptive windows.
// Returns (isConfident24h, blendedVelocity).
func CalculateVelocity(velocity24h, velocity7d float64, dataPoints24h int, change24h int64) (bool, float64) {
	// Confident if we have enough data points AND meaningful change
	confident := dataPoints24h >= 5 && change24h >= 10

	if confident {
		// Weight toward fresh data
		return true, (0.8 * velocity24h) + (0.2 * velocity7d)
	}
	// Fall back to longer window
	return false, (0.3 * velocity24h) + (0.7 * velocity7d)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/trending/... -v -run TestCalculateVelocity
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/trending/
git commit -m "feat: add velocity calculation with adaptive windows"
```

---

### Task 6: Add Weighted Signal Blend

**Files:**
- Modify: `internal/trending/trending.go`
- Modify: `internal/trending/trending_test.go`

**Step 1: Write the failing test**

Add to `internal/trending/trending_test.go`:

```go
func TestCalculateWeightedSignal(t *testing.T) {
	tests := []struct {
		name           string
		downloadSignal float64
		thumbsSignal   float64
		hasUpdate      bool
		want           float64
	}{
		{
			name:           "all signals with update",
			downloadSignal: 100.0,
			thumbsSignal:   50.0,
			hasUpdate:      true,
			want:           80.0 + 1.0, // 0.7*100 + 0.2*50 + 0.1*10
		},
		{
			name:           "all signals without update",
			downloadSignal: 100.0,
			thumbsSignal:   50.0,
			hasUpdate:      false,
			want:           80.0, // 0.7*100 + 0.2*50 + 0
		},
		{
			name:           "zero values",
			downloadSignal: 0,
			thumbsSignal:   0,
			hasUpdate:      false,
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateWeightedSignal(tt.downloadSignal, tt.thumbsSignal, tt.hasUpdate)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("CalculateWeightedSignal() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/trending/... -v -run TestCalculateWeightedSignal
```

Expected: FAIL

**Step 3: Write minimal implementation**

Add to `internal/trending/trending.go`:

```go
const (
	DownloadWeight = 0.7
	ThumbsWeight   = 0.2
	UpdateWeight   = 0.1
	UpdateBoost    = 10.0 // Boost value when addon has recent update
)

// CalculateWeightedSignal blends download, thumbs, and update signals.
// Signal blend: 70% downloads + 20% thumbs + 10% update boost.
func CalculateWeightedSignal(downloadSignal, thumbsSignal float64, hasRecentUpdate bool) float64 {
	updateBoost := 0.0
	if hasRecentUpdate {
		updateBoost = UpdateBoost
	}
	return (DownloadWeight * downloadSignal) + (ThumbsWeight * thumbsSignal) + (UpdateWeight * updateBoost)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/trending/... -v -run TestCalculateWeightedSignal
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/trending/
git commit -m "feat: add weighted signal blend calculation"
```

---

### Task 7: Add Final Score Calculations (Hot and Rising)

**Files:**
- Modify: `internal/trending/trending.go`
- Modify: `internal/trending/trending_test.go`

**Step 1: Write the failing tests**

Add to `internal/trending/trending_test.go`:

```go
func TestCalculateHotScore(t *testing.T) {
	tests := []struct {
		name                  string
		weightedVelocity      float64
		sizeMultiplier        float64
		maintenanceMultiplier float64
		ageHours              float64
		want                  float64
	}{
		{
			name:                  "new addon",
			weightedVelocity:      100.0,
			sizeMultiplier:        0.5,
			maintenanceMultiplier: 1.1,
			ageHours:              0,
			want:                  19.45, // (100 * 0.5 * 1.1) / (0+2)^1.5
		},
		{
			name:                  "24h old addon",
			weightedVelocity:      100.0,
			sizeMultiplier:        0.5,
			maintenanceMultiplier: 1.1,
			ageHours:              24,
			want:                  0.41, // (100 * 0.5 * 1.1) / (24+2)^1.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateHotScore(tt.weightedVelocity, tt.sizeMultiplier, tt.maintenanceMultiplier, tt.ageHours)
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("CalculateHotScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateRisingScore(t *testing.T) {
	tests := []struct {
		name                  string
		weightedGrowthPct     float64
		sizeMultiplier        float64
		maintenanceMultiplier float64
		ageHours              float64
		want                  float64
	}{
		{
			name:                  "new addon",
			weightedGrowthPct:     50.0,
			sizeMultiplier:        0.3,
			maintenanceMultiplier: 1.0,
			ageHours:              0,
			want:                  4.42, // (50 * 0.3 * 1.0) / (0+2)^1.8
		},
		{
			name:                  "48h old addon",
			weightedGrowthPct:     50.0,
			sizeMultiplier:        0.3,
			maintenanceMultiplier: 1.0,
			ageHours:              48,
			want:                  0.016, // (50 * 0.3 * 1.0) / (48+2)^1.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRisingScore(tt.weightedGrowthPct, tt.sizeMultiplier, tt.maintenanceMultiplier, tt.ageHours)
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("CalculateRisingScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/trending/... -v -run "TestCalculateHotScore|TestCalculateRisingScore"
```

Expected: FAIL

**Step 3: Write minimal implementation**

Add to `internal/trending/trending.go`:

```go
const (
	HotGravity    = 1.5
	RisingGravity = 1.8
	AgeOffset     = 2.0 // Prevents division by zero and smooths early decay
)

// CalculateHotScore computes the "Hot Right Now" score.
// Formula: (weighted_velocity * size_multiplier * maintenance_multiplier) / (age_hours + 2)^1.5
func CalculateHotScore(weightedVelocity, sizeMultiplier, maintenanceMultiplier, ageHours float64) float64 {
	numerator := weightedVelocity * sizeMultiplier * maintenanceMultiplier
	denominator := math.Pow(ageHours+AgeOffset, HotGravity)
	return numerator / denominator
}

// CalculateRisingScore computes the "Rising Stars" score.
// Formula: (weighted_growth_pct * size_multiplier * maintenance_multiplier) / (age_hours + 2)^1.8
func CalculateRisingScore(weightedGrowthPct, sizeMultiplier, maintenanceMultiplier, ageHours float64) float64 {
	numerator := weightedGrowthPct * sizeMultiplier * maintenanceMultiplier
	denominator := math.Pow(ageHours+AgeOffset, RisingGravity)
	return numerator / denominator
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/trending/... -v -run "TestCalculateHotScore|TestCalculateRisingScore"
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/trending/
git commit -m "feat: add hot and rising score calculations"
```

---

### Task 8: Add Trending Calculator Service

**Files:**
- Create: `internal/trending/calculator.go`

**Step 1: Create the calculator service**

Create `internal/trending/calculator.go`:

```go
package trending

import (
	"context"
	"log/slog"
	"math"
	"time"

	"addon-radar/internal/database"

	"github.com/jackc/pgx/v5/pgtype"
)

// Calculator computes and stores trending scores for all addons.
type Calculator struct {
	db *database.Queries
}

// NewCalculator creates a new trending calculator.
func NewCalculator(db *database.Queries) *Calculator {
	return &Calculator{db: db}
}

// CalculateAll recalculates trending scores for all active addons.
func (c *Calculator) CalculateAll(ctx context.Context) error {
	slog.Info("starting trending calculation")
	start := time.Now()

	// Get 95th percentile for size multiplier
	percentile, err := c.db.GetDownloadPercentile(ctx)
	if err != nil {
		return err
	}
	var percentile95 float64
	if percentile.Valid {
		percentile95, _ = percentile.Float64.Float64Value()
	}
	if percentile95 <= 0 {
		percentile95 = 500000 // Default fallback
	}
	slog.Info("using percentile", "percentile_95", percentile95)

	// Get all addons for calculation
	addons, err := c.db.ListAddonsForTrendingCalc(ctx)
	if err != nil {
		return err
	}
	slog.Info("calculating trending for addons", "count", len(addons))

	for _, addon := range addons {
		if err := c.calculateAddon(ctx, addon, percentile95); err != nil {
			slog.Warn("failed to calculate trending for addon", "addon_id", addon.ID, "error", err)
			continue
		}
	}

	// Clear trending age for addons that dropped off
	if err := c.db.ClearTrendingAgeForDroppedAddons(ctx); err != nil {
		slog.Warn("failed to clear hot age", "error", err)
	}
	if err := c.db.ClearRisingAgeForDroppedAddons(ctx); err != nil {
		slog.Warn("failed to clear rising age", "error", err)
	}

	slog.Info("trending calculation complete", "duration", time.Since(start))
	return nil
}

func (c *Calculator) calculateAddon(ctx context.Context, addon database.ListAddonsForTrendingCalcRow, percentile95 float64) error {
	downloads := float64(addon.DownloadCount.Int64)

	// Get snapshot stats for 24h window
	stats24h, err := c.db.GetSnapshotStats(ctx, database.GetSnapshotStatsParams{
		AddonID: addon.ID,
		Column2: "24",
	})
	if err != nil {
		return err
	}

	// Get snapshot stats for 7d window
	stats7d, err := c.db.GetSnapshotStats(ctx, database.GetSnapshotStatsParams{
		AddonID: addon.ID,
		Column2: "168",
	})
	if err != nil {
		return err
	}

	// Calculate velocities (downloads per hour)
	velocity24h := float64(stats24h.DownloadChange) / 24.0
	velocity7d := float64(stats7d.DownloadChange) / 168.0

	// Thumbs velocities
	thumbsVel24h := float64(stats24h.ThumbsChange) / 24.0
	thumbsVel7d := float64(stats7d.ThumbsChange) / 168.0

	// Apply adaptive windows
	_, downloadVelocity := CalculateVelocity(velocity24h, velocity7d, int(stats24h.SnapshotCount), stats24h.DownloadChange)
	_, thumbsVelocity := CalculateVelocity(thumbsVel24h, thumbsVel7d, int(stats24h.SnapshotCount), int64(stats24h.ThumbsChange))

	// Calculate growth percentages
	var downloadGrowthPct, thumbsGrowthPct float64
	if stats7d.MinDownloads > 0 {
		downloadGrowthPct = (float64(stats7d.DownloadChange) / float64(stats7d.MinDownloads)) * 100
	}
	thumbsBase := addon.ThumbsUpCount.Int32 - int32(stats7d.ThumbsChange)
	if thumbsBase > 0 {
		thumbsGrowthPct = (float64(stats7d.ThumbsChange) / float64(thumbsBase)) * 100
	}

	// Size multiplier
	sizeMultiplier := CalculateSizeMultiplier(downloads, percentile95)

	// Maintenance multiplier (count recent updates)
	updateCount, err := c.db.CountRecentFileUpdates(ctx, database.CountRecentFileUpdatesParams{
		AddonID: addon.ID,
		Column2: "90",
	})
	if err != nil {
		updateCount = 0
	}
	maintenanceMultiplier := CalculateMaintenanceMultiplier(int(updateCount))

	// Check for recent update (within 7 days)
	hasRecentUpdate := false
	if addon.LatestFileDate.Valid {
		hasRecentUpdate = time.Since(addon.LatestFileDate.Time) < 7*24*time.Hour
	}

	// Calculate weighted signals
	weightedVelocity := CalculateWeightedSignal(downloadVelocity, thumbsVelocity, hasRecentUpdate)
	weightedGrowthPct := CalculateWeightedSignal(downloadGrowthPct, thumbsGrowthPct, hasRecentUpdate)

	// Get existing trending score for age calculation
	existingScore, _ := c.db.GetTrendingScore(ctx, addon.ID)

	// Calculate age for hot score
	var hotAgeHours float64
	var firstHotAt pgtype.Timestamptz
	if downloads >= 500 && weightedVelocity > 0 {
		if existingScore.FirstHotAt.Valid {
			hotAgeHours = time.Since(existingScore.FirstHotAt.Time).Hours()
			firstHotAt = existingScore.FirstHotAt
		} else {
			hotAgeHours = 0
			firstHotAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}
	}

	// Calculate age for rising score
	var risingAgeHours float64
	var firstRisingAt pgtype.Timestamptz
	if downloads >= 50 && downloads <= 10000 && weightedGrowthPct > 0 {
		if existingScore.FirstRisingAt.Valid {
			risingAgeHours = time.Since(existingScore.FirstRisingAt.Time).Hours()
			firstRisingAt = existingScore.FirstRisingAt
		} else {
			risingAgeHours = 0
			firstRisingAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}
	}

	// Calculate final scores
	var hotScore, risingScore float64
	if downloads >= 500 && weightedVelocity > 0 {
		hotScore = CalculateHotScore(weightedVelocity, sizeMultiplier, maintenanceMultiplier, hotAgeHours)
	}
	if downloads >= 50 && downloads <= 10000 && weightedGrowthPct > 0 {
		risingScore = CalculateRisingScore(weightedGrowthPct, sizeMultiplier, maintenanceMultiplier, risingAgeHours)
	}

	// Convert to pgtype.Numeric for storage
	toNumeric := func(v float64) pgtype.Numeric {
		var n pgtype.Numeric
		n.Scan(v)
		return n
	}

	// Upsert trending score
	return c.db.UpsertTrendingScore(ctx, database.UpsertTrendingScoreParams{
		AddonID:               addon.ID,
		HotScore:              toNumeric(hotScore),
		RisingScore:           toNumeric(risingScore),
		DownloadVelocity:      toNumeric(downloadVelocity),
		ThumbsVelocity:        toNumeric(thumbsVelocity),
		DownloadGrowthPct:     toNumeric(downloadGrowthPct),
		ThumbsGrowthPct:       toNumeric(thumbsGrowthPct),
		SizeMultiplier:        toNumeric(sizeMultiplier),
		MaintenanceMultiplier: toNumeric(maintenanceMultiplier),
		FirstHotAt:            firstHotAt,
		FirstRisingAt:         firstRisingAt,
	})
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/trending/...
```

**Step 3: Commit**

```bash
git add internal/trending/
git commit -m "feat: add trending calculator service"
```

---

### Task 9: Update API Handlers to Use Real Trending Data

**Files:**
- Modify: `internal/api/handlers.go`

**Step 1: Update handleTrendingHot**

Replace `handleTrendingHot` in `internal/api/handlers.go`:

```go
type TrendingAddonResponse struct {
	AddonResponse
	Score float64 `json:"score"`
}

func (s *Server) handleTrendingHot(c *gin.Context) {
	ctx := c.Request.Context()

	addons, err := s.db.ListHotAddons(ctx, 20)
	if err != nil {
		slog.Error("failed to get hot addons", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]TrendingAddonResponse, len(addons))
	for i, a := range addons {
		response[i] = TrendingAddonResponse{
			AddonResponse: addonToResponse(database.Addon{
				ID:             a.ID,
				Name:           a.Name,
				Slug:           a.Slug,
				Summary:        a.Summary,
				AuthorName:     a.AuthorName,
				LogoUrl:        a.LogoUrl,
				DownloadCount:  a.DownloadCount,
				ThumbsUpCount:  a.ThumbsUpCount,
				PopularityRank: a.PopularityRank,
				GameVersions:   a.GameVersions,
				LastUpdatedAt:  a.LastUpdatedAt,
			}),
		}
		if a.HotScore.Valid {
			response[i].Score, _ = a.HotScore.Float64Value()
		}
	}

	respondWithData(c, response)
}
```

**Step 2: Update handleTrendingRising**

Replace `handleTrendingRising` in `internal/api/handlers.go`:

```go
func (s *Server) handleTrendingRising(c *gin.Context) {
	ctx := c.Request.Context()

	addons, err := s.db.ListRisingAddons(ctx, 20)
	if err != nil {
		slog.Error("failed to get rising addons", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]TrendingAddonResponse, len(addons))
	for i, a := range addons {
		response[i] = TrendingAddonResponse{
			AddonResponse: addonToResponse(database.Addon{
				ID:             a.ID,
				Name:           a.Name,
				Slug:           a.Slug,
				Summary:        a.Summary,
				AuthorName:     a.AuthorName,
				LogoUrl:        a.LogoUrl,
				DownloadCount:  a.DownloadCount,
				ThumbsUpCount:  a.ThumbsUpCount,
				PopularityRank: a.PopularityRank,
				GameVersions:   a.GameVersions,
				LastUpdatedAt:  a.LastUpdatedAt,
			}),
		}
		if a.RisingScore.Valid {
			response[i].Score, _ = a.RisingScore.Float64Value()
		}
	}

	respondWithData(c, response)
}
```

**Step 3: Verify it compiles**

```bash
go build ./cmd/web/...
```

**Step 4: Commit**

```bash
git add internal/api/handlers.go
git commit -m "feat: update trending handlers to use real calculated scores"
```

---

### Task 10: Add Trending Calculation to Sync Job

**Files:**
- Modify: `cmd/sync/main.go`

**Step 1: Import trending package and run calculation after sync**

Update `cmd/sync/main.go` to add trending calculation after the sync completes:

```go
// Add import
import "addon-radar/internal/trending"

// After sync.Run() succeeds, add:
slog.Info("starting trending calculation")
calculator := trending.NewCalculator(database.New(pool))
if err := calculator.CalculateAll(ctx); err != nil {
    slog.Error("trending calculation failed", "error", err)
    // Don't exit - sync succeeded, trending is secondary
}
```

**Step 2: Verify it compiles**

```bash
go build ./cmd/sync/...
```

**Step 3: Commit**

```bash
git add cmd/sync/main.go
git commit -m "feat: run trending calculation after each sync"
```

---

### Task 11: Test End-to-End Locally

**Files:** None (verification only)

**Step 1: Apply migration to local database**

```bash
psql $DATABASE_URL -c "
CREATE TABLE IF NOT EXISTS trending_scores (
    addon_id INTEGER PRIMARY KEY REFERENCES addons(id) ON DELETE CASCADE,
    hot_score DECIMAL(20,10) DEFAULT 0,
    rising_score DECIMAL(20,10) DEFAULT 0,
    download_velocity DECIMAL(15,5) DEFAULT 0,
    thumbs_velocity DECIMAL(15,5) DEFAULT 0,
    download_growth_pct DECIMAL(10,5) DEFAULT 0,
    thumbs_growth_pct DECIMAL(10,5) DEFAULT 0,
    size_multiplier DECIMAL(5,4) DEFAULT 1.0,
    maintenance_multiplier DECIMAL(5,4) DEFAULT 1.0,
    first_hot_at TIMESTAMPTZ,
    first_rising_at TIMESTAMPTZ,
    calculated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_trending_hot ON trending_scores(hot_score DESC) WHERE hot_score > 0;
CREATE INDEX IF NOT EXISTS idx_trending_rising ON trending_scores(rising_score DESC) WHERE rising_score > 0;
"
```

**Step 2: Run tests**

```bash
go test ./internal/trending/... -v
```

Expected: All tests pass

**Step 3: Run sync job to populate trending scores**

```bash
CURSEFORGE_API_KEY=your_key DATABASE_URL=your_url go run ./cmd/sync
```

**Step 4: Start API server and test endpoints**

```bash
DATABASE_URL=your_url go run ./cmd/web
```

Test in another terminal:
```bash
curl http://localhost:8080/api/v1/trending/hot | jq '.[0]'
curl http://localhost:8080/api/v1/trending/rising | jq '.[0]'
```

Expected: Both return addons with `score` field populated.

---

### Task 12: Deploy to Railway

**Files:** None (deployment only)

**Step 1: Apply migration to production database**

Connect to Railway's PostgreSQL and run the migration SQL from Task 1.

**Step 2: Push code**

```bash
git push origin main
```

**Step 3: Verify deployment**

Wait for Railway to redeploy both services, then test:
```bash
curl https://addon-radar-api-production.up.railway.app/api/v1/trending/hot | jq '.[0]'
```

Expected: Trending data with real scores.

**Step 4: Trigger sync to populate initial scores**

Either wait for the hourly cron or manually trigger the sync service in Railway.

---

### Task 13: Update Documentation

**Files:**
- Modify: `TODO.md`
- Modify: `PLAN.md`
- Modify: `docs/plans/2025-12-08-trending-algorithm-design.md`

**Step 1: Update TODO.md**

Mark trending algorithm tasks as complete.

**Step 2: Update PLAN.md**

Change Phase 3 status from "Next" to "Complete".

**Step 3: Update design doc status**

Change status from "TO IMPLEMENT" to "IMPLEMENTED".

**Step 4: Commit**

```bash
git add TODO.md PLAN.md docs/plans/
git commit -m "docs: mark trending algorithm as implemented"
git push origin main
```

---

## Summary

This plan implements the trending algorithm in 13 tasks:

1. **Database schema** - Add `trending_scores` table
2. **SQL queries** - Add aggregation and list queries
3. **Size multiplier** - Logarithmic scaling function
4. **Maintenance multiplier** - Update frequency reward
5. **Velocity calculation** - Adaptive windows
6. **Weighted signal** - Signal blend function
7. **Score calculations** - Hot and Rising formulas
8. **Calculator service** - Orchestrates all calculations
9. **API handlers** - Use real trending data
10. **Sync integration** - Run calculation after sync
11. **Local testing** - End-to-end verification
12. **Deployment** - Push to Railway
13. **Documentation** - Update project docs

---

Plan complete and saved to `docs/plans/2025-12-10-trending-algorithm-implementation.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
