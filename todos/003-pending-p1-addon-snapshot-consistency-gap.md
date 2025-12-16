---
status: pending
priority: p1
issue_id: "003"
tags: [code-review, data-integrity, pr-2]
dependencies: []
---

# P1: Addon-Snapshot Consistency Gap

## Problem Statement

An addon can be upserted successfully but its corresponding snapshot creation can fail, creating orphaned metadata without time-series data. This breaks trending calculations which rely on snapshots for velocity measurement.

**Why it matters:**
- Trending calculations fail or produce incorrect results
- Historical data incomplete for affected addons
- Detection requires cross-table consistency checks
- Silent data corruption

## Findings

**Source:** Data Integrity Guardian Agent

**Location:** `/internal/sync/sync.go:66-78`

**Evidence:**
```go
if err := s.upsertAddon(ctx, mod); err != nil {
    errorCount++
    continue
}

if err := s.createSnapshot(ctx, mod); err != nil {
    errorCount++
    continue  // Addon exists but has no snapshot!
}
```

**Scenario:**
1. Addon upsert succeeds (addon record updated)
2. Network hiccup or database error occurs
3. Snapshot creation fails
4. Next iteration continues
5. Addon now has updated metadata but missing latest snapshot

## Proposed Solutions

### Option 1: Transaction per Addon+Snapshot Pair (Recommended)
**Pros:** Guarantees consistency at addon level, reasonable transaction scope
**Cons:** Many small transactions, slight overhead
**Effort:** Medium (1 hour)
**Risk:** Low

```go
for _, mod := range mods {
    err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
        qtx := s.db.WithTx(tx)
        if err := qtx.UpsertAddon(ctx, params); err != nil {
            return err
        }
        return qtx.CreateSnapshot(ctx, snapshotParams)
    })
    if err != nil {
        slog.Error("failed to sync addon", "id", mod.ID, "error", err)
        errorCount++
    }
}
```

### Option 2: Batch Transaction (All or Nothing)
**Pros:** Single transaction, simpler logic
**Cons:** Large transaction, any failure rolls back everything
**Effort:** Medium (1 hour)
**Risk:** Medium (large transaction performance)

### Option 3: Snapshot Idempotency Check
**Pros:** Detect and fix missing snapshots on next run
**Cons:** Doesn't prevent the issue, just detects it
**Effort:** Small (30 minutes)
**Risk:** Low

## Recommended Action

<!-- Filled during triage -->

## Technical Details

**Affected Files:**
- `internal/sync/sync.go`

**Components:** Sync service

## Acceptance Criteria

- [ ] Addon and snapshot are created/updated atomically
- [ ] Failed snapshot rolls back corresponding addon update
- [ ] Error logging identifies which addon failed
- [ ] Existing tests pass

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2025-12-16 | Created from PR #2 code review | Data Integrity Guardian identified consistency gap |

## Resources

- PR #2: https://github.com/Postmodum37/addon-radar/pull/2
