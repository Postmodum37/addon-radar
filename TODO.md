# TODO

## Completed

### Research Phase
- [x] Research CurseForge's API → See `docs/plans/2025-12-08-curseforge-api-design.md`
- [x] Research tech stack options → See `docs/plans/2025-12-08-tech-stack-design.md`
- [x] Research trendiness algorithms → See `docs/plans/2025-12-08-trending-algorithm-design.md`

### Sync Job Implementation (Dec 2025)
- [x] Set up Go project structure (cmd/sync, internal/)
- [x] Create database schema (sql/schema.sql)
- [x] Implement sqlc code generation
- [x] Implement configuration loading (envconfig)
- [x] Implement CurseForge API client
  - Multi-query strategy (popularity, lastUpdated, totalDownloads) for 99.8% addon coverage
  - Overcomes CurseForge API 10k result limit
- [x] Build full sync job with category sync
  - Two-pass category insertion to handle FK constraints
- [x] Local testing with Docker PostgreSQL
- [x] Deploy sync job to Railway
- [x] Set up hourly cron schedule (0 * * * *)
- [x] Verify data accumulation (12,406 Retail addons synced)

## Next Steps

### Priority 1: Web UI
- [ ] Create cmd/web entry point
- [ ] Set up Gin router with basic routes
- [ ] Create HTML templates (layout, home, addon detail)
- [ ] Implement homepage showing trending addons
- [ ] Add HTMX for filtering/sorting
- [ ] Set up Tailwind CSS (standalone CLI)
- [ ] Deploy web server to Railway

### Priority 2: Trending Algorithm
- [ ] Implement velocity calculation from snapshots
- [ ] Implement confidence-based adaptive windows (24h vs 7d)
- [ ] Implement logarithmic size multiplier
- [ ] Implement maintenance multiplier
- [ ] Create "Hot Right Now" scoring (gravity 1.5)
- [ ] Create "Rising Stars" scoring (gravity 1.8)
- [ ] Add trending score columns to database
- [ ] Schedule hourly score recalculation

### Priority 3: Hot Detection & Optimization
- [ ] Implement "hot" addon detection (downloads_24h >= 100 OR growth >= 5%)
- [ ] Add batch endpoint for hot-only sync (POST /v1/mods)
- [ ] Implement hourly hot-only sync (faster than full sync)
- [ ] Keep full sync as daily job

### Future Enhancements
- [ ] Category filtering on homepage
- [ ] Search functionality
- [ ] Addon detail pages with historical charts
- [ ] SEO optimization (meta tags, sitemap)
- [ ] Database snapshot export for local dev
- [ ] Consider Svelte migration for richer UI

## Notes

- **Retail focus**: Currently syncing only Retail (gameVersionTypeId=517) addons
- **Data accumulating**: Hourly snapshots building historical data for trending
- **Go version**: Using Go 1.25.5 (current stable as of Dec 2025)
