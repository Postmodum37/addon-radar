<script lang="ts">
	import AddonCard from '$lib/components/AddonCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const hasPrevPage = $derived(data.results ? data.results.meta.page > 1 : false);
	const hasNextPage = $derived(
		data.results ? data.results.meta.page < data.results.meta.total_pages : false
	);

	function buildPageUrl(page: number): string {
		return `/search?q=${encodeURIComponent(data.query)}&page=${page}`;
	}
</script>

<svelte:head>
	<title>Search Addons - Addon Radar</title>
	<meta name="description" content="Search for World of Warcraft addons" />
	<meta property="og:title" content="Search Addons - Addon Radar" />
	<meta property="og:description" content="Search for World of Warcraft addons" />
	<meta property="og:type" content="website" />
</svelte:head>

<h1>Search Addons</h1>

<form method="get" action="/search" class="search-form">
	<input
		type="search"
		name="q"
		placeholder="Search for addons..."
		value={data.query}
		class="search-input"
		aria-label="Search addons"
		autofocus
	/>
	<button type="submit" class="search-button">Search</button>
</form>

{#if !data.query}
	<p class="empty">Enter a search query to find addons</p>
{:else if data.results === null}
	<p class="empty">Failed to load search results</p>
{:else if data.results.data.length === 0}
	<p class="empty">No addons found for "{data.query}"</p>
{:else}
	<div class="results-header">
		<p class="results-count">
			Found {data.results.meta.total} {data.results.meta.total === 1 ? 'addon' : 'addons'} for "{data.query}"
		</p>
	</div>

	<div class="addon-grid">
		{#each data.results.data as addon}
			<AddonCard {addon} />
		{/each}
	</div>

	{#if data.results.meta.total_pages > 1}
		<div class="pagination">
			{#if hasPrevPage}
				<a href={buildPageUrl(data.results.meta.page - 1)} class="page-link">Previous</a>
			{:else}
				<span class="page-link disabled">Previous</span>
			{/if}

			<span class="page-info">
				Page {data.results.meta.page} of {data.results.meta.total_pages}
			</span>

			{#if hasNextPage}
				<a href={buildPageUrl(data.results.meta.page + 1)} class="page-link">Next</a>
			{:else}
				<span class="page-link disabled">Next</span>
			{/if}
		</div>
	{/if}
{/if}

<style>
	h1 {
		font-size: 2rem;
		margin-bottom: 1.5rem;
	}

	.search-form {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 2rem;
		max-width: 600px;
	}

	.search-input {
		flex: 1;
		padding: 0.75rem 1rem;
		font-size: 1rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		color: var(--color-text);
	}

	.search-input:focus {
		outline: none;
		border-color: var(--color-accent);
	}

	.search-button {
		padding: 0.75rem 1.5rem;
		font-size: 1rem;
		background: var(--color-accent);
		color: white;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: opacity 0.2s;
	}

	.search-button:hover {
		opacity: 0.9;
	}

	.results-header {
		margin-bottom: 1rem;
	}

	.results-count {
		color: var(--color-text-muted);
		font-size: 0.9rem;
	}

	.addon-grid {
		display: grid;
		gap: 1rem;
		grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
		margin-bottom: 2rem;
	}

	.empty {
		color: var(--color-text-muted);
		padding: 2rem;
		text-align: center;
		background: var(--color-surface);
		border-radius: 8px;
	}

	.pagination {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: 1rem;
		margin-top: 2rem;
	}

	.page-link {
		padding: 0.5rem 1rem;
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
		font-size: 0.9rem;
	}
</style>
