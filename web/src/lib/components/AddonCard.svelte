<script lang="ts">
	import type { TrendingAddon, Addon } from '$lib/types';

	let { addon, showScore = false, scoreType = 'hot' }: {
		addon: TrendingAddon | Addon;
		showScore?: boolean;
		scoreType?: 'hot' | 'rising';
	} = $props();

	function formatDownloads(count: number): string {
		if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`;
		if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`;
		return String(count);
	}

	const score = $derived('score' in addon ? addon.score : 0);
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
