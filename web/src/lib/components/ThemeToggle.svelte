<script lang="ts">
	import { browser } from '$app/environment';

	function getInitialTheme(): 'light' | 'dark' {
		if (!browser) return 'light';
		// First check what the FOUC prevention script already set on the document
		const docTheme = document.documentElement.dataset.theme;
		if (docTheme === 'light' || docTheme === 'dark') return docTheme;
		// Fallback to localStorage or system preference
		const saved = localStorage.getItem('theme');
		if (saved === 'light' || saved === 'dark') return saved;
		return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
	}

	let theme = $state<'light' | 'dark'>(getInitialTheme());

	// Sync theme to DOM and localStorage only when it changes
	$effect(() => {
		if (browser && document.documentElement.dataset.theme !== theme) {
			document.documentElement.dataset.theme = theme;
			localStorage.setItem('theme', theme);
		}
	});

	// Listen for system theme changes when user has no saved preference
	$effect(() => {
		if (!browser) return;
		if (localStorage.getItem('theme')) return; // User has explicit preference

		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		const handleChange = (e: MediaQueryListEvent) => {
			theme = e.matches ? 'dark' : 'light';
		};
		mediaQuery.addEventListener('change', handleChange);
		return () => mediaQuery.removeEventListener('change', handleChange);
	});

	function toggle() {
		theme = theme === 'light' ? 'dark' : 'light';
	}
</script>

<button
	onclick={toggle}
	class="theme-toggle"
	data-testid="theme-toggle"
	data-theme={theme}
	aria-label={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
>
	{#if theme === 'light'}
		<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>
		</svg>
	{:else}
		<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			<circle cx="12" cy="12" r="4"/>
			<path d="M12 2v2"/>
			<path d="M12 20v2"/>
			<path d="m4.93 4.93 1.41 1.41"/>
			<path d="m17.66 17.66 1.41 1.41"/>
			<path d="M2 12h2"/>
			<path d="M20 12h2"/>
			<path d="m6.34 17.66-1.41 1.41"/>
			<path d="m19.07 4.93-1.41 1.41"/>
		</svg>
	{/if}
</button>

<style>
	.theme-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.5rem;
		border: none;
		background: transparent;
		color: inherit;
		cursor: pointer;
		border-radius: 6px;
		transition: background-color 0.15s ease;
	}

	.theme-toggle:hover {
		background: var(--color-header-hover);
	}

	.theme-toggle:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}
</style>
