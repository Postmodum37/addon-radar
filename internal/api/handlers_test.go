package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"addon-radar/internal/database"
	"addon-radar/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealth(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	server := NewServer(tdb.Queries)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	require.NoError(t, err)
	server.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestGetAddon(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed test addon
	err := tdb.Queries.UpsertAddon(ctx, database.UpsertAddonParams{
		ID:   123,
		Slug: "test-addon",
		Name: "Test Addon",
	})
	require.NoError(t, err)

	server := NewServer(tdb.Queries)

	t.Run("existing addon", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons/test-addon", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-addon", data["slug"])
		assert.Equal(t, "Test Addon", data["name"])
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons/nonexistent", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 404, w.Code)
		assert.Contains(t, w.Body.String(), "not_found")
	})
}

func TestListAddons(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed test addons
	for i := 1; i <= 25; i++ {
		err := tdb.Queries.UpsertAddon(ctx, database.UpsertAddonParams{
			ID:   int32(i), //nolint:gosec // Test data with small known values
			Slug: "addon-" + string(rune('a'+i-1)),
			Name: "Addon " + string(rune('A'+i-1)),
		})
		require.NoError(t, err)
	}

	server := NewServer(tdb.Queries)

	t.Run("default pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 1, resp.Meta.Page)
		assert.Equal(t, 20, resp.Meta.PerPage)
		assert.Equal(t, 25, resp.Meta.Total)
		assert.Equal(t, 2, resp.Meta.TotalPages)
	})

	t.Run("custom page size", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons?per_page=10&page=2", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 2, resp.Meta.Page)
		assert.Equal(t, 10, resp.Meta.PerPage)
	})

	t.Run("search", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons?search=Addon%20A", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		// Search should return results containing "Addon A"
	})
}

func TestListAddonsByCategory(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed addons with different categories
	// Category 100: addons 1-5
	// Category 200: addons 6-10
	// No category: addons 11-15
	for i := 1; i <= 15; i++ {
		var categories []int32
		if i <= 5 {
			categories = []int32{100}
		} else if i <= 10 {
			categories = []int32{200}
		}

		_, err := tdb.Pool.Exec(ctx, `
			INSERT INTO addons (id, slug, name, status, categories, download_count)
			VALUES ($1, $2, $3, 'active', $4, $5)
		`, i, fmt.Sprintf("addon-%d", i), fmt.Sprintf("Addon %d", i), categories, 1000-i)
		require.NoError(t, err)
	}

	server := NewServer(tdb.Queries)

	t.Run("filter by valid category", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons?category=100", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 5, resp.Meta.Total)
	})

	t.Run("filter by different category", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons?category=200", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 5, resp.Meta.Total)
	})

	t.Run("invalid category returns empty results", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons?category=abc", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 0, resp.Meta.Total)
	})

	t.Run("non-existent category returns empty results", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons?category=99999", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 0, resp.Meta.Total)
	})

	t.Run("pagination works with category filter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons?category=100&per_page=2&page=2", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 2, resp.Meta.Page)
		assert.Equal(t, 2, resp.Meta.PerPage)
		assert.Equal(t, 5, resp.Meta.Total)
		assert.Equal(t, 3, resp.Meta.TotalPages)
	})
}

func TestListCategories(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed test categories
	err := tdb.Queries.UpsertCategory(ctx, database.UpsertCategoryParams{
		ID:   1001,
		Name: "Test Category",
		Slug: "test-category",
	})
	require.NoError(t, err)

	server := NewServer(tdb.Queries)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/v1/categories", nil)
	require.NoError(t, err)
	server.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data, ok := resp["data"].([]interface{})
	require.True(t, ok)
	assert.Len(t, data, 1)
}

func TestTrendingHot(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed addon with required fields for hot query (status=active, download_count>=500)
	_, err := tdb.Pool.Exec(ctx, `
		INSERT INTO addons (id, slug, name, status, download_count)
		VALUES ($1, $2, $3, 'active', 1000)
	`, 123, "hot-addon", "Hot Addon")
	require.NoError(t, err)

	// Insert trending score using raw SQL
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO trending_scores (addon_id, hot_score, rising_score)
		VALUES ($1, 100.5, 0)
	`, 123)
	require.NoError(t, err)

	server := NewServer(tdb.Queries)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/v1/trending/hot", nil)
	require.NoError(t, err)
	server.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data, ok := resp["data"].([]interface{})
	require.True(t, ok)
	if assert.Len(t, data, 1) {
		addon, ok := data[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "hot-addon", addon["slug"])
		assert.InDelta(t, 100.5, addon["score"], 0.1)
	}
}

func TestTrendingRising(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed addon with required fields for rising query (status=active, 50<=download_count<=10000)
	_, err := tdb.Pool.Exec(ctx, `
		INSERT INTO addons (id, slug, name, status, download_count)
		VALUES ($1, $2, $3, 'active', 500)
	`, 456, "rising-addon", "Rising Addon")
	require.NoError(t, err)

	// Insert trending score (rising_score > 0, hot_score = 0)
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO trending_scores (addon_id, hot_score, rising_score)
		VALUES ($1, 0, 50.25)
	`, 456)
	require.NoError(t, err)

	server := NewServer(tdb.Queries)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/v1/trending/rising", nil)
	require.NoError(t, err)
	server.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data, ok := resp["data"].([]interface{})
	require.True(t, ok)
	if assert.Len(t, data, 1) {
		addon, ok := data[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "rising-addon", addon["slug"])
		assert.InDelta(t, 50.25, addon["score"], 0.1)
	}
}

func TestGetAddonHistory(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed addon
	err := tdb.Queries.UpsertAddon(ctx, database.UpsertAddonParams{
		ID:   789,
		Slug: "history-addon",
		Name: "History Addon",
	})
	require.NoError(t, err)

	// Add some snapshots
	for i := 0; i < 5; i++ {
		err = tdb.Queries.CreateSnapshot(ctx, database.CreateSnapshotParams{
			AddonID:       789,
			DownloadCount: int64(1000 + i*100),
		})
		require.NoError(t, err)
	}

	server := NewServer(tdb.Queries)

	t.Run("returns history", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons/history-addon/history", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp["data"].([]interface{})
		require.True(t, ok)
		assert.Len(t, data, 5)
	})

	t.Run("addon not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons/nonexistent/history", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 404, w.Code)
	})

	t.Run("respects limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/addons/history-addon/history?limit=2", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp["data"].([]interface{})
		require.True(t, ok)
		assert.Len(t, data, 2)
	})
}

func TestCORS(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	server := NewServer(tdb.Queries)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("OPTIONS", "/api/v1/health", nil)
	require.NoError(t, err)
	server.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name            string
		queryParams     map[string]string
		expectedPage    int
		expectedPerPage int
		expectedOffset  int
	}{
		{"defaults", nil, 1, 20, 0},
		{"page 2", map[string]string{"page": "2"}, 2, 20, 20},
		{"custom per_page", map[string]string{"per_page": "50"}, 1, 50, 0},
		{"page 3 with 10 per page", map[string]string{"page": "3", "per_page": "10"}, 3, 10, 20},
		{"negative page defaults to 1", map[string]string{"page": "-5"}, 1, 20, 0},
		{"zero page defaults to 1", map[string]string{"page": "0"}, 1, 20, 0},
		{"per_page exceeds max resets to 20", map[string]string{"per_page": "500"}, 1, 20, 0},
		{"per_page zero defaults to 20", map[string]string{"per_page": "0"}, 1, 20, 0},
		{"per_page negative defaults to 20", map[string]string{"per_page": "-10"}, 1, 20, 0},
		{"invalid page string defaults to 1", map[string]string{"page": "abc"}, 1, 20, 0},
		{"invalid per_page string defaults to 20", map[string]string{"per_page": "xyz"}, 1, 20, 0},
		{"per_page at max boundary", map[string]string{"per_page": "100"}, 1, 100, 0},
		{"per_page just over max", map[string]string{"per_page": "101"}, 1, 20, 0},
		{"large page number", map[string]string{"page": "100", "per_page": "50"}, 100, 50, 4950},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build query string
			url := "/test"
			if len(tt.queryParams) > 0 {
				url += "?"
				first := true
				for k, v := range tt.queryParams {
					if !first {
						url += "&"
					}
					url += k + "=" + v
					first = false
				}
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			// Create gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			page, perPage, offset := parsePaginationParams(c)

			assert.Equal(t, tt.expectedPage, page, "page mismatch")
			assert.Equal(t, tt.expectedPerPage, perPage, "perPage mismatch")
			assert.Equal(t, tt.expectedOffset, offset, "offset mismatch")
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestApplyRankChanges(t *testing.T) {
	tests := []struct {
		name              string
		currentRank       int16
		rank24hAgo        pgtype.Int2
		rank7dAgo         pgtype.Int2
		expected24hChange *int
		expected7dChange  *int
	}{
		{
			"both invalid (new addon)",
			5,
			pgtype.Int2{Valid: false},
			pgtype.Int2{Valid: false},
			nil, nil,
		},
		{
			"24h valid only",
			5,
			pgtype.Int2{Int16: 10, Valid: true},
			pgtype.Int2{Valid: false},
			intPtr(5), nil,
		},
		{
			"7d valid only",
			5,
			pgtype.Int2{Valid: false},
			pgtype.Int2{Int16: 3, Valid: true},
			nil, intPtr(-2),
		},
		{
			"both valid - ranking improved",
			5,
			pgtype.Int2{Int16: 10, Valid: true},
			pgtype.Int2{Int16: 15, Valid: true},
			intPtr(5), intPtr(10),
		},
		{
			"both valid - ranking declined",
			10,
			pgtype.Int2{Int16: 5, Valid: true},
			pgtype.Int2{Int16: 3, Valid: true},
			intPtr(-5), intPtr(-7),
		},
		{
			"unchanged position",
			5,
			pgtype.Int2{Int16: 5, Valid: true},
			pgtype.Int2{Int16: 5, Valid: true},
			intPtr(0), intPtr(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &TrendingAddonResponse{}
			rc := database.GetRankChangesRow{
				CurrentRank: tt.currentRank,
				Rank24hAgo:  tt.rank24hAgo,
				Rank7dAgo:   tt.rank7dAgo,
			}

			applyRankChanges(resp, rc)

			if tt.expected24hChange == nil {
				assert.Nil(t, resp.RankChange24h)
			} else {
				require.NotNil(t, resp.RankChange24h)
				assert.Equal(t, *tt.expected24hChange, *resp.RankChange24h)
			}

			if tt.expected7dChange == nil {
				assert.Nil(t, resp.RankChange7d)
			} else {
				require.NotNil(t, resp.RankChange7d)
				assert.Equal(t, *tt.expected7dChange, *resp.RankChange7d)
			}
		})
	}
}

func TestBuildRankChangeMap(t *testing.T) {
	rankChanges := []database.GetRankChangesRow{
		{AddonID: 1, Category: "hot", CurrentRank: 1},
		{AddonID: 2, Category: "rising", CurrentRank: 1},
		{AddonID: 3, Category: "hot", CurrentRank: 2},
		{AddonID: 4, Category: "rising", CurrentRank: 2},
	}

	t.Run("filters by hot category", func(t *testing.T) {
		hotMap := buildRankChangeMap(rankChanges, "hot")
		assert.Len(t, hotMap, 2)
		assert.Contains(t, hotMap, int32(1))
		assert.Contains(t, hotMap, int32(3))
		assert.NotContains(t, hotMap, int32(2))
	})

	t.Run("filters by rising category", func(t *testing.T) {
		risingMap := buildRankChangeMap(rankChanges, "rising")
		assert.Len(t, risingMap, 2)
		assert.Contains(t, risingMap, int32(2))
		assert.Contains(t, risingMap, int32(4))
	})

	t.Run("empty for unknown category", func(t *testing.T) {
		unknownMap := buildRankChangeMap(rankChanges, "unknown")
		assert.Empty(t, unknownMap)
	})

	t.Run("empty input returns empty map", func(t *testing.T) {
		emptyMap := buildRankChangeMap(nil, "hot")
		assert.Empty(t, emptyMap)
	})
}

func makeNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	err := n.Scan(f)
	if err != nil {
		// Scan from string as fallback
		//nolint:errcheck // Test helper, panic is acceptable
		n.Scan(fmt.Sprintf("%v", f))
	}
	return n
}

func TestNumericToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    pgtype.Numeric
		expected float64
	}{
		{"invalid returns 0", pgtype.Numeric{Valid: false}, 0},
		{"valid positive", makeNumeric(123.45), 123.45},
		{"valid zero", makeNumeric(0), 0},
		{"valid negative", makeNumeric(-50.5), -50.5},
		{"small decimal", makeNumeric(0.001), 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := numericToFloat64(tt.input)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestTrendingHotPagination(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Seed 25 hot addons
	for i := 1; i <= 25; i++ {
		_, err := tdb.Pool.Exec(ctx, `
			INSERT INTO addons (id, slug, name, status, download_count)
			VALUES ($1, $2, $3, 'active', 1000)
		`, i, "hot-addon-"+string(rune('a'+i-1)), "Hot Addon "+string(rune('A'+i-1)))
		require.NoError(t, err)

		// Insert trending score - higher score for lower ID to get predictable order
		_, err = tdb.Pool.Exec(ctx, `
			INSERT INTO trending_scores (addon_id, hot_score, rising_score)
			VALUES ($1, $2, 0)
		`, i, 100-i)
		require.NoError(t, err)
	}

	server := NewServer(tdb.Queries)

	t.Run("returns pagination metadata", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/trending/hot?per_page=10", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 1, resp.Meta.Page)
		assert.Equal(t, 10, resp.Meta.PerPage)
		assert.Equal(t, 25, resp.Meta.Total)
		assert.Equal(t, 3, resp.Meta.TotalPages)
	})

	t.Run("page 1 starts at rank 1", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/trending/hot?per_page=10", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp["data"].([]interface{})
		require.True(t, ok, "data should be array")
		require.Len(t, data, 10)

		firstAddon, ok := data[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), firstAddon["rank"])

		lastAddon, ok := data[9].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(10), lastAddon["rank"])
	})

	t.Run("page 2 has correct ranks starting at 11", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/trending/hot?page=2&per_page=10", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp["data"].([]interface{})
		require.True(t, ok)
		require.Len(t, data, 10)

		// Page 2, first item should have rank 11
		firstAddon, ok := data[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(11), firstAddon["rank"])

		lastAddon, ok := data[9].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(20), lastAddon["rank"])
	})

	t.Run("page 3 has remaining 5 items", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/trending/hot?page=3&per_page=10", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp["data"].([]interface{})
		require.True(t, ok)
		assert.Len(t, data, 5) // Only 5 remaining

		firstAddon, ok := data[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(21), firstAddon["rank"])
	})

	t.Run("response includes rank change fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/v1/trending/hot?per_page=1", nil)
		require.NoError(t, err)
		server.ServeHTTP(w, req)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp["data"].([]interface{})
		require.True(t, ok)
		require.Len(t, data, 1)

		addon, ok := data[0].(map[string]interface{})
		require.True(t, ok)

		// These fields should exist (may be null for new addons)
		_, hasRank := addon["rank"]
		_, hasScore := addon["score"]
		_, hasVelocity := addon["download_velocity"]

		assert.True(t, hasRank, "response should include rank")
		assert.True(t, hasScore, "response should include score")
		assert.True(t, hasVelocity, "response should include download_velocity")
	})
}
