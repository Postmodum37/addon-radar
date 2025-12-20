# SvelteKit Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a SvelteKit frontend with SSR that displays trending WoW addons, deployed on Railway.

**Architecture:** SvelteKit app with Bun runtime, making server-side API calls to existing Go REST API. Three pages: homepage (trending), addon detail, and search.

**Tech Stack:** SvelteKit, Bun, TypeScript, adapter-bun, Railway

---

## Task 1: Initialize SvelteKit Project

**Files:**
- Create: `web/` directory with SvelteKit scaffold
- Create: `web/package.json`
- Create: `web/svelte.config.js`
- Create: `web/bun.lockb`

**Step 1: Create SvelteKit project with Bun**

```bash
cd /Users/tomas/Workspace/addon-radar
bunx sv create web --template minimal --types ts --no-add-ons --no-install
```

Select when prompted:
- Template: minimal
- TypeScript: Yes
- Add-ons: None

**Step 2: Install dependencies**

```bash
cd web
bun install
```

**Step 3: Add Bun adapter**

```bash
bun add -d @sveltejs/adapter-bun
```

**Step 4: Configure adapter-bun**

Replace `web/svelte.config.js`:

```javascript
import adapter from '@sveltejs/adapter-bun';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter()
	}
};

export default config;
```

**Step 5: Verify dev server starts**

```bash
bun run dev
```

Expected: Server starts on http://localhost:5173

**Step 6: Commit**

```bash
cd /Users/tomas/Workspace/addon-radar
git add web/
git commit -m "feat(web): initialize SvelteKit project with Bun"
```

---

## Task 2: Create API Client and Types

**Files:**
- Create: `web/src/lib/api.ts`
- Create: `web/src/lib/types.ts`

**Step 1: Create TypeScript types matching Go API responses**

Create `web/src/lib/types.ts`:

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
}

export interface Snapshot {
	recorded_at: string;
	download_count: number;
	thumbs_up_count?: number;
	popularity_rank?: number;
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
	pagination: {
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

**Step 2: Create API client**

Create `web/src/lib/api.ts`:

```typescript
import type {
	Addon,
	TrendingAddon,
	Snapshot,
	PaginatedResponse,
	DataResponse
} from './types';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

async function fetchApi<T>(path: string): Promise<T | null> {
	try {
		const res = await fetch(`${API_URL}${path}`);
		if (!res.ok) return null;
		return res.json();
	} catch {
		return null;
	}
}

export async function getTrendingHot(): Promise<TrendingAddon[]> {
	const res = await fetchApi<DataResponse<TrendingAddon[]>>('/api/v1/trending/hot');
	return res?.data ?? [];
}

export async function getTrendingRising(): Promise<TrendingAddon[]> {
	const res = await fetchApi<DataResponse<TrendingAddon[]>>('/api/v1/trending/rising');
	return res?.data ?? [];
}

export async function getAddon(slug: string): Promise<Addon | null> {
	const res = await fetchApi<DataResponse<Addon>>(`/api/v1/addons/${slug}`);
	return res?.data ?? null;
}

export async function getAddonHistory(slug: string, limit = 168): Promise<Snapshot[]> {
	const res = await fetchApi<DataResponse<Snapshot[]>>(
		`/api/v1/addons/${slug}/history?limit=${limit}`
	);
	return res?.data ?? [];
}

export async function searchAddons(
	query: string,
	page = 1,
	perPage = 20
): Promise<PaginatedResponse<Addon> | null> {
	const params = new URLSearchParams({
		search: query,
		page: String(page),
		per_page: String(perPage)
	});
	return fetchApi<PaginatedResponse<Addon>>(`/api/v1/addons?${params}`);
}
```

**Step 3: Commit**

```bash
git add web/src/lib/
git commit -m "feat(web): add API client and TypeScript types"
```

---

## Task 3: Create Shared Layout

**Files:**
- Modify: `web/src/routes/+layout.svelte`
- Create: `web/src/app.css`

**Step 1: Create global styles**

Create `web/src/app.css`:

```css
:root {
	--color-bg: #0f0f0f;
	--color-surface: #1a1a1a;
	--color-border: #2a2a2a;
	--color-text: #e5e5e5;
	--color-text-muted: #888;
	--color-accent: #ff8c00;
	--color-hot: #ff4444;
	--color-rising: #44ff44;
}

* {
	box-sizing: border-box;
	margin: 0;
	padding: 0;
}

body {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
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

**Step 2: Create layout with header**

Replace `web/src/routes/+layout.svelte`:

```svelte
<script>
	import '../app.css';
</script>

<header>
	<nav>
		<a href="/" class="logo">Addon Radar</a>
		<form action="/search" method="get" class="search-form">
			<input type="search" name="q" placeholder="Search addons..." />
		</form>
	</nav>
</header>

<main>
	<slot />
</main>

<footer>
	<p>Data from <a href="https://www.curseforge.com/wow" target="_blank">CurseForge</a>. Updated hourly.</p>
</footer>

<style>
	header {
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
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
		font-weight: bold;
		color: var(--color-accent);
	}

	.logo:hover {
		text-decoration: none;
	}

	.search-form {
		flex: 1;
		max-width: 400px;
	}

	.search-form input {
		width: 100%;
		padding: 0.5rem 1rem;
		border: 1px solid var(--color-border);
		border-radius: 4px;
		background: var(--color-bg);
		color: var(--color-text);
	}

	main {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem 1rem;
		min-height: calc(100vh - 200px);
	}

	footer {
		background: var(--color-surface);
		border-top: 1px solid var(--color-border);
		padding: 2rem 1rem;
		text-align: center;
		color: var(--color-text-muted);
	}
</style>
```

**Step 3: Verify layout renders**

```bash
bun run dev
```

Visit http://localhost:5173 - should see header with logo and search, footer with CurseForge credit.

**Step 4: Commit**

```bash
git add web/src/
git commit -m "feat(web): add shared layout with header and footer"
```

---

## Task 4: Create AddonCard Component

**Files:**
- Create: `web/src/lib/components/AddonCard.svelte`

**Step 1: Create the addon card component**

Create `web/src/lib/components/AddonCard.svelte`:

```svelte
<script lang="ts">
	import type { TrendingAddon, Addon } from '$lib/types';

	export let addon: TrendingAddon | Addon;
	export let showScore = false;
	export let scoreType: 'hot' | 'rising' = 'hot';

	function formatDownloads(count: number): string {
		if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`;
		if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`;
		return String(count);
	}

	$: score = 'score' in addon ? addon.score : 0;
</script>

<a href="/addon/{addon.slug}" class="card">
	<div class="logo">
		{#if addon.logo_url}
			<img src={addon.logo_url} alt="{addon.name} logo" />
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
			{#if addon.thumbs_up_count > 0}
				<span class="thumbs">+{addon.thumbs_up_count}</span>
			{/if}
		</p>
	</div>
	{#if showScore && score > 0}
		<div class="score" class:hot={scoreType === 'hot'} class:rising={scoreType === 'rising'}>
			{score.toFixed(1)}
		</div>
	{/if}
</a>

<style>
	.card {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 1rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		color: var(--color-text);
		transition: border-color 0.2s;
	}

	.card:hover {
		border-color: var(--color-accent);
		text-decoration: none;
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
		border-radius: 4px;
	}

	.placeholder {
		width: 100%;
		height: 100%;
		background: var(--color-border);
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
		color: var(--color-text-muted);
	}

	.info {
		flex: 1;
		min-width: 0;
	}

	h3 {
		font-size: 1.1rem;
		margin-bottom: 0.25rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.author {
		color: var(--color-text-muted);
		font-size: 0.9rem;
	}

	.stats {
		font-size: 0.85rem;
		color: var(--color-text-muted);
		display: flex;
		gap: 1rem;
		margin-top: 0.25rem;
	}

	.thumbs {
		color: #4ade80;
	}

	.score {
		flex-shrink: 0;
		font-size: 1.25rem;
		font-weight: bold;
		padding: 0.5rem 0.75rem;
		border-radius: 4px;
	}

	.score.hot {
		background: rgba(255, 68, 68, 0.2);
		color: var(--color-hot);
	}

	.score.rising {
		background: rgba(68, 255, 68, 0.2);
		color: var(--color-rising);
	}
</style>
```

**Step 2: Commit**

```bash
git add web/src/lib/components/
git commit -m "feat(web): add AddonCard component"
```

---

## Task 5: Create Homepage with Trending Lists

**Files:**
- Modify: `web/src/routes/+page.svelte`
- Create: `web/src/routes/+page.server.ts`

**Step 1: Create server-side data loading**

Create `web/src/routes/+page.server.ts`:

```typescript
import { getTrendingHot, getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
	const [hot, rising] = await Promise.all([
		getTrendingHot(),
		getTrendingRising()
	]);

	return {
		hot,
		rising
	};
};
```

**Step 2: Create homepage UI**

Replace `web/src/routes/+page.svelte`:

```svelte
<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import type { PageData } from './$types';

	export let data: PageData;
</script>

<svelte:head>
	<title>Addon Radar - Trending WoW Addons</title>
	<meta name="description" content="Discover trending and rising World of Warcraft addons. Updated hourly." />
	<meta property="og:title" content="Addon Radar - Trending WoW Addons" />
	<meta property="og:description" content="Discover trending and rising World of Warcraft addons. Updated hourly." />
	<meta property="og:type" content="website" />
</svelte:head>

<h1>Trending WoW Addons</h1>
<p class="subtitle">Discover what's hot and rising in the World of Warcraft addon community</p>

<section class="trending-section">
	<h2 class="section-title hot">Hot Right Now</h2>
	<p class="section-desc">Established addons with high download velocity</p>

	{#if data.hot.length > 0}
		<div class="addon-grid">
			{#each data.hot as addon}
				<AddonCard {addon} showScore={true} scoreType="hot" />
			{/each}
		</div>
	{:else}
		<p class="empty">No trending data available</p>
	{/if}
</section>

<section class="trending-section">
	<h2 class="section-title rising">Rising Stars</h2>
	<p class="section-desc">Smaller addons gaining traction</p>

	{#if data.rising.length > 0}
		<div class="addon-grid">
			{#each data.rising as addon}
				<AddonCard {addon} showScore={true} scoreType="rising" />
			{/each}
		</div>
	{:else}
		<p class="empty">No rising addons available</p>
	{/if}
</section>

<style>
	h1 {
		font-size: 2rem;
		margin-bottom: 0.5rem;
	}

	.subtitle {
		color: var(--color-text-muted);
		margin-bottom: 2rem;
	}

	.trending-section {
		margin-bottom: 3rem;
	}

	.section-title {
		font-size: 1.5rem;
		margin-bottom: 0.25rem;
	}

	.section-title.hot {
		color: var(--color-hot);
	}

	.section-title.rising {
		color: var(--color-rising);
	}

	.section-desc {
		color: var(--color-text-muted);
		margin-bottom: 1rem;
		font-size: 0.9rem;
	}

	.addon-grid {
		display: grid;
		gap: 1rem;
		grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
	}

	.empty {
		color: var(--color-text-muted);
		padding: 2rem;
		text-align: center;
		background: var(--color-surface);
		border-radius: 8px;
	}
</style>
```

**Step 3: Test with local API**

Start the Go API in another terminal:
```bash
cd /Users/tomas/Workspace/addon-radar
source .env && ./bin/web
```

Start SvelteKit dev server:
```bash
cd web
VITE_API_URL=http://localhost:8080 bun run dev
```

Visit http://localhost:5173 - should see trending lists with real data.

**Step 4: Commit**

```bash
git add web/src/routes/
git commit -m "feat(web): add homepage with trending lists"
```

---

## Task 6: Create Addon Detail Page

**Files:**
- Create: `web/src/routes/addon/[slug]/+page.server.ts`
- Create: `web/src/routes/addon/[slug]/+page.svelte`

**Step 1: Create server-side data loading**

Create `web/src/routes/addon/[slug]/+page.server.ts`:

```typescript
import { error } from '@sveltejs/kit';
import { getAddon, getAddonHistory } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
	const [addon, history] = await Promise.all([
		getAddon(params.slug),
		getAddonHistory(params.slug)
	]);

	if (!addon) {
		throw error(404, 'Addon not found');
	}

	return {
		addon,
		history
	};
};
```

**Step 2: Create addon detail page UI**

Create `web/src/routes/addon/[slug]/+page.svelte`:

```svelte
<script lang="ts">
	import type { PageData } from './$types';

	export let data: PageData;

	function formatDownloads(count: number): string {
		return count.toLocaleString();
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	$: curseforgeUrl = `https://www.curseforge.com/wow/addons/${data.addon.slug}`;
</script>

<svelte:head>
	<title>{data.addon.name} - Addon Radar</title>
	<meta name="description" content={data.addon.summary || `${data.addon.name} WoW addon`} />
	<meta property="og:title" content="{data.addon.name} - Addon Radar" />
	<meta property="og:description" content={data.addon.summary || `${data.addon.name} WoW addon`} />
	{#if data.addon.logo_url}
		<meta property="og:image" content={data.addon.logo_url} />
	{/if}
</svelte:head>

<article class="addon-detail">
	<header>
		{#if data.addon.logo_url}
			<img src={data.addon.logo_url} alt="{data.addon.name} logo" class="logo" />
		{/if}
		<div class="header-info">
			<h1>{data.addon.name}</h1>
			{#if data.addon.author_name}
				<p class="author">by {data.addon.author_name}</p>
			{/if}
		</div>
	</header>

	{#if data.addon.summary}
		<p class="summary">{data.addon.summary}</p>
	{/if}

	<div class="stats-grid">
		<div class="stat">
			<span class="stat-value">{formatDownloads(data.addon.download_count)}</span>
			<span class="stat-label">Downloads</span>
		</div>
		<div class="stat">
			<span class="stat-value">+{data.addon.thumbs_up_count}</span>
			<span class="stat-label">Thumbs Up</span>
		</div>
		{#if data.addon.popularity_rank}
			<div class="stat">
				<span class="stat-value">#{data.addon.popularity_rank}</span>
				<span class="stat-label">Popularity</span>
			</div>
		{/if}
		{#if data.addon.last_updated_at}
			<div class="stat">
				<span class="stat-value">{formatDate(data.addon.last_updated_at)}</span>
				<span class="stat-label">Last Updated</span>
			</div>
		{/if}
	</div>

	{#if data.addon.game_versions?.length > 0}
		<div class="versions">
			<h3>Game Versions</h3>
			<div class="version-tags">
				{#each data.addon.game_versions as version}
					<span class="tag">{version}</span>
				{/each}
			</div>
		</div>
	{/if}

	<div class="actions">
		<a href={curseforgeUrl} target="_blank" rel="noopener" class="btn primary">
			View on CurseForge
		</a>
		<a href="/" class="btn secondary">Back to Trending</a>
	</div>

	{#if data.history.length > 0}
		<section class="history">
			<h3>Download History (Last 7 Days)</h3>
			<div class="history-list">
				{#each data.history.slice(0, 24) as snapshot}
					<div class="history-item">
						<span class="history-date">{formatDate(snapshot.recorded_at)}</span>
						<span class="history-downloads">{formatDownloads(snapshot.download_count)}</span>
					</div>
				{/each}
			</div>
		</section>
	{/if}
</article>

<style>
	.addon-detail {
		max-width: 800px;
	}

	header {
		display: flex;
		gap: 1.5rem;
		align-items: flex-start;
		margin-bottom: 1.5rem;
	}

	.logo {
		width: 128px;
		height: 128px;
		border-radius: 8px;
		object-fit: cover;
	}

	.header-info h1 {
		font-size: 2rem;
		margin-bottom: 0.5rem;
	}

	.author {
		color: var(--color-text-muted);
		font-size: 1.1rem;
	}

	.summary {
		font-size: 1.1rem;
		line-height: 1.7;
		margin-bottom: 2rem;
		color: var(--color-text);
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.stat {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		padding: 1rem;
		text-align: center;
	}

	.stat-value {
		display: block;
		font-size: 1.5rem;
		font-weight: bold;
		color: var(--color-accent);
	}

	.stat-label {
		color: var(--color-text-muted);
		font-size: 0.85rem;
	}

	.versions {
		margin-bottom: 2rem;
	}

	.versions h3 {
		margin-bottom: 0.5rem;
	}

	.version-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.tag {
		background: var(--color-border);
		padding: 0.25rem 0.75rem;
		border-radius: 4px;
		font-size: 0.85rem;
	}

	.actions {
		display: flex;
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.btn {
		padding: 0.75rem 1.5rem;
		border-radius: 6px;
		font-weight: 500;
		text-decoration: none;
	}

	.btn.primary {
		background: var(--color-accent);
		color: #000;
	}

	.btn.secondary {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		color: var(--color-text);
	}

	.btn:hover {
		opacity: 0.9;
		text-decoration: none;
	}

	.history h3 {
		margin-bottom: 1rem;
	}

	.history-list {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		overflow: hidden;
	}

	.history-item {
		display: flex;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--color-border);
	}

	.history-item:last-child {
		border-bottom: none;
	}

	.history-date {
		color: var(--color-text-muted);
	}

	.history-downloads {
		font-weight: 500;
	}
</style>
```

**Step 3: Test addon detail page**

Visit http://localhost:5173/addon/details (or any addon slug from homepage).

**Step 4: Commit**

```bash
git add web/src/routes/addon/
git commit -m "feat(web): add addon detail page"
```

---

## Task 7: Create Search Page

**Files:**
- Create: `web/src/routes/search/+page.server.ts`
- Create: `web/src/routes/search/+page.svelte`

**Step 1: Create server-side search**

Create `web/src/routes/search/+page.server.ts`:

```typescript
import { searchAddons } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const query = url.searchParams.get('q') || '';
	const page = parseInt(url.searchParams.get('page') || '1', 10);

	if (!query) {
		return { query, results: null };
	}

	const results = await searchAddons(query, page);

	return {
		query,
		results
	};
};
```

**Step 2: Create search page UI**

Create `web/src/routes/search/+page.svelte`:

```svelte
<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import type { PageData } from './$types';

	export let data: PageData;

	$: totalPages = data.results?.pagination.total_pages ?? 0;
	$: currentPage = data.results?.pagination.page ?? 1;
</script>

<svelte:head>
	<title>{data.query ? `Search: ${data.query}` : 'Search'} - Addon Radar</title>
	<meta name="description" content="Search for World of Warcraft addons" />
</svelte:head>

<h1>Search Addons</h1>

<form action="/search" method="get" class="search-box">
	<input
		type="search"
		name="q"
		value={data.query}
		placeholder="Search by addon name..."
		autofocus
	/>
	<button type="submit">Search</button>
</form>

{#if data.query && data.results}
	<p class="results-count">
		Found {data.results.pagination.total} addons for "{data.query}"
	</p>

	{#if data.results.data.length > 0}
		<div class="addon-list">
			{#each data.results.data as addon}
				<AddonCard {addon} />
			{/each}
		</div>

		{#if totalPages > 1}
			<nav class="pagination">
				{#if currentPage > 1}
					<a href="/search?q={encodeURIComponent(data.query)}&page={currentPage - 1}">
						Previous
					</a>
				{/if}
				<span class="page-info">Page {currentPage} of {totalPages}</span>
				{#if currentPage < totalPages}
					<a href="/search?q={encodeURIComponent(data.query)}&page={currentPage + 1}">
						Next
					</a>
				{/if}
			</nav>
		{/if}
	{:else}
		<p class="no-results">No addons found matching "{data.query}"</p>
	{/if}
{:else if data.query}
	<p class="no-results">Search failed. Please try again.</p>
{:else}
	<p class="hint">Enter a search term to find addons</p>
{/if}

<style>
	h1 {
		margin-bottom: 1.5rem;
	}

	.search-box {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 2rem;
		max-width: 600px;
	}

	.search-box input {
		flex: 1;
		padding: 0.75rem 1rem;
		border: 1px solid var(--color-border);
		border-radius: 6px;
		background: var(--color-surface);
		color: var(--color-text);
		font-size: 1rem;
	}

	.search-box button {
		padding: 0.75rem 1.5rem;
		background: var(--color-accent);
		color: #000;
		border: none;
		border-radius: 6px;
		font-weight: 500;
		cursor: pointer;
	}

	.search-box button:hover {
		opacity: 0.9;
	}

	.results-count {
		color: var(--color-text-muted);
		margin-bottom: 1rem;
	}

	.addon-list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.pagination {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1.5rem;
		margin-top: 2rem;
		padding: 1rem;
	}

	.pagination a {
		padding: 0.5rem 1rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 4px;
	}

	.page-info {
		color: var(--color-text-muted);
	}

	.no-results,
	.hint {
		color: var(--color-text-muted);
		padding: 2rem;
		text-align: center;
		background: var(--color-surface);
		border-radius: 8px;
	}
</style>
```

**Step 3: Test search functionality**

Visit http://localhost:5173/search and search for "details" or another addon name.

**Step 4: Commit**

```bash
git add web/src/routes/search/
git commit -m "feat(web): add search page with pagination"
```

---

## Task 8: Create Error Pages

**Files:**
- Create: `web/src/routes/+error.svelte`

**Step 1: Create error page**

Create `web/src/routes/+error.svelte`:

```svelte
<script lang="ts">
	import { page } from '$app/stores';
</script>

<svelte:head>
	<title>Error {$page.status} - Addon Radar</title>
</svelte:head>

<div class="error-page">
	<h1>{$page.status}</h1>
	<p class="message">{$page.error?.message || 'Something went wrong'}</p>

	{#if $page.status === 404}
		<p class="hint">The addon you're looking for doesn't exist or has been removed.</p>
	{/if}

	<div class="actions">
		<a href="/" class="btn">Back to Home</a>
		<a href="/search" class="btn secondary">Search Addons</a>
	</div>
</div>

<style>
	.error-page {
		text-align: center;
		padding: 4rem 1rem;
	}

	h1 {
		font-size: 6rem;
		color: var(--color-accent);
		margin-bottom: 0.5rem;
	}

	.message {
		font-size: 1.5rem;
		margin-bottom: 1rem;
	}

	.hint {
		color: var(--color-text-muted);
		margin-bottom: 2rem;
	}

	.actions {
		display: flex;
		gap: 1rem;
		justify-content: center;
	}

	.btn {
		padding: 0.75rem 1.5rem;
		background: var(--color-accent);
		color: #000;
		border-radius: 6px;
		font-weight: 500;
		text-decoration: none;
	}

	.btn.secondary {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		color: var(--color-text);
	}

	.btn:hover {
		opacity: 0.9;
		text-decoration: none;
	}
</style>
```

**Step 2: Test error page**

Visit http://localhost:5173/addon/nonexistent-addon-slug

**Step 3: Commit**

```bash
git add web/src/routes/+error.svelte
git commit -m "feat(web): add error page"
```

---

## Task 9: Add Railway Deployment Configuration

**Files:**
- Create: `Dockerfile.web`
- Modify: `railway.toml`

**Step 1: Create Dockerfile for web service**

Create `Dockerfile.web`:

```dockerfile
FROM oven/bun:1 AS builder
WORKDIR /app
COPY web/package.json web/bun.lockb* ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM oven/bun:1-alpine
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
RUN bun install --production --frozen-lockfile
ENV PORT=3000
EXPOSE 3000
CMD ["bun", "./build/index.js"]
```

**Step 2: Update railway.toml**

Add to existing `railway.toml`:

```toml
# Web frontend service
[services.addon-radar-web.build]
builder = "dockerfile"
dockerfilePath = "Dockerfile.web"

[services.addon-radar-web.deploy]
numReplicas = 1
```

**Step 3: Commit**

```bash
git add Dockerfile.web railway.toml
git commit -m "feat(web): add Railway deployment configuration"
```

---

## Task 10: Add Environment Configuration

**Files:**
- Modify: `web/.env.example` (create)
- Modify: `.gitignore`

**Step 1: Create environment example file**

Create `web/.env.example`:

```bash
# API URL for development
VITE_API_URL=http://localhost:8080

# Production (set in Railway):
# API_URL=http://addon-radar-api.railway.internal:8080
```

**Step 2: Update .gitignore**

Add to `.gitignore`:

```
# Web frontend
web/node_modules/
web/.svelte-kit/
web/build/
web/.env
```

**Step 3: Update API client for production**

Modify `web/src/lib/api.ts` to use server-side env:

```typescript
import { env } from '$env/dynamic/private';
import type {
	Addon,
	TrendingAddon,
	Snapshot,
	PaginatedResponse,
	DataResponse
} from './types';

// Server-side: use API_URL from Railway internal network
// Client-side fallback: use VITE_API_URL (only for dev)
const API_URL = env?.API_URL || import.meta.env.VITE_API_URL || 'http://localhost:8080';

// ... rest of the file unchanged
```

**Step 4: Commit**

```bash
git add web/.env.example .gitignore web/src/lib/api.ts
git commit -m "feat(web): add environment configuration"
```

---

## Task 11: Final Testing and Documentation

**Files:**
- Modify: `docs/PLAN.md`
- Modify: `CLAUDE.md`

**Step 1: Test full production build**

```bash
cd web
bun run build
bun run preview
```

Visit http://localhost:4173 and test all pages.

**Step 2: Update PLAN.md**

Add to Phase 4 section in `docs/PLAN.md`:

```markdown
### Phase 4: Frontend ✅
- [x] Choose framework (SvelteKit with Bun)
- [x] Homepage with trending lists
- [x] Addon detail pages
- [x] Search with pagination
- [x] Railway deployment config
- [ ] Deploy to Railway
```

**Step 3: Update CLAUDE.md**

Add Web Frontend section to tech stack and update project structure.

**Step 4: Commit**

```bash
git add docs/PLAN.md CLAUDE.md
git commit -m "docs: update project docs for frontend"
```

---

## Task 12: Deploy to Railway

**Step 1: Push feature branch**

```bash
git push -u origin feat/frontend-sveltekit
```

**Step 2: Create PR and merge**

```bash
gh pr create --title "feat: add SvelteKit frontend" --body "Adds SvelteKit frontend with:
- Homepage with trending hot/rising lists
- Addon detail pages with download history
- Search with pagination
- Error handling
- Railway deployment configuration

Uses Bun runtime for faster cold starts."
```

**Step 3: After merge, configure Railway**

In Railway dashboard:
1. New service will auto-detect from `railway.toml`
2. Add environment variable: `API_URL=http://addon-radar-api.railway.internal:8080`
3. Set up custom domain (optional)

**Step 4: Verify deployment**

Visit the deployed URL and test all pages.
