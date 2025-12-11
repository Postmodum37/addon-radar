# Testing Infrastructure

**Status**: ✅ Complete
**Implemented**: 2025-12-11

## Overview

Added automated tests and CI to prevent regressions. Uses real Postgres via testcontainers, stdlib httptest, and minimal dependencies.

## What Was Implemented

### Test Files Created

| File | Coverage |
|------|----------|
| `internal/api/handlers_test.go` | API endpoints (health, addons, categories, trending) |
| `internal/curseforge/client_test.go` | HTTP client (search, categories, error handling) |
| `internal/sync/sync_test.go` | Full sync, deduplication, category sync |
| `internal/trending/calculator_test.go` | Score calculation, thresholds, timestamps |
| `internal/testutil/db.go` | Shared test database setup |

### CI/CD

GitHub Actions workflow (`.github/workflows/test.yml`):
- Runs on push/PR to main
- Go 1.25 with race detector
- Dependency verification (`go mod verify`)
- 5-minute timeout
- Security permissions configured

### Architecture Decisions

1. **Real PostgreSQL** - All tests use testcontainers for real database behavior
2. **Shared testutil** - `internal/testutil.SetupTestDB()` eliminates duplication
3. **Interface-based DI** - `CurseForgeClient` interface allows mock injection
4. **http.Handler** - Server implements `ServeHTTP()` for clean testing

## Test Strategy

| Component | Approach |
|-----------|----------|
| `api/handlers.go` | Real DB + httptest |
| `curseforge/client.go` | httptest.Server (mock HTTP) |
| `sync/sync.go` | Real DB + mock client interface |
| `trending/calculator.go` | Real DB |

## Running Tests

```bash
# Requires Docker for testcontainers
go test ./... -race -timeout=5m
```

## Acceptance Criteria

- [x] API handlers tested (GET addon, list addons, trending endpoints)
- [x] CurseForge client tested (search, categories, error handling)
- [x] Sync deduplication tested
- [x] Trending calculator tested with real DB
- [x] CI runs tests on push/PR
- [x] All tests pass with `-race` flag

---

**Generated**: 2025-12-11
