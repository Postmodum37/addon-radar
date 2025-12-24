<script lang="ts">
	import type { PageData } from './$types';

	let { data } = $props<{ data: PageData }>();

	function formatDownloads(count: number): string {
		return count.toLocaleString('en-US');
	}

	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	function simplifyVersion(versions: string[]): string {
		if (!versions || versions.length === 0) return 'Unknown';

		// Find the highest version number (likely retail)
		const sorted = [...versions].sort((a, b) => {
			const aNum = parseFloat(a.split('.')[0]) || 0;
			const bNum = parseFloat(b.split('.')[0]) || 0;
			return bNum - aNum;
		});

		const highest = sorted[0];
		const major = highest.split('.')[0];
		return `${major}.0+`;
	}

	// Simple sparkline data
	const chartData = $derived(data.dailyHistory.map((d: { downloads_delta: number }) => d.downloads_delta));
	const maxDelta = $derived(Math.max(...chartData, 1));
</script>

<svelte:head>
	<title>{data.addon.name} - Addon Radar</title>
	<meta name="description" content={data.addon.summary} />
	<meta property="og:title" content="{data.addon.name} - Addon Radar" />
	<meta property="og:description" content={data.addon.summary} />
	<meta property="og:type" content="website" />
	{#if data.addon.logo_url}
		<meta property="og:image" content={data.addon.logo_url} />
	{/if}
</svelte:head>

<div class="page-header">
	<a href="/" class="back-link">← Back to Trending</a>
</div>

<div class="addon-header">
	{#if data.addon.logo_url}
		<img src={data.addon.logo_url} alt={data.addon.name} class="logo" />
	{/if}
	<div class="addon-info">
		<h1>{data.addon.name}</h1>
		{#if data.addon.author_name}
			<p class="author">by {data.addon.author_name}</p>
		{/if}
		{#if data.addon.summary}
			<p class="summary">{data.addon.summary}</p>
		{/if}
	</div>
</div>

<div class="stats-grid">
	<div class="stat-card">
		<div class="stat-label">Total Downloads</div>
		<div class="stat-value">{formatDownloads(data.addon.download_count)}</div>
	</div>

	{#if data.addon.last_updated_at}
		<div class="stat-card">
			<div class="stat-label">Last Updated</div>
			<div class="stat-value">{formatDate(data.addon.last_updated_at)}</div>
		</div>
	{/if}

	<div class="stat-card">
		<div class="stat-label">Game Version</div>
		<div class="stat-value">{simplifyVersion(data.addon.game_versions)}</div>
	</div>
</div>

{#if data.dailyHistory.length > 0}
	<div class="chart-section">
		<h2>Weekly Download Trend</h2>
		<div class="chart">
			{#each chartData as delta, i}
				<div
					class="bar"
					style="height: {Math.max((delta / maxDelta) * 100, 2)}%"
					title="{formatDate(data.dailyHistory[i].date)}: +{formatDownloads(delta)}"
				></div>
			{/each}
		</div>
		<p class="chart-label">Last {data.dailyHistory.length} days</p>
	</div>
{/if}

<div class="actions">
	<a
		href="https://www.curseforge.com/wow/addons/{data.addon.slug}"
		target="_blank"
		rel="noopener noreferrer"
		class="btn btn-primary"
	>
		View on CurseForge →
	</a>
</div>

<style>
	.page-header {
		margin-bottom: 1.5rem;
	}

	.back-link {
		font-size: 0.875rem;
		color: var(--color-text-muted);
	}

	.back-link:hover {
		color: var(--color-accent);
	}

	.addon-header {
		display: flex;
		gap: 1.5rem;
		align-items: flex-start;
		margin-bottom: 2rem;
	}

	.logo {
		width: 80px;
		height: 80px;
		border-radius: 12px;
		flex-shrink: 0;
	}

	.addon-info {
		flex: 1;
		min-width: 0;
	}

	h1 {
		font-size: 1.75rem;
		font-weight: 700;
		letter-spacing: -0.025em;
		margin-bottom: 0.25rem;
	}

	.author {
		color: var(--color-text-muted);
		margin-bottom: 0.5rem;
	}

	.summary {
		color: var(--color-text);
		line-height: 1.6;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.stat-card {
		background: var(--color-surface);
		padding: 1.25rem;
		border-radius: 12px;
		box-shadow: var(--shadow-sm);
	}

	.stat-label {
		font-size: 0.8125rem;
		color: var(--color-text-muted);
		margin-bottom: 0.25rem;
	}

	.stat-value {
		font-size: 1.25rem;
		font-weight: 600;
	}

	.chart-section {
		background: var(--color-surface);
		padding: 1.5rem;
		border-radius: 12px;
		box-shadow: var(--shadow-sm);
		margin-bottom: 2rem;
	}

	.chart-section h2 {
		font-size: 1rem;
		font-weight: 600;
		margin-bottom: 1rem;
	}

	.chart {
		display: flex;
		align-items: flex-end;
		gap: 2px;
		height: 100px;
	}

	.bar {
		flex: 1;
		background: var(--color-accent);
		border-radius: 2px 2px 0 0;
		min-height: 2px;
		transition: opacity 0.2s;
	}

	.bar:hover {
		opacity: 0.8;
	}

	.chart-label {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		text-align: center;
		margin-top: 0.5rem;
	}

	.actions {
		display: flex;
		gap: 1rem;
	}

	.btn {
		display: inline-block;
		padding: 0.75rem 1.5rem;
		border-radius: 8px;
		font-weight: 600;
		text-align: center;
		transition: all 0.2s;
	}

	.btn-primary {
		background: var(--color-accent);
		color: white;
	}

	.btn-primary:hover {
		background: var(--color-accent-hover);
		text-decoration: none;
	}
</style>
