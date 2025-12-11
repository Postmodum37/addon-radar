package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
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
		req, _ := http.NewRequest("GET", "/api/v1/addons/test-addon", nil)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "test-addon", data["slug"])
		assert.Equal(t, "Test Addon", data["name"])
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/addons/nonexistent", nil)
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
			ID:   int32(i),
			Slug: "addon-" + string(rune('a'+i-1)),
			Name: "Addon " + string(rune('A'+i-1)),
		})
		require.NoError(t, err)
	}

	server := NewServer(tdb.Queries)

	t.Run("default pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/addons", nil)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 1, resp.Meta.Page)
		assert.Equal(t, 20, resp.Meta.PerPage)
		assert.Equal(t, 25, resp.Meta.Total)
		assert.Equal(t, 2, resp.Meta.TotalPages)
	})

	t.Run("custom page size", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/addons?per_page=10&page=2", nil)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp PaginatedResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 2, resp.Meta.Page)
		assert.Equal(t, 10, resp.Meta.PerPage)
	})

	t.Run("search", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/addons?search=Addon%20A", nil)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		// Search should return results containing "Addon A"
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
	req, _ := http.NewRequest("GET", "/api/v1/categories", nil)
	server.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].([]interface{})
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
	req, _ := http.NewRequest("GET", "/api/v1/trending/hot", nil)
	server.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].([]interface{})
	if assert.Len(t, data, 1) {
		addon := data[0].(map[string]interface{})
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
	req, _ := http.NewRequest("GET", "/api/v1/trending/rising", nil)
	server.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].([]interface{})
	if assert.Len(t, data, 1) {
		addon := data[0].(map[string]interface{})
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
		req, _ := http.NewRequest("GET", "/api/v1/addons/history-addon/history", nil)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].([]interface{})
		assert.Len(t, data, 5)
	})

	t.Run("addon not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/addons/nonexistent/history", nil)
		server.ServeHTTP(w, req)

		assert.Equal(t, 404, w.Code)
	})

	t.Run("respects limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/addons/history-addon/history?limit=2", nil)
		server.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].([]interface{})
		assert.Len(t, data, 2)
	})
}

func TestCORS(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	server := NewServer(tdb.Queries)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/v1/health", nil)
	server.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
