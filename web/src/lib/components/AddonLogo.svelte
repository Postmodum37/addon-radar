<script lang="ts">
	let {
		url,
		name,
		size = 'md'
	}: {
		url: string | null | undefined;
		name: string;
		size?: 'sm' | 'md' | 'lg';
	} = $props();

	const sizeClass = $derived(`logo-${size}`);
</script>

<div class="logo {sizeClass}">
	{#if url}
		<img src={url} alt="{name} logo" loading="lazy" />
	{:else}
		<div class="placeholder">?</div>
	{/if}
</div>

<style>
	.logo {
		flex-shrink: 0;
	}

	.logo-sm {
		width: 48px;
		height: 48px;
	}

	.logo-md {
		width: 64px;
		height: 64px;
	}

	.logo-lg {
		width: 80px;
		height: 80px;
	}

	.logo img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: 8px;
	}

	.logo-lg img {
		border-radius: 12px;
	}

	.placeholder {
		width: 100%;
		height: 100%;
		background: var(--color-border);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.25rem;
		color: var(--color-text-muted);
	}

	.logo-md .placeholder {
		font-size: 1.5rem;
	}

	.logo-lg .placeholder {
		border-radius: 12px;
	}
</style>
