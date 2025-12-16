---
status: pending
priority: p2
issue_id: "007"
tags: [code-review, performance, pr-2]
dependencies: []
---

# P2: Sync Duration Already Exceeds Hourly Window

## Problem Statement

The best-case sync time (63.1 minutes) already exceeds the 60-minute hourly window, before any retry overhead. With retries, this compounds significantly.

**Why it matters:**
- Overlapping sync runs could cause database contention
- 1% failure rate: 70.9 minutes (18% overrun)
- 5% failure rate: 101.9 minutes (70% overrun)
- Retry logic makes existing timing problem more visible

## Findings

**Source:** Performance Oracle Agent

**Location:** Pre-existing issue, not introduced by PR #2

**Evidence:**
- 750 total API calls (250 pages x 3 sort orders)
- Average 5s per request + 50ms inter-page delay
- Best case: 63.1 minutes
- Available time: 60 minutes
- **Utilization: 105.2% of hourly window**

**With Retries (added by PR #2):**
- Single timeout adds 62s overhead (60s timeout + 2s backoff)
- 1% failure rate adds ~7.8 minutes
- 5% failure rate adds ~38.8 minutes

## Proposed Solutions

### Option 1: Add Sync Duration Monitoring (Recommended First Step)
**Pros:** Visibility into actual performance
**Cons:** Doesn't fix the issue
**Effort:** Small (15 minutes)
**Risk:** Low

```go
slog.Info("sync complete",
    "duration", duration,
    "apiCallsFailed", failureCount,
    "apiCallsRetried", retryCount,
)
```

### Option 2: Parallel Fetching
**Pros:** Significant speedup possible
**Cons:** More complex, need to manage concurrency
**Effort:** Large (4+ hours)
**Risk:** Medium (rate limiting concerns)

### Option 3: Reduce API Calls
**Pros:** Direct impact on duration
**Cons:** May reduce data coverage
**Effort:** Medium (2 hours)
**Risk:** Medium

## Recommended Action

<!-- Filled during triage -->

## Technical Details

**Affected Files:**
- `internal/sync/sync.go`
- `internal/curseforge/client.go`
- `cmd/sync/main.go`

## Acceptance Criteria

- [ ] Sync duration logged with each run
- [ ] Alert/warning when sync approaches 60 minute threshold
- [ ] Plan for optimization if monitoring shows consistent overruns

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2025-12-16 | Created from PR #2 code review | Performance Oracle identified pre-existing timing issue |

## Resources

- PR #2: https://github.com/Postmodum37/addon-radar/pull/2
