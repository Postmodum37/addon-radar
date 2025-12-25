<script lang="ts">
	import type { TrendingAddon } from '$lib/types';
	import { formatCount, formatVelocity, formatUpdatedAgo, truncateText } from '$lib/utils/format';
	import RankBadge from './RankBadge.svelte';
	import AddonLogo from './AddonLogo.svelte';

	let {
		addon,
		velocityLabel = 'day'
	}: {
		addon: TrendingAddon;
		velocityLabel?: 'day' | 'week';
	} = $props();

	const isNew = $derived(addon.rank_change_24h === null);
</script>

<a href="/addon/{addon.slug}" class="card">
	<div class="rank">#{addon.rank}</div>
	<div class="badge-container">
		<RankBadge rankChange={addon.rank_change_24h} {isNew} />
	</div>

	<div class="header">
		<AddonLogo url={addon.logo_url} name={addon.name} size="md" />
		<div class="title">
			<h3>{addon.name}</h3>
			{#if addon.author_name}
				<p class="author">by {addon.author_name}</p>
			{/if}
		</div>
	</div>

	{#if addon.summary}
		<p class="summary">{truncateText(addon.summary, 100)}</p>
	{/if}

	<div class="stats">
		<span>{formatCount(addon.download_count)} downloads</span>
		{#if addon.download_velocity > 0}
			<span class="separator">·</span>
			<span class="velocity">{formatVelocity(addon.download_velocity)}/{velocityLabel}</span>
		{/if}
	</div>

	<div class="meta">
		{#if addon.last_updated_at}
			<span>{formatUpdatedAgo(addon.last_updated_at)}</span>
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
