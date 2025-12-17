package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"addon-radar/internal/config"
	"addon-radar/internal/database"
	"addon-radar/internal/sync"
	"addon-radar/internal/trending"
)

const (
	// snapshotDeleteBatchSize is the number of old snapshots to delete per batch
	// to avoid long-running transactions that lock the table.
	snapshotDeleteBatchSize = 10000

	// minSyncedAddonsThreshold is the minimum number of addons that must be synced
	// before marking missing addons as inactive. Prevents catastrophic data loss
	// if CurseForge API returns empty response.
	minSyncedAddonsThreshold = 1000
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

	// Validate required config for sync
	if cfg.CurseForgeAPIKey == "" {
		slog.Error("CURSEFORGE_API_KEY is required for sync")
		os.Exit(1)
	}

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
	syncedIDs, err := syncService.RunFullSync(ctx)
	if err != nil {
		slog.Error("sync failed", "error", err)
		os.Exit(1)
	}

	slog.Info("sync complete")

	// Run trending calculation
	slog.Info("starting trending calculation")
	queries := database.New(pool)
	calculator := trending.NewCalculator(queries)
	if err := calculator.CalculateAll(ctx); err != nil {
		slog.Error("trending calculation failed", "error", err)
		// Don't exit - sync succeeded, trending is secondary
	}

	// Cleanup: delete old snapshots (95-day retention) in batches
	// to avoid long-running transactions that lock the table
	var totalDeleted int64
	for {
		deleted, err := queries.DeleteOldSnapshotsBatch(ctx, snapshotDeleteBatchSize)
		if err != nil {
			slog.Warn("snapshot cleanup batch failed", "error", err, "deleted_so_far", totalDeleted)
			break
		}
		totalDeleted += deleted
		if deleted == 0 {
			break
		}
		if deleted == int64(snapshotDeleteBatchSize) {
			// More batches to process, yield briefly to reduce contention
			time.Sleep(100 * time.Millisecond)
		}
	}
	if totalDeleted > 0 {
		slog.Info("snapshots cleaned", "count", totalDeleted)
	}

	// Cleanup: mark missing addons as inactive
	// Guard against empty or suspiciously small sync results to prevent catastrophic data loss
	if len(syncedIDs) < minSyncedAddonsThreshold {
		slog.Warn("skipping inactive marking: synced addon count below threshold",
			"synced", len(syncedIDs),
			"threshold", minSyncedAddonsThreshold,
		)
	} else {
		inactive, err := queries.MarkMissingAddonsInactive(ctx, syncedIDs)
		if err != nil {
			slog.Warn("mark inactive failed", "error", err)
		} else if inactive > 0 {
			slog.Info("addons marked inactive", "count", inactive)
		}
	}
}
