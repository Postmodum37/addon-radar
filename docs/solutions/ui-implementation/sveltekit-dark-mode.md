# SvelteKit Dark Mode with SSR

## Metadata

```yaml
category: ui-implementation
tags: [sveltekit, svelte5, dark-mode, ssr, theming, css-variables]
problem_type: implementation
difficulty: intermediate
date_solved: 2025-12-25
related_pr: "#12"
```

## Problem Statement

Implementing dark mode in a SvelteKit application with proper SSR support, avoiding flash of unstyled content (FOUC), and handling hydration mismatches between server and client.

### Symptoms

- Flash of wrong theme on page load (FOUC)
- Hydration mismatch warnings in console
- Theme not persisting across page refreshes
- OS theme preference changes not detected

## Root Cause

SSR renders with a default theme (usually 'light'), but the client may have a different preference stored in localStorage or from OS settings. This mismatch causes:

1. **FOUC**: Page renders with server theme, then flickers to client theme
2. **Hydration mismatch**: React/Svelte state differs from DOM on mount
3. **State desync**: Component state doesn't match what FOUC prevention script set

## Solution

### 1. CSS Variables for Theming

Use CSS custom properties with a `[data-theme="dark"]` selector:

```css
/* app.css */
:root {
  --color-bg: #FAFAFA;
  --color-surface: #FFFFFF;
  --color-text: #1A1A1A;
  --color-text-muted: #6B7280;
  --color-accent: #3B82F6;
  --color-header-hover: rgba(255, 255, 255, 0.1);
}

[data-theme="dark"] {
  --color-bg: #0F172A;
  --color-surface: #1E293B;
  --color-text: #F1F5F9;
  --color-text-muted: #94A3B8;
  --color-accent: #60A5FA;
  --color-header-hover: rgba(255, 255, 255, 0.1);
}
```

### 2. FOUC Prevention Script

Add an inline blocking script in `app.html` that runs before any rendering:

```html
<!-- app.html, before %sveltekit.body% -->
<script>
  (function() {
    const saved = localStorage.getItem('theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const theme = saved || (prefersDark ? 'dark' : 'light');
    document.documentElement.dataset.theme = theme;
  })();
</script>
```

This script:
- Runs synchronously before page renders
- Sets the correct theme attribute immediately
- Prevents any flash of wrong theme

### 3. Theme Toggle Component (Svelte 5)

The key insight is to **read from the document first** on client initialization:

```svelte
<script lang="ts">
  import { browser } from '$app/environment';

  function getInitialTheme(): 'light' | 'dark' {
    if (!browser) return 'light';
    // First check what the FOUC prevention script already set
    const docTheme = document.documentElement.dataset.theme;
    if (docTheme === 'light' || docTheme === 'dark') return docTheme;
    // Fallback to localStorage or system preference
    const saved = localStorage.getItem('theme');
    if (saved === 'light' || saved === 'dark') return saved;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  let theme = $state<'light' | 'dark'>(getInitialTheme());

  // Sync to DOM only when theme actually changes
  $effect(() => {
    if (browser && document.documentElement.dataset.theme !== theme) {
      document.documentElement.dataset.theme = theme;
      localStorage.setItem('theme', theme);
    }
  });

  // Listen for system theme changes (when no saved preference)
  $effect(() => {
    if (!browser) return;
    if (localStorage.getItem('theme')) return;

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
    <!-- Moon icon for "switch to dark" -->
  {:else}
    <!-- Sun icon for "switch to light" -->
  {/if}
</button>
```

## Key Patterns

### Reading Document State First

```typescript
// WRONG - causes hydration mismatch
function getInitialTheme() {
  if (!browser) return 'light';
  const saved = localStorage.getItem('theme');
  // ...
}

// RIGHT - syncs with what FOUC script already set
function getInitialTheme() {
  if (!browser) return 'light';
  const docTheme = document.documentElement.dataset.theme;
  if (docTheme === 'light' || docTheme === 'dark') return docTheme;
  // fallback...
}
```

### Conditional DOM Writes

```typescript
// WRONG - writes on every mount, even when unnecessary
$effect(() => {
  if (browser) {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem('theme', theme);
  }
});

// RIGHT - only writes when value differs
$effect(() => {
  if (browser && document.documentElement.dataset.theme !== theme) {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem('theme', theme);
  }
});
```

### System Theme Listener

```typescript
// Listen for OS theme changes when user hasn't set explicit preference
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
```

## Prevention Strategies

### 1. SSR State Synchronization

Always initialize client state from the actual DOM when dealing with values that may have been set by blocking scripts:

```typescript
// For any value set by blocking scripts in app.html
function getInitialValue() {
  if (!browser) return defaultValue;
  return document.documentElement.dataset.something || defaultValue;
}
```

### 2. Blocking Script Pattern

For critical render-blocking preferences (theme, locale, etc.), use inline scripts in `app.html`:

```html
<script>
  // Runs before ANY rendering
  document.documentElement.dataset.theme = /* determine theme */;
</script>
```

### 3. Testability

Always add data attributes for testing:

```svelte
<button
  data-testid="theme-toggle"
  data-theme={theme}
>
```

## Verification Checklist

- [ ] No FOUC on page load (test with cleared cache)
- [ ] Theme persists across refreshes
- [ ] Respects OS preference on first visit
- [ ] OS theme changes update when no saved preference
- [ ] No hydration warnings in console
- [ ] Toggle works correctly in both directions
- [ ] Accessible (aria-label, focus-visible states)

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    SvelteKit App (SSR)                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  1. Browser Requests Page                                    │
│     ↓                                                         │
│  2. Server Renders app.html                                  │
│     ├─ <head>: CSS loads (defines variables)                │
│     ├─ <body>: BLOCKING SCRIPT RUNS                         │
│     │  └─ Sets document.documentElement.dataset.theme       │
│     └─ %sveltekit.body%: Component hydrates                │
│     ↓                                                         │
│  3. Browser Renders with Correct Theme                       │
│     ↓                                                         │
│  4. Component Mounts                                         │
│     ├─ getInitialTheme() reads DOM state                    │
│     ├─ $state initialized from function                     │
│     └─ Effects set up listeners                             │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## State Priority Order

```
getInitialTheme() Decision Tree:

                    ┌─ Is browser environment?
                    │  └─ NO → Return 'light' (SSR safe default)
                    │
                    YES
                    │
                    ├─ 1st Priority: Document State
                    │  document.documentElement.dataset.theme
                    │  ├─ 'light' or 'dark'? → Use it
                    │  └─ No/empty? → Continue
                    │
                    ├─ 2nd Priority: Saved Preference
                    │  localStorage.getItem('theme')
                    │  ├─ 'light' or 'dark'? → Use it
                    │  └─ No/empty? → Continue
                    │
                    └─ 3rd Priority: System Preference
                       window.matchMedia('(prefers-color-scheme: dark)')
                       ├─ matches? → Use 'dark'
                       └─ not matches? → Use 'light'

Why this order?
1. DOM state (from FOUC script) → Ensures component hydration matches
2. localStorage → Respects user's explicit choice
3. System preference → Falls back to OS setting
```

## Testing Matrix

```
                    │ Initial Load │ After Toggle │ After Reload │ System Change
────────────────────┼──────────────┼──────────────┼──────────────┼──────────────
Light (Default)     │ ✓ Light      │ Toggle→Dark  │ ✓ Light      │ N/A
Light (Saved)       │ ✓ Light      │ Toggle→Dark  │ ✓ Light      │ Ignored
Light (System)      │ ✓ Light      │ Toggle→Dark  │ ✓ Light      │ Switches
Dark (Saved)        │ ✓ Dark       │ Toggle→Light │ ✓ Dark       │ Ignored
Dark (System)       │ ✓ Dark       │ Toggle→Light │ ✓ Dark       │ Switches

✓ = Theme matches expected
→ = State transition works
Ignored = System preference doesn't override saved
Switches = System preference updates theme
```

## Related Resources

- [SvelteKit docs on loading data](https://kit.svelte.dev/docs/load)
- [Svelte 5 runes documentation](https://svelte.dev/docs/svelte/$state)
- [prefers-color-scheme MDN](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-color-scheme)
