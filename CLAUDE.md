# Claude Code Instructions

## Working Guidelines

- Act as orchestrator for the Addon Radar project, providing guidance on architecture, technology choices, and implementation strategies
- Follow TDD principles when writing code
- Keep markdown files up-to-date after significant changes:
  - `CLAUDE.md` - Project overview and architecture (this file)
  - `PLAN.md` - Project roadmap and status
  - `TODO.md` - Task tracking
- Focus on the unique trendiness algorithm as the main differentiator
- Ensure scalable, maintainable code

## Project Overview

Addon Radar is a website displaying trending World of Warcraft addons for **Retail** version. It syncs data hourly from the CurseForge API and stores historical download snapshots to calculate velocity-based trending scores.

**Main Goal**: Help users discover lesser-known addons through a comprehensive trending algorithm.

## Current Status

**Phase 1 Complete**: Data collection infrastructure deployed and running.

- Sync job running hourly on Railway (cron: `0 * * * *`)
- 12,406 Retail addons synced
- Snapshots accumulating for trending algorithm

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.25 |
| Web Framework | Gin (planned) |
| Database | PostgreSQL |
| DB Library | sqlc + pgx/v5 |
| Frontend | HTMX + Tailwind (planned) |
| Config | envconfig |
| Logging | slog (stdlib) |
| Hosting | Railway |

## Project Structure

```
addon-radar/
├── cmd/
│   ├── sync/main.go        # Sync job (deployed)
│   └── web/main.go         # Web server (planned)
├── internal/
│   ├── config/             # Environment configuration
│   ├── database/           # sqlc generated code
│   ├── curseforge/         # API client
│   └── sync/               # Sync service
├── sql/
│   ├── schema.sql          # Database schema
│   └── queries.sql         # sqlc queries
├── scripts/
│   └── db-setup.sh         # Local Docker setup
├── docs/plans/             # Design documents
├── Dockerfile              # Railway deployment
├── railway.toml
├── sqlc.yaml
└── go.mod
```

## Key Implementation Details

### Multi-Query Strategy
CurseForge API limits results to 10k per query. We use 3 sort orders to achieve 99.8% coverage:
1. Popularity
2. Last Updated
3. Total Downloads

### Two-Pass Category Sync
Categories have parent references. Insert all without parent_id first, then update parent relationships to avoid FK constraint violations.

### Game Version Filtering
Only syncing Retail addons (gameVersionTypeId=517). Other version types:
- Classic: 67408
- Wrath Classic: 73713
- Cata Classic: 77522
- MoP Classic: 79434

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `CURSEFORGE_API_KEY` | CurseForge API key |
| `PORT` | Server port (default: 8080) |
| `ENV` | Environment (development/production) |

## Available Tools

- Railway CLI (`railway`)
- GitHub CLI (`gh`)
- sqlc (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)

## External Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)

## Design Documents

| Document | Status | Description |
|----------|--------|-------------|
| `2025-12-08-curseforge-api-design.md` | Reference | API integration design |
| `2025-12-08-trending-algorithm-design.md` | **Active** | Trending algorithm spec (to implement) |
| `2025-12-08-tech-stack-design.md` | Reference | Technology decisions |
| `2025-12-09-sync-job-implementation.md` | **Complete** | Implementation plan (done) |
