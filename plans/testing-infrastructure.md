# Testing Infrastructure - Simplified Plan

## Overview

Add automated tests and CI to prevent regressions. Simple approach: real Postgres, stdlib httptest, minimal dependencies.

**Time estimate**: 4-5 hours
**Dependencies**: testcontainers-go (for real Postgres in tests)

## Strategy

Use **real PostgreSQL** for all tests via testcontainers. No mocks. The existing `internal/trending/trending_test.go` is the pattern to follow for pure functions.

| Component | Test Approach |
|-----------|---------------|
| `trending/trending.go` | Pure functions (existing) |
| `trending/calculator.go` | Real DB (testcontainers) |
| `api/handlers.go` | Real DB + httptest |
| `curseforge/client.go` | httptest.Server |
| `sync/sync.go` | Real DB + httptest.Server |

## Prerequisites

Before writing tests, fix the API server to be testable:

```go
// internal/api/server.go - Change to accept queries directly
func NewServer(queries *database.Queries) *Server {
    // Instead of: func NewServer(pool *pgxpool.Pool)
}

func (s *Server) Router() *gin.Engine {
    return s.router
}
```

```go
// cmd/web/main.go - Update call site
pool, _ := pgxpool.New(ctx, dbURL)
server := api.NewServer(database.New(pool))
```

## Implementation

### 1. Add testcontainers dependency

```bash
go get github.com/testcontainers/testcontainers-go/modules/postgres
```

### 2. Create shared DB helper

```go
// internal/api/handlers_test.go (or inline where needed)
func setupTestDB(t *testing.T) *database.Queries {
    ctx := context.Background()

    postgres, err := pg.Run(ctx,
        "postgres:15-alpine",
        pg.WithDatabase("testdb"),
        pg.WithUsername("test"),
        pg.WithPassword("test"),
    )
    require.NoError(t, err)

    t.Cleanup(func() { postgres.Terminate(ctx) })

    connStr, _ := postgres.ConnectionString(ctx, "sslmode=disable")
    pool, _ := pgxpool.New(ctx, connStr)
    t.Cleanup(func() { pool.Close() })

    // Apply schema
    schema, _ := os.ReadFile("../../sql/schema.sql")
    _, err = pool.Exec(ctx, string(schema))
    require.NoError(t, err)

    return database.New(pool)
}
```

### 3. Write API handler tests

```go
// internal/api/handlers_test.go
func TestGetAddon(t *testing.T) {
    db := setupTestDB(t)

    // Seed test data
    ctx := context.Background()
    db.UpsertAddon(ctx, database.UpsertAddonParams{
        ID:   123,
        Slug: "test-addon",
        Name: "Test Addon",
    })

    server := NewServer(db)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/addons/test-addon", nil)
    server.Router().ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "test-addon")
}

func TestGetAddon_NotFound(t *testing.T) {
    db := setupTestDB(t)
    server := NewServer(db)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/addons/nonexistent", nil)
    server.Router().ServeHTTP(w, req)

    assert.Equal(t, 404, w.Code)
}
```

### 4. Write CurseForge client tests

```go
// internal/curseforge/client_test.go
func TestSearchMods(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/v1/mods/search", r.URL.Path)
        w.WriteHeader(200)
        w.Write([]byte(`{"data": [{"id": 1, "name": "Test"}]}`))
    }))
    defer server.Close()

    client := NewClient("fake-key")
    client.baseURL = server.URL  // Override for test

    addons, err := client.SearchMods(context.Background(), SearchParams{})

    assert.NoError(t, err)
    assert.Len(t, addons, 1)
}

func TestSearchMods_RateLimit(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(429)
    }))
    defer server.Close()

    client := NewClient("fake-key")
    client.baseURL = server.URL

    _, err := client.SearchMods(context.Background(), SearchParams{})

    assert.Error(t, err)
    // Verify error type or message
}
```

### 5. Write sync service tests

```go
// internal/sync/sync_test.go
func TestRunFullSync_Deduplication(t *testing.T) {
    db := setupTestDB(t)

    // Mock CurseForge server returning overlapping results
    cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return same addon from multiple sort queries
        w.Write([]byte(`{"data": [{"id": 1, "slug": "test"}], "pagination": {"totalCount": 1}}`))
    }))
    defer cfServer.Close()

    service := NewService(db, "fake-key")
    service.client.baseURL = cfServer.URL

    err := service.RunFullSync(context.Background())

    assert.NoError(t, err)

    // Verify deduplication - addon should only exist once
    addons, _ := db.ListAddons(context.Background(), database.ListAddonsParams{Limit: 10})
    assert.Len(t, addons, 1)
}
```

### 6. Add GitHub Actions CI

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run tests
        run: go test ./... -race -timeout=5m

      - name: Build
        run: go build ./cmd/...
```

## Test Files to Create

```
internal/
├── api/
│   └── handlers_test.go      # API endpoint tests
├── curseforge/
│   └── client_test.go        # HTTP client tests
├── sync/
│   └── sync_test.go          # Sync logic tests
└── trending/
    ├── trending_test.go      # ✅ Exists
    └── calculator_test.go    # Trending calculation tests
```

## What We're NOT Doing

- ❌ pgxmock (real DB is simpler and catches more bugs)
- ❌ testutil package (inline helpers are fine)
- ❌ Build tags (no artificial test separation)
- ❌ Coverage metrics (focus on critical paths)
- ❌ Codecov/badges (vanity metrics)
- ❌ go.uber.org/mock (no interface mocking)
- ❌ Makefile (go test ./... is simple enough)
- ❌ Phased rollout (just write the tests)

## Acceptance Criteria

- [ ] API handlers tested (GET addon, list addons, trending endpoints)
- [ ] CurseForge client tested (success, rate limit, timeout)
- [ ] Sync deduplication tested
- [ ] Trending calculator tested with real DB
- [ ] CI runs tests on push/PR
- [ ] All tests pass with `-race` flag

## Run Locally

```bash
# Requires Docker for testcontainers
go test ./... -race
```

---

**Generated**: 2025-12-11 (Simplified based on reviewer feedback)
