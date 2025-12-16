# Fix: CurseForge API Timeout During Sync

## Overview

The sync job fails with "context deadline exceeded" when fetching page 3900 from CurseForge API. The HTTP client has a hardcoded 30-second timeout with no retry logic.

## Problem Statement

```
time=2025-12-16T09:02:46.484Z level=ERROR msg="sync failed"
error="...context deadline exceeded (Client.Timeout exceeded while awaiting headers)"
```

**Root Cause:** Hardcoded 30s timeout at `internal/curseforge/client.go:27` with no retry logic.

## Solution

The simplest fix that could possibly work:

1. Increase timeout from 30s to 60s
2. Add inline retry logic with exponential backoff (3 retries)
3. Ship it and monitor

**No new files. No new dependencies. No configuration infrastructure.**

## Implementation

### Single File Change: `internal/curseforge/client.go`

#### Step 1: Increase Timeout

```go
// Line 27: Change this
Timeout: 30 * time.Second,

// To this
Timeout: 60 * time.Second,
```

#### Step 2: Add Retry Logic to doRequest

Replace the current `doRequest` method with a version that retries:

```go
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	const maxRetries = 3

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			slog.Warn("retrying request",
				"attempt", attempt,
				"maxRetries", maxRetries,
				"backoff", backoff,
				"path", path,
				"error", lastErr,
			)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		body, err := c.doRequestOnce(ctx, method, path, query)
		if err == nil {
			return body, nil
		}

		lastErr = err

		// Don't retry client errors (4xx) except rate limits (429)
		if isClientError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) doRequestOnce(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	return body, nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

func isClientError(err error) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		// Client errors (4xx) except rate limits (429)
		return httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && httpErr.StatusCode != 429
	}
	return false
}
```

### Required Import Addition

Add `errors` to imports:

```go
import (
	// existing imports...
	"errors"
)
```

## Acceptance Criteria

- [ ] Timeout increased from 30s to 60s
- [ ] Requests retry up to 3 times on timeout/server errors
- [ ] Retry attempts are logged with backoff duration
- [ ] Client errors (4xx except 429) fail immediately without retry
- [ ] Context cancellation stops retries immediately
- [ ] Existing tests pass

## Test Cases

```go
func TestDoRequest_RetriesOnTimeout(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test",
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    server.URL,
	}

	body, err := client.doRequest(context.Background(), "GET", "/test", nil)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.Contains(t, string(body), "data")
}

func TestDoRequest_NoRetryOn400(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test",
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    server.URL,
	}

	_, err := client.doRequest(context.Background(), "GET", "/test", nil)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // No retries for client errors
}

func TestDoRequest_GivesUpAfterMaxRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test",
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    server.URL,
	}

	_, err := client.doRequest(context.Background(), "GET", "/test", nil)

	assert.Error(t, err)
	assert.Equal(t, 4, attempts) // 1 initial + 3 retries
	assert.Contains(t, err.Error(), "failed after 3 retries")
}
```

## What We're NOT Doing (And Why)

| Feature | Why Not |
|---------|---------|
| Rate limiter | No evidence of hitting rate limits |
| Configurable timeout | Just pick a good default |
| Retry-After header parsing | Add later if we see 429s |
| Separate retry.go file | One place uses this |
| RetryConfig struct | Constants work fine |
| Transport-level tuning | Premature optimization |

## Rollback Plan

If issues occur: revert the commit. The change is small and isolated.

## Success Metrics

Monitor Railway logs after deployment:
- Sync should complete despite transient timeouts
- Retry log messages should appear occasionally (not constantly)
- If retries are constant, investigate deeper API issues

## Future Improvements (Only If Needed)

Add these **only if monitoring shows they're needed**:

1. Configurable timeout (if 60s proves wrong)
2. Rate limiter (if we start hitting 429s)
3. Retry-After header parsing (if CurseForge sends it)
4. Checkpoint/resume (if sync failures persist)

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)
