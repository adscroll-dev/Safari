package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vyro/backend/internal/models"
)

// TestPostgresStoreVerification exercises the v0.7 email-code and
// pending-user store methods against real PostgreSQL. It runs only
// when VYRO_TEST_DATABASE_URL is set (see TestPostgresStore).
func TestPostgresStoreVerification(t *testing.T) {
	dsn := os.Getenv("VYRO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VYRO_TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	ctx := context.Background()
	s, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer s.Close()
	_, _ = s.pool.Exec(ctx, `TRUNCATE email_codes, pending_users CASCADE`)
	defer func() { _, _ = s.pool.Exec(ctx, `TRUNCATE email_codes, pending_users CASCADE`) }()

	email := "pg-verify@example.com"
	now := time.Now().UTC()
	if err := s.SavePendingUser(&models.PendingUser{
		Email: email, Username: "pgverify", DisplayName: "PG Verify",
		PasswordHash: "hash", ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("SavePendingUser: %v", err)
	}
	p, err := s.GetPendingUser(email)
	if err != nil || p.Username != "pgverify" {
		t.Fatalf("GetPendingUser: %+v, %v", p, err)
	}
	// Case-insensitive email lookup, like the users table.
	if _, err := s.GetPendingUser("PG-VERIFY@EXAMPLE.COM"); err != nil {
		t.Fatalf("case-insensitive pending lookup: %v", err)
	}
	if err := s.SaveEmailCode(&models.EmailCode{
		Email: email, Purpose: "register", CodeHash: "deadbeef",
		ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("SaveEmailCode: %v", err)
	}
	row, err := s.GetEmailCode(email, "register")
	if err != nil || row.CodeHash != "deadbeef" || row.Attempts != 0 {
		t.Fatalf("GetEmailCode: %+v, %v", row, err)
	}
	// A resend replaces the row and resets attempts.
	if err := s.SaveEmailCode(&models.EmailCode{
		Email: email, Purpose: "register", CodeHash: "cafebabe",
		ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("SaveEmailCode resend: %v", err)
	}
	row, _ = s.GetEmailCode(email, "register")
	if row.CodeHash != "cafebabe" {
		t.Fatalf("resend must replace the code hash, got %q", row.CodeHash)
	}
	n, err := s.IncrementCodeAttempts(email, "register")
	if err != nil || n != 1 {
		t.Fatalf("IncrementCodeAttempts: %d, %v", n, err)
	}
	if err := s.ConsumeEmailCode(email, "register"); err != nil {
		t.Fatalf("ConsumeEmailCode: %v", err)
	}
	row, _ = s.GetEmailCode(email, "register")
	if row.ConsumedAt == nil {
		t.Fatal("consumed code must carry ConsumedAt")
	}
	if err := s.InvalidateEmailCode(email, "register"); err != nil {
		t.Fatalf("InvalidateEmailCode: %v", err)
	}
	if _, err := s.GetEmailCode(email, "register"); err != ErrNotFound {
		t.Fatalf("invalidated code: expected ErrNotFound, got %v", err)
	}
	if err := s.DeletePendingUser(email); err != nil {
		t.Fatalf("DeletePendingUser: %v", err)
	}
	if _, err := s.GetPendingUser(email); err != ErrNotFound {
		t.Fatalf("deleted pending user: expected ErrNotFound, got %v", err)
	}
}
