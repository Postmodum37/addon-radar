<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
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
		<div class="search-wrapper">
			<form action="/search" method="get" class="search-form">
				<svg class="search-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<circle cx="11" cy="11" r="8"></circle>
					<path d="m21 21-4.3-4.3"></path>
				</svg>
				<input type="search" name="q" placeholder="Search addons..." aria-label="Search addons" />
			</form>
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

	.search-wrapper {
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

	.search-form input {
		width: 100%;
		padding: 0.625rem 1rem 0.625rem 2.5rem;
		border: 1px solid var(--color-border);
		border-radius: 8px;
		background: var(--color-surface);
		color: var(--color-text);
		font-size: 0.9rem;
	}

	.search-form input::placeholder {
		color: var(--color-text-muted);
	}

	.search-form input:focus {
		outline: none;
		border-color: var(--color-accent);
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
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
