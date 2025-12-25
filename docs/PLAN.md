# Addon Radar - Project Plan

## Vision

A website that helps World of Warcraft players discover trending and rising addons for **Retail** version. The main focus is a unique trendiness algorithm that surfaces both established hot addons and lesser-known rising stars.

## Current Status: Frontend V3 ✅

**Live Production:**
- **Frontend**: https://addon-radar.com
- **API**: https://api.addon-radar.com
- **Sync Job**: Running hourly via Railway cron
- **Data**: 12,424 Retail addons with hourly snapshots

**Frontend V3 Features (Dec 25, 2025):**
- Renamed categories: "Trending" and "Rising" (clearer naming)
- Featured addon cards for top 3 (larger, more prominent)
- Enhanced cards with rank position, summary, likes, update time
- RankBadge shows all states: rising/falling/unchanged/new
- Server-side pagination with meta object (page, per_page, total, total_pages)
- Fixed rank history bug for accurate rank change tracking

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Sync Job   │────▶│ PostgreSQL  │◀────│  REST API   │
│ (hourly)    │     │  (Railway)  │     │  (Gin)      │
└─────────────┘     └─────────────┘     └─────────────┘
                                              │
                                              ▼
                                        ┌─────────────┐
                                        │  Frontend   │
                                        │ (SvelteKit) │
                                        └─────────────┘
```

| Component | Status | Description |
|-----------|--------|-------------|
| Sync Job | ✅ Deployed | Hourly CurseForge sync |
| PostgreSQL | ✅ Deployed | Hosted on Railway |
| REST API | ✅ Deployed | JSON endpoints for all data |
| Frontend | ✅ Deployed | SvelteKit + Bun on Railway |

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/health` | Health check |
| `GET /api/v1/addons` | List with pagination & search |
| `GET /api/v1/addons/:slug` | Single addon |
| `GET /api/v1/addons/:slug/history` | Download history |
| `GET /api/v1/categories` | All categories |
| `GET /api/v1/trending/hot` | Hot addons (real data) |
| `GET /api/v1/trending/rising` | Rising addons (real data) |

## Trending Algorithm Design

### Two Categories

**Hot Right Now** - Established addons with high download velocity
- Minimum 500 total downloads
- Moderate decay (gravity 1.5)
- Signal: 70% downloads + 20% thumbs up + 10% update activity

**Rising Stars** - Smaller addons gaining traction
- 50-10,000 total downloads
- Aggressive decay (gravity 1.8)
- Same signal blend

### Key Features (Implemented)
- Adaptive time windows (24h vs 7d confidence-based)
- Logarithmic size multiplier
- Maintenance reward (0.95x-1.15x)
- Age reset on re-entry

See `docs/plans/2025-12-08-trending-algorithm-design.md` for full spec.

## Implementation Roadmap

### Phase 1: Data Collection ✅
- [x] CurseForge API client with multi-query strategy
- [x] Database schema (addons, snapshots, categories)
- [x] Full sync job deployed to Railway
- [x] Hourly cron schedule configured

### Phase 2: REST API ✅
- [x] Gin server with versioned endpoints
- [x] Pagination and search
- [x] All CRUD endpoints
- [x] Placeholder trending endpoints
- [x] Deployed to Railway

### Phase 3: Trending Algorithm ✅
- [x] Implement score calculations
- [x] Replace placeholder endpoints
- [x] Schedule hourly recalculation
- [x] Deploy to production

### Phase 3.5: API Resilience (PR #2) ✅
- [x] Add retry logic with exponential backoff (2s, 4s, 8s)
- [x] Add Retry-After header parsing for 429 responses
- [x] Add circuit breaker (fail after 10 consecutive failures)
- [x] Add atomic transactions for addon+snapshot writes
- [x] Add comprehensive testing infrastructure (testutil, httptest)
- [x] Add error rate threshold (fail if >1% errors)

### Phase 4: Frontend ✅
- [x] Choose framework (SvelteKit with Bun)
- [x] Homepage with trending lists
- [x] Addon detail pages
- [x] Search with pagination
- [x] Railway deployment config
- [x] Deploy to Railway

### Phase 4.5: Frontend Redesign V2 (PR #9) ✅
- [x] Clean minimal light theme with dark header
- [x] Download velocity display instead of arbitrary scores
- [x] Rank change badges showing position movement
- [x] Paginated trending pages (/trending/hot, /trending/rising)
- [x] Search autocomplete with dropdown results
- [x] Weekly trend chart on addon detail pages
- [x] API: Added `download_velocity` to trending responses

### Phase 4.6: Frontend V3 (PR #10) ✅
- [x] Rename categories: "Hot Right Now" → "Trending", "Rising Stars" → "Rising"
- [x] Featured addon cards for top 3 (FeaturedAddonCard component)
- [x] Enhanced AddonCard with rank, summary, likes, update time
- [x] RankBadge shows all states: rising (green), falling (red), unchanged (gray), new (blue)
- [x] API: Server-side pagination with page/per_page params and meta object
- [x] Fix rank history bug: DISTINCT ON pattern for accurate rank changes

### Phase 5: Polish
- [ ] Hot addon detection for faster sync
- [x] Historical charts (trend chart on detail page)
- [ ] SEO optimization

## Tech Stack

| Component | Choice | Status |
|-----------|--------|--------|
| Language | Go 1.25 | ✅ |
| Web Framework | Gin | ✅ |
| Database | PostgreSQL | ✅ |
| DB Library | sqlc + pgx/v5 | ✅ |
| Frontend | SvelteKit + Bun | ✅ |
| Hosting | Railway | ✅ |

## Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)
- Design docs in `docs/plans/`
