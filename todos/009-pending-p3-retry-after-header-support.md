---
status: pending
priority: p3
issue_id: "009"
tags: [code-review, enhancement, pr-2]
dependencies: []
---

# P3: Add Retry-After Header Support for 429 Responses

## Problem Statement

The retry logic retries 429 (rate limit) errors with fixed exponential backoff but doesn't respect `Retry-After` headers that CurseForge might send.

**Why it matters:**
- May retry too aggressively if API specifies longer backoff
- Could lead to IP banning with strict rate limits
- Wastes resources retrying when we know it will fail

## Findings

**Source:** Security Sentinel Agent

**Location:** `/internal/curseforge/client.go:63`

**Evidence:**
```go
// Fixed backoff - ignores Retry-After header
backoff := time.Duration(1<<uint(attempt)) * c.backoffMultiply
```

**Current Mitigations:**
- Exponential backoff provides reasonable spacing (2s, 4s, 8s)
- Small delay between pagination requests (50ms)
- Fixed max retries (3) prevents infinite loops

## Proposed Solutions

### Option 1: Parse Retry-After Header
**Pros:** Respects API guidance
**Cons:** More code
**Effort:** Small (30 minutes)
**Risk:** Low

```go
if resp.StatusCode == http.StatusTooManyRequests {
    if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
        if seconds, err := strconv.Atoi(retryAfter); err == nil {
            backoff = time.Duration(seconds) * time.Second
        }
    }
}
```

## Recommended Action

<!-- Filled during triage - implement only if 429s observed in production -->

## Technical Details

**Affected Files:**
- `internal/curseforge/client.go`

## Acceptance Criteria

- [ ] Retry-After header parsed when present
- [ ] Falls back to exponential backoff when header absent
- [ ] Test covers Retry-After scenario

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2025-12-16 | Created from PR #2 code review | Nice-to-have if 429s become an issue |

## Resources

- PR #2: https://github.com/Postmodum37/addon-radar/pull/2
- RFC 7231 Retry-After header
