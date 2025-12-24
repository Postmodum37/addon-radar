<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Addon } from '$lib/types';

	let query = $state('');
	let results = $state<Addon[]>([]);
	let isOpen = $state(false);
	let selectedIndex = $state(-1);
	let debounceTimer: ReturnType<typeof setTimeout>;

	function formatDownloads(count: number): string {
		if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`;
		if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`;
		return String(count);
	}

	async function search(q: string) {
		if (q.length < 2) {
			results = [];
			isOpen = false;
			return;
		}

		try {
			const res = await fetch(`/api/search?q=${encodeURIComponent(q)}`);
			if (res.ok) {
				const data = await res.json();
				results = data.slice(0, 5);
				isOpen = results.length > 0;
				selectedIndex = -1;
			}
		} catch {
			results = [];
			isOpen = false;
		}
	}

	function handleInput() {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => search(query), 300);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!isOpen) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = Math.min(selectedIndex + 1, results.length - 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = Math.max(selectedIndex - 1, -1);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			if (selectedIndex >= 0 && results[selectedIndex]) {
				goToAddon(results[selectedIndex].slug);
			} else if (query.length >= 2) {
				goToSearch();
			}
		} else if (e.key === 'Escape') {
			isOpen = false;
		}
	}

	function goToAddon(slug: string) {
		isOpen = false;
		query = '';
		goto(`/addon/${slug}`);
	}

	function goToSearch() {
		isOpen = false;
		goto(`/search?q=${encodeURIComponent(query)}`);
	}

	function handleBlur() {
		// Delay to allow click on results
		setTimeout(() => {
			isOpen = false;
		}, 200);
	}
</script>

<div class="search-wrapper">
	<div class="search-form">
		<svg class="search-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			<circle cx="11" cy="11" r="8"></circle>
			<path d="m21 21-4.3-4.3"></path>
		</svg>
		<input
			type="search"
			bind:value={query}
			oninput={handleInput}
			onkeydown={handleKeydown}
			onblur={handleBlur}
			onfocus={() => query.length >= 2 && results.length > 0 && (isOpen = true)}
			placeholder="Search addons..."
			aria-label="Search addons"
			autocomplete="off"
		/>
	</div>

	{#if isOpen}
		<div class="dropdown">
			{#each results as addon, i}
				<button
					type="button"
					class="result"
					class:selected={i === selectedIndex}
					onmousedown={() => goToAddon(addon.slug)}
				>
					{#if addon.logo_url}
						<img src={addon.logo_url} alt="" class="result-logo" />
					{:else}
						<div class="result-logo placeholder">?</div>
					{/if}
					<div class="result-info">
						<div class="result-name">{addon.name}</div>
						<div class="result-meta">{formatDownloads(addon.download_count)} downloads</div>
					</div>
				</button>
			{/each}
			<button type="button" class="view-all" onmousedown={goToSearch}>
				View all results for "{query}" →
			</button>
		</div>
	{/if}
</div>

<style>
	.search-wrapper {
		position: relative;
		flex: 1;
		max-width: 400px;
	}

	.search-form {
		position: relative;
		display: flex;
		align-items: center;
	}

	.search-icon {
		position: absolute;
		left: 12px;
		color: var(--color-text-muted);
		pointer-events: none;
	}

	input {
		width: 100%;
		padding: 0.625rem 1rem 0.625rem 2.5rem;
		border: 1px solid var(--color-border);
		border-radius: 8px;
		background: var(--color-surface);
		color: var(--color-text);
		font-size: 0.9rem;
	}

	input::placeholder {
		color: var(--color-text-muted);
	}

	input:focus {
		outline: none;
		border-color: var(--color-accent);
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.dropdown {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		box-shadow: var(--shadow-md);
		z-index: 100;
		overflow: hidden;
	}

	.result {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		width: 100%;
		padding: 0.75rem;
		border: none;
		background: none;
		text-align: left;
		cursor: pointer;
		transition: background 0.1s;
	}

	.result:hover,
	.result.selected {
		background: var(--color-bg);
	}

	.result-logo {
		width: 32px;
		height: 32px;
		border-radius: 4px;
		object-fit: cover;
	}

	.result-logo.placeholder {
		background: var(--color-border);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.875rem;
		color: var(--color-text-muted);
	}

	.result-info {
		flex: 1;
		min-width: 0;
	}

	.result-name {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.result-meta {
		font-size: 0.75rem;
		color: var(--color-text-muted);
	}

	.view-all {
		display: block;
		width: 100%;
		padding: 0.75rem;
		border: none;
		border-top: 1px solid var(--color-border);
		background: none;
		text-align: center;
		font-size: 0.8125rem;
		color: var(--color-accent);
		cursor: pointer;
	}

	.view-all:hover {
		background: var(--color-bg);
	}
</style>
