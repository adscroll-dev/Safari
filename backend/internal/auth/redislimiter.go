// Package auth — Redis-backed rate limiter (v0.4).
//
// RedisLimiter is a fixed-window limiter on top of Redis (go-redis v9,
// free/open-source). It is used only when REDIS_URL is set and Redis
// answers PING; otherwise the server keeps the in-memory RateLimiter
// (see cmd/server/main.go). Every Redis error fails open to the
// in-memory limiter for that key, and is logged once — a Redis outage
// must never lock every user out, and must never silently disable
// limiting either: the fallback is in-memory, per-process.
package auth

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter is the interface handlers depend on. Both RateLimiter
// (in-memory) and RedisLimiter implement it.
type Limiter interface {
	Allow(key string) bool
	Reset(key string)
}

// RedisLimiter implements Limiter with Redis INCR + EXPIRE.
type RedisLimiter struct {
	client   *redis.Client
	limit    int
	window   time.Duration
	fallback *RateLimiter
	once     sync.Once
}

// NewRedisLimiter dials addr (redis:// URL or host:port) and returns a
// limiter plus an error when Redis cannot be reached, so the caller can
// fall back to the in-memory limiter.
func NewRedisLimiter(redisURL string, limit int, window time.Duration) (*RedisLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// Accept a bare host:port as well.
		opts = &redis.Options{Addr: redisURL}
	}
	// Redis < 7.2 (e.g. Ubuntu's Redis 7.0) does not know CLIENT
	// SETINFO, which go-redis sends on connect by default; the
	// handshake error would make every connection fail. Disable it —
	// it is only client identification, not functionality.
	opts.DisableIdentity = true
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &RedisLimiter{
		client: client, limit: limit, window: window,
		fallback: NewRateLimiter(limit, window),
	}, nil
}

// Allow increments the counter for key and reports whether the attempt
// is within the limit. On any Redis error it warns once and uses the
// in-memory fallback for that key.
func (r *RedisLimiter) Allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	n, err := r.client.Incr(ctx, "vyro:ratelimit:"+key).Result()
	if err != nil {
		r.once.Do(func() { log.Printf("redis limiter error (%v); using in-memory fallback", err) })
		return r.fallback.Allow(key)
	}
	if n == 1 {
		_ = r.client.Expire(ctx, "vyro:ratelimit:"+key, r.window).Err()
	}
	return n <= int64(r.limit)
}

func (r *RedisLimiter) Reset(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = r.client.Del(ctx, "vyro:ratelimit:"+key).Err()
	r.fallback.Reset(key)
}

// Close releases the Redis connection.
func (r *RedisLimiter) Close() error { return r.client.Close() }
