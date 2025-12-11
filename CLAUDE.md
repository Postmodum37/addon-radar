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

**Phase 3 Complete**: Trending algorithm implemented and deployed.

| Component | URL | Status |
|-----------|-----|--------|
| API | https://addon-radar-api-production.up.railway.app | ✅ Live |
| Sync Job | Railway cron (hourly) | ✅ Running |
| Trending Calculation | Part of sync job | ✅ Running |
| Frontend | TBD | Planned |

**Data**: 12,424 Retail addons with hourly snapshots and trending scores.

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
│   ├── sync/               # Sync service ✅
│   └── trending/           # Trending algorithm ✅
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
| `GET /api/v1/trending/hot` | Hot addons (real data) |
| `GET /api/v1/trending/rising` | Rising addons (real data) |

## Key Implementation Details

### Multi-Query Strategy
CurseForge API limits results to 10k per query. We use 3 sort orders (popularity, lastUpdated, totalDownloads) to achieve 99.8% coverage.

### Service-Specific Railway Configs
`railway.toml` uses `[services.name]` sections to configure different Dockerfiles for sync and API services.

### Game Version Filtering
Only syncing Retail addons (gameVersionTypeId=517).

### Trending Algorithm
Calculates "Hot Right Now" and "Rising Stars" scores using multi-signal blend (downloads, thumbs up, updates), adaptive time windows, logarithmic size multipliers, and maintenance rewards. Runs hourly as part of sync job.

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
| `2025-12-08-trending-algorithm-design.md` | **Implemented** |
| `2025-12-08-tech-stack-design.md` | Reference |
| `2025-12-09-sync-job-implementation.md` | Complete |
| `2025-12-10-rest-api-implementation.md` | Complete |
| `2025-12-10-trending-algorithm-implementation.md` | **Complete** |

## Serena MCP

Serena is a semantic code analysis MCP server with persistent memory. Use it for intelligent code exploration and to maintain project knowledge across sessions.

### When to Use Serena
- **First time in session**: Call `check_onboarding_performed` to load project context
- **Code exploration**: Use `find_symbol`, `get_symbols_overview`, `find_referencing_symbols` for semantic code navigation
- **Pattern search**: Use `search_for_pattern` for regex-based searches across codebase
- **After significant changes**: Update memories with `write_memory` to persist learnings

### Available Memories
| Memory | Content |
|--------|---------|
| `project_overview` | Purpose, tech stack, status, key features |
| `codebase_structure` | Directory layout, packages, API endpoints |
| `suggested_commands` | Build, run, test, deploy commands |
| `style_and_conventions` | Go style, patterns, gotchas (e.g., pgtype.Numeric) |
| `task_completion_checklist` | Pre-commit and deployment verification |

### Key Serena Tools
```
mcp__serena__check_onboarding_performed  # Start of session
mcp__serena__list_memories               # See available memories
mcp__serena__read_memory                 # Load specific memory
mcp__serena__find_symbol                 # Find code symbols by name
mcp__serena__get_symbols_overview        # Get file structure
mcp__serena__search_for_pattern          # Regex search
mcp__serena__write_memory                # Persist new knowledge
```

## Context7 MCP

Context7 fetches up-to-date documentation for libraries and frameworks. Use it instead of relying on potentially outdated training data.

### When to Use Context7
- **Using unfamiliar libraries**: Get current API docs and examples
- **Checking latest syntax**: Verify correct usage of library methods
- **Debugging library issues**: Fetch docs to understand expected behavior
- **Before implementing features**: Research library capabilities first

### Key Context7 Tools
```
mcp__context7__resolve-library-id   # Find library ID (required first step)
mcp__context7__get-library-docs     # Fetch documentation for a library
```

### Example Usage
```
# Step 1: Find the library ID
resolve-library-id("pgx")  → "/jackc/pgx"

# Step 2: Fetch docs (optionally with topic filter)
get-library-docs("/jackc/pgx", topic="connection pool")
```

### Libraries Used in This Project
| Library | Context7 ID | Use For |
|---------|-------------|---------|
| pgx/v5 | `/jackc/pgx` | PostgreSQL driver, connection pooling |
| Gin | `/gin-gonic/gin` | HTTP routing, middleware |
| sqlc | `/sqlc-dev/sqlc` | SQL code generation |

## External Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)
