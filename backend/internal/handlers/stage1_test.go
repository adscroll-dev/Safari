package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// Stage 1 chat gaps (WhatsApp parity): mute chat, pin message,
// contact share, delete-for-me / delete-for-everyone.
func TestStage1ChatGaps(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, _ := register(t, h, "carol@example.com", "carol")
	chatID := makeDM(t, h, aliceTok, bobID)

	listChats := func(tok string) []models.Chat {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/chats", nil, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("list chats: %d %s", rec.Code, rec.Body.String())
		}
		var chats []models.Chat
		mustDecode(t, rec, &chats)
		return chats
	}
	findChat := func(chats []models.Chat, id string) models.Chat {
		t.Helper()
		for _, c := range chats {
			if c.ID == id {
				return c
			}
		}
		t.Fatalf("chat %s not in list", id)
		return models.Chat{}
	}
	listMsgs := func(tok, cid string) []models.Message {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/chats/"+cid+"/messages", nil, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("list messages: %d %s", rec.Code, rec.Body.String())
		}
		var msgs []models.Message
		mustDecode(t, rec, &msgs)
		return msgs
	}

	// --- Mute chat ---
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/mute", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("mute: %d %s", rec.Code, rec.Body.String())
	}
	if c := findChat(listChats(aliceTok), chatID); !c.Muted {
		t.Fatalf("chat muted = false, want true")
	}
	// Mute is per-user: Bob is unaffected.
	if c := findChat(listChats(bobTok), chatID); c.Muted {
		t.Fatalf("bob chat muted = true, want false")
	}
	// A non-member cannot mute.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/mute", nil, carolTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member mute: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/unmute", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unmute: %d %s", rec.Code, rec.Body.String())
	}
	if c := findChat(listChats(aliceTok), chatID); c.Muted {
		t.Fatalf("after unmute, chat muted = true, want false")
	}

	// --- Pin message ---
	m1 := sendMsg(t, h, aliceTok, chatID, map[string]interface{}{"content": "pin me"})
	// Pinning a message from another chat is rejected.
	otherChat := makeDM(t, h, aliceTok, bobID) // second DM id differs
	_ = otherChat
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+otherChat+"/pin-message", map[string]string{"message_id": m1.ID}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("pin foreign message: %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/pin-message", map[string]string{"message_id": m1.ID}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("pin: %d %s", rec.Code, rec.Body.String())
	}
	if c := findChat(listChats(aliceTok), chatID); c.PinnedMessage == nil || c.PinnedMessage.ID != m1.ID {
		t.Fatalf("pinned_message = %+v, want id %s", c.PinnedMessage, m1.ID)
	}
	// A non-member cannot pin.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/pin-message", map[string]string{"message_id": m1.ID}, carolTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member pin: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodDelete, "/api/chats/"+chatID+"/pin-message", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unpin: %d %s", rec.Code, rec.Body.String())
	}
	if c := findChat(listChats(aliceTok), chatID); c.PinnedMessage != nil {
		t.Fatalf("after unpin, pinned_message = %+v, want nil", c.PinnedMessage)
	}

	// --- Contact share ---
	contact := sendMsg(t, h, aliceTok, chatID, map[string]interface{}{
		"kind": "contact", "content": fmt.Sprintf(`{"user_id": %q}`, bobID),
	})
	if contact.Kind != "contact" {
		t.Fatalf("contact kind = %q, want contact", contact.Kind)
	}
	// Bad contact bodies are rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/messages", map[string]interface{}{
		"kind": "contact", "content": "not json",
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("contact bad json: %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/messages", map[string]interface{}{
		"kind": "contact", "content": `{"user_id": "00000000-0000-0000-0000-000000000000"}`,
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("contact unknown user: %d, want 400", rec.Code)
	}

	// --- Delete-for-me ---
	// Bob hides Alice's message only for himself.
	m2 := sendMsg(t, h, aliceTok, chatID, map[string]interface{}{"content": "hide me from bob"})
	rec = doJSON(t, h, http.MethodPost, "/api/messages/"+m2.ID+"/delete-for-me", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete-for-me: %d %s", rec.Code, rec.Body.String())
	}
	for _, m := range listMsgs(bobTok, chatID) {
		if m.ID == m2.ID {
			t.Fatalf("bob still sees message he deleted for himself")
		}
	}
	found := false
	for _, m := range listMsgs(aliceTok, chatID) {
		if m.ID == m2.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("alice no longer sees a message bob deleted for himself")
	}
	// A non-member cannot delete-for-me.
	rec = doJSON(t, h, http.MethodPost, "/api/messages/"+m2.ID+"/delete-for-me", nil, carolTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member delete-for-me: %d, want 403", rec.Code)
	}
	// The preview skips a message the user deleted for themself:
	// Bob's last message is the contact (m2 is hidden for him).
	if c := findChat(listChats(bobTok), chatID); c.LastMessage == nil || c.LastMessage.Kind != "contact" {
		t.Fatalf("bob preview = %+v, want the contact message, not the deleted one", c.LastMessage)
	}
	// Unread count also skips it: Bob has not read m1, the contact
	// or m2, but m2 is hidden for him, so only 2 count as unread.
	if c := findChat(listChats(bobTok), chatID); c.UnreadCount != 2 {
		t.Fatalf("bob unread = %d, want 2 (m1 + contact)", c.UnreadCount)
	}
	// --- Delete-for-everyone (existing hard delete) ---
	rec = doJSON(t, h, http.MethodDelete, "/api/messages/"+m1.ID, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete-for-everyone: %d %s", rec.Code, rec.Body.String())
	}
	for _, tok := range []string{aliceTok, bobTok} {
		for _, m := range listMsgs(tok, chatID) {
			if m.ID == m1.ID {
				t.Fatalf("message deleted for everyone still visible")
			}
		}
	}
	_ = aliceID
}
