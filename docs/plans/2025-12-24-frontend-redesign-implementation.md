# Frontend Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign Addon Radar frontend with clean minimal design, meaningful data display (velocity + rank changes), and improved UX.

**Architecture:** Incremental refactoring of existing SvelteKit app. Update CSS variables first, then components, then add new pages. Backend already has velocity and rank data - just need to expose it in API responses and use it in frontend.

**Tech Stack:** SvelteKit 2.x, TypeScript, CSS custom properties, existing Go backend

---

## Task 1: Update Global CSS Variables (Color Palette)

**Files:**
- Modify: `web/src/app.css`

**Step 1: Update CSS variables for light theme**

Replace the entire content of `web/src/app.css`:

```css
:root {
	/* Core colors - Clean minimal with dark header accent */
	--color-bg: #FAFAFA;
	--color-surface: #FFFFFF;
	--color-header: #111827;
	--color-border: #E5E7EB;

	/* Text colors */
	--color-text: #1A1A1A;
	--color-text-muted: #6B7280;

	/* Accent colors */
	--color-accent: #3B82F6;
	--color-accent-hover: #2563EB;

	/* Badge colors */
	--color-rising: #10B981;
	--color-rising-bg: #D1FAE5;
	--color-new: #8B5CF6;
	--color-new-bg: #EDE9FE;
	--color-hot: #EF4444;

	/* Shadows */
	--shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
	--shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
}

* {
	box-sizing: border-box;
	margin: 0;
	padding: 0;
}

body {
	font-family: Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	background: var(--color-bg);
	color: var(--color-text);
	line-height: 1.6;
}

a {
	color: var(--color-accent);
	text-decoration: none;
}

a:hover {
	text-decoration: underline;
}
```

**Step 2: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/app.css
git commit -m "style: update color palette to clean minimal theme"
```

---

## Task 2: Update Layout Header (Dark Accent)

**Files:**
- Modify: `web/src/routes/+layout.svelte`

**Step 1: Update layout with dark header and improved search**

Replace the entire content of `web/src/routes/+layout.svelte`:

```svelte
<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet" />
</svelte:head>

<header>
	<nav>
		<a href="/" class="logo">Addon Radar</a>
		<div class="search-wrapper">
			<form action="/search" method="get" class="search-form">
				<svg class="search-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<circle cx="11" cy="11" r="8"></circle>
					<path d="m21 21-4.3-4.3"></path>
				</svg>
				<input type="search" name="q" placeholder="Search addons..." aria-label="Search addons" />
			</form>
		</div>
	</nav>
</header>

<main>
	{@render children()}
</main>

<footer>
	<p>Data from <a href="https://www.curseforge.com/wow" target="_blank" rel="noopener noreferrer">CurseForge</a>. Updated hourly.</p>
</footer>

<style>
	header {
		background: var(--color-header);
		padding: 1rem;
	}

	nav {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		align-items: center;
		gap: 2rem;
	}

	.logo {
		font-size: 1.5rem;
		font-weight: 700;
		color: #FFFFFF;
		letter-spacing: -0.025em;
	}

	.logo:hover {
		text-decoration: none;
		opacity: 0.9;
	}

	.search-wrapper {
		flex: 1;
		max-width: 400px;
	}

	.search-form {
		position: relative;
		display: flex;
		align-items: center;
	}

	.search-icon {
		position: absolute;
		left: 12px;
		color: var(--color-text-muted);
		pointer-events: none;
	}

	.search-form input {
		width: 100%;
		padding: 0.625rem 1rem 0.625rem 2.5rem;
		border: 1px solid var(--color-border);
		border-radius: 8px;
		background: var(--color-surface);
		color: var(--color-text);
		font-size: 0.9rem;
	}

	.search-form input::placeholder {
		color: var(--color-text-muted);
	}

	.search-form input:focus {
		outline: none;
		border-color: var(--color-accent);
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	main {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem 1rem;
		min-height: calc(100vh - 180px);
	}

	footer {
		background: var(--color-surface);
		border-top: 1px solid var(--color-border);
		padding: 2rem 1rem;
		text-align: center;
		color: var(--color-text-muted);
		font-size: 0.875rem;
	}
</style>
```

**Step 2: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/routes/+layout.svelte
git commit -m "style: update layout with dark header and improved search"
```

---

## Task 3: Update Frontend Types for Velocity and Rank

**Files:**
- Modify: `web/src/lib/types.ts`

**Step 1: Add velocity and rank fields to TrendingAddon**

Replace the entire content of `web/src/lib/types.ts`:

```typescript
export interface Addon {
	id: number;
	name: string;
	slug: string;
	summary?: string;
	author_name?: string;
	logo_url?: string;
	download_count: number;
	thumbs_up_count: number;
	popularity_rank?: number;
	game_versions: string[];
	last_updated_at?: string;
}

export interface TrendingAddon extends Addon {
	score: number;
	rank: number;
	rank_change_24h: number;
	rank_change_7d: number;
	download_velocity: number;
}

export interface Snapshot {
	recorded_at: string;
	download_count: number;
	thumbs_up_count?: number;
	popularity_rank?: number;
}

export interface DailySnapshot {
	date: string;
	download_count: number;
	downloads_delta: number;
}

export interface Category {
	id: number;
	name: string;
	slug: string;
	parent_id?: number;
	icon_url?: string;
}

export interface PaginatedResponse<T> {
	data: T[];
	meta: {
		page: number;
		per_page: number;
		total: number;
		total_pages: number;
	};
}

export interface DataResponse<T> {
	data: T;
}
```

**Step 2: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors (may have type errors in components that we'll fix next)

**Step 3: Commit**

```bash
git add web/src/lib/types.ts
git commit -m "feat: add velocity and rank fields to TrendingAddon type"
```

---

## Task 4: Create RankBadge Component

**Files:**
- Create: `web/src/lib/components/RankBadge.svelte`

**Step 1: Create the RankBadge component**

Create file `web/src/lib/components/RankBadge.svelte`:

```svelte
<script lang="ts">
	let { rankChange, isNew = false }: { rankChange: number; isNew?: boolean } = $props();

	const showBadge = $derived(isNew || rankChange > 0);
	const badgeText = $derived(isNew ? 'New' : `↑${rankChange}`);
	const badgeClass = $derived(isNew ? 'new' : 'rising');
</script>

{#if showBadge}
	<span class="badge {badgeClass}">
		{badgeText}
	</span>
{/if}

<style>
	.badge {
		position: absolute;
		top: 8px;
		left: 8px;
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		z-index: 1;
	}

	.rising {
		background: var(--color-rising-bg);
		color: var(--color-rising);
	}

	.new {
		background: var(--color-new-bg);
		color: var(--color-new);
	}
</style>
```

**Step 2: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/lib/components/RankBadge.svelte
git commit -m "feat: add RankBadge component for rank change display"
```

---

## Task 5: Redesign AddonCard Component

**Files:**
- Modify: `web/src/lib/components/AddonCard.svelte`

**Step 1: Update AddonCard with new design**

Replace the entire content of `web/src/lib/components/AddonCard.svelte`:

```svelte
<script lang="ts">
	import type { TrendingAddon, Addon } from '$lib/types';
	import RankBadge from './RankBadge.svelte';

	let {
		addon,
		showVelocity = false,
		velocityLabel = 'day'
	}: {
		addon: TrendingAddon | Addon;
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

	const isTrending = $derived('rank' in addon);
	const rankChange = $derived(isTrending ? (addon as TrendingAddon).rank_change_24h : 0);
	const velocity = $derived(isTrending ? (addon as TrendingAddon).download_velocity : 0);
	const isNew = $derived(isTrending && (addon as TrendingAddon).rank === 1 && rankChange === 0);
</script>

<a href="/addon/{addon.slug}" class="card">
	{#if isTrending}
		<RankBadge {rankChange} {isNew} />
	{/if}

	<div class="logo">
		{#if addon.logo_url}
			<img src={addon.logo_url} alt="{addon.name} logo" loading="lazy" />
		{:else}
			<div class="placeholder">?</div>
		{/if}
	</div>

	<div class="info">
		<h3>{addon.name}</h3>
		{#if addon.author_name}
			<p class="author">by {addon.author_name}</p>
		{/if}
		<p class="stats">
			<span class="downloads">{formatDownloads(addon.download_count)} downloads</span>
			{#if showVelocity && velocity > 0}
				<span class="velocity">{formatVelocity(velocity)}/{velocityLabel}</span>
			{/if}
		</p>
	</div>
</a>

<style>
	.card {
		position: relative;
		display: flex;
		align-items: center;
		gap: 1rem;
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

	.logo {
		flex-shrink: 0;
		width: 56px;
		height: 56px;
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

	.info {
		flex: 1;
		min-width: 0;
	}

	h3 {
		font-size: 1rem;
		font-weight: 600;
		margin-bottom: 0.125rem;
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

	.stats {
		font-size: 0.8125rem;
		color: var(--color-text-muted);
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.velocity {
		color: var(--color-rising);
		font-weight: 500;
	}
</style>
```

**Step 2: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/lib/components/AddonCard.svelte
git commit -m "feat: redesign AddonCard with velocity and rank badge"
```

---

## Task 6: Update Homepage with View All Links

**Files:**
- Modify: `web/src/routes/+page.svelte`

**Step 1: Update homepage with new design and view all links**

Replace the entire content of `web/src/routes/+page.svelte`:

```svelte
<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Addon Radar - Trending WoW Addons</title>
	<meta name="description" content="Discover trending and rising World of Warcraft addons. Updated hourly." />
	<meta property="og:title" content="Addon Radar - Trending WoW Addons" />
	<meta property="og:description" content="Discover trending and rising World of Warcraft addons. Updated hourly." />
	<meta property="og:type" content="website" />
</svelte:head>

<div class="hero">
	<h1>Trending WoW Addons</h1>
	<p class="subtitle">Discover what's hot and rising in the World of Warcraft addon community</p>
</div>

<section class="trending-section">
	<div class="section-header">
		<div>
			<h2>Hot Right Now</h2>
			<p class="section-desc">Established addons with high download velocity</p>
		</div>
		<a href="/trending/hot" class="view-all">View all →</a>
	</div>

	{#if data.hot.length > 0}
		<div class="addon-grid">
			{#each data.hot.slice(0, 10) as addon}
				<AddonCard {addon} showVelocity={true} velocityLabel="day" />
			{/each}
		</div>
	{:else}
		<p class="empty">No trending data available</p>
	{/if}
</section>

<section class="trending-section">
	<div class="section-header">
		<div>
			<h2>Rising Stars</h2>
			<p class="section-desc">Smaller addons gaining traction</p>
		</div>
		<a href="/trending/rising" class="view-all">View all →</a>
	</div>

	{#if data.rising.length > 0}
		<div class="addon-grid">
			{#each data.rising.slice(0, 10) as addon}
				<AddonCard {addon} showVelocity={true} velocityLabel="week" />
			{/each}
		</div>
	{:else}
		<p class="empty">No rising addons available</p>
	{/if}
</section>

<style>
	.hero {
		text-align: center;
		margin-bottom: 3rem;
	}

	h1 {
		font-size: 2.25rem;
		font-weight: 700;
		letter-spacing: -0.025em;
		margin-bottom: 0.5rem;
	}

	.subtitle {
		color: var(--color-text-muted);
		font-size: 1.125rem;
	}

	.trending-section {
		margin-bottom: 3rem;
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 1.25rem;
	}

	h2 {
		font-size: 1.5rem;
		font-weight: 600;
		letter-spacing: -0.025em;
		margin-bottom: 0.25rem;
	}

	.section-desc {
		color: var(--color-text-muted);
		font-size: 0.9rem;
	}

	.view-all {
		font-size: 0.9rem;
		font-weight: 500;
		color: var(--color-accent);
		white-space: nowrap;
	}

	.view-all:hover {
		text-decoration: underline;
	}

	.addon-grid {
		display: grid;
		gap: 1rem;
		grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
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

**Step 2: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/routes/+page.svelte
git commit -m "feat: update homepage with view all links and new styling"
```

---

## Task 7: Create Trending Hot Page

**Files:**
- Create: `web/src/routes/trending/hot/+page.server.ts`
- Create: `web/src/routes/trending/hot/+page.svelte`

**Step 1: Create the server loader**

Create file `web/src/routes/trending/hot/+page.server.ts`:

```typescript
import { getTrendingHot } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const perPage = 20;

	const allHot = await getTrendingHot();
	const totalPages = Math.ceil(allHot.length / perPage);
	const start = (page - 1) * perPage;
	const addons = allHot.slice(start, start + perPage);

	return {
		addons,
		meta: {
			page,
			perPage,
			total: allHot.length,
			totalPages
		}
	};
};
```

**Step 2: Create the page component**

Create file `web/src/routes/trending/hot/+page.svelte`:

```svelte
<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Hot Right Now - Addon Radar</title>
	<meta name="description" content="Established WoW addons with high download velocity. Updated hourly." />
</svelte:head>

<div class="page-header">
	<a href="/" class="back-link">← Back to home</a>
	<h1>Hot Right Now</h1>
	<p class="subtitle">Established addons with high download velocity</p>
</div>

{#if data.addons.length > 0}
	<div class="addon-list">
		{#each data.addons as addon}
			<AddonCard {addon} showVelocity={true} velocityLabel="day" />
		{/each}
	</div>

	{#if data.meta.totalPages > 1}
		<nav class="pagination">
			{#if data.meta.page > 1}
				<a href="/trending/hot?page={data.meta.page - 1}" class="page-link">← Previous</a>
			{:else}
				<span class="page-link disabled">← Previous</span>
			{/if}

			<span class="page-info">Page {data.meta.page} of {data.meta.totalPages}</span>

			{#if data.meta.page < data.meta.totalPages}
				<a href="/trending/hot?page={data.meta.page + 1}" class="page-link">Next →</a>
			{:else}
				<span class="page-link disabled">Next →</span>
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
		font-size: 0.9rem;
		font-weight: 500;
		color: var(--color-accent);
	}

	.page-link.disabled {
		color: var(--color-text-muted);
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

**Step 3: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/routes/trending/hot/+page.server.ts web/src/routes/trending/hot/+page.svelte
git commit -m "feat: add paginated Hot Right Now page"
```

---

## Task 8: Create Trending Rising Page

**Files:**
- Create: `web/src/routes/trending/rising/+page.server.ts`
- Create: `web/src/routes/trending/rising/+page.svelte`

**Step 1: Create the server loader**

Create file `web/src/routes/trending/rising/+page.server.ts`:

```typescript
import { getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const perPage = 20;

	const allRising = await getTrendingRising();
	const totalPages = Math.ceil(allRising.length / perPage);
	const start = (page - 1) * perPage;
	const addons = allRising.slice(start, start + perPage);

	return {
		addons,
		meta: {
			page,
			perPage,
			total: allRising.length,
			totalPages
		}
	};
};
```

**Step 2: Create the page component**

Create file `web/src/routes/trending/rising/+page.svelte`:

```svelte
<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Rising Stars - Addon Radar</title>
	<meta name="description" content="Smaller WoW addons gaining traction. Updated hourly." />
</svelte:head>

<div class="page-header">
	<a href="/" class="back-link">← Back to home</a>
	<h1>Rising Stars</h1>
	<p class="subtitle">Smaller addons gaining traction</p>
</div>

{#if data.addons.length > 0}
	<div class="addon-list">
		{#each data.addons as addon}
			<AddonCard {addon} showVelocity={true} velocityLabel="week" />
		{/each}
	</div>

	{#if data.meta.totalPages > 1}
		<nav class="pagination">
			{#if data.meta.page > 1}
				<a href="/trending/rising?page={data.meta.page - 1}" class="page-link">← Previous</a>
			{:else}
				<span class="page-link disabled">← Previous</span>
			{/if}

			<span class="page-info">Page {data.meta.page} of {data.meta.totalPages}</span>

			{#if data.meta.page < data.meta.totalPages}
				<a href="/trending/rising?page={data.meta.page + 1}" class="page-link">Next →</a>
			{:else}
				<span class="page-link disabled">Next →</span>
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
		font-size: 0.9rem;
		font-weight: 500;
		color: var(--color-accent);
	}

	.page-link.disabled {
		color: var(--color-text-muted);
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

**Step 3: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/routes/trending/rising/+page.server.ts web/src/routes/trending/rising/+page.svelte
git commit -m "feat: add paginated Rising Stars page"
```

---

## Task 9: Update Backend API to Include Velocity

**Files:**
- Modify: `internal/api/handlers.go`

**Step 1: Find and read the trending handler**

First, read the current implementation to understand structure.

**Step 2: Update TrendingAddonResponse to include download_velocity**

The struct already has rank fields. Add `DownloadVelocity` field and update the handler mapping.

In `internal/api/handlers.go`, update the `TrendingAddonResponse` struct to add:

```go
DownloadVelocity float64 `json:"download_velocity"`
```

And update the `handleTrendingHot` and `handleTrendingRising` methods to populate this field from the trending_scores table.

**Step 3: Verify with tests**

Run: `go test ./internal/api/... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/api/handlers.go
git commit -m "feat: include download_velocity in trending API response"
```

---

## Task 10: Redesign Addon Detail Page

**Files:**
- Modify: `web/src/routes/addon/[slug]/+page.svelte`
- Modify: `web/src/routes/addon/[slug]/+page.server.ts`

**Step 1: Update the page loader to get weekly aggregated data**

Modify `web/src/routes/addon/[slug]/+page.server.ts` to aggregate snapshots by day:

```typescript
import { getAddon, getAddonHistory } from '$lib/api';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
	const addon = await getAddon(params.slug);
	if (!addon) {
		throw error(404, 'Addon not found');
	}

	// Get 4 weeks of history (672 hours)
	const history = await getAddonHistory(params.slug, 672);

	// Aggregate by day
	const dailyData = aggregateByDay(history);

	return {
		addon,
		dailyHistory: dailyData
	};
};

function aggregateByDay(snapshots: { recorded_at: string; download_count: number }[]) {
	const byDay = new Map<string, { count: number; downloads: number[] }>();

	for (const snap of snapshots) {
		const date = snap.recorded_at.split('T')[0];
		if (!byDay.has(date)) {
			byDay.set(date, { count: 0, downloads: [] });
		}
		const day = byDay.get(date)!;
		day.count++;
		day.downloads.push(snap.download_count);
	}

	// Get last download of each day and calculate delta
	const result: { date: string; download_count: number; downloads_delta: number }[] = [];
	const sortedDates = [...byDay.keys()].sort().reverse();

	for (let i = 0; i < sortedDates.length && i < 28; i++) {
		const date = sortedDates[i];
		const day = byDay.get(date)!;
		const downloads = Math.max(...day.downloads);
		const prevDate = sortedDates[i + 1];
		const prevDownloads = prevDate ? Math.max(...byDay.get(prevDate)!.downloads) : downloads;

		result.push({
			date,
			download_count: downloads,
			downloads_delta: downloads - prevDownloads
		});
	}

	return result.reverse(); // oldest first for chart
}
```

**Step 2: Update the page component**

Replace content of `web/src/routes/addon/[slug]/+page.svelte` with simplified design - stat cards, version simplification, and trend chart.

**Step 3: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/routes/addon/[slug]/+page.server.ts web/src/routes/addon/[slug]/+page.svelte
git commit -m "feat: redesign addon detail page with trend chart"
```

---

## Task 11: Add Search Autocomplete Component

**Files:**
- Create: `web/src/lib/components/SearchAutocomplete.svelte`
- Modify: `web/src/routes/+layout.svelte`
- Modify: `web/src/lib/api.ts`

**Step 1: Add quick search API function**

Add to `web/src/lib/api.ts`:

```typescript
export async function searchAddonsQuick(query: string): Promise<Addon[]> {
	if (query.length < 2) return [];
	const res = await searchAddons(query, 1, 5);
	return res?.data ?? [];
}
```

**Step 2: Create SearchAutocomplete component**

Create `web/src/lib/components/SearchAutocomplete.svelte` with debounced input, dropdown results, and keyboard navigation.

**Step 3: Update layout to use autocomplete**

Replace the search form in `+layout.svelte` with the new component.

**Step 4: Verify the changes compile**

Run: `cd web && bun run check`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/lib/components/SearchAutocomplete.svelte web/src/lib/api.ts web/src/routes/+layout.svelte
git commit -m "feat: add search autocomplete with dropdown results"
```

---

## Task 12: Final Testing and Cleanup

**Files:**
- Review all changed files

**Step 1: Run full type check**

Run: `cd web && bun run check`
Expected: No errors

**Step 2: Run development server and visual test**

Run: `cd web && bun run dev`

Test:
1. Homepage loads with new clean design
2. Cards show velocity and rank badges
3. "View all" links work
4. Trending pages have pagination
5. Addon detail page shows trend chart
6. Search autocomplete works

**Step 3: Build for production**

Run: `cd web && bun run build`
Expected: Build succeeds

**Step 4: Final commit**

```bash
git add -A
git commit -m "chore: final cleanup for frontend redesign"
```

---

## Summary

This plan covers:

1. **CSS Variables** - New color palette (light theme with dark header)
2. **Layout** - Dark header, improved search styling
3. **Types** - Added velocity and rank fields
4. **RankBadge** - New component for position changes
5. **AddonCard** - Redesigned with velocity display
6. **Homepage** - View all links, 10 items per section
7. **Trending Hot Page** - Paginated list
8. **Trending Rising Page** - Paginated list
9. **Backend API** - Include velocity in response
10. **Addon Detail Page** - Simplified with trend chart
11. **Search Autocomplete** - Dropdown with top 5 results
12. **Testing** - Visual verification and build check
