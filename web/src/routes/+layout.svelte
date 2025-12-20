<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<header>
	<nav>
		<a href="/" class="logo">Addon Radar</a>
		<form action="/search" method="get" class="search-form">
			<input type="search" name="q" placeholder="Search addons..." aria-label="Search addons" />
		</form>
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
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
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
		font-weight: bold;
		color: var(--color-accent);
	}

	.logo:hover {
		text-decoration: none;
	}

	.search-form {
		flex: 1;
		max-width: 400px;
	}

	.search-form input {
		width: 100%;
		padding: 0.5rem 1rem;
		border: 1px solid var(--color-border);
		border-radius: 4px;
		background: var(--color-bg);
		color: var(--color-text);
	}

	main {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem 1rem;
		min-height: calc(100vh - 200px);
	}

	footer {
		background: var(--color-surface);
		border-top: 1px solid var(--color-border);
		padding: 2rem 1rem;
		text-align: center;
		color: var(--color-text-muted);
	}
</style>
