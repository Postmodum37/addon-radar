# Frontend V3 Design: Cards, Pagination & Naming

**Date:** 2025-12-25
**Status:** Approved

## Overview

Improve the frontend with richer addon cards, working pagination on trending pages, and clearer category naming.

## Changes Summary

| Area | Current | New |
|------|---------|-----|
| Category naming | "Hot Right Now", "Rising Stars" | "Trending", "Rising" |
| Card content | Logo, name, author, downloads, velocity | + rank position, summary, likes, updated date, game versions |
| Card hierarchy | All cards same size | Top 3 featured (larger), rest compact |
| Rank badges | Only shows upward movement | Shows up (green), down (red), unchanged (gray), new (blue) |
| Pagination | Broken (API returns only 20) | Working with page/per_page params |

## Card Design

### Top 3 Featured Cards (Larger)

```
+----------------------------------------------------------+
| #1                                             +5 (24h)  |
| +--------+                                               |
| |  LOGO  |  Deadly Boss Mods                             |
| |  64px  |  by MysticalOS                                |
| +--------+                                               |
| Boss encounter warnings and timers for all raids...      |
|                                                          |
| 72.8M downloads · 1.2K likes · +2.7K/day                 |
| Updated 2 days ago · Retail, Classic                     |
+----------------------------------------------------------+
```

### Regular Cards (#4 and beyond, Compact)

```
+----------------------------------------------------------+
| #4  +------+  Auctionator                      -2 (24h)  |
|     | LOGO |  by plusmouse                               |
|     | 48px |  Makes it easier to use the auction...      |
|     +------+  168M · 892 likes · +1.6K/day · 3d ago      |
+----------------------------------------------------------+
```

### Rank Badge States

| State | Display | Color |
|-------|---------|-------|
| Moving up | +5 | Green |
| Moving down | -3 | Red |
| No change | = | Gray |
| New to list | NEW | Blue |

## API Changes

### Updated Endpoints

```
GET /api/v1/trending/hot?page=1&per_page=20
GET /api/v1/trending/rising?page=1&per_page=20
```

### Response Format

```json
{
  "data": [
    {
      "id": 1592,
      "name": "Bagnon",
      "slug": "bagnon",
      "summary": "All in one displays for your inventory...",
      "author_name": "jaliborc",
      "logo_url": "https://...",
      "download_count": 149297583,
      "thumbs_up_count": 1024,
      "game_versions": ["Retail", "Classic"],
      "last_updated_at": "2025-12-20T23:25:10Z",
      "score": 472.83,
      "rank": 1,
      "rank_change_24h": 5,
      "rank_change_7d": null,
      "download_velocity": 1573.38
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 7888,
    "total_pages": 395
  }
}
```

### New SQL Queries

```sql
-- name: ListHotAddonsPaginated :many
SELECT a.*, t.hot_score, t.download_velocity
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 500
  AND t.hot_score > 0
ORDER BY t.hot_score DESC
LIMIT $1 OFFSET $2;

-- name: CountHotAddons :one
SELECT COUNT(*)
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 500
  AND t.hot_score > 0;

-- name: ListRisingAddonsPaginated :many
SELECT a.*, t.rising_score, t.download_velocity
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 50
  AND a.download_count <= 10000
  AND t.rising_score > 0
  AND a.id NOT IN (
      SELECT addon_id FROM trending_scores
      WHERE hot_score > 0
      ORDER BY hot_score DESC
      LIMIT 20
  )
ORDER BY t.rising_score DESC
LIMIT $1 OFFSET $2;

-- name: CountRisingAddons :one
SELECT COUNT(*)
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 50
  AND a.download_count <= 10000
  AND t.rising_score > 0
  AND a.id NOT IN (
      SELECT addon_id FROM trending_scores
      WHERE hot_score > 0
      ORDER BY hot_score DESC
      LIMIT 20
  );
```

## Files to Modify

### Backend

| File | Changes |
|------|---------|
| `sql/queries.sql` | Add paginated queries + count queries |
| `internal/database/queries.sql.go` | Regenerate with sqlc |
| `internal/api/handlers.go` | Add pagination params, return meta object |

### Frontend

| File | Changes |
|------|---------|
| `web/src/lib/components/AddonCard.svelte` | New card design with rank, summary, stats |
| `web/src/lib/components/FeaturedAddonCard.svelte` | New component for top 3 |
| `web/src/lib/components/RankBadge.svelte` | Support all states (up/down/unchanged/new) |
| `web/src/lib/api.ts` | Update trending functions for pagination |
| `web/src/routes/+page.svelte` | Rename categories, use featured cards for top 3 |
| `web/src/routes/trending/hot/+page.svelte` | Update title to "Trending", fix pagination |
| `web/src/routes/trending/rising/+page.svelte` | Update title to "Rising", fix pagination |
| `web/src/routes/trending/hot/+page.server.ts` | Fetch paginated data from API |
| `web/src/routes/trending/rising/+page.server.ts` | Fetch paginated data from API |

## Implementation Order

1. **Backend pagination** - Add SQL queries, update handlers
2. **RankBadge updates** - Support all badge states
3. **AddonCard redesign** - Add all new fields
4. **FeaturedAddonCard** - Create larger card variant
5. **Page updates** - Rename categories, integrate new cards
6. **Testing** - Verify pagination, badges, responsive design

## Design Decisions

### Why traditional pagination over infinite scroll?
- Matches existing Search page pattern
- Better for SEO (distinct URLs per page)
- Simpler implementation
- Users can bookmark/share specific pages

### Why top 3 featured instead of top 5?
- Keeps focus on truly trending addons
- Prevents page from feeling too heavy
- Common pattern (podium: gold, silver, bronze)

### Why show rank changes for 24h only on cards?
- 7-day changes add clutter
- 24h is more actionable/interesting
- 7-day available on detail page if needed
