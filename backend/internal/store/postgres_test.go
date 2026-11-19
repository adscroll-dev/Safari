package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// TestMigrationMatchesFile guards the embedded copy of the migration
// (internal/store/migrations/001_init.sql) against drifting from the
// canonical file (backend/migrations/001_init.sql). No database needed.
func TestMigrationMatchesFile(t *testing.T) {
	embedded := migrationSQL
	canonical, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read canonical migration: %v", err)
	}
	if strings.TrimSpace(embedded) != strings.TrimSpace(string(canonical)) {
		t.Fatal("embedded migration differs from backend/migrations/001_init.sql — copy the canonical file into internal/store/migrations/")
	}
	embedded002 := migration002SQL
	canonical002, err := os.ReadFile("../../migrations/002_uploads.sql")
	if err != nil {
		t.Fatalf("read canonical migration 002: %v", err)
	}
	if strings.TrimSpace(embedded002) != strings.TrimSpace(string(canonical002)) {
		t.Fatal("embedded migration 002 differs from backend/migrations/002_uploads.sql — copy the canonical file into internal/store/migrations/")
	}
	if strings.TrimSpace(migration003SQL) != strings.TrimSpace(mustRead(t, "../../migrations/003_v04.sql")) {
		t.Fatal("embedded migration 003 differs from backend/migrations/003_v04.sql — copy the canonical file into internal/store/migrations/")
	}
	if strings.TrimSpace(migration004SQL) != strings.TrimSpace(mustRead(t, "../../migrations/004_v05.sql")) {
		t.Fatal("embedded migration 004 differs from backend/migrations/004_v05.sql — copy the canonical file into internal/store/migrations/")
	}
	if strings.TrimSpace(migration005SQL) != strings.TrimSpace(mustRead(t, "../../migrations/005_v06.sql")) {
		t.Fatal("embedded migration 005 differs from backend/migrations/005_v06.sql — copy the canonical file into internal/store/migrations/")
	}
	if strings.TrimSpace(migration006SQL) != strings.TrimSpace(mustRead(t, "../../migrations/006_verification.sql")) {
		t.Fatal("embedded migration 006 differs from backend/migrations/006_verification.sql — copy the canonical file into internal/store/migrations/")
	}
	if strings.TrimSpace(migration007SQL) != strings.TrimSpace(mustRead(t, "../../migrations/007_forward.sql")) {
		t.Fatal("embedded migration 007 differs from backend/migrations/007_forward.sql — copy the canonical file into internal/store/migrations/")
	}
	if strings.TrimSpace(migration008SQL) != strings.TrimSpace(mustRead(t, "../../migrations/008_fcm.sql")) {
		t.Fatal("embedded migration 008 differs from backend/migrations/008_fcm.sql — copy the canonical file into internal/store/migrations/")
	}
	canonical019, err := os.ReadFile("../../migrations/019_status_exclusions.sql")
	if err != nil {
		t.Fatalf("read canonical migration 019: %v", err)
	}
	embedded019, err := os.ReadFile("migrations/019_status_exclusions.sql")
	if err != nil {
		t.Fatalf("read embedded migration 019: %v", err)
	}
	if string(embedded019) != string(canonical019) || strings.TrimSpace(migration019SQL) != strings.TrimSpace(string(canonical019)) {
		t.Fatal("migration 019 differs between backend/migrations and internal/store/migrations")
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestPostgresStore is an integration test. It runs only when
// VYRO_TEST_DATABASE_URL is set, e.g.:
//
//	VYRO_TEST_DATABASE_URL="postgres://vyro:vyro@localhost:5432/vyro_test?sslmode=disable" go test ./internal/store/ -run TestPostgresStore -v
//
// The test database must exist; the migration is applied automatically by
// NewPostgresStore. Tables are truncated before and after the test, so use
// a throwaway database, never a production one.
func TestPostgresStore(t *testing.T) {
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
	truncate := func() {
		_, _ = s.pool.Exec(ctx, `TRUNCATE users, chats, messages, reactions, blocks CASCADE`)
	}
	truncate()
	defer truncate()

	a := &models.User{Email: "pg-a@example.com", Username: "pgalice", DisplayName: "Alice", PasswordHash: "hash"}
	if err := s.CreateUser(a); err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	b := &models.User{Email: "pg-b@example.com", Username: "pgbob", DisplayName: "Bob", PasswordHash: "hash"}
	if err := s.CreateUser(b); err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}
	// Duplicates, case-insensitive, like MemoryStore.
	if err := s.CreateUser(&models.User{Email: "PG-A@EXAMPLE.COM", Username: "other"}); err != ErrAlreadyExists {
		t.Fatalf("duplicate email: expected ErrAlreadyExists, got %v", err)
	}
	if got, err := s.UserByEmail("PG-A@EXAMPLE.COM"); err != nil || got.ID != a.ID {
		t.Fatalf("case-insensitive email lookup: got %+v, err %v", got, err)
	}

	chat := &models.Chat{Type: models.ChatDM, OwnerID: a.ID, MemberIDs: []string{a.ID, b.ID}}
	if err := s.CreateChat(chat); err != nil {
		t.Fatalf("CreateChat: %v", err)
	}
	if !s.IsMember(chat.ID, a.ID) || s.IsMember(chat.ID, "00000000-0000-0000-0000-000000000000") {
		t.Fatal("IsMember mismatch")
	}
	m1 := &models.Message{ChatID: chat.ID, SenderID: a.ID, Content: "hello bob", Kind: "text"}
	if err := s.CreateMessage(m1); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	m2 := &models.Message{ChatID: chat.ID, SenderID: b.ID, Content: "hello alice", Kind: "text"}
	if err := s.CreateMessage(m2); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	// v0.3: a file message keeps its attachment metadata in Postgres.
	m3 := &models.Message{ChatID: chat.ID, SenderID: a.ID, Content: "عکس", Kind: "image",
		FileURL: "/uploads/test-123.png", FileName: "عکس.png", FileSize: 1234, MimeType: "image/png"}
	if err := s.CreateMessage(m3); err != nil {
		t.Fatalf("CreateMessage file: %v", err)
	}
	msgs := s.MessagesForChat(chat.ID, 50, 0)
	if len(msgs) != 3 || msgs[0].Content != "hello bob" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
	if msgs[2].FileURL != "/uploads/test-123.png" || msgs[2].FileName != "عکس.png" ||
		msgs[2].FileSize != 1234 || msgs[2].MimeType != "image/png" {
		t.Fatalf("file metadata lost: %+v", msgs[2])
	}
	if err := s.AddReaction(m1.ID, b.ID, "❤️"); err != nil {
		t.Fatalf("AddReaction: %v", err)
	}
	if err := s.AddReaction(m1.ID, b.ID, "❤️"); err != nil {
		t.Fatalf("duplicate reaction should be idempotent: %v", err)
	}
	got, _ := s.MessageByID(m1.ID)
	if len(got.Reactions["❤️"]) != 1 {
		t.Fatalf("expected 1 reaction, got %+v", got.Reactions)
	}
	m1.Content = "hello bob (edited)"
	if err := s.UpdateMessage(m1); err != nil {
		t.Fatalf("UpdateMessage: %v", err)
	}
	got, _ = s.MessageByID(m1.ID)
	if got.EditedAt == nil || got.Content != "hello bob (edited)" {
		t.Fatalf("edit not applied: %+v", got)
	}
	if err := s.DeleteMessage(m2.ID); err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}
	if msgs := s.MessagesForChat(chat.ID, 50, 0); len(msgs) != 2 {
		t.Fatalf("deleted message should be hidden, got %d", len(msgs))
	}
	if found := s.SearchMessages(a.ID, "hello bob", 10); len(found) != 1 {
		t.Fatalf("search expected 1 result, got %d", len(found))
	}
	if found := s.SearchMessages("00000000-0000-0000-0000-000000000000", "hello", 10); len(found) != 0 {
		t.Fatalf("search for non-member should return 0, got %d", len(found))
	}
	if err := s.BlockUser(a.ID, b.ID); err != nil {
		t.Fatalf("BlockUser: %v", err)
	}
	if !s.IsBlocked(a.ID, b.ID) || !s.IsBlocked(b.ID, a.ID) {
		t.Fatal("IsBlocked should be symmetric for the check")
	}

	// Persistence: a fresh store on the same database still sees the data.
	s2, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("reopen PostgresStore: %v", err)
	}
	defer s2.Close()
	if _, err := s2.UserByID(a.ID); err != nil {
		t.Fatalf("user lost after reopen: %v", err)
	}
	if msgs := s2.MessagesForChat(chat.ID, 50, 0); len(msgs) != 2 || msgs[0].Content != "hello bob (edited)" ||
		msgs[1].FileURL != "/uploads/test-123.png" || msgs[1].FileSize != 1234 {
		t.Fatalf("messages lost after reopen: %+v", msgs)
	}
}

// TestPostgresStoreForwardedMessage (v0.7): a forwarded message keeps
// its attribution through Postgres (migration 007 columns) and reads
// back the same via MessageByID and MessagesForChat.
func TestPostgresStoreForwardedMessage(t *testing.T) {
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
	_, _ = s.pool.Exec(ctx, `TRUNCATE users, chats, messages, reactions, blocks CASCADE`)
	defer func() {
		_, _ = s.pool.Exec(ctx, `TRUNCATE users, chats, messages, reactions, blocks CASCADE`)
	}()

	a := &models.User{Email: "fwd-a@example.com", Username: "fwdalice", DisplayName: "Alice", PasswordHash: "hash"}
	if err := s.CreateUser(a); err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	b := &models.User{Email: "fwd-b@example.com", Username: "fwdbob", DisplayName: "Bob", PasswordHash: "hash"}
	if err := s.CreateUser(b); err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}
	chat := &models.Chat{Type: models.ChatDM, OwnerID: a.ID, MemberIDs: []string{a.ID, b.ID}}
	if err := s.CreateChat(chat); err != nil {
		t.Fatalf("CreateChat: %v", err)
	}
	m := &models.Message{
		ChatID: chat.ID, SenderID: b.ID, Content: "hello again", Kind: "text",
		ForwardedFrom: "Alice", ForwardedFromID: a.ID,
	}
	if err := s.CreateMessage(m); err != nil {
		t.Fatalf("CreateMessage forwarded: %v", err)
	}
	got, err := s.MessageByID(m.ID)
	if err != nil {
		t.Fatalf("MessageByID: %v", err)
	}
	if got.ForwardedFrom != "Alice" || got.ForwardedFromID != a.ID {
		t.Fatalf("forwarded attribution lost: %+v", got)
	}
	msgs := s.MessagesForChat(chat.ID, 50, 0)
	if len(msgs) != 1 || msgs[0].ForwardedFrom != "Alice" {
		t.Fatalf("forwarded attribution lost in list: %+v", msgs)
	}
	// A plain message has empty forwarding fields (not nulls leaking).
	plain := &models.Message{ChatID: chat.ID, SenderID: a.ID, Content: "plain", Kind: "text"}
	if err := s.CreateMessage(plain); err != nil {
		t.Fatalf("CreateMessage plain: %v", err)
	}
	gotPlain, err := s.MessageByID(plain.ID)
	if err != nil {
		t.Fatalf("MessageByID plain: %v", err)
	}
	if gotPlain.ForwardedFrom != "" || gotPlain.ForwardedFromID != "" {
		t.Fatalf("plain message gained forwarding fields: %+v", gotPlain)
	}
}
