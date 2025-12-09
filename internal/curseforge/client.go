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

// SearchMods searches for mods with pagination
func (c *Client) SearchMods(ctx context.Context, gameID, index, pageSize int) (*SearchModsResponse, error) {
	query := url.Values{}
	query.Set("gameId", strconv.Itoa(gameID))
	query.Set("index", strconv.Itoa(index))
	query.Set("pageSize", strconv.Itoa(pageSize))
	query.Set("sortField", "2") // 2 = popularity
	query.Set("sortOrder", "desc")

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

// GetAllWoWAddons fetches all WoW addons with pagination
func (c *Client) GetAllWoWAddons(ctx context.Context) ([]Mod, error) {
	var allMods []Mod
	pageSize := 50
	index := 0

	for {
		slog.Info("fetching addons page", "index", index, "pageSize", pageSize)

		resp, err := c.SearchMods(ctx, GameIDWoW, index, pageSize)
		if err != nil {
			return nil, fmt.Errorf("fetch page at index %d: %w", index, err)
		}

		allMods = append(allMods, resp.Data...)

		slog.Info("fetched page",
			"count", len(resp.Data),
			"total", len(allMods),
			"totalAvailable", resp.Pagination.TotalCount,
		)

		// Check if we've fetched all results
		if len(resp.Data) < pageSize || index+pageSize >= resp.Pagination.TotalCount {
			break
		}

		index += pageSize

		// Small delay to be nice to the API
		time.Sleep(100 * time.Millisecond)
	}

	return allMods, nil
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
