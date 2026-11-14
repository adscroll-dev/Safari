package store

import (
	"testing"

	"github.com/vyro/backend/internal/models"
)

func mustUser(t *testing.T, s *MemoryStore, email, username string) *models.User {
	t.Helper()
	u := &models.User{Email: email, Username: username, DisplayName: username, PasswordHash: "hash"}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser %s: %v", email, err)
	}
	return u
}

func TestCreateUserDuplicateEmailAndUsername(t *testing.T) {
	s := NewMemoryStore()
	mustUser(t, s, "a@example.com", "alice")
	if err := s.CreateUser(&models.User{Email: "a@example.com", Username: "alice2"}); err != ErrAlreadyExists {
		t.Fatalf("duplicate email: expected ErrAlreadyExists, got %v", err)
	}
	if err := s.CreateUser(&models.User{Email: "b@example.com", Username: "alice"}); err != ErrAlreadyExists {
		t.Fatalf("duplicate username: expected ErrAlreadyExists, got %v", err)
	}
	if _, err := s.UserByEmail("A@EXAMPLE.COM"); err != nil {
		t.Fatalf("email lookup should be case-insensitive: %v", err)
	}
}

func TestChatAndMessageFlow(t *testing.T) {
	s := NewMemoryStore()
	a := mustUser(t, s, "a@example.com", "alice")
	b := mustUser(t, s, "b@example.com", "bob")

	chat := &models.Chat{Type: models.ChatDM, MemberIDs: []string{a.ID, b.ID}}
	if err := s.CreateChat(chat); err != nil {
		t.Fatalf("CreateChat: %v", err)
	}
	if !s.IsMember(chat.ID, a.ID) || s.IsMember(chat.ID, "nobody") {
		t.Fatal("IsMember mismatch")
	}
	if got := s.ChatsForUser(a.ID); len(got) != 1 {
		t.Fatalf("expected 1 chat for alice, got %d", len(got))
	}

	m1 := &models.Message{ChatID: chat.ID, SenderID: a.ID, Content: "hello bob", Kind: "text"}
	if err := s.CreateMessage(m1); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	m2 := &models.Message{ChatID: chat.ID, SenderID: b.ID, Content: "hello alice", Kind: "text"}
	if err := s.CreateMessage(m2); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	msgs := s.MessagesForChat(chat.ID, 50, 0)
	if len(msgs) != 2 || msgs[0].Content != "hello bob" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}

	// Reaction, edit, delete.
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
	if msgs := s.MessagesForChat(chat.ID, 50, 0); len(msgs) != 1 {
		t.Fatalf("deleted message should be hidden, got %d", len(msgs))
	}

	// Search only finds messages in the user's chats.
	if found := s.SearchMessages(a.ID, "hello bob", 10); len(found) != 1 {
		t.Fatalf("search expected 1 result, got %d", len(found))
	}
	if found := s.SearchMessages("stranger", "hello", 10); len(found) != 0 {
		t.Fatalf("search for non-member should return 0, got %d", len(found))
	}
}

func TestBlockUser(t *testing.T) {
	s := NewMemoryStore()
	a := mustUser(t, s, "a@example.com", "alice")
	b := mustUser(t, s, "b@example.com", "bob")
	if s.IsBlocked(a.ID, b.ID) {
		t.Fatal("should not be blocked initially")
	}
	if err := s.BlockUser(a.ID, b.ID); err != nil {
		t.Fatalf("BlockUser: %v", err)
	}
	if !s.IsBlocked(a.ID, b.ID) || !s.IsBlocked(b.ID, a.ID) {
		t.Fatal("IsBlocked should be symmetric for the check")
	}
}
