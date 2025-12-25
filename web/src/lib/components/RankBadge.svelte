<script lang="ts">
	let { rankChange, isNew = false }: { rankChange: number | null; isNew?: boolean } = $props();

	const state = $derived(() => {
		if (isNew) return 'new';
		if (rankChange === null) return 'new';
		if (rankChange > 0) return 'rising';
		if (rankChange < 0) return 'falling';
		return 'unchanged';
	});

	const badgeText = $derived(() => {
		const s = state();
		if (s === 'new') return 'NEW';
		if (s === 'rising') return `+${rankChange}`;
		if (s === 'falling') return `${rankChange}`;
		return '=';
	});

	const showBadge = $derived(state() !== 'unchanged');
</script>

{#if showBadge}
	<span class="badge {state()}">
		{badgeText()}
	</span>
{/if}

<style>
	.badge {
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		white-space: nowrap;
	}

	.rising {
		background: var(--color-rising-bg);
		color: var(--color-rising);
	}

	.falling {
		background: var(--color-falling-bg);
		color: var(--color-falling);
	}

	.new {
		background: var(--color-new-bg);
		color: var(--color-new);
	}

	.unchanged {
		background: var(--color-unchanged-bg);
		color: var(--color-unchanged);
	}
</style>
