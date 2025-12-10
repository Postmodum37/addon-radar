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

## Next Steps

### Priority 1: Trending Algorithm
- [ ] Implement velocity calculation from snapshots
- [ ] Implement confidence-based adaptive windows (24h vs 7d)
- [ ] Implement logarithmic size multiplier
- [ ] Implement maintenance multiplier
- [ ] Create "Hot Right Now" scoring (gravity 1.5)
- [ ] Create "Rising Stars" scoring (gravity 1.8)
- [ ] Replace placeholder trending endpoints

### Priority 2: API Enhancements
- [ ] Add category filtering to `/addons` endpoint
- [ ] Add sorting options (downloads, updated, name)
- [ ] Add rate limiting (optional)

### Priority 3: Frontend
- [ ] Choose frontend framework (Svelte, React, or HTMX)
- [ ] Build homepage with trending lists
- [ ] Add addon detail pages
- [ ] Add search functionality
- [ ] Deploy frontend

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
