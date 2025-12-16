# TODO

## Completed

### Research Phase (Dec 2025)
- [x] Research CurseForge's API → See `docs/plans/2025-12-08-curseforge-api-design.md`
- [x] Research tech stack options → See `docs/plans/2025-12-08-tech-stack-design.md`
- [x] Research trendiness algorithms → See `docs/plans/2025-12-08-trending-algorithm-design.md`

### Sync Job Implementation (Dec 2025)
- [x] Set up Go project structure (cmd/sync, internal/)
- [x] Create database schema (sql/schema.sql)
- [x] Implement sqlc code generation
- [x] Implement CurseForge API client with multi-query strategy
- [x] Build full sync job with category sync
- [x] Deploy sync job to Railway (hourly cron)

### REST API Implementation (Dec 2025)
- [x] Add Gin web framework
- [x] Create API server skeleton with health endpoint
- [x] Add response helpers (pagination, errors)
- [x] Add sqlc queries for listing/filtering
- [x] Implement `/api/v1/addons` with pagination & search
- [x] Implement `/api/v1/addons/:slug`
- [x] Implement `/api/v1/addons/:slug/history`
- [x] Implement `/api/v1/categories`
- [x] Implement `/api/v1/trending/hot` (placeholder)
- [x] Implement `/api/v1/trending/rising` (placeholder)
- [x] Create separate Dockerfiles (sync & api)
- [x] Configure service-specific Railway configs
- [x] Deploy API to Railway

### Trending Algorithm Implementation (Dec 2025)
- [x] Add trending_scores table for caching calculations
- [x] Add SQL queries for snapshot aggregation
- [x] Implement velocity calculation from snapshots
- [x] Implement confidence-based adaptive windows (24h vs 7d)
- [x] Implement logarithmic size multiplier
- [x] Implement maintenance multiplier
- [x] Create "Hot Right Now" scoring (gravity 1.5)
- [x] Create "Rising Stars" scoring (gravity 1.8)
- [x] Replace placeholder trending endpoints
- [x] Integrate trending calculation into sync job
- [x] Deploy to Railway and verify

### API Resilience (PR #2, Dec 2025)
- [x] Fix CurseForge API timeout errors (30s → 60s)
- [x] Add retry logic with exponential backoff (2s, 4s, 8s)
- [x] Add HTTPError type for proper error classification
- [x] Add Retry-After header parsing for 429 responses
- [x] Add circuit breaker (10 consecutive failures)
- [x] Add io.LimitReader (10MB) for memory protection
- [x] Add atomic transactions for addon+snapshot writes
- [x] Add error rate threshold (>1% fails sync)
- [x] Add comprehensive testing infrastructure (testutil package)
- [x] Add sync duration warning (>55 min)

### Development Hooks (Dec 2025)
- [x] Install golangci-lint v2.7.2
- [x] Create golangci-lint configuration (.golangci.yml)
- [x] Fix existing lint issues
- [x] Install Lefthook v2.0.12
- [x] Create Lefthook configuration (lefthook.yml)
- [x] Add GitHub Actions lint workflow
- [x] Update documentation (CLAUDE.md)

## Next Steps

### Priority 1: Frontend Development
- [ ] Choose frontend framework (Svelte, React, or HTMX)
- [ ] Build homepage with trending lists
- [ ] Add addon detail pages
- [ ] Add search functionality
- [ ] Deploy frontend

### Priority 2: API Enhancements
- [ ] Add category filtering to `/addons` endpoint
- [ ] Add sorting options (downloads, updated, name)
- [ ] Add rate limiting (optional)

### Future Enhancements
- [ ] Hot addon detection for faster sync
- [ ] Historical charts on addon pages
- [ ] SEO optimization
- [ ] Database snapshot export for local dev

## Production URLs

- **API**: https://addon-radar-api-production.up.railway.app
- **Sync Job**: Running hourly via Railway cron

## Notes

- **Retail focus**: Syncing only Retail (gameVersionTypeId=517) addons
- **Data**: 12,424 addons, snapshots accumulating hourly
- **Go version**: 1.25.5
