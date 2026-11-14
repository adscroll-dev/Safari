package auth

import (
	"sync"
	"time"
)

// RateLimiter is a simple in-memory fixed-window limiter, keyed by a string
// (e.g. client IP + email). It is sufficient for the dev build and for
// protecting /api/auth/login from brute force. In production this should be
// backed by Redis (see docs/ARCHITECTURE.md) so limits are shared across
// instances.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a limiter allowing `limit` attempts per `window`.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{attempts: make(map[string][]time.Time), limit: limit, window: window}
}

// Allow reports whether a new attempt for key is permitted, and records it.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-r.window)
	kept := r.attempts[key][:0]
	for _, t := range r.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= r.limit {
		r.attempts[key] = kept
		return false
	}
	r.attempts[key] = append(kept, now)
	return true
}

// Reset clears the history for a key (e.g. after a successful login).
func (r *RateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.attempts, key)
}
