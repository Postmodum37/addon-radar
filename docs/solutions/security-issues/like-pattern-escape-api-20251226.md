# LIKE Pattern Escape for Search API

## Metadata

```yaml
---
module: API
date: 2025-12-26
problem_type: security_issue
component: service_object
symptoms:
  - "LIKE pattern wildcards (%, _) not escaped in search"
  - "Potential pattern-based DoS via malicious search queries"
root_cause: missing_validation
resolution_type: code_fix
severity: high
tags: [sql-injection, like-pattern, dos-prevention, go, gin]
---
```

## Problem

The `/api/v1/addons?search=` endpoint passed user input directly to a SQL LIKE query without escaping wildcard characters. This allowed attackers to craft patterns like `%_%_%_%_%` that could cause expensive regex scans.

### Observable Symptoms

1. Search queries with `%` or `_` characters triggered unintended pattern matching
2. Malicious patterns could cause slow queries (DoS vector)

### Environment

- Go 1.25
- PostgreSQL with sqlc + pgx/v5
- Gin web framework

## Investigation

### Attempt 1: Rely on parameterized queries
**Result:** Did not help - LIKE patterns are still interpreted even in parameterized queries

### Root Cause

PostgreSQL LIKE queries interpret `%` (any characters) and `_` (single character) as wildcards. Even with parameterized queries (which prevent SQL injection), the LIKE pattern interpretation remains active.

User searching for `50% discount` would match any addon containing `50` followed by anything followed by ` discount`.

## Solution

### Code Fix

Added `escapeLikePattern` helper function in `internal/api/handlers.go`:

```go
// escapeLikePattern escapes LIKE wildcards (% and _) to prevent pattern-based DoS.
// PostgreSQL uses backslash as the default escape character.
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\") // Escape backslashes first
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
```

Applied in search handler:

```go
if search != "" {
	// Escape LIKE wildcards to prevent pattern-based DoS
	escapedSearch := escapeLikePattern(search)
	searchText := pgtype.Text{String: escapedSearch, Valid: true}
	// ... use escapedSearch in query
}
```

### Tests Added

```go
func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no special chars", "hello world", "hello world"},
		{"percent sign", "100% complete", "100\\% complete"},
		{"underscore", "test_addon", "test\\_addon"},
		{"multiple wildcards", "50%_test", "50\\%\\_test"},
		{"backslash", "path\\to\\file", "path\\\\to\\\\file"},
		{"all special chars", "50%\\test_", "50\\%\\\\test\\_"},
		{"empty string", "", ""},
		{"only wildcards", "%_", "\\%\\_"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeLikePattern(tt.input)
			if result != tt.expected {
				t.Errorf("escapeLikePattern(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
```

## Prevention

### How to Avoid in Future

1. **Always escape LIKE input**: Any user-provided string used in LIKE queries needs wildcard escaping
2. **Consider full-text search**: For complex search needs, PostgreSQL full-text search (`tsvector`) is more robust
3. **Code review checklist**: Add "LIKE pattern escaping" to security review checklist

### Detection

- Grep for LIKE queries with user input: `grep -r "ILIKE\|LIKE" sql/`
- Review search endpoints for unescaped input

## Verification

```bash
# Run tests
go test ./internal/api/... -v -run TestEscapeLikePattern

# Manual verification
curl "http://localhost:8080/api/v1/addons?search=50%25_test"
# Should treat % and _ as literal characters, not wildcards
```

## Related

- PR #13: Basic Polish (SEO + Category Filter + Security fixes)
- Also added: Category filtering with GIN index, canonical URLs, og:url meta tags
