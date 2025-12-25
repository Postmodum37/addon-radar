# Frontend V3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add pagination to trending endpoints, enrich addon cards with rank/summary/stats, and rename categories to "Trending"/"Rising".

**Architecture:** Backend-first approach - add paginated SQL queries and API handlers, then update frontend components and pages. Featured cards for top 3, compact cards for rest.

**Tech Stack:** Go 1.25, sqlc, Gin, SvelteKit, TypeScript

---

## Task 1: Add Paginated SQL Queries

**Files:**
- Modify: `sql/queries.sql`

**Step 1: Add ListHotAddonsPaginated query**

Add after the existing `ListHotAddons` query:

```sql
-- name: ListHotAddonsPaginated :many
SELECT a.*, t.hot_score, t.download_velocity
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 500
  AND t.hot_score > 0
ORDER BY t.hot_score DESC
LIMIT $1 OFFSET $2;

-- name: CountHotAddons :one
SELECT COUNT(*)
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 500
  AND t.hot_score > 0;
```

**Step 2: Add ListRisingAddonsPaginated query**

Add after the existing `ListRisingAddons` query:

```sql
-- name: ListRisingAddonsPaginated :many
SELECT a.*, t.rising_score, t.download_velocity
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 50
  AND a.download_count <= 10000
  AND t.rising_score > 0
  AND a.id NOT IN (
      SELECT addon_id FROM trending_scores
      WHERE hot_score > 0
      ORDER BY hot_score DESC
      LIMIT 20
  )
ORDER BY t.rising_score DESC
LIMIT $1 OFFSET $2;

-- name: CountRisingAddons :one
SELECT COUNT(*)
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 50
  AND a.download_count <= 10000
  AND t.rising_score > 0
  AND a.id NOT IN (
      SELECT addon_id FROM trending_scores
      WHERE hot_score > 0
      ORDER BY hot_score DESC
      LIMIT 20
  );
```

**Step 3: Regenerate sqlc**

Run: `sqlc generate`

**Step 4: Verify build**

Run: `go build ./...`

**Step 5: Commit**

```bash
git add sql/queries.sql internal/database/
git commit -m "feat(db): add paginated trending queries"
```

---

## Task 2: Update API Handlers for Pagination

**Files:**
- Modify: `internal/api/handlers.go`

**Step 1: Update handleTrendingHot to support pagination**

Replace the `handleTrendingHot` function with:

```go
func (s *Server) handleTrendingHot(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	ctx := c.Request.Context()

	// Get total count
	total, err := s.db.CountHotAddons(ctx)
	if err != nil {
		slog.Error("failed to count hot addons", "error", err)
		respondInternalError(c)
		return
	}

	// Get paginated addons
	addons, err := s.db.ListHotAddonsPaginated(ctx, database.ListHotAddonsPaginatedParams{
		Limit:  int32(perPage),
		Offset: int32(offset),
	})
	if err != nil {
		slog.Error("failed to get hot addons", "error", err)
		respondInternalError(c)
		return
	}

	// Get rank changes
	rankChanges, err := s.db.GetRankChanges(ctx)
	if err != nil {
		slog.Error("failed to get rank changes", "error", err)
		respondInternalError(c)
		return
	}

	// Build lookup map by addon_id for "hot" category
	rankChangeMap := make(map[int32]database.GetRankChangesRow)
	for _, rc := range rankChanges {
		if rc.Category == "hot" {
			rankChangeMap[rc.AddonID] = rc
		}
	}

	response := make([]TrendingAddonResponse, len(addons))
	for i, a := range addons {
		response[i] = TrendingAddonResponse{
			AddonResponse: addonToResponse(database.Addon{
				ID:             a.ID,
				Name:           a.Name,
				Slug:           a.Slug,
				Summary:        a.Summary,
				AuthorName:     a.AuthorName,
				LogoUrl:        a.LogoUrl,
				DownloadCount:  a.DownloadCount,
				ThumbsUpCount:  a.ThumbsUpCount,
				PopularityRank: a.PopularityRank,
				GameVersions:   a.GameVersions,
				LastUpdatedAt:  a.LastUpdatedAt,
			}),
			Rank: offset + i + 1, // Calculate correct rank based on page
		}
		if a.HotScore.Valid {
			f8, err := a.HotScore.Float64Value()
			if err == nil {
				response[i].Score = f8.Float64
			}
		}
		if a.DownloadVelocity.Valid {
			f8, err := a.DownloadVelocity.Float64Value()
			if err == nil {
				response[i].DownloadVelocity = f8.Float64
			}
		}

		// Add rank changes if available
		if rc, ok := rankChangeMap[a.ID]; ok {
			if rc.Rank24hAgo.Valid {
				change := int(rc.Rank24hAgo.Int16 - rc.CurrentRank)
				response[i].RankChange24h = &change
			}
			if rc.Rank7dAgo.Valid {
				change := int(rc.Rank7dAgo.Int16 - rc.CurrentRank)
				response[i].RankChange7d = &change
			}
		}
	}

	respondWithPagination(c, response, page, perPage, int(total))
}
```

**Step 2: Update handleTrendingRising similarly**

Replace the `handleTrendingRising` function with:

```go
func (s *Server) handleTrendingRising(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	ctx := c.Request.Context()

	// Get total count
	total, err := s.db.CountRisingAddons(ctx)
	if err != nil {
		slog.Error("failed to count rising addons", "error", err)
		respondInternalError(c)
		return
	}

	// Get paginated addons
	addons, err := s.db.ListRisingAddonsPaginated(ctx, database.ListRisingAddonsPaginatedParams{
		Limit:  int32(perPage),
		Offset: int32(offset),
	})
	if err != nil {
		slog.Error("failed to get rising addons", "error", err)
		respondInternalError(c)
		return
	}

	// Get rank changes
	rankChanges, err := s.db.GetRankChanges(ctx)
	if err != nil {
		slog.Error("failed to get rank changes", "error", err)
		respondInternalError(c)
		return
	}

	// Build lookup map by addon_id for "rising" category
	rankChangeMap := make(map[int32]database.GetRankChangesRow)
	for _, rc := range rankChanges {
		if rc.Category == "rising" {
			rankChangeMap[rc.AddonID] = rc
		}
	}

	response := make([]TrendingAddonResponse, len(addons))
	for i, a := range addons {
		response[i] = TrendingAddonResponse{
			AddonResponse: addonToResponse(database.Addon{
				ID:             a.ID,
				Name:           a.Name,
				Slug:           a.Slug,
				Summary:        a.Summary,
				AuthorName:     a.AuthorName,
				LogoUrl:        a.LogoUrl,
				DownloadCount:  a.DownloadCount,
				ThumbsUpCount:  a.ThumbsUpCount,
				PopularityRank: a.PopularityRank,
				GameVersions:   a.GameVersions,
				LastUpdatedAt:  a.LastUpdatedAt,
			}),
			Rank: offset + i + 1,
		}
		if a.RisingScore.Valid {
			f8, err := a.RisingScore.Float64Value()
			if err == nil {
				response[i].Score = f8.Float64
			}
		}
		if a.DownloadVelocity.Valid {
			f8, err := a.DownloadVelocity.Float64Value()
			if err == nil {
				response[i].DownloadVelocity = f8.Float64
			}
		}

		// Add rank changes if available
		if rc, ok := rankChangeMap[a.ID]; ok {
			if rc.Rank24hAgo.Valid {
				change := int(rc.Rank24hAgo.Int16 - rc.CurrentRank)
				response[i].RankChange24h = &change
			}
			if rc.Rank7dAgo.Valid {
				change := int(rc.Rank7dAgo.Int16 - rc.CurrentRank)
				response[i].RankChange7d = &change
			}
		}
	}

	respondWithPagination(c, response, page, perPage, int(total))
}
```

**Step 3: Verify build and run tests**

Run: `go build ./... && go test ./internal/api/... -v`

**Step 4: Test API manually**

Run: `curl "http://localhost:8080/api/v1/trending/hot?page=1&per_page=5" | jq '.meta'`

Expected: `{"page":1,"per_page":5,"total":7888,"total_pages":1578}`

**Step 5: Commit**

```bash
git add internal/api/handlers.go
git commit -m "feat(api): add pagination to trending endpoints"
```

---

## Task 3: Add Falling/Unchanged Badge Colors to CSS

**Files:**
- Modify: `web/src/app.css`

**Step 1: Add new badge color variables**

Add after `--color-hot: #EF4444;`:

```css
	--color-falling: #EF4444;
	--color-falling-bg: #FEE2E2;
	--color-unchanged: #6B7280;
	--color-unchanged-bg: #F3F4F6;
```

**Step 2: Commit**

```bash
git add web/src/app.css
git commit -m "style: add falling and unchanged badge colors"
```

---

## Task 4: Update RankBadge Component

**Files:**
- Modify: `web/src/lib/components/RankBadge.svelte`

**Step 1: Replace entire component**

```svelte
<script lang="ts">
	let { rankChange, isNew = false }: { rankChange: number | null; isNew?: boolean } = $props();

	const state = $derived(() => {
		if (isNew) return 'new';
		if (rankChange === null) return 'new'; // No history = new to list
		if (rankChange > 0) return 'rising';
		if (rankChange < 0) return 'falling';
		return 'unchanged';
	});

	const badgeText = $derived(() => {
		const s = state();
		if (s === 'new') return 'NEW';
		if (s === 'rising') return `+${rankChange}`;
		if (s === 'falling') return `${rankChange}`;
		return '=';
	});

	const showBadge = $derived(state() !== 'unchanged');
</script>

{#if showBadge}
	<span class="badge {state()}">
		{badgeText()}
	</span>
{/if}

<style>
	.badge {
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		white-space: nowrap;
	}

	.rising {
		background: var(--color-rising-bg);
		color: var(--color-rising);
	}

	.falling {
		background: var(--color-falling-bg);
		color: var(--color-falling);
	}

	.new {
		background: var(--color-new-bg);
		color: var(--color-new);
	}

	.unchanged {
		background: var(--color-unchanged-bg);
		color: var(--color-unchanged);
	}
</style>
```

**Step 2: Commit**

```bash
git add web/src/lib/components/RankBadge.svelte
git commit -m "feat(ui): update RankBadge to show all states"
```

---

## Task 5: Update Frontend API Client

**Files:**
- Modify: `web/src/lib/api.ts`

**Step 1: Update getTrendingHot function**

Replace the existing function:

```typescript
export async function getTrendingHot(
	page = 1,
	perPage = 20
): Promise<PaginatedResponse<TrendingAddon> | null> {
	const params = new URLSearchParams({
		page: String(page),
		per_page: String(perPage)
	});
	return fetchApi<PaginatedResponse<TrendingAddon>>(`/api/v1/trending/hot?${params}`);
}
```

**Step 2: Update getTrendingRising function**

Replace the existing function:

```typescript
export async function getTrendingRising(
	page = 1,
	perPage = 20
): Promise<PaginatedResponse<TrendingAddon> | null> {
	const params = new URLSearchParams({
		page: String(page),
		per_page: String(perPage)
	});
	return fetchApi<PaginatedResponse<TrendingAddon>>(`/api/v1/trending/rising?${params}`);
}
```

**Step 3: Commit**

```bash
git add web/src/lib/api.ts
git commit -m "feat(api): update trending functions for pagination"
```

---

## Task 6: Create FeaturedAddonCard Component

**Files:**
- Create: `web/src/lib/components/FeaturedAddonCard.svelte`

**Step 1: Create the component**

```svelte
<script lang="ts">
	import type { TrendingAddon } from '$lib/types';
	import RankBadge from './RankBadge.svelte';

	let {
		addon,
		velocityLabel = 'day'
	}: {
		addon: TrendingAddon;
		velocityLabel?: 'day' | 'week';
	} = $props();

	function formatDownloads(count: number): string {
		if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`;
		if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`;
		return String(count);
	}

	function formatVelocity(velocity: number): string {
		if (velocity >= 1_000) return `+${(velocity / 1_000).toFixed(1)}K`;
		return `+${Math.round(velocity)}`;
	}

	function formatTimeAgo(dateStr: string | undefined): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
		if (diffDays === 0) return 'Updated today';
		if (diffDays === 1) return 'Updated yesterday';
		if (diffDays < 7) return `Updated ${diffDays}d ago`;
		if (diffDays < 30) return `Updated ${Math.floor(diffDays / 7)}w ago`;
		return `Updated ${Math.floor(diffDays / 30)}mo ago`;
	}

	function truncateSummary(text: string | undefined, maxLen = 100): string {
		if (!text) return '';
		if (text.length <= maxLen) return text;
		return text.slice(0, maxLen).trimEnd() + '...';
	}

	const isNew = $derived(addon.rank_change_24h === null);
</script>

<a href="/addon/{addon.slug}" class="card">
	<div class="rank">#{addon.rank}</div>
	<div class="badge-container">
		<RankBadge rankChange={addon.rank_change_24h} {isNew} />
	</div>

	<div class="header">
		<div class="logo">
			{#if addon.logo_url}
				<img src={addon.logo_url} alt="{addon.name} logo" loading="lazy" />
			{:else}
				<div class="placeholder">?</div>
			{/if}
		</div>
		<div class="title">
			<h3>{addon.name}</h3>
			{#if addon.author_name}
				<p class="author">by {addon.author_name}</p>
			{/if}
		</div>
	</div>

	{#if addon.summary}
		<p class="summary">{truncateSummary(addon.summary)}</p>
	{/if}

	<div class="stats">
		<span>{formatDownloads(addon.download_count)} downloads</span>
		<span class="separator">·</span>
		<span>{formatDownloads(addon.thumbs_up_count)} likes</span>
		{#if addon.download_velocity > 0}
			<span class="separator">·</span>
			<span class="velocity">{formatVelocity(addon.download_velocity)}/{velocityLabel}</span>
		{/if}
	</div>

	<div class="meta">
		{#if addon.last_updated_at}
			<span>{formatTimeAgo(addon.last_updated_at)}</span>
		{/if}
		{#if addon.game_versions && addon.game_versions.length > 0}
			<span class="separator">·</span>
			<span class="versions">{addon.game_versions.slice(0, 3).join(', ')}</span>
		{/if}
	</div>
</a>

<style>
	.card {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 1.25rem;
		background: var(--color-surface);
		border-radius: 12px;
		color: var(--color-text);
		box-shadow: var(--shadow-sm);
		transition: box-shadow 0.2s, transform 0.2s;
	}

	.card:hover {
		box-shadow: var(--shadow-md);
		transform: translateY(-2px);
		text-decoration: none;
	}

	.rank {
		position: absolute;
		top: 12px;
		left: 12px;
		font-size: 0.875rem;
		font-weight: 700;
		color: var(--color-text-muted);
	}

	.badge-container {
		position: absolute;
		top: 12px;
		right: 12px;
	}

	.header {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-top: 0.5rem;
	}

	.logo {
		flex-shrink: 0;
		width: 64px;
		height: 64px;
	}

	.logo img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: 8px;
	}

	.placeholder {
		width: 100%;
		height: 100%;
		background: var(--color-border);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
		color: var(--color-text-muted);
	}

	.title {
		flex: 1;
		min-width: 0;
	}

	h3 {
		font-size: 1.125rem;
		font-weight: 600;
		margin-bottom: 0.125rem;
		letter-spacing: -0.01em;
	}

	.author {
		color: var(--color-text-muted);
		font-size: 0.875rem;
	}

	.summary {
		color: var(--color-text-muted);
		font-size: 0.875rem;
		line-height: 1.5;
	}

	.stats {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.8125rem;
		color: var(--color-text-muted);
	}

	.separator {
		color: var(--color-border);
	}

	.velocity {
		color: var(--color-rising);
		font-weight: 500;
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.75rem;
		color: var(--color-text-muted);
	}

	.versions {
		max-width: 200px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
```

**Step 2: Commit**

```bash
git add web/src/lib/components/FeaturedAddonCard.svelte
git commit -m "feat(ui): add FeaturedAddonCard component for top 3"
```

---

## Task 7: Update AddonCard Component

**Files:**
- Modify: `web/src/lib/components/AddonCard.svelte`

**Step 1: Replace entire component**

```svelte
<script lang="ts">
	import type { TrendingAddon, Addon } from '$lib/types';
	import RankBadge from './RankBadge.svelte';

	let {
		addon,
		showRank = false,
		showVelocity = false,
		velocityLabel = 'day'
	}: {
		addon: TrendingAddon | Addon;
		showRank?: boolean;
		showVelocity?: boolean;
		velocityLabel?: 'day' | 'week';
	} = $props();

	function formatDownloads(count: number): string {
		if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`;
		if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`;
		return String(count);
	}

	function formatVelocity(velocity: number): string {
		if (velocity >= 1_000) return `+${(velocity / 1_000).toFixed(1)}K`;
		return `+${Math.round(velocity)}`;
	}

	function formatTimeAgo(dateStr: string | undefined): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
		if (diffDays === 0) return 'today';
		if (diffDays === 1) return '1d ago';
		if (diffDays < 7) return `${diffDays}d ago`;
		if (diffDays < 30) return `${Math.floor(diffDays / 7)}w ago`;
		return `${Math.floor(diffDays / 30)}mo ago`;
	}

	function truncateSummary(text: string | undefined, maxLen = 60): string {
		if (!text) return '';
		if (text.length <= maxLen) return text;
		return text.slice(0, maxLen).trimEnd() + '...';
	}

	const isTrending = $derived('rank' in addon);
	const trendingAddon = $derived(isTrending ? (addon as TrendingAddon) : null);
	const rankChange = $derived(trendingAddon?.rank_change_24h ?? null);
	const velocity = $derived(trendingAddon?.download_velocity ?? 0);
	const isNew = $derived(isTrending && rankChange === null);
</script>

<a href="/addon/{addon.slug}" class="card">
	{#if showRank && trendingAddon}
		<div class="rank">#{trendingAddon.rank}</div>
	{/if}

	<div class="logo">
		{#if addon.logo_url}
			<img src={addon.logo_url} alt="{addon.name} logo" loading="lazy" />
		{:else}
			<div class="placeholder">?</div>
		{/if}
	</div>

	<div class="content">
		<div class="header">
			<h3>{addon.name}</h3>
			{#if isTrending}
				<RankBadge {rankChange} {isNew} />
			{/if}
		</div>
		{#if addon.author_name}
			<p class="author">by {addon.author_name}</p>
		{/if}
		{#if addon.summary}
			<p class="summary">{truncateSummary(addon.summary)}</p>
		{/if}
		<p class="stats">
			<span>{formatDownloads(addon.download_count)}</span>
			<span class="separator">·</span>
			<span>{formatDownloads(addon.thumbs_up_count)} likes</span>
			{#if showVelocity && velocity > 0}
				<span class="separator">·</span>
				<span class="velocity">{formatVelocity(velocity)}/{velocityLabel}</span>
			{/if}
			{#if addon.last_updated_at}
				<span class="separator">·</span>
				<span>{formatTimeAgo(addon.last_updated_at)}</span>
			{/if}
		</p>
	</div>
</a>

<style>
	.card {
		display: flex;
		align-items: flex-start;
		gap: 0.875rem;
		padding: 1rem;
		background: var(--color-surface);
		border-radius: 12px;
		color: var(--color-text);
		box-shadow: var(--shadow-sm);
		transition: box-shadow 0.2s, transform 0.2s;
	}

	.card:hover {
		box-shadow: var(--shadow-md);
		transform: translateY(-1px);
		text-decoration: none;
	}

	.rank {
		flex-shrink: 0;
		width: 2.5rem;
		font-size: 0.875rem;
		font-weight: 700;
		color: var(--color-text-muted);
		text-align: center;
		padding-top: 0.25rem;
	}

	.logo {
		flex-shrink: 0;
		width: 48px;
		height: 48px;
	}

	.logo img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: 8px;
	}

	.placeholder {
		width: 100%;
		height: 100%;
		background: var(--color-border);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.25rem;
		color: var(--color-text-muted);
	}

	.content {
		flex: 1;
		min-width: 0;
	}

	.header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.125rem;
	}

	h3 {
		font-size: 1rem;
		font-weight: 600;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		letter-spacing: -0.01em;
	}

	.author {
		color: var(--color-text-muted);
		font-size: 0.8125rem;
		margin-bottom: 0.25rem;
	}

	.summary {
		color: var(--color-text-muted);
		font-size: 0.8125rem;
		margin-bottom: 0.25rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.stats {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.25rem;
	}

	.separator {
		color: var(--color-border);
	}

	.velocity {
		color: var(--color-rising);
		font-weight: 500;
	}
</style>
```

**Step 2: Commit**

```bash
git add web/src/lib/components/AddonCard.svelte
git commit -m "feat(ui): update AddonCard with rank, summary, stats"
```

---

## Task 8: Update Trending Hot Page

**Files:**
- Modify: `web/src/routes/trending/hot/+page.server.ts`
- Modify: `web/src/routes/trending/hot/+page.svelte`

**Step 1: Update page.server.ts**

Replace entire file:

```typescript
import { getTrendingHot } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const result = await getTrendingHot(page, 20);

	return {
		addons: result?.data ?? [],
		meta: result?.meta ?? { page: 1, per_page: 20, total: 0, total_pages: 0 }
	};
};
```

**Step 2: Update page.svelte**

Replace entire file:

```svelte
<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import FeaturedAddonCard from '$lib/components/FeaturedAddonCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const isFirstPage = $derived(data.meta.page === 1);
	const hasPrevPage = $derived(data.meta.page > 1);
	const hasNextPage = $derived(data.meta.page < data.meta.total_pages);

	// On first page, split into featured (top 3) and rest
	const featuredAddons = $derived(isFirstPage ? data.addons.slice(0, 3) : []);
	const regularAddons = $derived(isFirstPage ? data.addons.slice(3) : data.addons);
</script>

<svelte:head>
	<title>Trending Addons - Addon Radar</title>
	<meta
		name="description"
		content="Discover trending World of Warcraft addons with high download velocity. Updated hourly."
	/>
</svelte:head>

<div class="page-header">
	<a href="/" class="back-link">Back to home</a>
	<h1>Trending</h1>
	<p class="subtitle">Established addons with high download velocity</p>
</div>

{#if data.addons.length > 0}
	{#if featuredAddons.length > 0}
		<div class="featured-grid">
			{#each featuredAddons as addon}
				<FeaturedAddonCard {addon} velocityLabel="day" />
			{/each}
		</div>
	{/if}

	{#if regularAddons.length > 0}
		<div class="addon-list">
			{#each regularAddons as addon}
				<AddonCard {addon} showRank={true} showVelocity={true} velocityLabel="day" />
			{/each}
		</div>
	{/if}

	{#if data.meta.total_pages > 1}
		<nav class="pagination">
			{#if hasPrevPage}
				<a href="/trending/hot?page={data.meta.page - 1}" class="page-link">Previous</a>
			{:else}
				<span class="page-link disabled">Previous</span>
			{/if}

			<span class="page-info">Page {data.meta.page} of {data.meta.total_pages}</span>

			{#if hasNextPage}
				<a href="/trending/hot?page={data.meta.page + 1}" class="page-link">Next</a>
			{:else}
				<span class="page-link disabled">Next</span>
			{/if}
		</nav>
	{/if}
{:else}
	<p class="empty">No trending addons available</p>
{/if}

<style>
	.page-header {
		margin-bottom: 2rem;
	}

	.back-link {
		display: inline-block;
		font-size: 0.875rem;
		color: var(--color-text-muted);
		margin-bottom: 1rem;
	}

	.back-link:hover {
		color: var(--color-accent);
	}

	h1 {
		font-size: 2rem;
		font-weight: 700;
		letter-spacing: -0.025em;
		margin-bottom: 0.25rem;
	}

	.subtitle {
		color: var(--color-text-muted);
	}

	.featured-grid {
		display: grid;
		gap: 1rem;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		margin-bottom: 1.5rem;
	}

	.addon-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.pagination {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: 1.5rem;
		margin-top: 2rem;
		padding: 1rem;
	}

	.page-link {
		padding: 0.5rem 1rem;
		font-size: 0.9rem;
		font-weight: 500;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		color: var(--color-text);
		transition: border-color 0.2s;
	}

	.page-link:hover:not(.disabled) {
		border-color: var(--color-accent);
		text-decoration: none;
	}

	.page-link.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.page-info {
		color: var(--color-text-muted);
		font-size: 0.875rem;
	}

	.empty {
		color: var(--color-text-muted);
		padding: 3rem;
		text-align: center;
		background: var(--color-surface);
		border-radius: 12px;
	}
</style>
```

**Step 3: Commit**

```bash
git add web/src/routes/trending/hot/
git commit -m "feat(ui): update Trending page with featured cards and pagination"
```

---

## Task 9: Update Trending Rising Page

**Files:**
- Modify: `web/src/routes/trending/rising/+page.server.ts`
- Modify: `web/src/routes/trending/rising/+page.svelte`

**Step 1: Update page.server.ts**

Replace entire file:

```typescript
import { getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const result = await getTrendingRising(page, 20);

	return {
		addons: result?.data ?? [],
		meta: result?.meta ?? { page: 1, per_page: 20, total: 0, total_pages: 0 }
	};
};
```

**Step 2: Update page.svelte**

Replace entire file:

```svelte
<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import FeaturedAddonCard from '$lib/components/FeaturedAddonCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const isFirstPage = $derived(data.meta.page === 1);
	const hasPrevPage = $derived(data.meta.page > 1);
	const hasNextPage = $derived(data.meta.page < data.meta.total_pages);

	// On first page, split into featured (top 3) and rest
	const featuredAddons = $derived(isFirstPage ? data.addons.slice(0, 3) : []);
	const regularAddons = $derived(isFirstPage ? data.addons.slice(3) : data.addons);
</script>

<svelte:head>
	<title>Rising Addons - Addon Radar</title>
	<meta
		name="description"
		content="Discover rising World of Warcraft addons gaining traction. Updated hourly."
	/>
</svelte:head>

<div class="page-header">
	<a href="/" class="back-link">Back to home</a>
	<h1>Rising</h1>
	<p class="subtitle">Smaller addons gaining traction</p>
</div>

{#if data.addons.length > 0}
	{#if featuredAddons.length > 0}
		<div class="featured-grid">
			{#each featuredAddons as addon}
				<FeaturedAddonCard {addon} velocityLabel="week" />
			{/each}
		</div>
	{/if}

	{#if regularAddons.length > 0}
		<div class="addon-list">
			{#each regularAddons as addon}
				<AddonCard {addon} showRank={true} showVelocity={true} velocityLabel="week" />
			{/each}
		</div>
	{/if}

	{#if data.meta.total_pages > 1}
		<nav class="pagination">
			{#if hasPrevPage}
				<a href="/trending/rising?page={data.meta.page - 1}" class="page-link">Previous</a>
			{:else}
				<span class="page-link disabled">Previous</span>
			{/if}

			<span class="page-info">Page {data.meta.page} of {data.meta.total_pages}</span>

			{#if hasNextPage}
				<a href="/trending/rising?page={data.meta.page + 1}" class="page-link">Next</a>
			{:else}
				<span class="page-link disabled">Next</span>
			{/if}
		</nav>
	{/if}
{:else}
	<p class="empty">No rising addons available</p>
{/if}

<style>
	.page-header {
		margin-bottom: 2rem;
	}

	.back-link {
		display: inline-block;
		font-size: 0.875rem;
		color: var(--color-text-muted);
		margin-bottom: 1rem;
	}

	.back-link:hover {
		color: var(--color-accent);
	}

	h1 {
		font-size: 2rem;
		font-weight: 700;
		letter-spacing: -0.025em;
		margin-bottom: 0.25rem;
	}

	.subtitle {
		color: var(--color-text-muted);
	}

	.featured-grid {
		display: grid;
		gap: 1rem;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		margin-bottom: 1.5rem;
	}

	.addon-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.pagination {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: 1.5rem;
		margin-top: 2rem;
		padding: 1rem;
	}

	.page-link {
		padding: 0.5rem 1rem;
		font-size: 0.9rem;
		font-weight: 500;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		color: var(--color-text);
		transition: border-color 0.2s;
	}

	.page-link:hover:not(.disabled) {
		border-color: var(--color-accent);
		text-decoration: none;
	}

	.page-link.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.page-info {
		color: var(--color-text-muted);
		font-size: 0.875rem;
	}

	.empty {
		color: var(--color-text-muted);
		padding: 3rem;
		text-align: center;
		background: var(--color-surface);
		border-radius: 12px;
	}
</style>
```

**Step 3: Commit**

```bash
git add web/src/routes/trending/rising/
git commit -m "feat(ui): update Rising page with featured cards and pagination"
```

---

## Task 10: Update Homepage Category Names

**Files:**
- Modify: `web/src/routes/+page.svelte`

**Step 1: Update section headings and links**

Find and replace:
- `"Hot Right Now"` → `"Trending"`
- `"Rising Stars"` → `"Rising"`
- Update hrefs if needed

**Step 2: Update to use featured cards for top 3**

Update the template to use FeaturedAddonCard for first 3 in each section, AddonCard for rest.

**Step 3: Commit**

```bash
git add web/src/routes/+page.svelte
git commit -m "feat(ui): update homepage with new category names and cards"
```

---

## Task 11: Build and Test Frontend

**Step 1: Run type check**

Run: `cd web && bun run check`

**Step 2: Run build**

Run: `bun run build`

**Step 3: Test locally**

Run: `bun run dev`

Visit:
- http://localhost:5173/ - Check homepage
- http://localhost:5173/trending/hot - Check pagination, featured cards
- http://localhost:5173/trending/rising - Check pagination, featured cards

**Step 4: Commit any fixes**

```bash
git add -A
git commit -m "fix: address build issues"
```

---

## Task 12: Final Testing and PR

**Step 1: Run all backend tests**

Run: `go test ./... -race -timeout=5m`

**Step 2: Run linter**

Run: `golangci-lint run ./...`

**Step 3: Test API endpoints**

```bash
curl "https://api.addon-radar.com/api/v1/trending/hot?page=1" | jq '.meta'
curl "https://api.addon-radar.com/api/v1/trending/rising?page=2" | jq '.data | length'
```

**Step 4: Create PR**

```bash
git push -u origin feature/frontend-v3
gh pr create --title "Frontend V3: Pagination, Rich Cards, Category Rename" --body "## Summary
- Add pagination to trending endpoints (7888 hot, 7225 rising addons)
- Enrich addon cards with rank position, summary, likes, update date
- Featured cards for top 3 addons on each page
- Rank badges show all states: up (green), down (red), unchanged (gray), new (blue)
- Rename categories: 'Hot Right Now' -> 'Trending', 'Rising Stars' -> 'Rising'

## Test Plan
- [ ] Verify pagination works on /trending/hot and /trending/rising
- [ ] Verify featured cards display correctly for top 3
- [ ] Verify rank badges show correct colors for all states
- [ ] Verify card stats (downloads, likes, velocity, update date)
- [ ] Verify responsive design on mobile

Generated with Claude Code"
```
