# Phase 5: Polish - SEO Basics & Category Filter

## Overview

Minimal polish improvements based on reviewer feedback. Focused on high-value, low-complexity changes.

**Features:**
1. **SEO Basics** - robots.txt, canonical URLs, og:url
2. **Category Filter** - Wire existing query to API endpoint

**Deferred (per reviewer consensus):**
- Hot addon sync service (premature optimization)
- Dynamic sorting options (no user demand)
- JSON-LD structured data (schema issues)
- Full dynamic sitemap (over-engineering)

## Problem Statement

- Missing `robots.txt` file
- No canonical URLs (potential duplicate content issues)
- Missing `og:url` meta tag
- No way to filter addons by category via API

## Proposed Solution

### 1. SEO Basics

**File: `web/static/robots.txt`**
```
User-agent: *
Allow: /

Sitemap: https://addon-radar.com/sitemap.xml
```

Note: Sitemap URL referenced but not implemented yet. Will add static sitemap in future iteration if needed.

**File: `web/src/routes/+layout.svelte`** (add to svelte:head):
```svelte
<script>
  import { page } from '$app/stores';
</script>

<svelte:head>
  <link rel="canonical" href="https://addon-radar.com{$page.url.pathname}" />
  <meta property="og:url" content="https://addon-radar.com{$page.url.pathname}" />
  <meta property="og:site_name" content="Addon Radar" />
</svelte:head>
```

### 2. Category Filter

Use existing `ListAddonsByCategory` query (already in `sql/queries.sql:69-74`):
```sql
-- name: ListAddonsByCategory :many
SELECT a.* FROM addons a
WHERE a.status = 'active'
  AND $3 = ANY(a.categories)
ORDER BY a.download_count DESC
LIMIT $1 OFFSET $2;
```

**File: `internal/api/handlers.go`** (update handleListAddons):
```go
func (s *Server) handleListAddons(c *gin.Context) {
    page, perPage, offset := parsePaginationParams(c)
    search := c.Query("search")
    categoryStr := c.Query("category")

    ctx := c.Request.Context()
    var addons []database.Addon
    var total int64
    var err error

    if search != "" {
        // Existing search logic
        addons, err = s.queries.SearchAddons(ctx, database.SearchAddonsParams{...})
        total, _ = s.queries.CountSearchAddons(ctx, search)
    } else if categoryStr != "" {
        // New: category filter
        categoryID, parseErr := strconv.ParseInt(categoryStr, 10, 32)
        if parseErr != nil {
            // Invalid category - return empty results (lenient, matches existing pattern)
            categoryID = -1
        }
        addons, err = s.queries.ListAddonsByCategory(ctx, database.ListAddonsByCategoryParams{
            Limit:  int32(perPage),
            Offset: int32(offset),
            Column3: int32(categoryID),
        })
        total, _ = s.queries.CountAddonsByCategory(ctx, int32(categoryID))
    } else {
        // Existing list logic
        addons, err = s.queries.ListAddons(ctx, database.ListAddonsParams{...})
        total, _ = s.queries.CountActiveAddons(ctx)
    }
    // ... rest of handler
}
```

## Acceptance Criteria

### SEO Basics
- [ ] `/robots.txt` returns valid robots file
- [ ] All pages have canonical URLs
- [ ] All pages have og:url meta tag

### Category Filter
- [ ] `/api/v1/addons?category=5` returns addons in that category
- [ ] Invalid category returns empty results (not error)
- [ ] Pagination works with category filter

## Files to Create/Modify

### Create
- `web/static/robots.txt` (5 lines)

### Modify
- `web/src/routes/+layout.svelte` (add ~5 lines)
- `internal/api/handlers.go` (add ~15 lines)

## Testing

1. Verify `robots.txt` accessible at `/robots.txt`
2. Check canonical URLs render correctly in page source
3. Test `/api/v1/addons?category=X` with valid/invalid category IDs

## What Was Cut (Per Reviewer Feedback)

| Feature | Reason |
|---------|--------|
| Hot sync service | Premature optimization - current hourly full sync is sufficient |
| Dynamic sorting | No user demand; adds SQL complexity |
| JSON-LD structured data | Schema misuse issues (aggregateRating semantics) |
| Full dynamic sitemap | Over-engineering; static pages sufficient |
| JsonLd.svelte component | Unnecessary abstraction |
| Twitter meta tags | No @AddonRadar account exists |

## References

- `sql/queries.sql:69-74` - Existing ListAddonsByCategory query
- `sql/queries.sql:77-79` - Existing CountAddonsByCategory query
