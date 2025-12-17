# Data Retention and Cleanup Strategy

**Date**: 2025-12-18
**Status**: Proposed (Revised after review)
**Type**: Enhancement

## Overview

Implement data retention and cleanup to prevent unbounded database growth. Snapshots accumulate indefinitely (~108M rows/year) but the trending algorithm only needs 95 days of data.

## Problem Statement

- **No cleanup exists**: Snapshots grow ~300k rows/day, ~108M rows/year
- **Stale addons**: Addons removed from CurseForge remain active forever
- **Algorithm needs**: Only 24h, 7d, and 90d windows (using 95 days for buffer)

## Solution

Two simple SQL queries + ~15 lines of Go. No stored procedures, no new packages, no stats endpoints.

### 1. Add SQL Queries

**File**: `sql/queries.sql`

```sql
-- name: DeleteOldSnapshots :execrows
DELETE FROM snapshots
WHERE recorded_at < NOW() - INTERVAL '95 days';

-- name: MarkMissingAddonsInactive :execrows
WITH synced_ids AS (SELECT unnest($1::integer[]) AS id)
UPDATE addons
SET status = 'inactive', last_synced_at = NOW()
WHERE status = 'active'
  AND NOT EXISTS (SELECT 1 FROM synced_ids WHERE synced_ids.id = addons.id);
```

### 2. Fix UpsertAddon (Critical)

**File**: `sql/queries.sql` - Modify existing UpsertAddon

Add `status = 'active'` to the ON CONFLICT clause so addons can recover if they reappear:

```sql
-- name: UpsertAddon :exec
INSERT INTO addons (...)
VALUES (...)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    -- ... existing fields ...
    last_synced_at = NOW(),
    status = 'active';  -- ADD THIS LINE
```

### 3. Integrate into Sync Job

**File**: `cmd/sync/main.go` - Add after trending calculation (~15 lines)

```go
// Cleanup old snapshots (95-day retention)
deleted, err := queries.DeleteOldSnapshots(ctx)
if err != nil {
    slog.Warn("snapshot cleanup failed", "error", err)
} else if deleted > 0 {
    slog.Info("snapshots cleaned", "count", deleted)
}

// Mark missing addons as inactive
inactive, err := queries.MarkMissingAddonsInactive(ctx, syncedIDs)
if err != nil {
    slog.Warn("mark inactive failed", "error", err)
} else if inactive > 0 {
    slog.Info("addons marked inactive", "count", inactive)
}
```

**File**: `internal/sync/sync.go` - Expose syncedIDs

The sync service already iterates through mods. Collect the IDs and return them:

```go
func (s *Service) RunFullSync(ctx context.Context) ([]int32, error) {
    // ... existing fetch logic ...

    syncedIDs := make([]int32, 0, len(mods))
    for _, mod := range mods {
        // ... existing upsert logic ...
        syncedIDs = append(syncedIDs, int32(mod.ID))
    }

    return syncedIDs, nil
}
```

## Files to Modify

| File | Changes |
|------|---------|
| `sql/queries.sql` | Add DeleteOldSnapshots, MarkMissingAddonsInactive; fix UpsertAddon |
| `internal/sync/sync.go` | Return syncedIDs from RunFullSync |
| `cmd/sync/main.go` | Add cleanup calls after trending |

Then run: `sqlc generate`

## Why This Design

Per reviewer feedback:

1. **No stored procedure**: PostgreSQL handles DELETE fine. Batching/SKIP LOCKED unnecessary for hourly cleanup with minimal contention.

2. **No cleanup service package**: A 2-line function call doesn't need a struct, config, and constructor.

3. **No stats endpoint**: Use psql or logs. YAGNI.

4. **No config env vars**: 95 days is a requirement (algorithm needs 90d + buffer), not configurable.

5. **NOT EXISTS over NOT IN**: Safer NULL handling, better query plan.

6. **status='active' in UpsertAddon**: Prevents permanent data loss when addons temporarily disappear from API.

## Storage Impact

| Scenario | Rows | Storage |
|----------|------|---------|
| No cleanup (current) | ~108M/year | ~55 GB/year |
| 95-day retention | ~28M (steady) | ~14 GB |

## Testing

1. Run `DELETE FROM snapshots WHERE recorded_at < NOW() - INTERVAL '95 days'` on dev
2. Verify oldest snapshot is ~95 days old after cleanup
3. Test addon reactivation: mark inactive → re-sync → verify status='active'
4. Test with empty syncedIDs array (no crash)

## Implementation Order

1. Add queries to `sql/queries.sql`
2. Fix UpsertAddon to set `status = 'active'`
3. Run `sqlc generate`
4. Update `internal/sync/sync.go` to return syncedIDs
5. Add cleanup calls to `cmd/sync/main.go`
6. Test locally
7. Deploy

## References

- Original design: `docs/plans/2025-12-08-curseforge-api-design.md`
- Trending algorithm: `docs/ALGORITHM.md`
