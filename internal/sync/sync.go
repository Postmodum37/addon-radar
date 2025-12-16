package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"addon-radar/internal/curseforge"
	"addon-radar/internal/database"
)

// CurseForgeClient defines the interface for CurseForge API operations
type CurseForgeClient interface {
	GetAllWoWAddons(ctx context.Context) ([]curseforge.Mod, error)
	GetCategories(ctx context.Context, gameID int) ([]curseforge.Category, error)
}

// Service handles the sync process
type Service struct {
	pool   *pgxpool.Pool
	db     *database.Queries
	client CurseForgeClient
}

// NewService creates a new sync service
func NewService(pool *pgxpool.Pool, apiKey string) *Service {
	return &Service{
		pool:   pool,
		db:     database.New(pool),
		client: curseforge.NewClient(apiKey),
	}
}

// NewServiceWithClient creates a sync service with a custom client (for testing)
func NewServiceWithClient(pool *pgxpool.Pool, db *database.Queries, client CurseForgeClient) *Service {
	return &Service{
		pool:   pool,
		db:     db,
		client: client,
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

	// Upsert each addon and create snapshot atomically
	var successCount, errorCount int
	for _, mod := range mods {
		if err := s.syncAddon(ctx, mod); err != nil {
			slog.Error("failed to sync addon", "id", mod.ID, "name", mod.Name, "error", err)
			errorCount++
			continue
		}
		successCount++
	}

	duration := time.Since(startTime)

	// Warn if sync is approaching or exceeding hourly window
	if duration > 55*time.Minute {
		slog.Warn("sync duration approaching hourly limit",
			"duration", duration,
			"limit", "60m",
		)
	}

	slog.Info("full sync complete",
		"duration", duration,
		"total", len(mods),
		"success", successCount,
		"errors", errorCount,
	)

	// Fail if error rate exceeds 1%
	if errorCount > 0 && float64(errorCount)/float64(len(mods)) > 0.01 {
		return fmt.Errorf("sync had too many errors: %d/%d (%.1f%%)",
			errorCount, len(mods), float64(errorCount)/float64(len(mods))*100)
	}

	return nil
}

// syncAddon upserts an addon and creates a snapshot atomically
func (s *Service) syncAddon(ctx context.Context, mod curseforge.Mod) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.db.WithTx(tx)

	if err := s.upsertAddonWithTx(ctx, qtx, mod); err != nil {
		return fmt.Errorf("upsert addon: %w", err)
	}

	if err := s.createSnapshotWithTx(ctx, qtx, mod); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// syncCategories fetches and stores all WoW addon categories
func (s *Service) syncCategories(ctx context.Context) error {
	categories, err := s.client.GetCategories(ctx, curseforge.GameIDWoW)
	if err != nil {
		return fmt.Errorf("fetch categories: %w", err)
	}

	// Sort categories so parents come before children
	// First pass: insert all categories without parent references
	for _, cat := range categories {
		var iconURL pgtype.Text
		if cat.IconURL != "" {
			iconURL = pgtype.Text{String: cat.IconURL, Valid: true}
		}

		err := s.db.UpsertCategory(ctx, database.UpsertCategoryParams{
			ID:       int32(cat.ID),
			Name:     cat.Name,
			Slug:     cat.Slug,
			ParentID: pgtype.Int4{}, // No parent reference initially
			IconUrl:  iconURL,
		})
		if err != nil {
			slog.Warn("failed to upsert category", "id", cat.ID, "error", err)
		}
	}

	// Second pass: update parent references
	for _, cat := range categories {
		if cat.ParentID > 0 {
			var iconURL pgtype.Text
			if cat.IconURL != "" {
				iconURL = pgtype.Text{String: cat.IconURL, Valid: true}
			}

			err := s.db.UpsertCategory(ctx, database.UpsertCategoryParams{
				ID:       int32(cat.ID),
				Name:     cat.Name,
				Slug:     cat.Slug,
				ParentID: pgtype.Int4{Int32: int32(cat.ParentID), Valid: true},
				IconUrl:  iconURL,
			})
			if err != nil {
				slog.Warn("failed to update category parent", "id", cat.ID, "parentId", cat.ParentID, "error", err)
			}
		}
	}

	slog.Info("synced categories", "count", len(categories))
	return nil
}

// upsertAddonWithTx inserts or updates an addon within a transaction
func (s *Service) upsertAddonWithTx(ctx context.Context, qtx *database.Queries, mod curseforge.Mod) error {
	// Extract primary author
	var authorName pgtype.Text
	var authorID pgtype.Int4
	if len(mod.Authors) > 0 {
		authorName = pgtype.Text{String: mod.Authors[0].Name, Valid: true}
		authorID = pgtype.Int4{Int32: int32(mod.Authors[0].ID), Valid: true}
	}

	// Extract logo URL
	var logoURL pgtype.Text
	if mod.Logo != nil {
		logoURL = pgtype.Text{String: mod.Logo.ThumbnailURL, Valid: true}
	}

	// Extract category IDs
	categoryIDs := make([]int32, len(mod.Categories))
	var primaryCategoryID pgtype.Int4
	for i, cat := range mod.Categories {
		categoryIDs[i] = int32(cat.ID)
		if i == 0 {
			primaryCategoryID = pgtype.Int4{Int32: int32(cat.ID), Valid: true}
		}
	}

	// Extract game versions from latest files
	gameVersions := extractGameVersions(mod)

	// Get latest file date
	var latestFileDate pgtype.Timestamptz
	if len(mod.LatestFiles) > 0 {
		latestFileDate = pgtype.Timestamptz{Time: mod.LatestFiles[0].FileDate, Valid: true}
	}

	// Summary
	var summary pgtype.Text
	if mod.Summary != "" {
		summary = pgtype.Text{String: mod.Summary, Valid: true}
	}

	// Created at and last updated at
	createdAt := pgtype.Timestamptz{Time: mod.DateCreated, Valid: true}
	lastUpdatedAt := pgtype.Timestamptz{Time: mod.DateModified, Valid: true}

	// Download count
	downloadCount := pgtype.Int8{Int64: mod.DownloadCount, Valid: true}

	// Thumbs up count
	thumbsUpCount := pgtype.Int4{Int32: int32(mod.ThumbsUpCount), Valid: true}

	// Popularity rank
	popularityRank := pgtype.Int4{Int32: int32(mod.PopularityRank), Valid: true}

	// Calculate rating as numeric
	var rating pgtype.Numeric
	if mod.Rating > 0 {
		if err := rating.Scan(fmt.Sprintf("%.2f", mod.Rating)); err != nil {
			slog.Warn("failed to convert rating", "rating", mod.Rating, "error", err)
		}
	}

	return qtx.UpsertAddon(ctx, database.UpsertAddonParams{
		ID:                int32(mod.ID),
		Name:              mod.Name,
		Slug:              mod.Slug,
		Summary:           summary,
		AuthorName:        authorName,
		AuthorID:          authorID,
		LogoUrl:           logoURL,
		PrimaryCategoryID: primaryCategoryID,
		Categories:        categoryIDs,
		GameVersions:      gameVersions,
		CreatedAt:         createdAt,
		LastUpdatedAt:     lastUpdatedAt,
		DownloadCount:     downloadCount,
		ThumbsUpCount:     thumbsUpCount,
		PopularityRank:    popularityRank,
		Rating:            rating,
		LatestFileDate:    latestFileDate,
	})
}

// createSnapshotWithTx creates a point-in-time snapshot within a transaction
func (s *Service) createSnapshotWithTx(ctx context.Context, qtx *database.Queries, mod curseforge.Mod) error {
	var latestFileDate pgtype.Timestamptz
	if len(mod.LatestFiles) > 0 {
		latestFileDate = pgtype.Timestamptz{Time: mod.LatestFiles[0].FileDate, Valid: true}
	}

	var rating pgtype.Numeric
	if mod.Rating > 0 {
		if err := rating.Scan(fmt.Sprintf("%.2f", mod.Rating)); err != nil {
			slog.Warn("failed to convert rating", "rating", mod.Rating, "error", err)
		}
	}

	thumbsUpCount := pgtype.Int4{Int32: int32(mod.ThumbsUpCount), Valid: true}
	popularityRank := pgtype.Int4{Int32: int32(mod.PopularityRank), Valid: true}

	return qtx.CreateSnapshot(ctx, database.CreateSnapshotParams{
		AddonID:        int32(mod.ID),
		DownloadCount:  mod.DownloadCount,
		ThumbsUpCount:  thumbsUpCount,
		PopularityRank: popularityRank,
		Rating:         rating,
		LatestFileDate: latestFileDate,
	})
}

// upsertAddon is a convenience wrapper for testing (uses transaction internally)
func (s *Service) upsertAddon(ctx context.Context, mod curseforge.Mod) error {
	return s.upsertAddonWithTx(ctx, s.db, mod)
}

// createSnapshot is a convenience wrapper for testing (uses transaction internally)
func (s *Service) createSnapshot(ctx context.Context, mod curseforge.Mod) error {
	return s.createSnapshotWithTx(ctx, s.db, mod)
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
