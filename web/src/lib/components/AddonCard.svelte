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
