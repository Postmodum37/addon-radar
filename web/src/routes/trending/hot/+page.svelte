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
