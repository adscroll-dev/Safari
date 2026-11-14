package auth

import (
	"strings"
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "correct-horse-battery" {
		t.Fatal("password stored in plaintext")
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Fatalf("expected bcrypt hash, got %q", hash)
	}
	if !CheckPassword("correct-horse-battery", hash) {
		t.Fatal("correct password did not verify")
	}
	if CheckPassword("wrong-password", hash) {
		t.Fatal("wrong password verified")
	}
}

func TestHashPasswordTooShort(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	token, err := GenerateJWT("test-secret", "u1", "a@example.com", "alice", time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	claims, err := ValidateJWT("test-secret", token)
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}
	if claims.UserID != "u1" || claims.Email != "a@example.com" || claims.Username != "alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTWrongSecretRejected(t *testing.T) {
	token, _ := GenerateJWT("secret-a", "u1", "a@example.com", "alice", time.Hour)
	if _, err := ValidateJWT("secret-b", token); err == nil {
		t.Fatal("token signed with another secret was accepted")
	}
}

func TestJWTExpiredRejected(t *testing.T) {
	token, err := GenerateJWT("test-secret", "u1", "a@example.com", "alice", -time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	if _, err := ValidateJWT("test-secret", token); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !rl.Allow("1.2.3.4|a@example.com") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if rl.Allow("1.2.3.4|a@example.com") {
		t.Fatal("4th attempt within window should be blocked")
	}
	// A different key is unaffected.
	if !rl.Allow("5.6.7.8|a@example.com") {
		t.Fatal("different key should be allowed")
	}
	rl.Reset("1.2.3.4|a@example.com")
	if !rl.Allow("1.2.3.4|a@example.com") {
		t.Fatal("attempt after Reset should be allowed")
	}
}
