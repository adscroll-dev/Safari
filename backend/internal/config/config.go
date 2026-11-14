// Package config reads server configuration from the environment.
// No secrets are committed; JWT_SECRET and DATABASE_URL must be supplied
// via environment (see .env.example). Defaults are safe for local dev only.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         string
	JWTSecret    string
	JWTExpiry    time.Duration
	DatabaseURL  string // empty = in-memory store (dev). Set to enable PostgreSQL later.
	RedisAddr    string // empty = no Redis. Set to enable Redis-backed rate limiting later.
	RedisURL     string // REDIS_URL (redis://...); falls back to REDIS_ADDR.
	UploadDir    string
	MaxUploadMB  int64
	AllowOrigins string
	// Web Push (VAPID, RFC 8292). Empty keys = push disabled; main
	// generates an ephemeral pair and warns when none are configured.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
	// SMTP for verification-code email (v0.7). Empty host/user = not
	// configured; the server then returns 503 from the endpoints that
	// must send a code, and never fakes one.
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	SMTPAllowNoAuth bool
}

func Load() Config {
	expiry, _ := strconv.Atoi(getenv("JWT_EXPIRY_HOURS", "24"))
	maxMB, _ := strconv.ParseInt(getenv("MAX_UPLOAD_MB", "50"), 10, 64)
	smtpPort, _ := strconv.Atoi(getenv("SMTP_PORT", "587"))
	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = os.Getenv("SMTP_USER") // From defaults to the SMTP user
	}
	allowNoAuth, _ := strconv.ParseBool(os.Getenv("SMTP_ALLOW_NOAUTH"))
	return Config{
		Port:            getenv("PORT", "8080"),
		JWTSecret:       getenv("JWT_SECRET", "vyro-dev-secret-change-me-in-production"),
		JWTExpiry:       time.Duration(expiry) * time.Hour,
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisAddr:       os.Getenv("REDIS_ADDR"),
		RedisURL:        redisURL(),
		UploadDir:       getenv("UPLOAD_DIR", "./uploads"),
		MaxUploadMB:     maxMB,
		AllowOrigins:    getenv("CORS_ALLOW_ORIGINS", "http://localhost:3000"),
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:    getenv("VAPID_SUBJECT", "mailto:vyro@example.invalid"),
		SMTPHost:        getenv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:        smtpPort,
		SMTPUser:        os.Getenv("SMTP_USER"),
		SMTPPass:        os.Getenv("SMTP_PASS"),
		SMTPFrom:        smtpFrom,
		SMTPAllowNoAuth: allowNoAuth,
	}
}

// redisURL prefers REDIS_URL, then REDIS_ADDR (bare host:port is accepted
// by the Redis client too).
func redisURL() string {
	if v := os.Getenv("REDIS_URL"); v != "" {
		return v
	}
	return os.Getenv("REDIS_ADDR")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
