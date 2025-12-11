package trending

import (
	"context"
	"fmt"
	"log/slog"
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
	percentile95, err := c.db.GetDownloadPercentile(ctx)
	if err != nil {
		return err
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
	// Extract downloads from pgtype.Int8
	var downloads float64
	if addon.DownloadCount.Valid {
		downloads = float64(addon.DownloadCount.Int64)
	}

	// Get snapshot stats for 24h window
	stats24h, err := c.db.GetSnapshotStats(ctx, database.GetSnapshotStatsParams{
		AddonID: addon.ID,
		Column2: pgtype.Text{String: "24", Valid: true},
	})
	if err != nil {
		return err
	}

	// Get snapshot stats for 7d window
	stats7d, err := c.db.GetSnapshotStats(ctx, database.GetSnapshotStatsParams{
		AddonID: addon.ID,
		Column2: pgtype.Text{String: "168", Valid: true},
	})
	if err != nil {
		return err
	}

	// Extract values from interface{} with type assertions
	downloadChange24h := extractInt64(stats24h.DownloadChange)
	thumbsChange24h := extractInt64(stats24h.ThumbsChange)
	downloadChange7d := extractInt64(stats7d.DownloadChange)
	thumbsChange7d := extractInt64(stats7d.ThumbsChange)
	minDownloads := extractInt64(stats7d.MinDownloads)

	// Calculate velocities (downloads per hour)
	velocity24h := float64(downloadChange24h) / 24.0
	velocity7d := float64(downloadChange7d) / 168.0

	// Thumbs velocities
	thumbsVel24h := float64(thumbsChange24h) / 24.0
	thumbsVel7d := float64(thumbsChange7d) / 168.0

	// Apply adaptive windows
	_, downloadVelocity := CalculateVelocity(velocity24h, velocity7d, int(stats24h.SnapshotCount), downloadChange24h)
	_, thumbsVelocity := CalculateVelocity(thumbsVel24h, thumbsVel7d, int(stats24h.SnapshotCount), thumbsChange24h)

	// Calculate growth percentages
	var downloadGrowthPct, thumbsGrowthPct float64
	if minDownloads > 0 {
		downloadGrowthPct = (float64(downloadChange7d) / float64(minDownloads)) * 100
	}

	// Calculate thumbs base from addon current count minus change
	var thumbsBase int32
	if addon.ThumbsUpCount.Valid {
		thumbsBase = addon.ThumbsUpCount.Int32 - int32(thumbsChange7d)
	}
	if thumbsBase > 0 {
		thumbsGrowthPct = (float64(thumbsChange7d) / float64(thumbsBase)) * 100
	}

	// Size multiplier
	sizeMultiplier := CalculateSizeMultiplier(downloads, percentile95)

	// Maintenance multiplier (count recent updates)
	updateCount, err := c.db.CountRecentFileUpdates(ctx, database.CountRecentFileUpdatesParams{
		AddonID: addon.ID,
		Column2: pgtype.Text{String: "90", Valid: true},
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

// extractInt64 safely extracts an int64 from interface{} returned by database queries
func extractInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case pgtype.Numeric:
		if !val.Valid {
			return 0
		}
		f8, err := val.Float64Value()
		if err != nil {
			return 0
		}
		return int64(f8.Float64)
	default:
		// Log unknown type for debugging
		slog.Debug("extractInt64: unknown type", "type", fmt.Sprintf("%T", v), "value", v)
		return 0
	}
}
