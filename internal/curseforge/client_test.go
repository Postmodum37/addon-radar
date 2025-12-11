package curseforge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchMods(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		response := SearchModsResponse{
			Data: []Mod{
				{ID: 1, Name: "Test Addon", Slug: "test-addon"},
				{ID: 2, Name: "Another Addon", Slug: "another-addon"},
			},
			Pagination: Pagination{
				Index:       0,
				PageSize:    50,
				ResultCount: 2,
				TotalCount:  2,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/mods/search", r.URL.Path)
			assert.Equal(t, "fake-key", r.Header.Get("x-api-key"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			// Verify query params
			assert.Equal(t, "1", r.URL.Query().Get("gameId"))
			assert.Equal(t, "50", r.URL.Query().Get("pageSize"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("fake-key")
		client.baseURL = server.URL

		result, err := client.SearchMods(context.Background(), SearchModsParams{
			GameID:   GameIDWoW,
			PageSize: 50,
		})

		require.NoError(t, err)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, "test-addon", result.Data[0].Slug)
		assert.Equal(t, 2, result.Pagination.TotalCount)
	})

	t.Run("error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
		}))
		defer server.Close()

		client := NewClient("fake-key")
		client.baseURL = server.URL

		_, err := client.SearchMods(context.Background(), SearchModsParams{
			GameID:   GameIDWoW,
			PageSize: 50,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 500")
	})

	t.Run("rate limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limit exceeded"}`))
		}))
		defer server.Close()

		client := NewClient("fake-key")
		client.baseURL = server.URL

		_, err := client.SearchMods(context.Background(), SearchModsParams{
			GameID:   GameIDWoW,
			PageSize: 50,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 429")
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{invalid json}`))
		}))
		defer server.Close()

		client := NewClient("fake-key")
		client.baseURL = server.URL

		_, err := client.SearchMods(context.Background(), SearchModsParams{
			GameID:   GameIDWoW,
			PageSize: 50,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal")
	})

	t.Run("with game version filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "517", r.URL.Query().Get("gameVersionTypeId"))

			response := SearchModsResponse{
				Data:       []Mod{},
				Pagination: Pagination{TotalCount: 0},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("fake-key")
		client.baseURL = server.URL

		_, err := client.SearchMods(context.Background(), SearchModsParams{
			GameID:            GameIDWoW,
			GameVersionTypeID: GameVersionTypeRetail,
			PageSize:          50,
		})

		require.NoError(t, err)
	})
}

func TestGetCategories(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		response := GetCategoriesResponse{
			Data: []Category{
				{ID: 1001, Name: "Bags & Inventory", Slug: "bags-inventory"},
				{ID: 1002, Name: "Action Bars", Slug: "action-bars"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/categories", r.URL.Path)
			assert.Equal(t, "1", r.URL.Query().Get("gameId"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("fake-key")
		client.baseURL = server.URL

		categories, err := client.GetCategories(context.Background(), GameIDWoW)

		require.NoError(t, err)
		assert.Len(t, categories, 2)
		assert.Equal(t, "Bags & Inventory", categories[0].Name)
	})

	t.Run("error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "forbidden"}`))
		}))
		defer server.Close()

		client := NewClient("fake-key")
		client.baseURL = server.URL

		_, err := client.GetCategories(context.Background(), GameIDWoW)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 403")
	})
}

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key")

	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, BaseURL, client.baseURL)
	assert.NotNil(t, client.httpClient)
}
