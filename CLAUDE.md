# Claude Code Instructions

## Working Guidelines

- Act as orchestrator for the Addon Radar project
- Follow TDD principles when writing code. Write tests before implementation. Don't overdo testing, but ensure critical paths are well-covered. Focus on high-value tests, not 100% coverage. Prefer integration tests for key flows, unit tests for complex logic.
- Keep markdown files up-to-date after significant changes:
  - `CLAUDE.md` - Project overview (this file)
  - `/docs/PLAN.md` - Project roadmap and status
  - `/docs/TODO.md` - Task tracking
  - `/docs/ALGORITHM.md` - Trending algorithm details
  - `/docs/plans/*` - Design documents, feature specs and plans
- Focus on the unique trendiness algorithm as the main differentiator
- Create plans and design docs in docs/plans/
- Always develop new features in feature branches

## Project Overview

Addon Radar is a website displaying trending World of Warcraft addons for **Retail** version. It syncs data hourly from the CurseForge API and provides a REST API for frontend consumption.

**Main Goal**: Help users discover lesser-known addons through a comprehensive trending algorithm.

## Current Status

**Phase 4 Complete**: Frontend deployed and live.

| Component | URL | Status |
|-----------|-----|--------|
| Frontend | https://addon-radar.com | ✅ Live |
| API | https://api.addon-radar.com | ✅ Live |
| Sync Job | Railway cron (hourly) | ✅ Running |
| Trending Calculation | Part of sync job | ✅ Running |

**Data**: 12,424 Retail addons with hourly snapshots and trending scores.

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.25 |
| Web Framework | Gin |
| Database | PostgreSQL |
| DB Library | sqlc + pgx/v5 |
| Frontend | SvelteKit + Bun |
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
│   ├── testutil/           # Test utilities ✅
│   └── trending/           # Trending algorithm ✅
├── web/                    # SvelteKit frontend ✅
│   ├── src/
│   │   ├── lib/            # Components, API client
│   │   └── routes/         # Pages
│   └── bun.lock            # Bun lockfile
├── Dockerfile.web          # Frontend container
├── sql/
│   ├── schema.sql
│   └── queries.sql
├── .github/workflows/      # CI/CD ✅
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

## Development Setup

### Prerequisites
- Go 1.25+
- golangci-lint v2.7.2+
- Lefthook v2.0.12+

### Install Development Tools

```bash
# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.7.2

# Install Lefthook (macOS)
brew install lefthook

# Install git hooks
lefthook install
```

### Git Hooks (via Lefthook)

| Hook | Commands | Description |
|------|----------|-------------|
| pre-commit | lint, fmt | Run golangci-lint and gofmt on staged Go files |
| pre-push | test, lint-all | Run full test suite and lint check |

### Local Database Setup

```bash
# Create local PostgreSQL database
createdb addon_radar

# Apply schema
psql addon_radar < sql/schema.sql

# Copy environment template
cp .env.example .env
# Edit .env with your local DATABASE_URL and CURSEFORGE_API_KEY
```

### Regenerating Database Code

After modifying `sql/schema.sql` or `sql/queries.sql`:

```bash
# Regenerate sqlc code
sqlc generate
```

### Manual Commands

```bash
# Run linter
golangci-lint run ./...

# Run tests
go test ./... -race -timeout=5m

# Run single package tests
go test ./internal/trending/... -v

# Format code
gofmt -l -w .

# Build binaries
go build -o bin/sync ./cmd/sync
go build -o bin/web ./cmd/web

# Run locally
./bin/web        # API server on :8080
./bin/sync       # One-time sync job
```

### API Examples

```bash
# Health check
curl http://localhost:8080/api/v1/health

# List addons (paginated)
curl "http://localhost:8080/api/v1/addons?page=1&per_page=10"

# Search addons
curl "http://localhost:8080/api/v1/addons?search=details"

# Get single addon
curl http://localhost:8080/api/v1/addons/details

# Get trending hot
curl http://localhost:8080/api/v1/trending/hot

# Get rising stars
curl http://localhost:8080/api/v1/trending/rising

# Production API
curl https://api.addon-radar.com/api/v1/trending/hot
```

### Railway Deployment

Deployments are automatic via GitHub integration:

```bash
# Push to main triggers deployment
git push origin main

# Check deployment status in Railway dashboard
# https://railway.app/dashboard
```

### Frontend Development

The web frontend is built with SvelteKit and uses Bun as the runtime and package manager.

```bash
# Navigate to web directory
cd web

# Install dependencies
bun install

# Run development server
bun run dev           # Starts on http://localhost:5173

# Build for production
bun run build         # Output in .svelte-kit/output

# Preview production build
bun run preview       # Test production build locally

# Type checking
bun run check         # Run svelte-check
```

#### Frontend Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_URL` | Backend API URL (development) | `http://localhost:8080` |
| `API_URL` | Backend API URL (production, Railway internal) | Set in Railway |

## Design Documents

| Document | Status |
|----------|--------|
| `2025-12-08-curseforge-api-design.md` | Reference |
| `2025-12-08-trending-algorithm-design.md` | **Implemented** |
| `2025-12-08-tech-stack-design.md` | Reference |
| `2025-12-09-sync-job-implementation.md` | Complete |
| `2025-12-10-rest-api-implementation.md` | Complete |
| `2025-12-10-trending-algorithm-implementation.md` | **Complete** |
| `2025-12-11-trending-calculation-optimization.md` | Complete |
| `2025-12-11-testing-infrastructure.md` | **Complete** |
| `2025-12-16-fix-curseforge-api-timeout.md` | Complete |
| `2025-12-16-dev-hooks-implementation.md` | Complete |
| `2025-12-20-frontend-architecture-design.md` | **Complete** |
| `2025-12-20-frontend-sveltekit-implementation.md` | **Complete** |

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

## Ref MCP

Ref searches documentation from the web, GitHub, and private resources. **Prefer Ref over Context7** for finding documentation as it provides broader coverage and more up-to-date results.

### When to Use Ref
- **Finding documentation**: Search across web, GitHub, and private docs (preferred method)
- **Reading specific docs**: Fetch and read content from documentation URLs
- **API research**: Find official documentation for external services
- **Library docs**: First choice before falling back to Context7

### Key Ref Tools
```
mcp__Ref__ref_search_documentation  # Search for documentation
mcp__Ref__ref_read_url              # Read content from a documentation URL
```

### Example Usage
```
# Step 1: Search for documentation
ref_search_documentation("Go pgx PostgreSQL driver")

# Step 2: Read a specific documentation URL from results
ref_read_url("https://pkg.go.dev/github.com/jackc/pgx/v5#section-readme")
```

### Tips
- Include programming language and framework names in search queries
- Add `ref_src=private` to search queries to include private docs
- Pass the exact URL (including `#hash`) from search results to `ref_read_url`
- Use Ref first; fall back to Context7 only if Ref doesn't have the library indexed

## PAL MCP

PAL (Peer AI Layer) provides multi-model collaboration for complex analysis tasks. Use it to get second opinions, perform deep investigations, or leverage specialized analysis capabilities.

### When to Use PAL
- **Code review**: Get external model review of code changes
- **Debugging**: Deep investigation with hypothesis testing
- **Architecture decisions**: Multi-model consensus on design choices
- **Security audits**: Comprehensive vulnerability assessment
- **Complex analysis**: When you need deeper reasoning or validation

### Key PAL Tools

| Tool | Purpose |
|------|---------|
| `mcp__pal__chat` | General chat and brainstorming with external models |
| `mcp__pal__codereview` | Systematic code review with expert validation |
| `mcp__pal__debug` | Root cause analysis with hypothesis testing |
| `mcp__pal__analyze` | Comprehensive code analysis (architecture, performance, security) |
| `mcp__pal__thinkdeep` | Multi-stage investigation for complex problems |
| `mcp__pal__consensus` | Multi-model debate for architectural decisions |
| `mcp__pal__precommit` | Validate git changes before committing |
| `mcp__pal__secaudit` | Security audit with OWASP Top 10 analysis |
| `mcp__pal__testgen` | Generate comprehensive test suites |
| `mcp__pal__refactor` | Identify refactoring opportunities |
| `mcp__pal__listmodels` | List available models |

### Example Usage
```
# Get available models first
mcp__pal__listmodels()

# Chat with an external model for brainstorming
mcp__pal__chat(
  prompt="Review this API design approach...",
  model="google/gemini-2.5-pro",
  working_directory_absolute_path="/Users/tomas/Workspace/addon-radar"
)

# Deep code review
mcp__pal__codereview(
  step="Reviewing the trending algorithm implementation",
  step_number=1,
  total_steps=2,
  next_step_required=true,
  findings="Initial analysis of algorithm...",
  model="google/gemini-2.5-pro",
  relevant_files=["/Users/tomas/Workspace/addon-radar/internal/trending/calculator.go"]
)
```

### Tips
- Use `listmodels` first to see available models when no specific model is requested
- Pass absolute file paths to `relevant_files` for code context
- Use `continuation_id` to maintain context across multiple tool calls
- Set `thinking_mode` to "high" or "max" for complex analysis

## Tavily MCP

Tavily provides AI-powered web search and content extraction. **Use Tavily for real-time web searches**, current events, and extracting content from URLs.

### When to Use Tavily
- **Web search**: Find current information, news, documentation not in other MCPs
- **Content extraction**: Get clean markdown from web pages
- **Site crawling**: Explore website structure and content
- **Research**: Gather information from multiple sources

### Key Tavily Tools

| Tool | Purpose |
|------|---------|
| `mcp__tavily__tavily-search` | Search the web with AI-powered results |
| `mcp__tavily__tavily-extract` | Extract content from specific URLs |
| `mcp__tavily__tavily-crawl` | Crawl websites starting from a URL |
| `mcp__tavily__tavily-map` | Map website structure and URLs |

### Example Usage
```
# Web search
mcp__tavily__tavily-search(
  query="SvelteKit Railway deployment 2025",
  max_results=10
)

# Extract content from URL
mcp__tavily__tavily-extract(
  urls=["https://kit.svelte.dev/docs/adapter-node"]
)

# Search with domain filter
mcp__tavily__tavily-search(
  query="bun install frozen lockfile",
  include_domains=["bun.sh"]
)
```

### Tips
- Use `topic="news"` for recent news articles
- Set `search_depth="advanced"` for more thorough results
- Use `include_domains` to restrict search to specific sites
- For time-sensitive queries, use `time_range="week"` or `"month"`

## External Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)
