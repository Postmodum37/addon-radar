<script lang="ts">
	import type { TrendingAddon, Addon } from '$lib/types';
	import { formatCount, formatVelocity, formatTimeAgo, truncateText } from '$lib/utils/format';
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
			<p class="summary">{truncateText(addon.summary, 60)}</p>
		{/if}
		<p class="stats">
			<span>{formatCount(addon.download_count)}</span>
			<span class="separator">·</span>
			<span>{formatCount(addon.thumbs_up_count)} likes</span>
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
