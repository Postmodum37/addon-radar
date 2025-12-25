<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import SearchAutocomplete from '$lib/components/SearchAutocomplete.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet" />
</svelte:head>

<header>
	<nav>
		<a href="/" class="logo">Addon Radar</a>
		<div class="header-actions">
			<SearchAutocomplete />
			<ThemeToggle />
		</div>
	</nav>
</header>

<main>
	{@render children()}
</main>

<footer>
	<p>Data from <a href="https://www.curseforge.com/wow" target="_blank" rel="noopener noreferrer">CurseForge</a>. Updated hourly.</p>
</footer>

<style>
	header {
		background: var(--color-header);
		padding: 1rem;
	}

	nav {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		align-items: center;
		gap: 2rem;
	}

	.logo {
		font-size: 1.5rem;
		font-weight: 700;
		color: #FFFFFF;
		letter-spacing: -0.025em;
	}

	.logo:hover {
		text-decoration: none;
		opacity: 0.9;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex: 1;
	}

	main {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem 1rem;
		min-height: calc(100vh - 180px);
	}

	footer {
		background: var(--color-surface);
		border-top: 1px solid var(--color-border);
		padding: 2rem 1rem;
		text-align: center;
		color: var(--color-text-muted);
		font-size: 0.875rem;
	}
</style>
