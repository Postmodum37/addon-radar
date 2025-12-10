# Claude Code Instructions

## Working Guidelines

- Act as orchestrator for the Addon Radar project
- Follow TDD principles when writing code
- Keep markdown files up-to-date after significant changes:
  - `CLAUDE.md` - Project overview (this file)
  - `PLAN.md` - Project roadmap and status
  - `TODO.md` - Task tracking
- Focus on the unique trendiness algorithm as the main differentiator

## Project Overview

Addon Radar is a website displaying trending World of Warcraft addons for **Retail** version. It syncs data hourly from the CurseForge API and provides a REST API for frontend consumption.

**Main Goal**: Help users discover lesser-known addons through a comprehensive trending algorithm.

## Current Status

**Phase 2 Complete**: REST API deployed and serving data.

| Component | URL | Status |
|-----------|-----|--------|
| API | https://addon-radar-api-production.up.railway.app | ✅ Live |
| Sync Job | Railway cron (hourly) | ✅ Running |
| Frontend | TBD | Planned |

**Data**: 12,424 Retail addons with hourly snapshots accumulating.

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.25 |
| Web Framework | Gin |
| Database | PostgreSQL |
| DB Library | sqlc + pgx/v5 |
| Config | envconfig |
| Logging | slog (stdlib) |
| Hosting | Railway |

## Project Structure

```
addon-radar/
├── cmd/
│   ├── sync/main.go        # Sync job ✅
│   └── web/main.go         # API server ✅
├── internal/
│   ├── api/                # API handlers ✅
│   ├── config/             # Configuration ✅
│   ├── database/           # sqlc generated ✅
│   ├── curseforge/         # API client ✅
│   └── sync/               # Sync service ✅
├── sql/
│   ├── schema.sql
│   └── queries.sql
├── Dockerfile.sync         # Sync service
├── Dockerfile.api          # API service
├── railway.toml            # Service configs
└── docs/plans/             # Design documents
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/health` | Health check |
| `GET /api/v1/addons` | List (paginated, searchable) |
| `GET /api/v1/addons/:slug` | Single addon |
| `GET /api/v1/addons/:slug/history` | Download history |
| `GET /api/v1/categories` | All categories |
| `GET /api/v1/trending/hot` | Hot addons (placeholder) |
| `GET /api/v1/trending/rising` | Rising addons (placeholder) |

## Key Implementation Details

### Multi-Query Strategy
CurseForge API limits results to 10k per query. We use 3 sort orders (popularity, lastUpdated, totalDownloads) to achieve 99.8% coverage.

### Service-Specific Railway Configs
`railway.toml` uses `[services.name]` sections to configure different Dockerfiles for sync and API services.

### Game Version Filtering
Only syncing Retail addons (gameVersionTypeId=517).

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `CURSEFORGE_API_KEY` | CurseForge API key (sync only) |
| `PORT` | Server port (default: 8080) |
| `ENV` | Environment (development/production) |

## Design Documents

| Document | Status |
|----------|--------|
| `2025-12-08-curseforge-api-design.md` | Reference |
| `2025-12-08-trending-algorithm-design.md` | **To Implement** |
| `2025-12-08-tech-stack-design.md` | Reference |
| `2025-12-09-sync-job-implementation.md` | Complete |
| `2025-12-10-rest-api-implementation.md` | **Complete** |

## External Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)
