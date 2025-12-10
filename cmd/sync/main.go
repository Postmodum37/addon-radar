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
	if err := syncService.RunFullSync(ctx); err != nil {
		slog.Error("sync failed", "error", err)
		os.Exit(1)
	}

	slog.Info("sync complete")
}
