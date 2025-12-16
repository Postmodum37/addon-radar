---
status: pending
priority: p2
issue_id: "006"
tags: [code-review, architecture, testing, pr-2]
dependencies: []
---

# P2: Testing Seam Violates Encapsulation

## Problem Statement

The `backoffMultiply` field is exposed on the Client struct solely for testing purposes. This is "test-induced design damage" that violates encapsulation and sets a precedent for future development.

**Why it matters:**
- Production code modified to accommodate testing
- Field could be accidentally set to 0 in production
- Sets precedent for adding more test-specific fields
- Breaks encapsulation of internal implementation

## Findings

**Source:** Architecture Strategist Agent, Pattern Recognition Specialist, Code Simplicity Reviewer

**Location:** `/internal/curseforge/client.go:21, 32`

**Evidence:**
```go
type Client struct {
    apiKey          string
    httpClient      *http.Client
    baseURL         string
    backoffMultiply time.Duration // For testing: set to 0 to disable backoff
}
```

**Note from Simplicity Reviewer:** Tests only wait 14ms max (2+4+8) with real backoff. The optimization may not be necessary.

## Proposed Solutions

### Option 1: Accept Current Approach (Pragmatic)
**Pros:** Already implemented, tests run fast
**Cons:** Technical debt, encapsulation violation
**Effort:** None
**Risk:** Low (isolated case)

### Option 2: Use Functional Options Pattern
**Pros:** Cleaner API, encapsulation preserved
**Cons:** More verbose
**Effort:** Medium (30 minutes)
**Risk:** Low

```go
type ClientOption func(*Client)

func WithBackoffMultiplier(d time.Duration) ClientOption {
    return func(c *Client) { c.backoffMultiply = d }
}

func NewClient(apiKey string, opts ...ClientOption) *Client {
    c := &Client{...}
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

### Option 3: Remove Testing Seam Entirely
**Pros:** Simplest code, no encapsulation violation
**Cons:** Tests wait 14ms (negligible)
**Effort:** Small (15 minutes)
**Risk:** Low

## Recommended Action

<!-- Filled during triage -->

## Technical Details

**Affected Files:**
- `internal/curseforge/client.go`
- `internal/curseforge/client_test.go`

## Acceptance Criteria

- [ ] Client struct doesn't expose testing-only fields
- [ ] Tests still run in reasonable time (<5s)
- [ ] Production code unaffected

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2025-12-16 | Created from PR #2 code review | Multiple reviewers flagged encapsulation concern |

## Resources

- PR #2: https://github.com/Postmodum37/addon-radar/pull/2
- Go functional options pattern
