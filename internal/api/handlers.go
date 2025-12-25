package api

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"addon-radar/internal/database"
)

type AddonResponse struct {
	ID             int32    `json:"id"`
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	Summary        string   `json:"summary,omitempty"`
	AuthorName     string   `json:"author_name,omitempty"`
	LogoURL        string   `json:"logo_url,omitempty"`
	DownloadCount  int64    `json:"download_count"`
	ThumbsUpCount  int32    `json:"thumbs_up_count"`
	PopularityRank int32    `json:"popularity_rank,omitempty"`
	GameVersions   []string `json:"game_versions"`
	LastUpdatedAt  string   `json:"last_updated_at,omitempty"`
}

type TrendingAddonResponse struct {
	AddonResponse
	Score            float64 `json:"score"`
	Rank             int     `json:"rank"`
	RankChange24h    *int    `json:"rank_change_24h"` // nil = new to list
	RankChange7d     *int    `json:"rank_change_7d"`  // nil = new to list
	DownloadVelocity float64 `json:"download_velocity"`
}

func addonToResponse(a database.Addon) AddonResponse {
	resp := AddonResponse{
		ID:            a.ID,
		Name:          a.Name,
		Slug:          a.Slug,
		DownloadCount: a.DownloadCount.Int64,
		ThumbsUpCount: a.ThumbsUpCount.Int32,
		GameVersions:  a.GameVersions,
	}

	if a.Summary.Valid {
		resp.Summary = a.Summary.String
	}
	if a.AuthorName.Valid {
		resp.AuthorName = a.AuthorName.String
	}
	if a.LogoUrl.Valid {
		resp.LogoURL = a.LogoUrl.String
	}
	if a.PopularityRank.Valid {
		resp.PopularityRank = a.PopularityRank.Int32
	}
	if a.LastUpdatedAt.Valid {
		resp.LastUpdatedAt = a.LastUpdatedAt.Time.Format("2006-01-02T15:04:05Z")
	}

	return resp
}

// parsePaginationParams extracts and validates page, perPage, and calculates offset.
func parsePaginationParams(c *gin.Context) (page, perPage, offset int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err = strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset = (page - 1) * perPage
	return page, perPage, offset
}

// buildRankChangeMap creates a lookup map for rank changes by addon ID for a specific category.
func buildRankChangeMap(rankChanges []database.GetRankChangesRow, category string) map[int32]database.GetRankChangesRow {
	m := make(map[int32]database.GetRankChangesRow)
	for _, rc := range rankChanges {
		if rc.Category == category {
			m[rc.AddonID] = rc
		}
	}
	return m
}

// applyRankChanges applies rank change data to a trending addon response.
func applyRankChanges(resp *TrendingAddonResponse, rc database.GetRankChangesRow) {
	if rc.Rank24hAgo.Valid {
		change := int(rc.Rank24hAgo.Int16 - rc.CurrentRank)
		resp.RankChange24h = &change
	}
	if rc.Rank7dAgo.Valid {
		change := int(rc.Rank7dAgo.Int16 - rc.CurrentRank)
		resp.RankChange7d = &change
	}
}

// numericToFloat64 converts a pgtype.Numeric to float64, returning 0 on error.
func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f8, err := n.Float64Value()
	if err != nil {
		slog.Debug("failed to convert numeric to float64", "error", err)
		return 0
	}
	return f8.Float64
}

func (s *Server) handleListAddons(c *gin.Context) {
	page, perPage, offset := parsePaginationParams(c)
	search := c.Query("search")
	categoryStr := c.Query("category")
	ctx := c.Request.Context()

	var addons []database.Addon
	var total int64
	var err error

	if search != "" {
		// Convert search string to pgtype.Text
		searchText := pgtype.Text{String: search, Valid: true}

		addons, err = s.db.SearchAddons(ctx, database.SearchAddonsParams{
			Limit:   int32(perPage), //nolint:gosec // perPage validated to be <= 100
			Offset:  int32(offset),  //nolint:gosec // offset validated via perPage <= 100
			Column3: searchText,
		})
		if err != nil {
			slog.Error("failed to search addons", "error", err)
			respondInternalError(c)
			return
		}
		total, err = s.db.CountSearchAddons(ctx, searchText)
	} else if categoryStr != "" {
		// Filter by category
		categoryID, parseErr := strconv.ParseInt(categoryStr, 10, 32)
		if parseErr != nil {
			// Invalid category - use -1 which will match nothing (lenient behavior)
			categoryID = -1
		}

		addons, err = s.db.ListAddonsByCategory(ctx, database.ListAddonsByCategoryParams{
			Limit:   int32(perPage),    //nolint:gosec // perPage validated to be <= 100
			Offset:  int32(offset),     //nolint:gosec // offset validated via perPage <= 100
			Column3: int32(categoryID), //nolint:gosec // validated via ParseInt
		})
		if err != nil {
			slog.Error("failed to list addons by category", "error", err)
			respondInternalError(c)
			return
		}
		total, err = s.db.CountAddonsByCategory(ctx, int32(categoryID)) //nolint:gosec // validated via ParseInt
	} else {
		addons, err = s.db.ListAddons(ctx, database.ListAddonsParams{
			Limit:  int32(perPage), //nolint:gosec // perPage validated to be <= 100
			Offset: int32(offset),  //nolint:gosec // offset validated via perPage <= 100
		})
		if err != nil {
			slog.Error("failed to list addons", "error", err)
			respondInternalError(c)
			return
		}
		total, err = s.db.CountActiveAddons(ctx)
	}

	if err != nil {
		slog.Error("failed to count addons", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]AddonResponse, len(addons))
	for i, a := range addons {
		response[i] = addonToResponse(a)
	}

	respondWithPagination(c, response, page, perPage, int(total))
}

func (s *Server) handleGetAddon(c *gin.Context) {
	slug := c.Param("slug")
	ctx := c.Request.Context()

	addon, err := s.db.GetAddonBySlug(ctx, slug)
	if err != nil {
		respondNotFound(c, "Addon not found")
		return
	}

	respondWithData(c, addonToResponse(addon))
}

type SnapshotResponse struct {
	RecordedAt     string `json:"recorded_at"`
	DownloadCount  int64  `json:"download_count"`
	ThumbsUpCount  int32  `json:"thumbs_up_count,omitempty"`
	PopularityRank int32  `json:"popularity_rank,omitempty"`
}

func (s *Server) handleGetAddonHistory(c *gin.Context) {
	slug := c.Param("slug")
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "168")) // Default 7 days of hourly data
	if err != nil {
		limit = 168
	}
	if limit < 1 || limit > 720 {
		limit = 168
	}

	ctx := c.Request.Context()

	addon, err := s.db.GetAddonBySlug(ctx, slug)
	if err != nil {
		respondNotFound(c, "Addon not found")
		return
	}

	snapshots, err := s.db.GetAddonSnapshots(ctx, database.GetAddonSnapshotsParams{
		AddonID: addon.ID,
		Limit:   int32(limit), //nolint:gosec // limit validated to be <= 720
	})
	if err != nil {
		slog.Error("failed to get snapshots", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]SnapshotResponse, len(snapshots))
	for i, snap := range snapshots {
		response[i] = SnapshotResponse{
			RecordedAt:    snap.RecordedAt.Time.Format("2006-01-02T15:04:05Z"),
			DownloadCount: snap.DownloadCount,
		}
		if snap.ThumbsUpCount.Valid {
			response[i].ThumbsUpCount = snap.ThumbsUpCount.Int32
		}
		if snap.PopularityRank.Valid {
			response[i].PopularityRank = snap.PopularityRank.Int32
		}
	}

	respondWithData(c, response)
}

type CategoryResponse struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ParentID int32  `json:"parent_id,omitempty"`
	IconURL  string `json:"icon_url,omitempty"`
}

func (s *Server) handleListCategories(c *gin.Context) {
	ctx := c.Request.Context()

	categories, err := s.db.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		respondInternalError(c)
		return
	}

	response := make([]CategoryResponse, len(categories))
	for i, cat := range categories {
		response[i] = CategoryResponse{
			ID:   cat.ID,
			Name: cat.Name,
			Slug: cat.Slug,
		}
		if cat.ParentID.Valid {
			response[i].ParentID = cat.ParentID.Int32
		}
		if cat.IconUrl.Valid {
			response[i].IconURL = cat.IconUrl.String
		}
	}

	respondWithData(c, response)
}

func (s *Server) handleTrendingHot(c *gin.Context) {
	page, perPage, offset := parsePaginationParams(c)
	ctx := c.Request.Context()

	total, err := s.db.CountHotAddons(ctx)
	if err != nil {
		slog.Error("failed to count hot addons", "error", err)
		respondInternalError(c)
		return
	}

	addons, err := s.db.ListHotAddonsPaginated(ctx, database.ListHotAddonsPaginatedParams{
		Limit:  int32(perPage), //nolint:gosec // perPage validated to be <= 100
		Offset: int32(offset),  //nolint:gosec // offset validated via perPage <= 100
	})
	if err != nil {
		slog.Error("failed to get hot addons", "error", err)
		respondInternalError(c)
		return
	}

	rankChanges, err := s.db.GetRankChanges(ctx)
	if err != nil {
		slog.Error("failed to get rank changes", "error", err)
		respondInternalError(c)
		return
	}
	rankChangeMap := buildRankChangeMap(rankChanges, "hot")

	response := make([]TrendingAddonResponse, len(addons))
	for i, a := range addons {
		response[i] = TrendingAddonResponse{
			AddonResponse: addonToResponse(database.Addon{
				ID: a.ID, Name: a.Name, Slug: a.Slug, Summary: a.Summary,
				AuthorName: a.AuthorName, LogoUrl: a.LogoUrl, DownloadCount: a.DownloadCount,
				ThumbsUpCount: a.ThumbsUpCount, PopularityRank: a.PopularityRank,
				GameVersions: a.GameVersions, LastUpdatedAt: a.LastUpdatedAt,
			}),
			Rank:             offset + i + 1,
			Score:            numericToFloat64(a.HotScore),
			DownloadVelocity: numericToFloat64(a.DownloadVelocity),
		}
		if rc, ok := rankChangeMap[a.ID]; ok {
			applyRankChanges(&response[i], rc)
		}
	}

	respondWithPagination(c, response, page, perPage, int(total))
}

func (s *Server) handleTrendingRising(c *gin.Context) {
	page, perPage, offset := parsePaginationParams(c)
	ctx := c.Request.Context()

	total, err := s.db.CountRisingAddons(ctx)
	if err != nil {
		slog.Error("failed to count rising addons", "error", err)
		respondInternalError(c)
		return
	}

	addons, err := s.db.ListRisingAddonsPaginated(ctx, database.ListRisingAddonsPaginatedParams{
		Limit:  int32(perPage), //nolint:gosec // perPage validated to be <= 100
		Offset: int32(offset),  //nolint:gosec // offset validated via perPage <= 100
	})
	if err != nil {
		slog.Error("failed to get rising addons", "error", err)
		respondInternalError(c)
		return
	}

	rankChanges, err := s.db.GetRankChanges(ctx)
	if err != nil {
		slog.Error("failed to get rank changes", "error", err)
		respondInternalError(c)
		return
	}
	rankChangeMap := buildRankChangeMap(rankChanges, "rising")

	response := make([]TrendingAddonResponse, len(addons))
	for i, a := range addons {
		response[i] = TrendingAddonResponse{
			AddonResponse: addonToResponse(database.Addon{
				ID: a.ID, Name: a.Name, Slug: a.Slug, Summary: a.Summary,
				AuthorName: a.AuthorName, LogoUrl: a.LogoUrl, DownloadCount: a.DownloadCount,
				ThumbsUpCount: a.ThumbsUpCount, PopularityRank: a.PopularityRank,
				GameVersions: a.GameVersions, LastUpdatedAt: a.LastUpdatedAt,
			}),
			Rank:             offset + i + 1,
			Score:            numericToFloat64(a.RisingScore),
			DownloadVelocity: numericToFloat64(a.DownloadVelocity),
		}
		if rc, ok := rankChangeMap[a.ID]; ok {
			applyRankChanges(&response[i], rc)
		}
	}

	respondWithPagination(c, response, page, perPage, int(total))
}
