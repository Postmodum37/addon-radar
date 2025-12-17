# Data Retention and Cleanup Strategy

**Date**: 2025-12-18
**Status**: Implemented

## Problem

- Snapshots grow ~300k rows/day (~108M/year) unbounded
- Stale addons remain active after removal from CurseForge
- Algorithm only needs 90 days of data (using 95-day buffer)

## Solution

Three SQL queries + cleanup logic in sync job:

1. **DeleteOldSnapshotsBatch** - Batched delete (10k/batch) to avoid locking
2. **MarkMissingAddonsInactive** - Mark addons not in sync response
3. **UpsertAddon fix** - Set `status='active'` on conflict (allows reactivation)

Safety guards:
- Minimum 1000 synced addons required before marking inactive (prevents API failure catastrophe)
- Batched deletes with 100ms sleep between batches (prevents table lock)

## Impact

| Scenario | Rows/Year | Storage |
|----------|-----------|---------|
| Before | ~108M | ~55 GB |
| After | ~28M (steady) | ~14 GB |

**75% storage reduction**

## Files Modified

- `sql/queries.sql` - 3 queries
- `internal/sync/sync.go` - Return syncedIDs
- `cmd/sync/main.go` - Cleanup logic with guards

## References

- Trending algorithm: `docs/ALGORITHM.md` (requires 90-day snapshots)
