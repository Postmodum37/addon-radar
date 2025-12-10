# Addon Radar - Project Plan

## Vision

A website that helps World of Warcraft players discover trending and rising addons for **Retail** version. The main focus is a unique trendiness algorithm that surfaces both established hot addons and lesser-known rising stars.

## Current Status: API Complete ✅

**Live Production:**
- **API**: https://addon-radar-api-production.up.railway.app
- **Sync Job**: Running hourly via Railway cron
- **Data**: 12,424 Retail addons with hourly snapshots

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
                                        │  (Planned)  │
                                        └─────────────┘
```

| Component | Status | Description |
|-----------|--------|-------------|
| Sync Job | ✅ Deployed | Hourly CurseForge sync |
| PostgreSQL | ✅ Deployed | Hosted on Railway |
| REST API | ✅ Deployed | JSON endpoints for all data |
| Frontend | 🔜 Next | To be built |

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/health` | Health check |
| `GET /api/v1/addons` | List with pagination & search |
| `GET /api/v1/addons/:slug` | Single addon |
| `GET /api/v1/addons/:slug/history` | Download history |
| `GET /api/v1/categories` | All categories |
| `GET /api/v1/trending/hot` | Hot addons (placeholder) |
| `GET /api/v1/trending/rising` | Rising addons (placeholder) |

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

### Key Features (To Implement)
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

### Phase 4: Frontend (Next)
- [ ] Choose framework (Svelte, React, or HTMX)
- [ ] Homepage with trending lists
- [ ] Addon detail pages
- [ ] Search and filtering
- [ ] Deploy

### Phase 5: Polish
- [ ] Hot addon detection for faster sync
- [ ] Historical charts
- [ ] SEO optimization

## Tech Stack

| Component | Choice | Status |
|-----------|--------|--------|
| Language | Go 1.25 | ✅ |
| Web Framework | Gin | ✅ |
| Database | PostgreSQL | ✅ |
| DB Library | sqlc + pgx/v5 | ✅ |
| Frontend | TBD | Planned |
| Hosting | Railway | ✅ |

## Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)
- Design docs in `docs/plans/`
