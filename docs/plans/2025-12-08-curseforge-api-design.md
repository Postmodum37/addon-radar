# CurseForge API Integration Design

> **Status**: Reference document. Core design implemented with enhancements noted below.
>
> **Implementation Notes** (December 2025):
> - Uses multi-query strategy (3 sort orders) instead of single query to overcome 10k API limit
> - Achieves 99.8% catalog coverage (12,406 of ~12,427 Retail addons)
> - Currently syncing Retail only (gameVersionTypeId=517)
> - Hot-only sync not yet implemented (full sync runs hourly)

## Overview

This document describes how Addon Radar syncs and stores data from the CurseForge API to support trending algorithm development.

## Goals

- Store comprehensive data for future algorithm experimentation
- Full catalog coverage of all WoW addons (~7,000)
- Smart snapshot frequency: hourly for "hot" addons, daily for stable ones

## Data Model

### Addons Table

Stores addon metadata, updated on each sync.

| Column | Type | Description |
|--------|------|-------------|
| `id` | integer | CurseForge ID (primary key) |
| `name` | string | Addon display name |
| `slug` | string | URL-friendly identifier |
| `summary` | text | Short description |
| `author_name` | string | Primary author display name |
| `author_id` | integer | Primary author CurseForge ID |
| `logo_url` | string | Addon logo image URL |
| `primary_category_id` | integer | Main category ID |
| `categories` | integer[] | All assigned category IDs |
| `game_versions` | string[] | Supported WoW versions |
| `created_at` | timestamp | When addon was created (from CF) |
| `last_updated_at` | timestamp | Last update on CurseForge |
| `last_synced_at` | timestamp | Our last successful sync |
| `is_hot` | boolean | Whether addon gets hourly snapshots |
| `hot_until` | timestamp | Cooldown expiry for hot status |
| `status` | string | active/inactive |

### Snapshots Table

Time-series metrics for trending calculations.

| Column | Type | Description |
|--------|------|-------------|
| `id` | bigint | Primary key |
| `addon_id` | integer | Foreign key to addons |
| `recorded_at` | timestamp | When snapshot was taken |
| `download_count` | bigint | Total downloads |
| `thumbs_up_count` | integer | Total thumbs up |
| `popularity_rank` | integer | CurseForge popularity rank |
| `rating` | decimal | Average rating |
| `latest_file_date` | timestamp | Most recent file release |

Index on `(addon_id, recorded_at)` for time-range queries.

### Categories Table

Reference data for addon categories.

| Column | Type | Description |
|--------|------|-------------|
| `id` | integer | CurseForge category ID |
| `name` | string | Category display name |
| `slug` | string | URL-friendly identifier |
| `parent_id` | integer | Parent category (nullable) |

## Sync Strategy

### Daily Full Sync

Runs once per day at 3 AM.

1. Fetch all WoW addons via `GET /v1/mods/search`
   - Game ID: 1 (WoW)
   - Page size: 50
   - Sort by: popularity
   - ~160 paginated requests
2. Upsert addon metadata
3. Create snapshot record for each addon
4. Recalculate `is_hot` flag for all addons
5. Mark missing addons as `inactive`

### Hourly Hot Sync

Runs every hour (except during daily sync).

1. Query addons where `is_hot = true`
2. Fetch fresh data via `POST /v1/mods` (batch endpoint)
   - Up to 50 addon IDs per request
3. Create snapshot record for hot addons only

### "Hot" Detection Logic

An addon becomes "hot" if either condition is met:

```
is_hot = (downloads_24h >= 100) OR (growth_pct_24h >= 5.0)
```

Where:
- `downloads_24h` = current download count - download count from 24 hours ago
- `growth_pct_24h` = (downloads_24h / download_count_24h_ago) * 100

**Cooldown**: Once hot, an addon stays hot for 48 hours after last qualifying. This prevents flapping between hourly/daily sync.

## API Considerations

### Rate Limiting

CurseForge doesn't publish explicit limits. Safe operating parameters:
- Keep requests under 100/minute
- Daily sync completes in ~2-3 minutes
- Hourly sync completes in <30 seconds

### Pagination

- Use `index` and `pageSize=50` parameters
- Maximum 10,000 results (WoW catalog fits within this)
- Store total count from first response to detect catalog changes

### Authentication

- Header: `x-api-key`
- Key stored in `CURSEFORGE_API_KEY` environment variable

## Error Handling

### Retry Strategy

- Retry failed requests with exponential backoff
- Maximum 3 attempts per request
- Delays: 1s, 2s, 4s

### Partial Failure Recovery

- If full sync fails mid-way, log last successful page
- Resume from last page on next attempt
- Track failed addon IDs for retry

### Alerting

- Log warning if >1% of requests fail
- Log error if >5% of requests fail
- Track `last_successful_sync_at` to detect stale data

### Deleted Addons

- If addon disappears from API, mark as `status = 'inactive'`
- Retain historical data for analysis
- Do not hard-delete

## Storage Estimates

Assuming ~7,000 addons:

| Data | Records/Year | Estimated Size |
|------|--------------|----------------|
| Addons metadata | 7,000 | ~5 MB |
| Daily snapshots | 2.55M | ~150 MB |
| Hourly snapshots (hot) | 2.6M | ~160 MB |
| **Total Year 1** | ~5.2M records | **~315 MB** |

### Scaling Considerations

- Partition snapshots table by month after year 1
- Add index on `recorded_at` for time-range queries
- Consider TimescaleDB extension for large-scale time-series

## API Endpoints Used

| Endpoint | Purpose | Frequency |
|----------|---------|-----------|
| `GET /v1/mods/search` | Full catalog fetch | Daily |
| `POST /v1/mods` | Batch fetch by IDs | Hourly (hot only) |
| `GET /v1/categories` | Category reference data | Weekly or on-demand |
