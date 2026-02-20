package extractor

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter throttles outgoing requests to a configurable rate.
type RateLimiter struct {
	interval time.Duration
	last     time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a RateLimiter that allows at most requestsPerSecond
// requests per second.  A value <= 0 disables throttling (Wait returns
// immediately).
func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
	var interval time.Duration
	if requestsPerSecond > 0 {
		interval = time.Duration(float64(time.Second) / requestsPerSecond)
	}
	return &RateLimiter{
		interval: interval,
	}
}

// Wait blocks until enough time has elapsed since the last call so that the
// configured rate is respected.
func (r *RateLimiter) Wait() {
	if r.interval <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if !r.last.IsZero() {
		elapsed := now.Sub(r.last)
		if elapsed < r.interval {
			time.Sleep(r.interval - elapsed)
		}
	}
	r.last = time.Now()
}

const (
	retryMaxAttempts = 3
	retryBaseDelay   = 500 * time.Millisecond
	retryMaxDelay    = 10 * time.Second
)

// httpGetWithRetry performs an HTTP GET with exponential backoff retry.
// It retries on network errors, HTTP 429 (Too Many Requests), and HTTP 5xx.
// On 429 responses, it respects the Retry-After header if present.
func (e *Extractor) httpGetWithRetry(url string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= retryMaxAttempts; attempt++ {
		resp, err := e.apiClient.Get(url)
		if err != nil {
			lastErr = err
			if attempt < retryMaxAttempts {
				time.Sleep(retryDelay(attempt))
			}
			continue
		}

		// Success range or client error (except 429): return as-is.
		if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// 429: respect Retry-After header if present.
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP 429 Too Many Requests")
			if attempt < retryMaxAttempts {
				delay := retryAfterDelay(resp)
				if delay <= 0 {
					delay = retryDelay(attempt)
				}
				time.Sleep(delay)
			}
			continue
		}

		// 5xx: retry with backoff.
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		if attempt < retryMaxAttempts {
			time.Sleep(retryDelay(attempt))
		}
	}
	return nil, fmt.Errorf("after %d retries: %w", retryMaxAttempts, lastErr)
}

// retryDelay returns the backoff delay for the given attempt:
// min(baseDelay * 2^attempt, maxDelay) + random jitter (0-25%).
func retryDelay(attempt int) time.Duration {
	delay := float64(retryBaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(retryMaxDelay) {
		delay = float64(retryMaxDelay)
	}
	jitter := delay * 0.25 * rand.Float64()
	return time.Duration(delay + jitter)
}

// retryAfterDelay parses the Retry-After header from a 429 response.
// Returns the delay duration, or 0 if the header is absent or unparseable.
func retryAfterDelay(resp *http.Response) time.Duration {
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return 0
	}
	if secs, err := strconv.Atoi(ra); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(ra); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
