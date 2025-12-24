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

<h1>Trending WoW Addons</h1>
<p class="subtitle">Discover what's hot and rising in the World of Warcraft addon community</p>

<section class="trending-section">
	<h2 class="section-title hot">Hot Right Now</h2>
	<p class="section-desc">Established addons with high download velocity</p>

	{#if data.hot.length > 0}
		<div class="addon-grid">
			{#each data.hot as addon}
				<AddonCard {addon} showVelocity={true} velocityLabel="day" />
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
				<AddonCard {addon} showVelocity={true} velocityLabel="day" />
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
