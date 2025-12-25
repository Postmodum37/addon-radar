<script lang="ts">
	import type { PageData } from './$types';
	import { formatDelta, formatAxisNumber, formatAxisDate, niceRound } from '$lib/utils/format';

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

	// Chart data
	const chartData = $derived(data.dailyHistory.map((d: { downloads_delta: number }) => d.downloads_delta));
	const maxDelta = $derived(Math.max(...chartData, 1));
	const maxDeltaNice = $derived(niceRound(maxDelta));
	const midDeltaNice = $derived(niceRound(maxDelta / 2));

	// Hover state
	let hoveredIndex = $state<number | null>(null);
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
		<h2>Download Trend</h2>
		<div class="chart-container">
			<!-- Y-axis scale -->
			<div class="y-axis">
				<span class="y-tick">{formatAxisNumber(maxDeltaNice)}</span>
				<span class="y-tick">{formatAxisNumber(midDeltaNice)}</span>
				<span class="y-tick">0</span>
			</div>

			<!-- Chart area -->
			<div class="chart-wrapper">
				<!-- Delta label on hover -->
				{#if hoveredIndex !== null}
					<div
						class="delta-label"
						style="left: calc({(hoveredIndex + 0.5) / chartData.length * 100}%)"
					>
						{formatDelta(chartData[hoveredIndex])}
					</div>
				{/if}

				<!-- Bar chart -->
				<div class="chart" role="group" aria-label="Download trend bars" onmouseleave={() => (hoveredIndex = null)}>
					{#each chartData as delta, i}
						<button
							type="button"
							class="bar"
							class:hovered={hoveredIndex === i}
							style="height: {Math.max((delta / maxDelta) * 100, 2)}%"
							onmouseenter={() => (hoveredIndex = i)}
							onmouseleave={() => (hoveredIndex = null)}
							onfocus={() => (hoveredIndex = i)}
							onblur={() => (hoveredIndex = null)}
							title="{formatDate(data.dailyHistory[i].date)}: {formatDelta(delta)}"
						></button>
					{/each}
				</div>

				<!-- X-axis dates -->
				<div class="x-axis">
					{#each data.dailyHistory as item, i}
						{#if i % 7 === 0 || i === data.dailyHistory.length - 1}
							<span class="x-tick" style="left: calc({(i + 0.5) / data.dailyHistory.length * 100}%)">
								{formatAxisDate(item.date)}
							</span>
						{/if}
					{/each}
				</div>
			</div>
		</div>
		<p class="chart-footer">Last {data.dailyHistory.length} days · Daily downloads</p>
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

	.chart-container {
		display: flex;
		gap: 0.75rem;
	}

	.y-axis {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		padding-top: 1.5rem;
		padding-bottom: 1.5rem;
		min-width: 45px;
	}

	.y-tick {
		font-size: 0.6875rem;
		color: var(--color-text-muted);
		text-align: right;
	}

	.chart-wrapper {
		flex: 1;
		position: relative;
		min-width: 0;
	}

	.delta-label {
		position: absolute;
		top: 0;
		transform: translateX(-50%);
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--color-accent);
		white-space: nowrap;
		pointer-events: none;
		z-index: 10;
	}

	.chart {
		display: flex;
		align-items: flex-end;
		gap: 2px;
		height: 120px;
		margin-top: 1.5rem;
	}

	.bar {
		flex: 1;
		background: var(--color-accent);
		border: none;
		border-radius: 2px 2px 0 0;
		min-height: 2px;
		padding: 0;
		cursor: pointer;
		transition: opacity 0.15s ease;
	}

	.bar:hover,
	.bar.hovered {
		opacity: 0.7;
	}

	.bar:focus {
		outline: none;
	}

	.x-axis {
		position: relative;
		height: 1.5rem;
		margin-top: 0.5rem;
	}

	.x-tick {
		position: absolute;
		transform: translateX(-50%);
		font-size: 0.6875rem;
		color: var(--color-text-muted);
	}

	.chart-footer {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		text-align: center;
		margin-top: 0.75rem;
	}

	@media (max-width: 640px) {
		.y-axis {
			min-width: 35px;
		}

		.y-tick {
			font-size: 0.625rem;
		}

		.chart {
			gap: 1px;
		}

		.x-tick {
			font-size: 0.5625rem;
		}
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
