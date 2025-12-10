package api

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"addon-radar/internal/database"
)

type AddonResponse struct {
	ID              int32    `json:"id"`
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	Summary         string   `json:"summary,omitempty"`
	AuthorName      string   `json:"author_name,omitempty"`
	LogoURL         string   `json:"logo_url,omitempty"`
	DownloadCount   int64    `json:"download_count"`
	ThumbsUpCount   int32    `json:"thumbs_up_count"`
	PopularityRank  int32    `json:"popularity_rank,omitempty"`
	GameVersions    []string `json:"game_versions"`
	LastUpdatedAt   string   `json:"last_updated_at,omitempty"`
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

func (s *Server) handleListAddons(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	ctx := c.Request.Context()

	var addons []database.Addon
	var total int64
	var err error

	if search != "" {
		// Convert search string to pgtype.Text
		searchText := pgtype.Text{String: search, Valid: true}

		addons, err = s.db.SearchAddons(ctx, database.SearchAddonsParams{
			Limit:   int32(perPage),
			Offset:  int32(offset),
			Column3: searchText,
		})
		if err != nil {
			slog.Error("failed to search addons", "error", err)
			respondInternalError(c)
			return
		}
		total, err = s.db.CountSearchAddons(ctx, searchText)
	} else {
		addons, err = s.db.ListAddons(ctx, database.ListAddonsParams{
			Limit:  int32(perPage),
			Offset: int32(offset),
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
