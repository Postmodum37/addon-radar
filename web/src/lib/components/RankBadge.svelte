<script lang="ts">
	let { rankChange, isNew = false }: { rankChange: number; isNew?: boolean } = $props();

	const showBadge = $derived(isNew || rankChange > 0);
	const badgeText = $derived(isNew ? 'New' : `↑${rankChange}`);
	const badgeClass = $derived(isNew ? 'new' : 'rising');
</script>

{#if showBadge}
	<span class="badge {badgeClass}">
		{badgeText}
	</span>
{/if}

<style>
	.badge {
		position: absolute;
		top: 8px;
		left: 8px;
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		z-index: 1;
	}

	.rising {
		background: var(--color-rising-bg);
		color: var(--color-rising);
	}

	.new {
		background: var(--color-new-bg);
		color: var(--color-new);
	}
</style>
