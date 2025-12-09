package curseforge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is a CurseForge API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new CurseForge API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: BaseURL,
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// SearchModsParams configures the search query
type SearchModsParams struct {
	GameID            int
	GameVersionTypeID int // 0 means no filter
	SortField         int
	Index             int
	PageSize          int
}

// SearchMods searches for mods with pagination and filters
func (c *Client) SearchMods(ctx context.Context, params SearchModsParams) (*SearchModsResponse, error) {
	query := url.Values{}
	query.Set("gameId", strconv.Itoa(params.GameID))
	query.Set("index", strconv.Itoa(params.Index))
	query.Set("pageSize", strconv.Itoa(params.PageSize))
	query.Set("sortField", strconv.Itoa(params.SortField))
	query.Set("sortOrder", "desc")

	if params.GameVersionTypeID > 0 {
		query.Set("gameVersionTypeId", strconv.Itoa(params.GameVersionTypeID))
	}

	body, err := c.doRequest(ctx, http.MethodGet, "/v1/mods/search", query)
	if err != nil {
		return nil, fmt.Errorf("search mods: %w", err)
	}

	var result SearchModsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// GetAllAddonsForVersion fetches all addons for a specific game version type
// Uses multiple sort orders to overcome the 10k result limit
func (c *Client) GetAllAddonsForVersion(ctx context.Context, gameVersionTypeID int) ([]Mod, error) {
	seen := make(map[int]bool)
	var allMods []Mod

	// Use multiple sort orders to get different subsets of addons
	sortOrders := []struct {
		field int
		name  string
	}{
		{SortFieldPopularity, "popularity"},
		{SortFieldLastUpdated, "lastUpdated"},
		{SortFieldTotalDownloads, "totalDownloads"},
	}

	for _, sort := range sortOrders {
		slog.Info("fetching addons", "sortBy", sort.name, "gameVersionTypeId", gameVersionTypeID)

		mods, err := c.fetchWithSort(ctx, gameVersionTypeID, sort.field)
		if err != nil {
			return nil, fmt.Errorf("fetch by %s: %w", sort.name, err)
		}

		// Deduplicate
		newCount := 0
		for _, mod := range mods {
			if !seen[mod.ID] {
				seen[mod.ID] = true
				allMods = append(allMods, mod)
				newCount++
			}
		}

		slog.Info("fetched addons",
			"sortBy", sort.name,
			"fetched", len(mods),
			"new", newCount,
			"totalUnique", len(allMods),
		)
	}

	return allMods, nil
}

// fetchWithSort fetches up to 10k addons using a specific sort order
func (c *Client) fetchWithSort(ctx context.Context, gameVersionTypeID, sortField int) ([]Mod, error) {
	var mods []Mod
	pageSize := 50
	index := 0

	for {
		params := SearchModsParams{
			GameID:            GameIDWoW,
			GameVersionTypeID: gameVersionTypeID,
			SortField:         sortField,
			Index:             index,
			PageSize:          pageSize,
		}

		resp, err := c.SearchMods(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("fetch page at index %d: %w", index, err)
		}

		mods = append(mods, resp.Data...)

		// Check if we've fetched all results
		if len(resp.Data) < pageSize || index+pageSize >= resp.Pagination.TotalCount {
			break
		}

		// CurseForge API has a hard limit of 10,000 results
		if index+pageSize >= MaxSearchResults {
			slog.Info("reached API limit",
				"sortField", sortField,
				"fetched", len(mods),
				"totalAvailable", resp.Pagination.TotalCount,
			)
			break
		}

		index += pageSize

		// Small delay to be nice to the API
		time.Sleep(50 * time.Millisecond)
	}

	return mods, nil
}

// GetAllWoWAddons fetches all WoW Retail addons (convenience method)
func (c *Client) GetAllWoWAddons(ctx context.Context) ([]Mod, error) {
	return c.GetAllAddonsForVersion(ctx, GameVersionTypeRetail)
}

// GetCategories fetches all categories for a game
func (c *Client) GetCategories(ctx context.Context, gameID int) ([]Category, error) {
	query := url.Values{}
	query.Set("gameId", strconv.Itoa(gameID))

	body, err := c.doRequest(ctx, http.MethodGet, "/v1/categories", query)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}

	var result GetCategoriesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal categories: %w", err)
	}

	return result.Data, nil
}
