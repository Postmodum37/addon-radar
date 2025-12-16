---
status: pending
priority: p1
issue_id: "002"
tags: [code-review, data-integrity, pr-2]
dependencies: []
---

# P1: Partial Sync Creates Inconsistent Database State

## Problem Statement

If the sync fails after writing some addons but before completing all addons, the database will contain an inconsistent state with some addons marked as recently synced while others remain stale. This affects trending calculations and API consumers.

**Why it matters:**
- Trending scores calculated on incomplete data
- API consumers see mixture of fresh and stale data
- No way to detect or recover from partial sync without manual intervention
- Best-case sync time (63.1 min) already exceeds hourly window

## Findings

**Source:** Data Integrity Guardian Agent

**Location:** `/internal/sync/sync.go:45-90`

**Evidence:**
```go
func (s *Service) RunFullSync(ctx context.Context) error {
    mods, err := s.client.GetAllWoWAddons(ctx)
    if err != nil {
        return fmt.Errorf("fetch addons: %w", err) // FAIL: No DB writes happen
    }

    // Individual upserts - no transaction!
    for _, mod := range mods {
        if err := s.upsertAddon(ctx, mod); err != nil {
            errorCount++
            continue  // Partial writes accumulate
        }
        if err := s.createSnapshot(ctx, mod); err != nil {
            errorCount++
            continue  // Snapshot can fail even if addon succeeded
        }
    }
}
```

**Scenario:**
1. Sync processes addons 1-3900 successfully
2. API timeout occurs on page 3901 (now with retries, less likely but still possible)
3. All retries exhausted
4. Entire sync fails
5. Database has 3900 addons marked as synced "now", ~8600 with stale timestamps

## Proposed Solutions

### Option 1: Database Transaction Wrapper (Recommended)
**Pros:** Atomic operation, consistent state guaranteed
**Cons:** Large transaction, potential lock contention
**Effort:** Medium (2 hours)
**Risk:** Medium (need to test transaction performance)

```go
func (s *Service) RunFullSync(ctx context.Context) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    qtx := s.db.WithTx(tx)
    // ... fetch and process addons ...

    return tx.Commit(ctx)
}
```

### Option 2: Checkpoint/Resume Capability
**Pros:** Resilient to failures, can resume from last success
**Cons:** More complex, needs state tracking
**Effort:** Large (4+ hours)
**Risk:** Medium

### Option 3: Accept Current Behavior with Monitoring
**Pros:** No code changes needed
**Cons:** Data inconsistency risk remains
**Effort:** Small (add monitoring)
**Risk:** High (silent data corruption)

## Recommended Action

<!-- Filled during triage -->

## Technical Details

**Affected Files:**
- `internal/sync/sync.go`

**Components:** Sync service, database layer

**Database Changes:** None for Option 1

## Acceptance Criteria

- [ ] Sync either completes fully or rolls back completely
- [ ] No partial sync states possible
- [ ] Performance acceptable (sync still completes in reasonable time)
- [ ] Monitoring shows transaction success/failure

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2025-12-16 | Created from PR #2 code review | Data Integrity Guardian identified atomicity gap |

## Resources

- PR #2: https://github.com/Postmodum37/addon-radar/pull/2
- Related: Performance Oracle noted sync already exceeds hourly window
