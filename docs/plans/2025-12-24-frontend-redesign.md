# Frontend Redesign

**Date:** 2025-12-24
**Status:** Approved
**Goal:** Quick discovery - help users find trending addons at a glance

## Overview

Redesign Addon Radar's frontend to fix current UX issues and create a clean, professional interface that doesn't feel AI-generated. Focus on meaningful data display and proper visual hierarchy.

## Problems Addressed

| Current Issue | Solution |
|---------------|----------|
| Scores (1442.8) are meaningless | Replace with rank badge + download velocity |
| Thumbs up shown but unused | Remove entirely |
| Generic AI-made aesthetic | Clean minimal design with dark header accent |
| Game versions unordered | Simplified version display (e.g., "Retail 11.0+") |
| Hourly download history cluttered | Weekly trend chart |
| No search autocomplete | Add dropdown with top 5 matches |
| Can't view more trending addons | Add paginated /trending/hot and /trending/rising pages |

---

## Card Design

### Structure

```
┌─────────────────────────────────────────────┐
│ [↑5]                                        │  ← Rank badge (top-left corner)
│  ┌────┐                                     │
│  │Logo│  Addon Name                         │
│  └────┘  by AuthorName                      │
│                                             │
│  149.3M downloads · +2.3K/day               │  ← Velocity replaces score
└─────────────────────────────────────────────┘
```

### Rank Badge Behavior

- **Green badge with ↑N** - Addon rose N positions
- **"New" badge** - First appearance in top 100
- **No badge** - Position unchanged or dropped (keep it positive)

### Velocity Display

- **Hot addons:** Daily velocity ("+2.3K/day") - established, high volume
- **Rising addons:** Weekly velocity ("+847/week") - smaller, need longer window

### Removed Elements

- Score numbers (1442.8, etc.)
- Thumbs up indicator
- Purple gradient backgrounds

---

## Visual Design System

### Color Palette

| Role | Hex | Usage |
|------|-----|-------|
| Background | `#FAFAFA` | Page background |
| Surface | `#FFFFFF` | Cards, elevated elements |
| Text primary | `#1A1A1A` | Headings, addon names |
| Text secondary | `#6B7280` | Author names, metadata |
| Accent | `#3B82F6` | Links, interactive elements |
| Rising badge | `#10B981` | Green for positive rank changes |
| New badge | `#8B5CF6` | Purple for "New" entries |
| Header | `#111827` | Dark header bar |

### Typography

- **Headings:** Inter or system font, semi-bold, tight letter-spacing
- **Body:** Regular weight, comfortable line-height
- **Metadata:** Smaller size, secondary color

### Spacing

- Generous whitespace between sections
- Consistent card gaps (16-20px)
- Cards don't touch - breathing room matters

### Dark Accent Usage

- Dark header/nav bar contrasts against light page
- Subtle card shadows instead of borders
- No gradients on cards - solid white background

---

## Addon Detail Page

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  [← Back to Trending]                                   │
│                                                         │
│  ┌────────┐                                             │
│  │  Logo  │  Addon Name                                 │
│  │  80px  │  by AuthorName                              │
│  └────────┘                                             │
│                                                         │
│  Short description text from CurseForge...              │
│                                                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │
│  │ 149.3M      │ │ Dec 17      │ │ Retail      │        │
│  │ downloads   │ │ last update │ │ 11.0+       │        │
│  └─────────────┘ └─────────────┘ └─────────────┘        │
│                                                         │
│  ┌─────────────────────────────────────────────┐        │
│  │  Weekly Download Trend (simple line chart)  │        │
│  │  4 weeks of daily data                      │        │
│  └─────────────────────────────────────────────┘        │
│                                                         │
│  [View on CurseForge →]                                 │
└─────────────────────────────────────────────────────────┘
```

### Key Changes

- **Three stat cards:** Total downloads, last updated, simplified version
- **Simplified version:** "Retail 11.0+" instead of listing every patch number
- **Weekly trend chart:** Replaces hourly download history
- **Clear CTA:** Link to CurseForge for full details/download

### Removed Elements

- Raw hourly download history
- Unordered version list
- Thumbs up count

---

## Search Autocomplete

### Dropdown Behavior

```
┌──────────────────────────────────────┐
│  🔍  deta|                           │
└──────────────────────────────────────┘
    ┌──────────────────────────────────┐
    │  [Logo] Details! Damage Meter    │
    │         305M downloads           │
    │                                  │
    │  [Logo] Details! Streamer        │
    │         1.2M downloads           │
    │                                  │
    │  [Logo] DetailedDeathRecap       │
    │         89K downloads            │
    │                                  │
    │  ────────────────────────────    │
    │  View all results for "deta" →   │
    └──────────────────────────────────┘
```

### Specifications

- Triggers after 2+ characters typed
- Shows top 5 matches by relevance
- Each result clickable → addon page
- "View all results" link → full search page
- Keyboard navigation: arrow keys + Enter
- Debounced API calls (300ms delay)
- Matching: Prioritize prefix matches over substring

---

## Trending List Pages

### New Routes

- `/trending/hot` - Full paginated Hot Right Now list
- `/trending/rising` - Full paginated Rising Stars list

### Page Layout

```
┌─────────────────────────────────────────────────────────┐
│  [Header with search]                                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Hot Right Now                                          │
│  Established addons with high download velocity         │
│                                                         │
│  ┌─────────────────────────────────────────────────┐    │
│  │ [Card 1]                                        │    │
│  ├─────────────────────────────────────────────────┤    │
│  │ [Card 2]                                        │    │
│  ├─────────────────────────────────────────────────┤    │
│  │ ...                                             │    │
│  └─────────────────────────────────────────────────┘    │
│                                                         │
│  [← Previous]                [Page 1 of 10]    [Next →] │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Specifications

- Same card design as homepage
- 20 addons per page
- Simple pagination (not infinite scroll)
- Back link to homepage

### Homepage Integration

- Show top 10 Hot and top 10 Rising on homepage
- "View all Hot →" link below Hot section
- "View all Rising →" link below Rising section

---

## Implementation Notes

### Backend Changes Required

1. **New API fields:** Add `rank_change` and `velocity` to addon list responses
2. **Simplified versions:** Add endpoint or field for human-readable version (e.g., "Retail 11.0+")
3. **Weekly trend data:** Aggregate hourly snapshots into daily/weekly for chart
4. **Search endpoint:** Add lightweight search endpoint for autocomplete (name prefix matching)

### Frontend Components

1. **AddonCard** - Redesigned with badge + velocity
2. **RankBadge** - Corner badge component
3. **TrendChart** - Simple line chart for weekly downloads
4. **SearchAutocomplete** - Dropdown with results
5. **Pagination** - Reusable for trending pages

### Data Considerations

- Rank changes calculated during sync job (compare current vs previous position)
- Velocity calculated from existing hourly snapshots
- Version simplification can be done client-side by parsing version strings
