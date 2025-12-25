# Frontend Design Update: Dark Mode

**Date**: 2025-12-25
**Status**: Ready for Implementation
**Type**: Enhancement
**Estimated Time**: 2-3 hours

---

## Overview

Add dark mode support to Addon Radar. No new dependencies - just CSS variables and a toggle component.

---

## Background

### Why Only Dark Mode?

After thorough review, the current frontend is in good shape:
- Clean 51-line CSS with semantic variables
- Well-structured Svelte 5 components
- Working DIY chart
- Good UX (debounced search, keyboard navigation)
- Lighthouse score ~95

The only user-facing feature that adds value is dark mode. Everything else (Tailwind, component libraries, chart replacements) would be replacing working code with different working code.

---

## Implementation

### Files to Modify

```
web/
├── src/
│   ├── app.css                           # Add dark theme variables
│   ├── app.html                          # Add theme init script (prevent FOUC)
│   ├── lib/
│   │   └── components/
│   │       └── ThemeToggle.svelte        # NEW: Toggle component
│   └── routes/
│       └── +layout.svelte                # Add toggle to header
```

### Step 1: Dark Theme CSS Variables

Add dark mode color palette to `web/src/app.css`:

```css
/* After existing :root block */

[data-theme="dark"] {
  /* Core colors */
  --color-bg: #0F172A;
  --color-surface: #1E293B;
  --color-header: #0F172A;
  --color-border: #334155;

  /* Text colors */
  --color-text: #F1F5F9;
  --color-text-muted: #94A3B8;

  /* Accent colors - keep same for brand consistency */
  --color-accent: #60A5FA;
  --color-accent-hover: #3B82F6;

  /* Badge colors - adjusted for dark backgrounds */
  --color-rising: #34D399;
  --color-rising-bg: #064E3B;
  --color-new: #A78BFA;
  --color-new-bg: #4C1D95;
  --color-hot: #F87171;
  --color-falling: #F87171;
  --color-falling-bg: #7F1D1D;

  /* Shadows - more subtle in dark mode */
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.3);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.4), 0 2px 4px -2px rgb(0 0 0 / 0.3);
}
```

### Step 2: Prevent Flash of Unstyled Content (FOUC)

Add inline script to `web/src/app.html` before `%sveltekit.body%`:

```html
<script>
  (function() {
    const saved = localStorage.getItem('theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const theme = saved || (prefersDark ? 'dark' : 'light');
    document.documentElement.dataset.theme = theme;
  })();
</script>
```

### Step 3: Theme Toggle Component

Create `web/src/lib/components/ThemeToggle.svelte`:

```svelte
<script lang="ts">
  import { browser } from '$app/environment';

  function getInitialTheme(): 'light' | 'dark' {
    if (!browser) return 'light';
    const saved = localStorage.getItem('theme');
    if (saved === 'light' || saved === 'dark') return saved;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  let theme = $state<'light' | 'dark'>(getInitialTheme());

  $effect(() => {
    if (browser) {
      document.documentElement.dataset.theme = theme;
      localStorage.setItem('theme', theme);
    }
  });

  function toggle() {
    theme = theme === 'light' ? 'dark' : 'light';
  }
</script>

<button
  onclick={toggle}
  class="theme-toggle"
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
    background: rgba(255, 255, 255, 0.1);
  }

  .theme-toggle:focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
  }
</style>
```

### Step 4: Add Toggle to Header

Modify `web/src/routes/+layout.svelte`:

```svelte
<script lang="ts">
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';
  // ... existing imports
</script>

<!-- In the header, next to search -->
<header>
  <div class="header-content">
    <a href="/" class="logo">Addon Radar</a>
    <div class="header-actions">
      <SearchAutocomplete />
      <ThemeToggle />
    </div>
  </div>
</header>
```

Add CSS for header-actions layout:

```css
.header-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
```

---

## Acceptance Criteria

- [ ] Dark theme toggle visible in header
- [ ] Theme persists across page refreshes (localStorage)
- [ ] Respects system preference on first visit
- [ ] No flash of wrong theme on page load (FOUC prevention)
- [ ] All components render correctly in dark mode
- [ ] Chart is readable in dark mode
- [ ] Focus states visible in both themes
- [ ] Mobile: toggle accessible and tappable

---

## Testing Checklist

- [ ] Toggle switches between light and dark
- [ ] Refresh page - theme persists
- [ ] Clear localStorage, set OS to dark mode - site defaults to dark
- [ ] Clear localStorage, set OS to light mode - site defaults to light
- [ ] Check all pages: homepage, trending/hot, trending/rising, addon detail, search
- [ ] Check addon cards, featured cards, rank badges, chart
- [ ] Test on mobile Safari and Chrome
- [ ] Run Lighthouse - performance should stay > 90

---

## Color Reference

| Element | Light | Dark |
|---------|-------|------|
| Background | #FAFAFA | #0F172A |
| Surface (cards) | #FFFFFF | #1E293B |
| Header | #111827 | #0F172A |
| Border | #E5E7EB | #334155 |
| Text | #1A1A1A | #F1F5F9 |
| Text muted | #6B7280 | #94A3B8 |
| Accent | #3B82F6 | #60A5FA |
| Rising badge | #10B981 / #D1FAE5 | #34D399 / #064E3B |
| New badge | #8B5CF6 / #EDE9FE | #A78BFA / #4C1D95 |
| Hot/Falling badge | #EF4444 / #FEE2E2 | #F87171 / #7F1D1D |

---

## What We're NOT Doing (and why)

| Feature | Why Not |
|---------|---------|
| Tailwind CSS | Current CSS works, adds 15KB+ for no user benefit |
| shadcn-svelte | Components already work, just different syntax |
| LayerChart | DIY chart is 70 lines and works fine |
| Lucide icons | Only need 2 icons (sun/moon), inline SVG is simpler |
| Loading skeletons | SSR handles loading, no visible loading states |
| Icon system | No icons to replace, future problem if ever needed |

---

## Future Considerations

If we later need:
- **More icons**: Evaluate Lucide at that time
- **Component library**: Evaluate if we're building many new components
- **Tailwind**: Evaluate if CSS maintenance becomes painful

For now, keep it simple. Ship dark mode.

---

## References

- Current CSS: `web/src/app.css`
- Layout: `web/src/routes/+layout.svelte`
- Review feedback: DHH, Kieran, and Simplicity reviewers all agreed on this approach
