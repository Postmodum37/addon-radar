# Addon Radar - Project Plan

## Vision

A website that helps World of Warcraft players discover trending and rising addons for **Retail** version. The main focus is a unique trendiness algorithm that surfaces both established hot addons and lesser-known rising stars.

## Current Status: Data Collection Phase ✅

**Deployed**: Sync job running hourly on Railway, collecting data from CurseForge API.

- 12,406 Retail addons synced
- Hourly snapshots accumulating (downloads, thumbs up, popularity rank, rating)
- Categories synced with parent relationships
- Multi-query strategy achieving 99.8% catalog coverage

## Architecture

### Deployed Components
- **Sync Job** (`cmd/sync`) - Runs hourly via Railway cron
- **PostgreSQL** - Hosted on Railway

### Planned Components
- **Web Server** (`cmd/web`) - Gin + HTMX + Tailwind
- **Trending Service** - Score calculation and caching

## Trending Algorithm Design

### Two Categories

**Hot Right Now** - Established addons with high download velocity
- Minimum 500 total downloads
- Moderate decay (gravity 1.5) for stable list presence
- Signal: 70% downloads + 20% thumbs up + 10% update activity

**Rising Stars** - Smaller addons gaining traction
- 50-10,000 total downloads
- Aggressive decay (gravity 1.8) for quick cycling
- Same signal blend as Hot Right Now

### Key Features
- **Adaptive time windows**: Use 24h data when confident (5+ snapshots, 10+ change), otherwise blend with 7d
- **Logarithmic size multiplier**: Smooth curve instead of arbitrary tiers
- **Maintenance reward**: 0.95x-1.15x based on update frequency
- **Age reset on re-entry**: Fair chance for returning addons

See `docs/plans/2025-12-08-trending-algorithm-design.md` for full details.

## Implementation Roadmap

### Phase 1: Data Collection ✅
- [x] CurseForge API client with multi-query strategy
- [x] Database schema (addons, snapshots, categories)
- [x] Full sync job deployed to Railway
- [x] Hourly cron schedule configured

### Phase 2: Web UI (Next)
- [ ] Gin server with HTML templates
- [ ] Homepage showing trending lists
- [ ] HTMX for filtering/sorting
- [ ] Tailwind CSS styling
- [ ] Deploy to Railway

### Phase 3: Trending Algorithm
- [ ] Implement score calculations
- [ ] Add trending columns to database
- [ ] Schedule hourly recalculation
- [ ] Integrate with web UI

### Phase 4: Optimization
- [ ] Hot addon detection
- [ ] Hourly hot-only sync (faster)
- [ ] Daily full sync
- [ ] Performance tuning

### Phase 5: Polish
- [ ] Addon detail pages
- [ ] Search functionality
- [ ] Category filtering
- [ ] SEO optimization
- [ ] Historical charts

## Tech Stack

| Component | Choice | Status |
|-----------|--------|--------|
| Language | Go 1.25 | ✅ |
| Web Framework | Gin | Planned |
| Database | PostgreSQL | ✅ |
| DB Library | sqlc + pgx/v5 | ✅ |
| Frontend | HTMX + Tailwind | Planned |
| Hosting | Railway | ✅ (sync), Planned (web) |

## Data Model

### addons
Core addon metadata from CurseForge, updated each sync.

### snapshots
Time-series metrics for trending calculations. Growing hourly.

### categories
Reference data with parent relationships for filtering.

## API Coverage

Using multi-query strategy with 3 sort orders:
1. Popularity (most popular addons)
2. Last Updated (recently active addons)
3. Total Downloads (high download count addons)

This achieves 99.8% catalog coverage (12,406 of ~12,427 addons) despite CurseForge's 10k result limit per query.

## Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)
- Design docs in `docs/plans/`
