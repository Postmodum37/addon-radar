# Trending Algorithm V2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve trending algorithm with position tracking, remove useless thumbs_up signal, and rework Rising Stars to favor smaller addons.

**Architecture:** Add new `trending_rank_history` table for position tracking. Modify trending formulas: Hot uses 85% downloads + 15% update boost; Rising uses relative growth (downloads_gained/total_downloads) with no size penalty. API returns rank changes.

**Tech Stack:** Go 1.25, PostgreSQL, sqlc, Gin

---

## Task 1: Add Database Schema for Rank History

**Files:**
- Modify: `sql/schema.sql`

**Step 1: Add trending_rank_history table to schema**

Add at end of `sql/schema.sql`:

```sql
-- Trending rank history: tracks position changes over time
CREATE TABLE trending_rank_history (
    addon_id INTEGER NOT NULL REFERENCES addons(id) ON DELETE CASCADE,
    category TEXT NOT NULL CHECK (category IN ('hot', 'rising')),
    rank SMALLINT NOT NULL,
    score DECIMAL(20,10) NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (addon_id, category, recorded_at)
);

CREATE INDEX idx_rank_history_time
    ON trending_rank_history(category, recorded_at DESC);

CREATE INDEX idx_rank_history_recorded
    ON trending_rank_history(recorded_at);
```

**Step 2: Commit schema change**

```bash
git add sql/schema.sql
git commit -m "feat(db): add trending_rank_history table for position tracking"
```

---

## Task 2: Add SQL Queries for Rank History

**Files:**
- Modify: `sql/queries.sql`

**Step 1: Add rank history queries**

Add at end of `sql/queries.sql`:

```sql
-- name: InsertRankHistory :exec
-- Record current rank for an addon in a category
INSERT INTO trending_rank_history (addon_id, category, rank, score, recorded_at)
VALUES ($1, $2, $3, $4, NOW());

-- name: GetRankAt :one
-- Get the rank of an addon at a specific time (closest record before that time)
SELECT rank FROM trending_rank_history
WHERE addon_id = $1
  AND category = $2
  AND recorded_at <= $3
ORDER BY recorded_at DESC
LIMIT 1;

-- name: DeleteOldRankHistory :execrows
-- Delete rank history older than 7 days
DELETE FROM trending_rank_history
WHERE recorded_at < NOW() - INTERVAL '7 days';

-- name: GetRankChanges :many
-- Get rank changes for top addons (24h and 7d ago)
WITH current_ranks AS (
    SELECT addon_id, category, rank, score
    FROM trending_rank_history
    WHERE recorded_at = (
        SELECT MAX(recorded_at) FROM trending_rank_history
    )
),
ranks_24h AS (
    SELECT DISTINCT ON (addon_id, category) addon_id, category, rank
    FROM trending_rank_history
    WHERE recorded_at <= NOW() - INTERVAL '24 hours'
    ORDER BY addon_id, category, recorded_at DESC
),
ranks_7d AS (
    SELECT DISTINCT ON (addon_id, category) addon_id, category, rank
    FROM trending_rank_history
    WHERE recorded_at <= NOW() - INTERVAL '7 days'
    ORDER BY addon_id, category, recorded_at DESC
)
SELECT
    c.addon_id,
    c.category,
    c.rank AS current_rank,
    c.score,
    r24.rank AS rank_24h_ago,
    r7.rank AS rank_7d_ago
FROM current_ranks c
LEFT JOIN ranks_24h r24 ON c.addon_id = r24.addon_id AND c.category = r24.category
LEFT JOIN ranks_7d r7 ON c.addon_id = r7.addon_id AND c.category = r7.category;
```

**Step 2: Regenerate sqlc**

```bash
sqlc generate
```

**Step 3: Commit queries**

```bash
git add sql/queries.sql internal/database/
git commit -m "feat(db): add queries for rank history tracking"
```

---

## Task 3: Update Trending Constants

**Files:**
- Modify: `internal/trending/trending.go`
- Test: `internal/trending/trending_test.go`

**Step 1: Update constants**

Replace the const block in `internal/trending/trending.go`:

```go
const (
	// Hot Right Now weights (total = 1.0)
	HotDownloadWeight = 0.85
	HotUpdateWeight   = 0.15
	UpdateBoost       = 10.0 // Boost value when addon has recent update

	// Rising Stars weights (total = 1.0)
	RisingGrowthWeight      = 0.70
	RisingMaintenanceWeight = 0.30

	HotGravity    = 1.5
	RisingGravity = 1.8
	AgeOffset     = 2.0 // Prevents division by zero and smooths early decay
)
```

**Step 2: Run existing tests to see what breaks**

```bash
go test ./internal/trending/... -v
```

Expected: Some tests will fail due to changed constants.

**Step 3: Commit constant changes**

```bash
git add internal/trending/trending.go
git commit -m "refactor(trending): update constants for v2 algorithm

- Hot: 85% downloads, 15% update boost (was 70/20/10)
- Rising: 70% relative growth, 30% maintenance
- Remove ThumbsWeight constant"
```

---

## Task 4: Add New Signal Functions for Hot

**Files:**
- Modify: `internal/trending/trending.go`
- Test: `internal/trending/trending_test.go`

**Step 1: Write test for CalculateHotSignal**

Add to `internal/trending/trending_test.go`:

```go
func TestCalculateHotSignal(t *testing.T) {
	tests := []struct {
		name           string
		downloadSignal float64
		hasUpdate      bool
		want           float64
	}{
		{
			name:           "with update boost",
			downloadSignal: 100.0,
			hasUpdate:      true,
			want:           86.5, // 0.85*100 + 0.15*10
		},
		{
			name:           "without update",
			downloadSignal: 100.0,
			hasUpdate:      false,
			want:           85.0, // 0.85*100 + 0
		},
		{
			name:           "zero velocity",
			downloadSignal: 0,
			hasUpdate:      true,
			want:           1.5, // 0 + 0.15*10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateHotSignal(tt.downloadSignal, tt.hasUpdate)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("CalculateHotSignal() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/trending/... -run TestCalculateHotSignal -v
```

Expected: FAIL - undefined: CalculateHotSignal

**Step 3: Implement CalculateHotSignal**

Add to `internal/trending/trending.go`:

```go
// CalculateHotSignal computes the signal for Hot Right Now.
// Signal blend: 85% downloads + 15% update boost.
func CalculateHotSignal(downloadSignal float64, hasRecentUpdate bool) float64 {
	updateBoost := 0.0
	if hasRecentUpdate {
		updateBoost = UpdateBoost
	}
	return (HotDownloadWeight * downloadSignal) + (HotUpdateWeight * updateBoost)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/trending/... -run TestCalculateHotSignal -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/trending/trending.go internal/trending/trending_test.go
git commit -m "feat(trending): add CalculateHotSignal function

Replaces CalculateWeightedSignal for Hot: 85% downloads + 15% update boost"
```

---

## Task 5: Add New Signal Functions for Rising

**Files:**
- Modify: `internal/trending/trending.go`
- Test: `internal/trending/trending_test.go`

**Step 1: Write test for CalculateRelativeGrowth**

Add to `internal/trending/trending_test.go`:

```go
func TestCalculateRelativeGrowth(t *testing.T) {
	tests := []struct {
		name           string
		downloadsGained int64
		totalDownloads int64
		want           float64
	}{
		{
			name:           "small addon doubling",
			downloadsGained: 100,
			totalDownloads: 100,
			want:           1.0, // 100% growth
		},
		{
			name:           "large addon small gain",
			downloadsGained: 1000,
			totalDownloads: 100000,
			want:           0.01, // 1% growth
		},
		{
			name:           "zero downloads - avoid division by zero",
			downloadsGained: 50,
			totalDownloads: 0,
			want:           0.0,
		},
		{
			name:           "no gain",
			downloadsGained: 0,
			totalDownloads: 1000,
			want:           0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRelativeGrowth(tt.downloadsGained, tt.totalDownloads)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CalculateRelativeGrowth() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/trending/... -run TestCalculateRelativeGrowth -v
```

Expected: FAIL - undefined: CalculateRelativeGrowth

**Step 3: Implement CalculateRelativeGrowth**

Add to `internal/trending/trending.go`:

```go
// CalculateRelativeGrowth computes growth as a fraction of total downloads.
// Returns downloads_gained / total_downloads, naturally favoring smaller addons.
func CalculateRelativeGrowth(downloadsGained, totalDownloads int64) float64 {
	if totalDownloads <= 0 {
		return 0.0
	}
	return float64(downloadsGained) / float64(totalDownloads)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/trending/... -run TestCalculateRelativeGrowth -v
```

Expected: PASS

**Step 5: Write test for CalculateRisingSignal**

Add to `internal/trending/trending_test.go`:

```go
func TestCalculateRisingSignal(t *testing.T) {
	tests := []struct {
		name                  string
		relativeGrowth        float64
		maintenanceMultiplier float64
		want                  float64
	}{
		{
			name:                  "high growth active addon",
			relativeGrowth:        0.5, // 50% growth
			maintenanceMultiplier: 1.15,
			want:                  0.695, // 0.7*0.5 + 0.3*1.15
		},
		{
			name:                  "low growth stale addon",
			relativeGrowth:        0.01,
			maintenanceMultiplier: 0.95,
			want:                  0.292, // 0.7*0.01 + 0.3*0.95
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRisingSignal(tt.relativeGrowth, tt.maintenanceMultiplier)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("CalculateRisingSignal() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 6: Implement CalculateRisingSignal**

Add to `internal/trending/trending.go`:

```go
// CalculateRisingSignal computes the signal for Rising Stars.
// Signal blend: 70% relative growth + 30% maintenance multiplier.
// Maintenance is included in signal (not as separate multiplier) for Rising.
func CalculateRisingSignal(relativeGrowth, maintenanceMultiplier float64) float64 {
	return (RisingGrowthWeight * relativeGrowth) + (RisingMaintenanceWeight * maintenanceMultiplier)
}
```

**Step 7: Run all new tests**

```bash
go test ./internal/trending/... -run "TestCalculateRelativeGrowth|TestCalculateRisingSignal" -v
```

Expected: PASS

**Step 8: Commit**

```bash
git add internal/trending/trending.go internal/trending/trending_test.go
git commit -m "feat(trending): add Rising Stars signal functions

- CalculateRelativeGrowth: downloads_gained / total_downloads
- CalculateRisingSignal: 70% growth + 30% maintenance"
```

---

## Task 6: Update Score Calculation Functions

**Files:**
- Modify: `internal/trending/trending.go`
- Test: `internal/trending/trending_test.go`

**Step 1: Update CalculateHotScore signature and implementation**

Replace `CalculateHotScore` in `internal/trending/trending.go`:

```go
// CalculateHotScore computes the "Hot Right Now" score.
// Formula: (hot_signal * size_multiplier * maintenance_multiplier) / (age_hours + 2)^1.5
func CalculateHotScore(hotSignal, sizeMultiplier, maintenanceMultiplier, ageHours float64) float64 {
	numerator := hotSignal * sizeMultiplier * maintenanceMultiplier
	denominator := math.Pow(ageHours+AgeOffset, HotGravity)
	return numerator / denominator
}
```

**Step 2: Update CalculateRisingScore - remove size multiplier**

Replace `CalculateRisingScore` in `internal/trending/trending.go`:

```go
// CalculateRisingScore computes the "Rising Stars" score.
// Formula: rising_signal / (age_hours + 2)^1.8
// Note: No size multiplier - relative growth already handles this.
// Note: Maintenance is included in rising_signal, not separate.
func CalculateRisingScore(risingSignal, ageHours float64) float64 {
	denominator := math.Pow(ageHours+AgeOffset, RisingGravity)
	return risingSignal / denominator
}
```

**Step 3: Update tests for new signatures**

Replace `TestCalculateHotScore` and `TestCalculateRisingScore` in `internal/trending/trending_test.go`:

```go
func TestCalculateHotScore(t *testing.T) {
	tests := []struct {
		name                  string
		hotSignal             float64
		sizeMultiplier        float64
		maintenanceMultiplier float64
		ageHours              float64
		want                  float64
	}{
		{
			name:                  "new addon",
			hotSignal:             86.5, // 0.85*100 + 0.15*10
			sizeMultiplier:        0.5,
			maintenanceMultiplier: 1.1,
			ageHours:              0,
			want:                  16.82, // (86.5 * 0.5 * 1.1) / (0+2)^1.5
		},
		{
			name:                  "24h old addon",
			hotSignal:             86.5,
			sizeMultiplier:        0.5,
			maintenanceMultiplier: 1.1,
			ageHours:              24,
			want:                  0.36, // (86.5 * 0.5 * 1.1) / (24+2)^1.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateHotScore(tt.hotSignal, tt.sizeMultiplier, tt.maintenanceMultiplier, tt.ageHours)
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("CalculateHotScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateRisingScore(t *testing.T) {
	tests := []struct {
		name         string
		risingSignal float64
		ageHours     float64
		want         float64
	}{
		{
			name:         "new addon high signal",
			risingSignal: 0.695, // 0.7*0.5 + 0.3*1.15
			ageHours:     0,
			want:         0.20, // 0.695 / (0+2)^1.8
		},
		{
			name:         "48h old addon",
			risingSignal: 0.695,
			ageHours:     48,
			want:         0.00075, // 0.695 / (48+2)^1.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRisingScore(tt.risingSignal, tt.ageHours)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("CalculateRisingScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 4: Run all trending tests**

```bash
go test ./internal/trending/... -v
```

Expected: PASS (some old tests may need updating - see next step)

**Step 5: Remove obsolete test and function**

Remove `TestCalculateWeightedSignal` from test file and `CalculateWeightedSignal` from trending.go (no longer used).

**Step 6: Run tests again**

```bash
go test ./internal/trending/... -v
```

Expected: PASS

**Step 7: Commit**

```bash
git add internal/trending/trending.go internal/trending/trending_test.go
git commit -m "refactor(trending): update score functions for v2 algorithm

- CalculateHotScore: uses new hot_signal parameter
- CalculateRisingScore: simplified, no size multiplier
- Remove CalculateWeightedSignal (replaced by specific functions)"
```

---

## Task 7: Update Calculator to Use New Functions

**Files:**
- Modify: `internal/trending/calculator.go`

**Step 1: Read current calculator implementation**

Review `internal/trending/calculator.go` to understand current flow.

**Step 2: Update calculateHotScore method**

Find and update the `calculateHotScore` method to use `CalculateHotSignal` instead of `CalculateWeightedSignal`:

```go
func (c *Calculator) calculateHotScore(
	downloadVelocity float64,
	hasRecentUpdate bool,
	sizeMultiplier, maintenanceMultiplier float64,
	firstHotAt *time.Time,
) float64 {
	hotSignal := CalculateHotSignal(downloadVelocity, hasRecentUpdate)
	ageHours := c.calculateHotAge(firstHotAt)
	return CalculateHotScore(hotSignal, sizeMultiplier, maintenanceMultiplier, ageHours)
}
```

**Step 3: Update calculateRisingScore method**

Update to use relative growth and new signal function:

```go
func (c *Calculator) calculateRisingScore(
	relativeGrowth float64,
	maintenanceMultiplier float64,
	firstRisingAt *time.Time,
) float64 {
	risingSignal := CalculateRisingSignal(relativeGrowth, maintenanceMultiplier)
	ageHours := c.calculateRisingAge(firstRisingAt)
	return CalculateRisingScore(risingSignal, ageHours)
}
```

**Step 4: Update calculateAndUpsert to compute relative growth**

In the main calculation loop, compute relative growth from download change and total downloads:

```go
// For Rising Stars: calculate relative growth
relativeGrowth := CalculateRelativeGrowth(stats.DownloadChange24h, addon.DownloadCount)
```

**Step 5: Remove thumbs_velocity from calculations**

Remove all references to `thumbsVelocity` and `thumbsGrowthPct` from the calculator. Keep the DB fields for now but don't compute them.

**Step 6: Build and test**

```bash
go build ./...
go test ./internal/trending/... -v
```

Expected: Build succeeds, tests pass

**Step 7: Commit**

```bash
git add internal/trending/calculator.go
git commit -m "refactor(trending): update calculator for v2 algorithm

- Use CalculateHotSignal for Hot Right Now
- Use relative growth for Rising Stars
- Remove thumbs velocity calculations"
```

---

## Task 8: Add Rank History Recording

**Files:**
- Modify: `internal/trending/calculator.go`

**Step 1: Add method to record rank history**

Add to `internal/trending/calculator.go`:

```go
func (c *Calculator) recordRankHistory(ctx context.Context, hotAddons, risingAddons []database.ListHotAddonsRow) error {
	// Record hot addon ranks
	for i, addon := range hotAddons {
		err := c.db.InsertRankHistory(ctx, database.InsertRankHistoryParams{
			AddonID:  addon.ID,
			Category: "hot",
			Rank:     int16(i + 1),
			Score:    addon.HotScore,
		})
		if err != nil {
			return fmt.Errorf("insert hot rank history: %w", err)
		}
	}

	// Record rising addon ranks
	for i, addon := range risingAddons {
		err := c.db.InsertRankHistory(ctx, database.InsertRankHistoryParams{
			AddonID:  addon.ID,
			Category: "rising",
			Rank:     int16(i + 1),
			Score:    addon.RisingScore,
		})
		if err != nil {
			return fmt.Errorf("insert rising rank history: %w", err)
		}
	}

	return nil
}
```

**Step 2: Add cleanup method**

Add to `internal/trending/calculator.go`:

```go
func (c *Calculator) cleanupOldRankHistory(ctx context.Context) error {
	deleted, err := c.db.DeleteOldRankHistory(ctx)
	if err != nil {
		return fmt.Errorf("delete old rank history: %w", err)
	}
	if deleted > 0 {
		slog.Info("cleaned up old rank history", "deleted", deleted)
	}
	return nil
}
```

**Step 3: Call from CalculateAll**

At the end of `CalculateAll`, after scores are calculated, fetch top addons and record history:

```go
// Record rank history
hotAddons, err := c.db.ListHotAddons(ctx, 20)
if err != nil {
	return fmt.Errorf("list hot addons for history: %w", err)
}
risingAddons, err := c.db.ListRisingAddons(ctx, 20)
if err != nil {
	return fmt.Errorf("list rising addons for history: %w", err)
}
if err := c.recordRankHistory(ctx, hotAddons, risingAddons); err != nil {
	return fmt.Errorf("record rank history: %w", err)
}

// Cleanup old history
if err := c.cleanupOldRankHistory(ctx); err != nil {
	slog.Warn("failed to cleanup rank history", "error", err)
}
```

**Step 4: Build**

```bash
go build ./...
```

Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/trending/calculator.go
git commit -m "feat(trending): add rank history recording

Records top 20 hot and rising addon ranks after each calculation.
Cleans up history older than 7 days."
```

---

## Task 9: Update API Response with Rank Changes

**Files:**
- Modify: `internal/api/handlers.go`

**Step 1: Update TrendingAddonResponse struct**

Update in `internal/api/handlers.go`:

```go
type TrendingAddonResponse struct {
	AddonResponse
	Score         float64 `json:"score"`
	Rank          int     `json:"rank"`
	RankChange24h *int    `json:"rank_change_24h"` // nil = new to list
	RankChange7d  *int    `json:"rank_change_7d"`  // nil = new to list
}
```

**Step 2: Create helper to build response with rank changes**

Add helper function:

```go
func (s *Server) buildTrendingResponse(ctx context.Context, category string, addons interface{}) ([]TrendingAddonResponse, error) {
	// Get rank changes from database
	rankChanges, err := s.db.GetRankChanges(ctx)
	if err != nil {
		return nil, err
	}

	// Build lookup map
	type rankInfo struct {
		rank24hAgo *int16
		rank7dAgo  *int16
	}
	rankMap := make(map[int32]rankInfo)
	for _, rc := range rankChanges {
		if rc.Category == category {
			info := rankInfo{}
			if rc.Rank24hAgo.Valid {
				v := rc.Rank24hAgo.Int16
				info.rank24hAgo = &v
			}
			if rc.Rank7dAgo.Valid {
				v := rc.Rank7dAgo.Int16
				info.rank7dAgo = &v
			}
			rankMap[rc.AddonID] = info
		}
	}

	// Build response based on addon type
	// (Implementation depends on whether hot or rising addons)
	// ...
}
```

**Step 3: Update handleTrendingHot**

Update to include rank and rank changes in response.

**Step 4: Update handleTrendingRising**

Update to include rank and rank changes in response.

**Step 5: Build and test manually**

```bash
go build ./cmd/web
```

**Step 6: Commit**

```bash
git add internal/api/handlers.go
git commit -m "feat(api): add rank and rank changes to trending response

- rank: current position in list (1-20)
- rank_change_24h: positions moved since 24h ago (positive = up)
- rank_change_7d: positions moved since 7 days ago
- null values indicate addon is new to the list"
```

---

## Task 10: Update Documentation

**Files:**
- Modify: `docs/ALGORITHM.md`

**Step 1: Update algorithm documentation**

Update `docs/ALGORITHM.md` to reflect v2 changes:
- New formulas for Hot and Rising
- Position tracking feature
- Removal of thumbs_up signal
- Relative growth for Rising Stars

**Step 2: Commit**

```bash
git add docs/ALGORITHM.md
git commit -m "docs: update ALGORITHM.md for v2 trending algorithm"
```

---

## Task 11: Run Full Test Suite and Lint

**Files:**
- All modified files

**Step 1: Run linter**

```bash
golangci-lint run ./...
```

Expected: No errors

**Step 2: Run all tests**

```bash
go test ./... -race -timeout=5m
```

Expected: All tests pass

**Step 3: Build all binaries**

```bash
go build -o bin/sync ./cmd/sync
go build -o bin/web ./cmd/web
```

Expected: Build succeeds

**Step 4: Final commit if any fixes needed**

```bash
git add .
git commit -m "fix: address lint issues and test failures"
```

---

## Summary

| Task | Description | Commits |
|------|-------------|---------|
| 1 | Add rank history table schema | 1 |
| 2 | Add SQL queries for rank history | 1 |
| 3 | Update trending constants | 1 |
| 4 | Add CalculateHotSignal function | 1 |
| 5 | Add Rising signal functions | 1 |
| 6 | Update score calculation functions | 1 |
| 7 | Update calculator for v2 | 1 |
| 8 | Add rank history recording | 1 |
| 9 | Update API response | 1 |
| 10 | Update documentation | 1 |
| 11 | Final lint and test | 0-1 |

**Total: ~10-11 commits**
