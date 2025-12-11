# Trending Calculation Optimization Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce trending calculation time from ~10+ minutes to under 30 seconds by eliminating per-addon database queries.

**Architecture:** Replace N+1 query pattern (5 queries × 12,433 addons = 62,165 queries) with bulk SQL operations: one query to compute all snapshot stats, one to load existing scores, and batch upserts.

**Tech Stack:** Go 1.25, PostgreSQL, sqlc, pgx/v5 batch operations

---

## Current Problem

Each addon requires 5 database round-trips:
1. `GetSnapshotStats` (24h window)
2. `GetSnapshotStats` (7d window)
3. `CountRecentFileUpdates`
4. `GetTrendingScore`
5. `UpsertTrendingScore`

With 12,433 addons over network = **62,165 queries** = extremely slow.

---

### Task 1: Add Bulk Snapshot Stats Query

**Files:**
- Modify: `sql/queries.sql`

**Step 1: Add bulk query**

Add to `sql/queries.sql`:

```sql
-- name: GetAllSnapshotStats :many
-- Bulk fetch snapshot stats for all addons in both time windows
WITH stats_24h AS (
    SELECT
        addon_id,
        COALESCE(MAX(download_count) - MIN(download_count), 0)::bigint AS download_change,
        COALESCE(MAX(thumbs_up_count) - MIN(thumbs_up_count), 0)::int AS thumbs_change,
        COUNT(*)::int AS snapshot_count,
        MIN(download_count)::bigint AS min_downloads
    FROM snapshots
    WHERE recorded_at >= NOW() - INTERVAL '24 hours'
    GROUP BY addon_id
),
stats_7d AS (
    SELECT
        addon_id,
        COALESCE(MAX(download_count) - MIN(download_count), 0)::bigint AS download_change,
        COALESCE(MAX(thumbs_up_count) - MIN(thumbs_up_count), 0)::int AS thumbs_change,
        MIN(download_count)::bigint AS min_downloads
    FROM snapshots
    WHERE recorded_at >= NOW() - INTERVAL '7 days'
    GROUP BY addon_id
)
SELECT
    a.id AS addon_id,
    a.download_count,
    a.thumbs_up_count,
    a.latest_file_date,
    a.created_at,
    COALESCE(s24.download_change, 0) AS download_change_24h,
    COALESCE(s24.thumbs_change, 0) AS thumbs_change_24h,
    COALESCE(s24.snapshot_count, 0) AS snapshot_count_24h,
    COALESCE(s7.download_change, 0) AS download_change_7d,
    COALESCE(s7.thumbs_change, 0) AS thumbs_change_7d,
    COALESCE(s7.min_downloads, a.download_count) AS min_downloads_7d
FROM addons a
LEFT JOIN stats_24h s24 ON a.id = s24.addon_id
LEFT JOIN stats_7d s7 ON a.id = s7.addon_id
WHERE a.status = 'active';

-- name: GetAllTrendingScores :many
-- Bulk fetch all existing trending scores
SELECT addon_id, first_hot_at, first_rising_at
FROM trending_scores;

-- name: CountAllRecentFileUpdates :many
-- Bulk count file updates for all addons
SELECT
    addon_id,
    COUNT(DISTINCT DATE(latest_file_date))::int AS update_count
FROM snapshots
WHERE recorded_at >= NOW() - INTERVAL '90 days'
  AND latest_file_date IS NOT NULL
GROUP BY addon_id;
```

**Step 2: Regenerate sqlc**

```bash
/Users/tomas/go/bin/sqlc generate
```

**Step 3: Verify compilation**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add sql/queries.sql internal/database/
git commit -m "feat: add bulk queries for trending calculation"
```

---

### Task 2: Create Optimized Calculator

**Files:**
- Create: `internal/trending/calculator_fast.go`

**Step 1: Create the optimized calculator**

Create `internal/trending/calculator_fast.go`:

```go
package trending

import (
	"context"
	"log/slog"
	"time"

	"addon-radar/internal/database"

	"github.com/jackc/pgx/v5/pgtype"
)

// CalculateAllFast recalculates trending scores using bulk queries.
// Much faster than CalculateAll by eliminating per-addon queries.
func (c *Calculator) CalculateAllFast(ctx context.Context) error {
	slog.Info("starting fast trending calculation")
	start := time.Now()

	// Step 1: Get 95th percentile
	percentile95, err := c.db.GetDownloadPercentile(ctx)
	if err != nil {
		return err
	}
	if percentile95 <= 0 {
		percentile95 = 500000
	}
	slog.Info("percentile", "p95", percentile95)

	// Step 2: Bulk fetch all snapshot stats (1 query instead of 2N)
	allStats, err := c.db.GetAllSnapshotStats(ctx)
	if err != nil {
		return err
	}
	slog.Info("loaded snapshot stats", "count", len(allStats))

	// Step 3: Bulk fetch existing trending scores (1 query instead of N)
	existingScores, err := c.db.GetAllTrendingScores(ctx)
	if err != nil {
		return err
	}
	scoreMap := make(map[int32]database.GetAllTrendingScoresRow)
	for _, s := range existingScores {
		scoreMap[s.AddonID] = s
	}
	slog.Info("loaded existing scores", "count", len(existingScores))

	// Step 4: Bulk fetch update counts (1 query instead of N)
	updateCounts, err := c.db.CountAllRecentFileUpdates(ctx)
	if err != nil {
		return err
	}
	updateMap := make(map[int32]int32)
	for _, u := range updateCounts {
		updateMap[u.AddonID] = u.UpdateCount
	}
	slog.Info("loaded update counts", "count", len(updateCounts))

	// Step 5: Calculate and upsert scores
	processed := 0
	for _, stat := range allStats {
		if err := c.calculateAndUpsert(ctx, stat, percentile95, scoreMap, updateMap); err != nil {
			slog.Warn("failed addon", "id", stat.AddonID, "err", err)
			continue
		}
		processed++
		if processed%1000 == 0 {
			slog.Info("progress", "processed", processed, "total", len(allStats))
		}
	}

	// Step 6: Clear ages for dropped addons
	if err := c.db.ClearTrendingAgeForDroppedAddons(ctx); err != nil {
		slog.Warn("clear hot age failed", "err", err)
	}
	if err := c.db.ClearRisingAgeForDroppedAddons(ctx); err != nil {
		slog.Warn("clear rising age failed", "err", err)
	}

	slog.Info("fast trending complete", "duration", time.Since(start), "processed", processed)
	return nil
}

func (c *Calculator) calculateAndUpsert(
	ctx context.Context,
	stat database.GetAllSnapshotStatsRow,
	percentile95 float64,
	scoreMap map[int32]database.GetAllTrendingScoresRow,
	updateMap map[int32]int32,
) error {
	// Extract downloads
	var downloads float64
	if stat.DownloadCount.Valid {
		downloads = float64(stat.DownloadCount.Int64)
	}

	// Get values from bulk stats
	downloadChange24h := stat.DownloadChange24h
	thumbsChange24h := int64(stat.ThumbsChange24h)
	downloadChange7d := stat.DownloadChange7d
	thumbsChange7d := int64(stat.ThumbsChange7d)
	minDownloads := stat.MinDownloads7d
	snapshotCount24h := stat.SnapshotCount24h

	// Calculate velocities
	velocity24h := float64(downloadChange24h) / 24.0
	velocity7d := float64(downloadChange7d) / 168.0
	thumbsVel24h := float64(thumbsChange24h) / 24.0
	thumbsVel7d := float64(thumbsChange7d) / 168.0

	// Adaptive windows
	_, downloadVelocity := CalculateVelocity(velocity24h, velocity7d, int(snapshotCount24h), downloadChange24h)
	_, thumbsVelocity := CalculateVelocity(thumbsVel24h, thumbsVel7d, int(snapshotCount24h), thumbsChange24h)

	// Growth percentages
	var downloadGrowthPct, thumbsGrowthPct float64
	if minDownloads > 0 {
		downloadGrowthPct = (float64(downloadChange7d) / float64(minDownloads)) * 100
	}

	var thumbsBase int32
	if stat.ThumbsUpCount.Valid {
		thumbsBase = stat.ThumbsUpCount.Int32 - int32(thumbsChange7d)
	}
	if thumbsBase > 0 {
		thumbsGrowthPct = (float64(thumbsChange7d) / float64(thumbsBase)) * 100
	}

	// Multipliers
	sizeMultiplier := CalculateSizeMultiplier(downloads, percentile95)
	updateCount := updateMap[stat.AddonID]
	maintenanceMultiplier := CalculateMaintenanceMultiplier(int(updateCount))

	// Recent update check
	hasRecentUpdate := false
	if stat.LatestFileDate.Valid {
		hasRecentUpdate = time.Since(stat.LatestFileDate.Time) < 7*24*time.Hour
	}

	// Weighted signals
	weightedVelocity := CalculateWeightedSignal(downloadVelocity, thumbsVelocity, hasRecentUpdate)
	weightedGrowthPct := CalculateWeightedSignal(downloadGrowthPct, thumbsGrowthPct, hasRecentUpdate)

	// Age calculation
	existing := scoreMap[stat.AddonID]

	var hotAgeHours float64
	var firstHotAt pgtype.Timestamptz
	if downloads >= 500 && weightedVelocity > 0 {
		if existing.FirstHotAt.Valid {
			hotAgeHours = time.Since(existing.FirstHotAt.Time).Hours()
			firstHotAt = existing.FirstHotAt
		} else {
			firstHotAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}
	}

	var risingAgeHours float64
	var firstRisingAt pgtype.Timestamptz
	if downloads >= 50 && downloads <= 10000 && weightedGrowthPct > 0 {
		if existing.FirstRisingAt.Valid {
			risingAgeHours = time.Since(existing.FirstRisingAt.Time).Hours()
			firstRisingAt = existing.FirstRisingAt
		} else {
			firstRisingAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}
	}

	// Final scores
	var hotScore, risingScore float64
	if downloads >= 500 && weightedVelocity > 0 {
		hotScore = CalculateHotScore(weightedVelocity, sizeMultiplier, maintenanceMultiplier, hotAgeHours)
	}
	if downloads >= 50 && downloads <= 10000 && weightedGrowthPct > 0 {
		risingScore = CalculateRisingScore(weightedGrowthPct, sizeMultiplier, maintenanceMultiplier, risingAgeHours)
	}

	// Upsert
	toNumeric := func(v float64) pgtype.Numeric {
		var n pgtype.Numeric
		n.Scan(v)
		return n
	}

	return c.db.UpsertTrendingScore(ctx, database.UpsertTrendingScoreParams{
		AddonID:               stat.AddonID,
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

**Step 2: Verify compilation**

```bash
go build ./internal/trending/...
```

**Step 3: Commit**

```bash
git add internal/trending/calculator_fast.go
git commit -m "feat: add optimized bulk trending calculator"
```

---

### Task 3: Update cmd/calculate to Use Fast Calculator

**Files:**
- Modify: `cmd/calculate/main.go`

**Step 1: Update to use CalculateAllFast**

Replace the calculator call:

```go
// Change from:
if err := calc.CalculateAll(ctx); err != nil {

// To:
if err := calc.CalculateAllFast(ctx); err != nil {
```

**Step 2: Verify compilation**

```bash
go build ./cmd/calculate/...
```

**Step 3: Commit**

```bash
git add cmd/calculate/main.go
git commit -m "feat: use fast calculator in cmd/calculate"
```

---

### Task 4: Update Sync Job to Use Fast Calculator

**Files:**
- Modify: `cmd/sync/main.go`

**Step 1: Update sync to use CalculateAllFast**

```go
// Change from:
if err := calculator.CalculateAll(ctx); err != nil {

// To:
if err := calculator.CalculateAllFast(ctx); err != nil {
```

**Step 2: Verify compilation**

```bash
go build ./cmd/sync/...
```

**Step 3: Commit**

```bash
git add cmd/sync/main.go
git commit -m "feat: use fast calculator in sync job"
```

---

### Task 5: Test and Deploy

**Step 1: Run tests**

```bash
go test ./internal/trending/... -v
```

**Step 2: Test locally**

```bash
DATABASE_URL='postgres://...' go run ./cmd/calculate
```

Expected: Complete in under 30 seconds (vs 10+ minutes before).

**Step 3: Push and deploy**

```bash
git push origin main
```

**Step 4: Verify production**

```bash
curl https://addon-radar-api-production.up.railway.app/api/v1/trending/hot | jq '.data | length'
```

Expected: Returns 20 (or fewer if not enough qualifying addons).

---

## Performance Comparison

| Operation | Before (per addon) | After (bulk) |
|-----------|-------------------|--------------|
| Snapshot stats 24h | 12,433 queries | 1 query |
| Snapshot stats 7d | 12,433 queries | (combined) |
| Update counts | 12,433 queries | 1 query |
| Existing scores | 12,433 queries | 1 query |
| Upserts | 12,433 queries | 12,433 queries* |
| **Total** | **62,165 queries** | **~12,436 queries** |

*Upserts still per-addon but could be batched further if needed.

**Expected speedup: ~5x from query reduction + significant latency reduction**
