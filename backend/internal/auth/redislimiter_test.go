package auth

import (
	"os"
	"testing"
	"time"
)

// TestRedisLimiterUnreachable: with no Redis running, NewRedisLimiter must
// fail fast (3s PING timeout) so the server can fall back to the in-memory
// limiter instead of hanging or crashing. The in-memory limiter itself
// keeps working.
func TestRedisLimiterUnreachable(t *testing.T) {
	_, err := NewRedisLimiter("redis://127.0.0.1:6399", 5, time.Minute)
	if err == nil {
		t.Fatal("NewRedisLimiter against a dead Redis should return an error")
	}
	// The fallback path is exercised by the in-memory limiter directly.
	lim := NewRateLimiter(2, time.Minute)
	if !lim.Allow("k") || !lim.Allow("k") || lim.Allow("k") {
		t.Fatal("in-memory limiter should allow 2 then block")
	}
}

// TestRedisLimiterLive runs only when a real Redis is reachable via
// VYRO_TEST_REDIS_URL. In this environment Redis is not installed, so
// this test is expected to skip; it must never be faked.
func TestRedisLimiterLive(t *testing.T) {
	url := os.Getenv("VYRO_TEST_REDIS_URL")
	if url == "" {
		t.Skip("VYRO_TEST_REDIS_URL not set; skipping live Redis test (no Redis in this environment)")
	}
	rl, err := NewRedisLimiter(url, 2, time.Minute)
	if err != nil {
		t.Fatalf("NewRedisLimiter: %v", err)
	}
	defer rl.client.Close()
	key := "test-live-key"
	rl.Reset(key)
	if !rl.Allow(key) || !rl.Allow(key) || rl.Allow(key) {
		t.Fatal("Redis limiter should allow 2 then block")
	}
	rl.Reset(key)
	if !rl.Allow(key) {
		t.Fatal("after Reset, Redis limiter should allow again")
	}
}
