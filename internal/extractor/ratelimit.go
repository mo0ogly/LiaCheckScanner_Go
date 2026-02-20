package extractor

import (
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
