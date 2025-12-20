# Frontend Architecture Design

## Overview

SvelteKit frontend for Addon Radar with server-side rendering, deployed on Railway alongside the existing Go API.

## Decisions Made

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Framework | SvelteKit | Small bundles, excellent SSR, simple DX |
| Runtime | Bun | Faster cold starts, native TypeScript |
| Deployment | Separate Railway service | Independent scaling, cleaner separation |
| Caching | HTTP Cache-Control (Phase 1) | Simple, no infrastructure, upgrade path to in-memory/Redis |
| SEO | Full SSR with meta tags | Critical for discoverability |

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   SvelteKit     │────▶│   Go REST API   │────▶│   PostgreSQL    │
│   Frontend      │     │   (existing)    │     │   (Railway)     │
│   (Railway)     │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
     Bun :3000               :8080
```

- SvelteKit runs as separate Railway service
- Server-side rendering fetches data from Go API
- API calls happen server-to-server via Railway internal network
- Pre-rendered HTML sent to browser with minimal JS hydration

## Pages and Routes

| Route | Description | Data Source |
|-------|-------------|-------------|
| `/` | Homepage with trending lists | `/trending/hot` + `/trending/rising` |
| `/addon/[slug]` | Addon detail page | `/addons/:slug` + `/addons/:slug/history` |
| `/search` | Search results | `/addons?search=` |

### Homepage
- Two sections: "Hot Right Now" and "Rising Stars"
- Top 20 from each trending endpoint
- Addon cards: logo, name, author, download count, score badge

### Addon Detail
- Full addon info: name, summary, author, downloads, thumbs up
- Download history chart (last 7 days)
- Link to CurseForge page

### Search
- Search input with paginated results
- Server-rendered initial results

### Not Building Initially
- User accounts/authentication
- Favorites or bookmarks
- Category filtering
- Dark mode toggle (use system preference)

## Project Structure

```
web/                          # New directory in repo root
├── src/
│   ├── lib/
│   │   ├── api.ts            # API client
│   │   ├── types.ts          # TypeScript types (match Go API)
│   │   └── components/
│   │       ├── AddonCard.svelte
│   │       ├── TrendingList.svelte
│   │       └── SearchBar.svelte
│   ├── routes/
│   │   ├── +layout.svelte    # Shared header/footer
│   │   ├── +page.svelte      # Homepage UI
│   │   ├── +page.server.ts   # Homepage data loading
│   │   ├── addon/[slug]/
│   │   │   ├── +page.svelte
│   │   │   └── +page.server.ts
│   │   └── search/
│   │       ├── +page.svelte
│   │       └── +page.server.ts
│   └── app.html              # HTML shell
├── static/
│   └── favicon.ico
├── bun.lockb
├── package.json
├── svelte.config.js          # Uses adapter-bun
├── tsconfig.json
└── vite.config.ts
```

## Caching Strategy

### Phase 1 (Now): HTTP Caching
- Go API adds `Cache-Control: public, max-age=300` to responses
- SvelteKit's `fetch` respects these headers
- Zero infrastructure

### Phase 2 (If needed): In-Memory Cache in Go
- Add `ristretto` or `groupcache` to Go API
- Invalidate after sync job completes
- Still zero additional services

### Phase 3 (Scale): Redis/Valkey
- Swap in-memory for Redis when shared state needed
- Railway has one-click Redis/Valkey

## SEO

### Meta Tags

**Homepage:**
```html
<title>Addon Radar - Trending WoW Addons</title>
<meta name="description" content="Discover trending and rising World of Warcraft addons. Updated hourly." />
```

**Addon Detail:**
```html
<title>{addon.name} - Addon Radar</title>
<meta name="description" content="{addon.summary}" />
<meta property="og:image" content="{addon.logo_url}" />
```

### Additional Elements
- `robots.txt` - Allow all crawlers
- Semantic HTML with proper heading hierarchy
- Structured data (JSON-LD) - Phase 2

## Deployment

### Dockerfile.web
```dockerfile
FROM oven/bun:1 AS builder
WORKDIR /app
COPY web/package*.json web/bun.lockb* ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM oven/bun:1-alpine
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package*.json ./
RUN bun install --production --frozen-lockfile
EXPOSE 3000
CMD ["bun", "./build/index.js"]
```

### railway.toml Addition
```toml
[services.addon-radar-web.build]
builder = "dockerfile"
dockerfilePath = "Dockerfile.web"

[services.addon-radar-web.deploy]
numReplicas = 1
```

### Environment Variables
| Variable | Value |
|----------|-------|
| `API_URL` | `http://addon-radar-api.railway.internal:8080` |
| `PORT` | `3000` |

## Development

```bash
cd web
bun install              # Install dependencies
bun run dev              # Dev server with hot reload
bun run build            # Production build
bun run preview          # Test production build locally
```

### Project Initialization
```bash
bun create svelte@latest web
# Select: Skeleton project, TypeScript, ESLint, Prettier
cd web
bun add @sveltejs/adapter-bun
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| API timeout | Show friendly error, suggest refresh |
| API 500 | Show error page, log server-side |
| Addon not found | 404 page with search suggestion |
| Network error | Retry once, then error page |

### API Client Pattern
```typescript
export async function fetchApi<T>(path: string): Promise<T | null> {
  try {
    const res = await fetch(`${API_URL}${path}`);
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}
```

## Testing

- Unit tests for API client with mocked responses (`bun test`)
- Playwright e2e tests optional for later
- Pages handle `null` gracefully with fallback UI

## Future Considerations

- Category filtering page
- Dynamic sitemap.xml
- JSON-LD structured data
- Historical charts with more detail
- Dark mode with toggle
