<script lang="ts">
	import type { PageData } from './$types';

	let { data } = $props<{ data: PageData }>();

	// Format download count with commas
	function formatDownloads(count: number): string {
		return count.toLocaleString('en-US');
	}

	// Format date to human-readable format
	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}
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

<div class="container">
	<!-- Header -->
	<div class="header-card">
		<div class="header-content">
			<img
				src={data.addon.logo_url}
				alt={data.addon.name}
				class="logo"
			/>
			<div class="header-info">
				<h1>{data.addon.name}</h1>
				{#if data.addon.author_name}
					<p class="author">by {data.addon.author_name}</p>
				{/if}
				{#if data.addon.summary}
					<p class="summary">{data.addon.summary}</p>
				{/if}
			</div>
		</div>

		<!-- Stats Grid -->
		<div class="stats-grid">
			<div class="stat-card stat-downloads">
				<div class="stat-label">Total Downloads</div>
				<div class="stat-value">
					{formatDownloads(data.addon.download_count)}
				</div>
			</div>

			<div class="stat-card stat-thumbs">
				<div class="stat-label">Thumbs Up</div>
				<div class="stat-value">
					{formatDownloads(data.addon.thumbs_up_count)}
				</div>
			</div>

			{#if data.addon.popularity_rank}
				<div class="stat-card stat-rank">
					<div class="stat-label">Popularity Rank</div>
					<div class="stat-value">
						#{formatDownloads(data.addon.popularity_rank)}
					</div>
				</div>
			{/if}

			{#if data.addon.last_updated_at}
				<div class="stat-card stat-updated">
					<div class="stat-label">Last Updated</div>
					<div class="stat-value stat-value-small">
						{formatDate(data.addon.last_updated_at)}
					</div>
				</div>
			{/if}
		</div>

		<!-- Game Versions -->
		{#if data.addon.game_versions && data.addon.game_versions.length > 0}
			<div class="versions-section">
				<h2>Supported Versions</h2>
				<div class="versions">
					{#each data.addon.game_versions as version}
						<span class="version-tag">
							{version}
						</span>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Actions -->
		<div class="actions">
			<a
				href="https://www.curseforge.com/wow/addons/{data.addon.slug}"
				target="_blank"
				rel="noopener noreferrer"
				class="btn btn-primary"
			>
				View on CurseForge
			</a>
			<a
				href="/"
				class="btn btn-secondary"
			>
				Back to Trending
			</a>
		</div>
	</div>

	<!-- Download History -->
	<div class="history-card">
		<h2>Download History</h2>

		{#if data.history && data.history.length > 0}
			<div class="history-list">
				{#each data.history.slice(0, 24) as entry}
					<div class="history-entry">
						<div class="history-date">
							{formatDate(entry.recorded_at)}
						</div>
						<div class="history-downloads">
							{formatDownloads(entry.download_count)} downloads
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<p class="empty">No download history available yet.</p>
		{/if}
	</div>
</div>

<style>
	.container {
		max-width: 960px;
		margin: 0 auto;
	}

	.header-card {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		padding: 2rem;
		margin-bottom: 1.5rem;
	}

	.header-content {
		display: flex;
		align-items: flex-start;
		gap: 1.5rem;
		margin-bottom: 1.5rem;
	}

	.logo {
		width: 128px;
		height: 128px;
		border-radius: 8px;
		flex-shrink: 0;
		border: 1px solid var(--color-border);
	}

	.header-info {
		flex: 1;
		min-width: 0;
	}

	h1 {
		font-size: 2rem;
		margin-bottom: 0.5rem;
		color: var(--color-text);
	}

	.author {
		font-size: 1.1rem;
		color: var(--color-text-muted);
		margin-bottom: 0.75rem;
	}

	.summary {
		color: var(--color-text);
		line-height: 1.6;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.stat-card {
		padding: 1rem;
		border-radius: 6px;
		border: 1px solid var(--color-border);
	}

	.stat-downloads {
		background: rgba(59, 130, 246, 0.1);
	}

	.stat-thumbs {
		background: rgba(34, 197, 94, 0.1);
	}

	.stat-rank {
		background: rgba(168, 85, 247, 0.1);
	}

	.stat-updated {
		background: rgba(251, 146, 60, 0.1);
	}

	.stat-label {
		font-size: 0.85rem;
		color: var(--color-text-muted);
		margin-bottom: 0.25rem;
	}

	.stat-value {
		font-size: 1.5rem;
		font-weight: bold;
		color: var(--color-text);
	}

	.stat-value-small {
		font-size: 0.9rem;
		margin-top: 0.5rem;
	}

	.versions-section {
		margin-bottom: 1.5rem;
	}

	.versions-section h2 {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--color-text-muted);
		margin-bottom: 0.5rem;
	}

	.versions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.version-tag {
		background: var(--color-bg);
		color: var(--color-text);
		padding: 0.25rem 0.75rem;
		border-radius: 16px;
		font-size: 0.85rem;
		font-weight: 500;
		border: 1px solid var(--color-border);
	}

	.actions {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.btn {
		padding: 0.75rem 1.5rem;
		border-radius: 6px;
		font-weight: 600;
		transition: all 0.2s;
		border: 1px solid transparent;
		cursor: pointer;
	}

	.btn-primary {
		background: var(--color-accent);
		color: var(--color-bg);
		border-color: var(--color-accent);
	}

	.btn-primary:hover {
		background: #ff9d1a;
		border-color: #ff9d1a;
		text-decoration: none;
	}

	.btn-secondary {
		background: transparent;
		color: var(--color-text);
		border-color: var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-bg);
		border-color: var(--color-accent);
		text-decoration: none;
	}

	.history-card {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		padding: 2rem;
	}

	.history-card h2 {
		font-size: 1.5rem;
		margin-bottom: 1.5rem;
		color: var(--color-text);
	}

	.history-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.history-entry {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem;
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		transition: border-color 0.2s;
	}

	.history-entry:hover {
		border-color: var(--color-accent);
	}

	.history-date {
		color: var(--color-text);
		font-weight: 500;
	}

	.history-downloads {
		color: var(--color-text);
		font-weight: 600;
	}

	.empty {
		color: var(--color-text-muted);
		padding: 2rem;
		text-align: center;
		background: var(--color-bg);
		border-radius: 6px;
	}
</style>
