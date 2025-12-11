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

	// Get values from bulk stats (already native Go types due to SQL casts)
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
