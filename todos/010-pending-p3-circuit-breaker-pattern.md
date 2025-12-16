---
status: pending
priority: p3
issue_id: "010"
tags: [code-review, architecture, pr-2]
dependencies: []
---

# P3: Add Circuit Breaker Pattern

## Problem Statement

If CurseForge API is completely down, the sync will waste time retrying every single request. A circuit breaker would detect cascading failures and fail-fast.

**Why it matters:**
- Worst case: 750 requests x (60s timeout + 14s backoff) = 15+ hours wasted
- No early termination on total API failure
- Resources wasted on guaranteed failures

## Findings

**Source:** Architecture Strategist Agent, Performance Oracle Agent

**Observation:** Sync job runs hourly in cron environment. Total failure is observable in logs, and job can be re-run manually. Circuit breaker would help but isn't critical.

## Proposed Solutions

### Option 1: Simple Consecutive Failure Counter
**Pros:** Simple, no dependencies
**Cons:** Basic circuit breaker
**Effort:** Small (30 minutes)
**Risk:** Low

```go
consecutiveFailures := 0
maxConsecutiveFailures := 10

for _, sort := range sortOrders {
    mods, err := c.fetchWithSort(ctx, gameVersionTypeID, sort.field)
    if err != nil {
        consecutiveFailures++
        if consecutiveFailures >= maxConsecutiveFailures {
            return nil, fmt.Errorf("circuit breaker: %d consecutive failures", consecutiveFailures)
        }
    } else {
        consecutiveFailures = 0
    }
}
```

### Option 2: Use External Library (sony/gobreaker)
**Pros:** Full-featured, battle-tested
**Cons:** New dependency
**Effort:** Medium (1 hour)
**Risk:** Low

## Recommended Action

<!-- Filled during triage - implement only if API outages observed -->

## Acceptance Criteria

- [ ] Sync fails fast after N consecutive failures
- [ ] Clear error message about circuit breaker activation
- [ ] Reset on successful request

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2025-12-16 | Created from PR #2 code review | Defensive pattern for API outages |

## Resources

- PR #2: https://github.com/Postmodum37/addon-radar/pull/2
