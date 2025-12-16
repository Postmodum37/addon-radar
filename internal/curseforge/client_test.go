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

// newTestClient creates a client with no backoff delay for fast tests
func newTestClient(apiKey string) *Client {
	c := NewClient(apiKey)
	c.backoffMultiply = 0 // No delay between retries in tests
	return c
}

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

		client := newTestClient("fake-key")
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

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{invalid json}`))
		}))
		defer server.Close()

		client := newTestClient("fake-key")
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

		client := newTestClient("fake-key")
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

		client := newTestClient("fake-key")
		client.baseURL = server.URL

		categories, err := client.GetCategories(context.Background(), GameIDWoW)

		require.NoError(t, err)
		assert.Len(t, categories, 2)
		assert.Equal(t, "Bags & Inventory", categories[0].Name)
	})

	t.Run("client error no retry", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "forbidden"}`))
		}))
		defer server.Close()

		client := newTestClient("fake-key")
		client.baseURL = server.URL

		_, err := client.GetCategories(context.Background(), GameIDWoW)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 403:")
		assert.Equal(t, 1, attempts) // Should not retry on 403
	})
}

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key")

	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, BaseURL, client.baseURL)
	assert.NotNil(t, client.httpClient)
}

func TestDoRequest_RetryBehavior(t *testing.T) {
	t.Run("retries on server error and eventually succeeds", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":[]}`))
		}))
		defer server.Close()

		client := newTestClient("test-key")
		client.baseURL = server.URL

		body, err := client.doRequest(context.Background(), "GET", "/test", nil)

		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
		assert.Contains(t, string(body), "data")
	})

	t.Run("no retry on client error", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"bad request"}`))
		}))
		defer server.Close()

		client := newTestClient("test-key")
		client.baseURL = server.URL

		_, err := client.doRequest(context.Background(), "GET", "/test", nil)

		require.Error(t, err)
		assert.Equal(t, 1, attempts) // No retries for 4xx errors
	})

	t.Run("retries on 429 rate limit", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":[]}`))
		}))
		defer server.Close()

		client := newTestClient("test-key")
		client.baseURL = server.URL

		body, err := client.doRequest(context.Background(), "GET", "/test", nil)

		require.NoError(t, err)
		assert.Equal(t, 2, attempts) // 429 should retry
		assert.Contains(t, string(body), "data")
	})

	t.Run("gives up after max retries", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := newTestClient("test-key")
		client.baseURL = server.URL

		_, err := client.doRequest(context.Background(), "GET", "/test", nil)

		require.Error(t, err)
		assert.Equal(t, 4, attempts) // 1 initial + 3 retries
		assert.Contains(t, err.Error(), "failed after 3 retries")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := newTestClient("test-key")
		client.baseURL = server.URL

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := client.doRequest(ctx, "GET", "/test", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestHTTPError(t *testing.T) {
	err := &HTTPError{StatusCode: 404, Body: "not found"}
	assert.Equal(t, "HTTP 404: not found", err.Error())
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"400 is client error", &HTTPError{StatusCode: 400}, true},
		{"401 is client error", &HTTPError{StatusCode: 401}, true},
		{"403 is client error", &HTTPError{StatusCode: 403}, true},
		{"404 is client error", &HTTPError{StatusCode: 404}, true},
		{"429 is NOT client error (rate limit)", &HTTPError{StatusCode: 429}, false},
		{"500 is NOT client error", &HTTPError{StatusCode: 500}, false},
		{"502 is NOT client error", &HTTPError{StatusCode: 502}, false},
		{"nil error is NOT client error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isClientError(tt.err))
		})
	}
}
