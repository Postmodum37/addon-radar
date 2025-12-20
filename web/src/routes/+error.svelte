<script lang="ts">
	import { page } from '$app/stores';
</script>

<svelte:head>
	<title>Error {$page.status} - Addon Radar</title>
</svelte:head>

<div class="error-page">
	<h1>{$page.status}</h1>
	<p class="message">{$page.error?.message || 'Something went wrong'}</p>

	{#if $page.status === 404}
		<p class="hint">The addon you're looking for doesn't exist or has been removed.</p>
	{/if}

	<div class="actions">
		<a href="/" class="btn">Back to Home</a>
		<a href="/search" class="btn secondary">Search Addons</a>
	</div>
</div>

<style>
	.error-page {
		text-align: center;
		padding: 4rem 1rem;
	}

	h1 {
		font-size: 6rem;
		color: var(--color-accent);
		margin-bottom: 0.5rem;
	}

	.message {
		font-size: 1.5rem;
		margin-bottom: 1rem;
	}

	.hint {
		color: var(--color-text-muted);
		margin-bottom: 2rem;
	}

	.actions {
		display: flex;
		gap: 1rem;
		justify-content: center;
		flex-wrap: wrap;
	}

	.btn {
		padding: 0.75rem 1.5rem;
		border-radius: 6px;
		font-weight: 600;
		transition: all 0.2s;
		border: 1px solid transparent;
		cursor: pointer;
		text-decoration: none;
	}

	.btn {
		background: var(--color-accent);
		color: var(--color-bg);
		border-color: var(--color-accent);
	}

	.btn:hover {
		background: #ff9d1a;
		border-color: #ff9d1a;
		text-decoration: none;
	}

	.btn.secondary {
		background: transparent;
		color: var(--color-text);
		border-color: var(--color-border);
	}

	.btn.secondary:hover {
		background: var(--color-bg);
		border-color: var(--color-accent);
		text-decoration: none;
	}
</style>
