package store

import (
	"context"
	"os"
	"testing"

	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/push"
)

// FCM tokens (Firebase Cloud Messaging): one token per (user, device),
// idempotent by token value. Schema: migrations/008_fcm.sql.
func TestMemoryStoreFCMTokens(t *testing.T) {
	s := NewMemoryStore()
	u := &models.User{Email: "fcm-a@example.com", Username: "fcmalice", DisplayName: "Alice", PasswordHash: "hash"}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	tok := push.FCMToken{Token: "fcm-token-1", DeviceID: "phone", Platform: "android"}
	if err := s.SaveFCMToken(u.ID, tok); err != nil {
		t.Fatalf("SaveFCMToken: %v", err)
	}
	// Re-registering the same token updates it instead of duplicating.
	if err := s.SaveFCMToken(u.ID, push.FCMToken{Token: "fcm-token-1", DeviceID: "phone", Platform: "ios"}); err != nil {
		t.Fatalf("SaveFCMToken re-register: %v", err)
	}
	if err := s.SaveFCMToken(u.ID, push.FCMToken{Token: "fcm-token-2", DeviceID: "tablet", Platform: "android"}); err != nil {
		t.Fatalf("SaveFCMToken second: %v", err)
	}
	got := s.ListFCMTokens(u.ID)
	if len(got) != 2 {
		t.Fatalf("ListFCMTokens = %d, want 2", len(got))
	}
	if got[0].Token != "fcm-token-1" || got[0].Platform != "ios" {
		t.Fatalf("re-register did not update token: %+v", got[0])
	}
	if err := s.DeleteFCMToken(u.ID, "fcm-token-1"); err != nil {
		t.Fatalf("DeleteFCMToken: %v", err)
	}
	if got := s.ListFCMTokens(u.ID); len(got) != 1 || got[0].Token != "fcm-token-2" {
		t.Fatalf("tokens after delete: %+v", got)
	}
	// Unknown user: saving must fail, listing must be empty.
	if err := s.SaveFCMToken("nobody", tok); err != ErrNotFound {
		t.Fatalf("SaveFCMToken unknown user: got %v, want ErrNotFound", err)
	}
	if got := s.ListFCMTokens("nobody"); len(got) != 0 {
		t.Fatalf("ListFCMTokens unknown user: %+v", got)
	}
}

// TestPostgresStoreFCMTokens: tokens survive a restart (reopen) through
// Postgres. Runs only when VYRO_TEST_DATABASE_URL is set.
func TestPostgresStoreFCMTokens(t *testing.T) {
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
	_, _ = s.pool.Exec(ctx, `TRUNCATE users, fcm_tokens CASCADE`)
	defer func() {
		_, _ = s.pool.Exec(ctx, `TRUNCATE users, fcm_tokens CASCADE`)
	}()

	u := &models.User{Email: "fcm-pg@example.com", Username: "fcmpg", DisplayName: "Pg", PasswordHash: "hash"}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	tok := push.FCMToken{Token: "fcm-pg-token", DeviceID: "phone", Platform: "android"}
	if err := s.SaveFCMToken(u.ID, tok); err != nil {
		t.Fatalf("SaveFCMToken: %v", err)
	}
	// Reopen: the token must still be there.
	s2, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("reopen PostgresStore: %v", err)
	}
	defer s2.Close()
	got := s2.ListFCMTokens(u.ID)
	if len(got) != 1 || got[0].Token != "fcm-pg-token" || got[0].DeviceID != "phone" {
		t.Fatalf("FCM token lost after reopen: %+v", got)
	}
	if err := s2.DeleteFCMToken(u.ID, "fcm-pg-token"); err != nil {
		t.Fatalf("DeleteFCMToken: %v", err)
	}
	if got := s2.ListFCMTokens(u.ID); len(got) != 0 {
		t.Fatalf("tokens after delete: %+v", got)
	}
}
