package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vyro/backend/internal/models"
)

// TestPostgresStoreV04 covers the v0.4 store methods against real
// PostgreSQL. It runs only when VYRO_TEST_DATABASE_URL is set; in this
// environment PostgreSQL is not installed, so it is expected to skip.
func TestPostgresStoreV04(t *testing.T) {
	dsn := os.Getenv("VYRO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VYRO_TEST_DATABASE_URL not set; skipping Postgres v0.4 integration test")
	}
	ctx := context.Background()
	s, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer s.Close()
	_, _ = s.pool.Exec(ctx, `TRUNCATE users, chats, messages, reactions, blocks, message_receipts, chat_settings, scheduled_messages, bots, identity_keys, one_time_prekeys, stories CASCADE`)
	defer s.pool.Exec(ctx, `TRUNCATE users, chats, messages, reactions, blocks, message_receipts, chat_settings, scheduled_messages, bots, identity_keys, one_time_prekeys, stories CASCADE`)

	a := &models.User{Email: "v4-a@example.com", Username: "v4alice", DisplayName: "Alice", PasswordHash: "hash"}
	if err := s.CreateUser(a); err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	b := &models.User{Email: "v4-b@example.com", Username: "v4bob", DisplayName: "Bob", PasswordHash: "hash"}
	if err := s.CreateUser(b); err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}
	// The handler always sets OwnerID to the creating user; the store
	// grants the "owner" role only to that member.
	chat := &models.Chat{Type: "dm", OwnerID: a.ID, MemberIDs: []string{a.ID, b.ID}}
	if err := s.CreateChat(chat); err != nil {
		t.Fatalf("CreateChat: %v", err)
	}

	// Location message persists with lat/lng.
	lat, lng := 35.6892, 51.3890
	m := &models.Message{ChatID: chat.ID, SenderID: a.ID, Content: "here", Kind: "location", Lat: &lat, Lng: &lng, LocationLabel: "Tehran"}
	if err := s.CreateMessage(m); err != nil {
		t.Fatalf("CreateMessage location: %v", err)
	}
	got, err := s.MessageByID(m.ID)
	if err != nil || got.Lat == nil || *got.Lat != lat || got.LocationLabel != "Tehran" {
		t.Fatalf("location round-trip: %+v, %v", got, err)
	}

	// Receipts: delivered then read.
	if err := s.MarkDelivered(m.ID, b.ID); err != nil {
		t.Fatalf("MarkDelivered: %v", err)
	}
	if err := s.MarkRead(m.ID, b.ID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	got, _ = s.MessageByID(m.ID)
	if got.Status != "read" || len(got.ReadBy) != 1 || got.ReadBy[0] != b.ID {
		t.Fatalf("receipts: status=%q read_by=%v", got.Status, got.ReadBy)
	}

	// Roles: owner (creator) can promote.
	if role := s.MemberRole(chat.ID, a.ID); role != "owner" {
		t.Fatalf("creator role = %q; want owner", role)
	}
	if err := s.SetMemberRole(chat.ID, b.ID, "admin"); err != nil {
		t.Fatalf("SetMemberRole: %v", err)
	}
	if role := s.MemberRole(chat.ID, b.ID); role != "admin" {
		t.Fatalf("role after promote = %q, want admin", role)
	}

	// Chat settings: pin + folder.
	if err := s.SetChatSettings(models.ChatSettings{ChatID: chat.ID, UserID: a.ID, Pinned: true, Folder: "work"}); err != nil {
		t.Fatalf("SetChatSettings: %v", err)
	}
	cs := s.GetChatSettings(chat.ID, a.ID)
	if !cs.Pinned || cs.Folder != "work" {
		t.Fatalf("chat settings = %+v", cs)
	}

	// Stories with 24h expiry.
	st := &models.Story{UserID: a.ID, MediaURL: "/uploads/s.jpg", Caption: "hi", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}
	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	if stories := s.ListStories(); len(stories) != 1 {
		t.Fatalf("ListStories = %d, want 1", len(stories))
	}

	// Scheduled message, due now, then marked sent.
	sm := &models.ScheduledMessage{ChatID: chat.ID, SenderID: a.ID, Content: "later", Kind: "text", SendAt: time.Now().UTC().Add(-time.Second)}
	if err := s.CreateScheduled(sm); err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	if due := s.DueScheduled(time.Now().UTC()); len(due) != 1 {
		t.Fatalf("DueScheduled = %d, want 1", len(due))
	}
	if err := s.MarkScheduledSent(sm.ID, ""); err != nil {
		t.Fatalf("MarkScheduledSent: %v", err)
	}

	// Expired message is purged.
	past := time.Now().UTC().Add(-time.Hour)
	exp := &models.Message{ChatID: chat.ID, SenderID: a.ID, Content: "old", Kind: "text", ExpiresAt: &past}
	if err := s.CreateMessage(exp); err != nil {
		t.Fatalf("CreateMessage expired: %v", err)
	}
	if n := s.PurgeExpired(time.Now().UTC()); n != 1 {
		t.Fatalf("PurgeExpired = %d, want 1", n)
	}

	// E2EE foundation: identity key + one-time pre-key claimed exactly once.
	ik := &models.IdentityKey{UserID: a.ID, DeviceID: "phone", IdentityKey: "aGVsbG8=", SignedPreKey: "aGVsbG8="}
	if err := s.UploadIdentityKey(ik); err != nil {
		t.Fatalf("UploadIdentityKey: %v", err)
	}
	if _, err := s.GetIdentityKey(a.ID, "phone"); err != nil {
		t.Fatalf("GetIdentityKey: %v", err)
	}
	pk := &models.OneTimePreKey{UserID: a.ID, DeviceID: "phone", KeyID: 1, PublicKey: "aGVsbG8="}
	if err := s.UploadOneTimePreKeys(a.ID, "phone", []*models.OneTimePreKey{pk}); err != nil {
		t.Fatalf("UploadOneTimePreKeys: %v", err)
	}
	if _, err := s.FetchOneTimePreKey(a.ID); err != nil {
		t.Fatalf("FetchOneTimePreKey: %v", err)
	}
	if _, err := s.FetchOneTimePreKey(a.ID); err != ErrNotFound {
		t.Fatalf("second fetch should be ErrNotFound, got %v", err)
	}

	// Bots: token stored only as a hash, retrievable by token.
	bot := &models.Bot{OwnerID: a.ID, Username: "echo", Name: "Echo", Token: "test-token-123"}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	byToken, err := s.BotByToken("test-token-123")
	if err != nil || byToken.Username != "echo" {
		t.Fatalf("BotByToken: %+v, %v", byToken, err)
	}
	if _, err := s.BotByToken("wrong"); err != ErrNotFound {
		t.Fatalf("BotByToken wrong token should be ErrNotFound, got %v", err)
	}
}
