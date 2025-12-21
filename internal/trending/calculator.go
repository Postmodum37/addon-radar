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

// CalculateAll recalculates trending scores for all active addons using bulk queries.
func (c *Calculator) CalculateAll(ctx context.Context) error {
	slog.Info("starting trending calculation")
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

	// Step 7: Record rank history
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

	// Step 8: Cleanup old history
	if err := c.cleanupOldRankHistory(ctx); err != nil {
		slog.Warn("failed to cleanup rank history", "error", err)
	}

	slog.Info("trending calculation complete", "duration", time.Since(start), "processed", processed)
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

	// Calculate velocities and growth
	downloadVelocity, thumbsVelocity := c.calculateVelocities(stat)
	downloadGrowthPct, thumbsGrowthPct := c.calculateGrowthPercentages(stat)

	// Multipliers
	sizeMultiplier := CalculateSizeMultiplier(downloads, percentile95)
	updateCount := updateMap[stat.AddonID]
	maintenanceMultiplier := CalculateMaintenanceMultiplier(int(updateCount))

	// Recent update check
	hasRecentUpdate := false
	if stat.LatestFileDate.Valid {
		hasRecentUpdate = time.Since(stat.LatestFileDate.Time) < 7*24*time.Hour
	}

	// Calculate signals using new v2 functions
	hotSignal := CalculateHotSignal(downloadVelocity, hasRecentUpdate)
	relativeGrowth := CalculateRelativeGrowth(stat.DownloadChange7d, stat.MinDownloads7d)
	risingSignal := CalculateRisingSignal(relativeGrowth, maintenanceMultiplier)

	// Calculate age and timestamps
	existing := scoreMap[stat.AddonID]
	hotAgeHours, firstHotAt := c.calculateHotAge(downloads, hotSignal, existing)
	risingAgeHours, firstRisingAt := c.calculateRisingAge(downloads, risingSignal, existing)

	// Final scores
	hotScore := c.calculateHotScore(downloads, hotSignal, sizeMultiplier, maintenanceMultiplier, hotAgeHours)
	risingScore := c.calculateRisingScore(downloads, risingSignal, risingAgeHours)

	// Upsert
	return c.upsertScore(ctx, stat.AddonID, hotScore, risingScore, downloadVelocity, thumbsVelocity,
		downloadGrowthPct, thumbsGrowthPct, sizeMultiplier, maintenanceMultiplier, firstHotAt, firstRisingAt)
}

func (c *Calculator) calculateVelocities(stat database.GetAllSnapshotStatsRow) (float64, float64) {
	downloadChange24h := stat.DownloadChange24h
	thumbsChange24h := int64(stat.ThumbsChange24h)
	downloadChange7d := stat.DownloadChange7d
	thumbsChange7d := int64(stat.ThumbsChange7d)
	snapshotCount24h := stat.SnapshotCount24h

	velocity24h := float64(downloadChange24h) / 24.0
	velocity7d := float64(downloadChange7d) / 168.0
	thumbsVel24h := float64(thumbsChange24h) / 24.0
	thumbsVel7d := float64(thumbsChange7d) / 168.0

	_, downloadVelocity := CalculateVelocity(velocity24h, velocity7d, int(snapshotCount24h), downloadChange24h)
	_, thumbsVelocity := CalculateVelocity(thumbsVel24h, thumbsVel7d, int(snapshotCount24h), thumbsChange24h)

	return downloadVelocity, thumbsVelocity
}

func (c *Calculator) calculateGrowthPercentages(stat database.GetAllSnapshotStatsRow) (float64, float64) {
	downloadChange7d := stat.DownloadChange7d
	thumbsChange7d := int64(stat.ThumbsChange7d)
	minDownloads := stat.MinDownloads7d

	var downloadGrowthPct, thumbsGrowthPct float64
	if minDownloads > 0 {
		downloadGrowthPct = (float64(downloadChange7d) / float64(minDownloads)) * 100
	}

	var thumbsBase int32
	if stat.ThumbsUpCount.Valid {
		thumbsBase = stat.ThumbsUpCount.Int32 - int32(thumbsChange7d) //nolint:gosec // thumbsChange7d from DB is safe
	}
	if thumbsBase > 0 {
		thumbsGrowthPct = (float64(thumbsChange7d) / float64(thumbsBase)) * 100
	}

	return downloadGrowthPct, thumbsGrowthPct
}

func (c *Calculator) calculateHotAge(downloads, hotSignal float64, existing database.GetAllTrendingScoresRow) (float64, pgtype.Timestamptz) {
	var hotAgeHours float64
	var firstHotAt pgtype.Timestamptz
	if downloads >= 500 && hotSignal > 0 {
		if existing.FirstHotAt.Valid {
			hotAgeHours = time.Since(existing.FirstHotAt.Time).Hours()
			firstHotAt = existing.FirstHotAt
		} else {
			firstHotAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}
	}
	return hotAgeHours, firstHotAt
}

func (c *Calculator) calculateRisingAge(downloads, risingSignal float64, existing database.GetAllTrendingScoresRow) (float64, pgtype.Timestamptz) {
	var risingAgeHours float64
	var firstRisingAt pgtype.Timestamptz
	if downloads >= 50 && downloads <= 10000 && risingSignal > 0 {
		if existing.FirstRisingAt.Valid {
			risingAgeHours = time.Since(existing.FirstRisingAt.Time).Hours()
			firstRisingAt = existing.FirstRisingAt
		} else {
			firstRisingAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}
	}
	return risingAgeHours, firstRisingAt
}

func (c *Calculator) calculateHotScore(downloads, hotSignal, sizeMultiplier, maintenanceMultiplier, hotAgeHours float64) float64 {
	if downloads >= 500 && hotSignal > 0 {
		return CalculateHotScore(hotSignal, sizeMultiplier, maintenanceMultiplier, hotAgeHours)
	}
	return 0
}

func (c *Calculator) calculateRisingScore(downloads, risingSignal, risingAgeHours float64) float64 {
	if downloads >= 50 && downloads <= 10000 && risingSignal > 0 {
		return CalculateRisingScore(risingSignal, risingAgeHours)
	}
	return 0
}

func (c *Calculator) upsertScore(ctx context.Context, addonID int32, hotScore, risingScore, downloadVelocity, thumbsVelocity,
	downloadGrowthPct, thumbsGrowthPct, sizeMultiplier, maintenanceMultiplier float64,
	firstHotAt, firstRisingAt pgtype.Timestamptz) error {

	toNumeric := func(v float64) pgtype.Numeric {
		var n pgtype.Numeric
		n.Scan(fmt.Sprintf("%f", v)) //nolint:errcheck // Scan from formatted string is safe
		return n
	}

	return c.db.UpsertTrendingScore(ctx, database.UpsertTrendingScoreParams{
		AddonID:               addonID,
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

func (c *Calculator) recordRankHistory(ctx context.Context, hotAddons []database.ListHotAddonsRow, risingAddons []database.ListRisingAddonsRow) error {
	// Record hot addon ranks
	for i, addon := range hotAddons {
		err := c.db.InsertRankHistory(ctx, database.InsertRankHistoryParams{
			AddonID:  addon.ID,
			Category: "hot",
			Rank:     int16(i + 1), //nolint:gosec // i is bounded by query limit (20)
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
			Rank:     int16(i + 1), //nolint:gosec // i is bounded by query limit (20)
			Score:    addon.RisingScore,
		})
		if err != nil {
			return fmt.Errorf("insert rising rank history: %w", err)
		}
	}

	return nil
}

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
