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
