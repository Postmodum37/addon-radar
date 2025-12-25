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
			<h2>Trending</h2>
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
			<h2>Rising</h2>
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
