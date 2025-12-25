<script lang="ts">
	let { rankChange, isNew = false }: { rankChange: number | null; isNew?: boolean } = $props();

	function getState(isNew: boolean, rankChange: number | null): 'new' | 'rising' | 'falling' | 'unchanged' {
		if (isNew || rankChange === null) return 'new';
		if (rankChange > 0) return 'rising';
		if (rankChange < 0) return 'falling';
		return 'unchanged';
	}

	function getBadgeText(state: string, rankChange: number | null): string {
		if (state === 'new') return 'NEW';
		if (state === 'rising') return `+${rankChange}`;
		if (state === 'falling') return `${rankChange}`;
		return '=';
	}

	const state = $derived(getState(isNew, rankChange));
	const badgeText = $derived(getBadgeText(state, rankChange));
	const showBadge = $derived(state !== 'unchanged');
</script>

{#if showBadge}
	<span class="badge {state}">
		{badgeText}
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
</style>
