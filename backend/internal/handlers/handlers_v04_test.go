package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vyro/backend/internal/models"
)

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode response (status %d, body %s): %v", rec.Code, rec.Body.String(), err)
	}
}

func makeDM(t *testing.T, h *Handler, tok, otherID string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{otherID},
	}, tok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create DM: status %d body %s", rec.Code, rec.Body.String())
	}
	var chat models.Chat
	mustDecode(t, rec, &chat)
	return chat.ID
}

func makeGroup(t *testing.T, h *Handler, tok string, memberIDs []string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "group", "title": "Test Group", "member_ids": memberIDs,
	}, tok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: status %d body %s", rec.Code, rec.Body.String())
	}
	var chat models.Chat
	mustDecode(t, rec, &chat)
	return chat.ID
}

func sendMsg(t *testing.T, h *Handler, tok, chatID string, body map[string]interface{}) *models.Message {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/messages", body, tok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send message: status %d body %s", rec.Code, rec.Body.String())
	}
	var m models.Message
	mustDecode(t, rec, &m)
	return &m
}

// TestV04Receipts: ticks — sent -> delivered (when Bob lists the chat)
// -> read (after Bob POSTs /read). Presence endpoint is exercised too.
func TestV04Receipts(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	chatID := makeDM(t, h, aliceTok, bobID)

	m := sendMsg(t, h, aliceTok, chatID, map[string]interface{}{"content": "سلام"})
	if m.Status != "sent" {
		t.Fatalf("new message status = %q, want sent", m.Status)
	}

	// Bob lists the chat: messages from others are marked delivered.
	rec := doJSON(t, h, http.MethodGet, "/api/chats/"+chatID+"/messages", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list messages: %d %s", rec.Code, rec.Body.String())
	}
	var msgs []*models.Message
	mustDecode(t, rec, &msgs)
	if len(msgs) != 1 || msgs[0].Status != "delivered" {
		t.Fatalf("after Bob reads list, status = %q, want delivered", msgs[0].Status)
	}
	if len(msgs[0].DeliveredTo) != 1 || msgs[0].DeliveredTo[0] != bobID {
		t.Fatalf("delivered_to = %v, want [%s]", msgs[0].DeliveredTo, bobID)
	}

	// Bob marks it read.
	rec = doJSON(t, h, http.MethodPost, "/api/messages/"+m.ID+"/read", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("mark read: %d %s", rec.Code, rec.Body.String())
	}
	var read models.Message
	mustDecode(t, rec, &read)
	if read.Status != "read" {
		t.Fatalf("after read, status = %q, want read", read.Status)
	}

	// Presence: Alice is offline in this unit test (no WS connection).
	rec = doJSON(t, h, http.MethodGet, "/api/users/"+bobID+"/presence", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("presence: %d %s", rec.Code, rec.Body.String())
	}
	var pres struct {
		Online bool `json:"online"`
	}
	mustDecode(t, rec, &pres)
	if pres.Online {
		t.Fatalf("presence online = true, want false (no live WS connection)")
	}
}

// TestV04LocationAndDisappearing: location messages, TTL validation and purge.
func TestV04LocationAndDisappearing(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	_, bobID := register(t, h, "bob@example.com", "bob")
	chatID := makeDM(t, h, aliceTok, bobID)

	m := sendMsg(t, h, aliceTok, chatID, map[string]interface{}{
		"kind": "location", "lat": 35.6892, "lng": 51.3890, "location_label": "Tehran",
	})
	if m.Kind != "location" || m.Lat == nil || *m.Lat != 35.6892 || m.LocationLabel != "Tehran" {
		t.Fatalf("location message = %+v", m)
	}

	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/messages",
		map[string]interface{}{"kind": "location", "lat": 400.0, "lng": 0.0}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid lat: status %d, want 400", rec.Code)
	}

	// Disappearing message with a 60s TTL.
	dis := sendMsg(t, h, aliceTok, chatID, map[string]interface{}{
		"content": "boom", "ttl_seconds": 60,
	})
	if dis.ExpiresAt == nil {
		t.Fatal("ttl message has no expires_at")
	}
	// TTL below the minimum is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/messages",
		map[string]interface{}{"content": "x", "ttl_seconds": 5}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ttl=5: status %d, want 400", rec.Code)
	}

	// Expired messages are purged from the store.
	expired := &models.Message{ChatID: chatID, SenderID: "x", Content: "old", Kind: "text"}
	past := time.Now().UTC().Add(-time.Minute)
	expired.ExpiresAt = &past
	if err := h.Store.CreateMessage(expired); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	if n := h.Store.PurgeExpired(time.Now().UTC()); n != 1 {
		t.Fatalf("PurgeExpired = %d; want 1", n)
	}
}

// TestV04Admin: owner/admin/member enforcement for add/remove/promote.
func TestV04Admin(t *testing.T) {
	h := newTestHandler()
	ownerTok, _ := register(t, h, "owner@example.com", "owner")
	memberTok, memberID := register(t, h, "member@example.com", "member")
	_, outsiderID := register(t, h, "outsider@example.com", "outsider")
	_, daveID := register(t, h, "dave@example.com", "dave")

	chatID := makeGroup(t, h, ownerTok, []string{memberID})

	// A plain member cannot add anyone.
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/members",
		map[string]interface{}{"user_id": outsiderID}, memberTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member add: status %d, want 403", rec.Code)
	}
	// Owner promotes the member to admin.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/members/"+memberID+"/role",
		map[string]interface{}{"role": "admin"}, ownerTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("promote: %d %s", rec.Code, rec.Body.String())
	}
	// Admin can now add the outsider.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/members",
		map[string]interface{}{"user_id": outsiderID}, memberTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin add: %d %s", rec.Code, rec.Body.String())
	}
	// Admin can remove a plain member.
	rec = doJSON(t, h, http.MethodDelete, "/api/chats/"+chatID+"/members/"+outsiderID, nil, memberTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin remove member: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/members/"+memberID+"/role",
		map[string]interface{}{"role": "member"}, memberTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin self-demote: status %d, want 403", rec.Code)
	}
	// Owner demotes the admin back to member.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/members/"+memberID+"/role",
		map[string]interface{}{"role": "member"}, ownerTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner demote: %d %s", rec.Code, rec.Body.String())
	}
	// Member cannot remove anyone now.
	rec = doJSON(t, h, http.MethodDelete, "/api/chats/"+chatID+"/members/"+daveID, nil, memberTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member remove: status %d, want 403", rec.Code)
	}
	// List members shows the roles.
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chatID+"/members", nil, ownerTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list members: %d %s", rec.Code, rec.Body.String())
	}
	var members []struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	mustDecode(t, rec, &members)
	if len(members) != 2 {
		t.Fatalf("members = %v, want 2", members)
	}
}

// TestV04PinArchiveFolder: pin, archive, folder are persisted per user.
func TestV04PinArchiveFolder(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	_, bobID := register(t, h, "bob@example.com", "bob")
	chatID := makeDM(t, h, aliceTok, bobID)

	for _, action := range []string{"pin", "archive"} {
		rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/"+action, nil, aliceTok)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", action, rec.Code, rec.Body.String())
		}
	}
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/folder",
		map[string]interface{}{"folder": "work"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("folder: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/chats", nil, aliceTok)
	var chats []*models.Chat
	mustDecode(t, rec, &chats)
	if len(chats) != 1 || !chats[0].Pinned || !chats[0].Archived || chats[0].Folder != "work" {
		t.Fatalf("chat settings = %+v, want pinned+archived+folder=work", chats[0])
	}
}

// TestV04Scheduled: schedule a message, cancel it, and deliver a due one.
func TestV04Scheduled(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	_, bobID := register(t, h, "bob@example.com", "bob")
	chatID := makeDM(t, h, aliceTok, bobID)

	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/scheduled",
		map[string]interface{}{"content": "later", "send_at": future}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("schedule: %d %s", rec.Code, rec.Body.String())
	}
	var sm models.ScheduledMessage
	mustDecode(t, rec, &sm)

	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chatID+"/scheduled", nil, aliceTok)
	var list []*models.ScheduledMessage
	mustDecode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("scheduled list = %d, want 1", len(list))
	}

	// Cancel it.
	rec = doJSON(t, h, http.MethodDelete, "/api/scheduled/"+sm.ID, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body.String())
	}

	// A due message is delivered by the dispatcher.
	due := &models.ScheduledMessage{ChatID: chatID, SenderID: "alice", Content: "due now",
		Kind: "text", SendAt: time.Now().UTC().Add(-time.Second)}
	if err := h.Store.CreateScheduled(due); err != nil {
		t.Fatalf("create due scheduled: %v", err)
	}
	if sent := h.ProcessDueScheduled(); len(sent) != 1 {
		t.Fatalf("ProcessDueScheduled sent %d messages, want 1", len(sent))
	}
	msgs := h.Store.MessagesForChat(chatID, 10, 0)
	found := false
	for _, m := range msgs {
		if m.Content == "due now" {
			found = true
		}
	}
	if !found {
		t.Fatal("due scheduled message was not delivered as a real message")
	}
}

// TestV04Stories: create a story, list it; a non-upload URL is rejected.
func TestV04Stories(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	register(t, h, "bob@example.com", "bob")

	rec := doJSON(t, h, http.MethodPost, "/api/stories",
		map[string]interface{}{"media_url": "https://evil.example/x.jpg"}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("external story URL: status %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories",
		map[string]interface{}{"media_url": "/uploads/story.jpg", "caption": "hello"}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create story: %d %s", rec.Code, rec.Body.String())
	}
	var story models.Story
	mustDecode(t, rec, &story)
	if story.ExpiresAt.Sub(story.CreatedAt) < 23*time.Hour {
		t.Fatalf("story lifetime too short: %v", story.ExpiresAt.Sub(story.CreatedAt))
	}
	rec = doJSON(t, h, http.MethodGet, "/api/stories?user_id="+aliceID, nil, aliceTok)
	var stories []*models.Story
	mustDecode(t, rec, &stories)
	if len(stories) != 1 {
		t.Fatalf("stories = %d, want 1", len(stories))
	}
}

// TestV04BotsAndKeys: bot lifecycle + one-time pre-key claim + identity keys.
func TestV04BotsAndKeys(t *testing.T) {
	h := newTestHandler()
	ownerTok, _ := register(t, h, "owner@example.com", "owner")
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")

	// Create a bot; the token is shown exactly once.
	rec := doJSON(t, h, http.MethodPost, "/api/bots",
		map[string]interface{}{"username": "echo", "display_name": "Echo Bot"}, ownerTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create bot: %d %s", rec.Code, rec.Body.String())
	}
	var bot models.Bot
	mustDecode(t, rec, &bot)
	if bot.Token == "" {
		t.Fatal("bot token was not returned at creation")
	}
	botID := bot.ID
	token := bot.Token

	// Listing bots never leaks the token.
	rec = doJSON(t, h, http.MethodGet, "/api/bots", nil, ownerTok)
	var bots []*models.Bot
	mustDecode(t, rec, &bots)
	if len(bots) != 1 || bots[0].Token != "" {
		t.Fatalf("list bots leaks token or wrong count: %+v", bots)
	}

	// Bot polling with a wrong token is rejected.
	req := httptest.NewRequest(http.MethodGet, "/api/bot/updates", nil)
	req.Header.Set("X-Bot-Token", "wrong-token")
	w := httptest.NewRecorder()
	h.BotUpdates(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad bot token: status %d, want 401", w.Code)
	}
	// With the right token, polling works (empty for now).
	req = httptest.NewRequest(http.MethodGet, "/api/bot/updates", nil)
	req.Header.Set("X-Bot-Token", token)
	w = httptest.NewRecorder()
	h.BotUpdates(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("bot updates: status %d", w.Code)
	}

	// Set a webhook URL.
	rec = doJSON(t, h, http.MethodPost, "/api/bots/"+botID+"/webhook",
		map[string]interface{}{"webhook_url": "https://example.com/hook"}, ownerTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("set webhook: %d %s", rec.Code, rec.Body.String())
	}

	// The bot is a real user: Alice DMs it, then the bot replies via token.
	botUser, err := h.Store.UserByUsername("bot_echo")
	if err != nil {
		t.Fatalf("bot user not found: %v", err)
	}
	chatID := makeDM(t, h, aliceTok, botUser.ID)
	sendMsg(t, h, aliceTok, chatID, map[string]interface{}{"content": "hi bot"})

	req = httptest.NewRequest(http.MethodPost, "/api/bot/send",
		strings.NewReader(`{"chat_id":"`+chatID+`","content":"hello human"}`))
	req.Header.Set("X-Bot-Token", token)
	w = httptest.NewRecorder()
	h.BotSend(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("bot send: status %d body %s", w.Code, w.Body.String())
	}

	// Bot updates now include Alice's message.
	req = httptest.NewRequest(http.MethodGet, "/api/bot/updates", nil)
	req.Header.Set("X-Bot-Token", token)
	w = httptest.NewRecorder()
	h.BotUpdates(w, req)
	var updates []*models.Message
	if err := json.NewDecoder(w.Body).Decode(&updates); err != nil {
		t.Fatalf("decode updates: %v", err)
	}
	if len(updates) != 1 || updates[0].Content != "hi bot" {
		t.Fatalf("bot updates = %+v, want Alice's message", updates)
	}
	_ = aliceID

	// E2EE foundation: identity key upload + one-time pre-key claim.
	keyB64 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh4="
	rec = doJSON(t, h, http.MethodPost, "/api/keys/identity",
		map[string]interface{}{"device_id": "phone", "identity_key": keyB64, "signed_prekey": keyB64}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload identity key: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/keys/prekeys",
		map[string]interface{}{"device_id": "phone", "keys": []map[string]interface{}{
			{"key_id": 1, "public_key": keyB64},
			{"key_id": 2, "public_key": keyB64},
		}}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload prekeys: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/keys/"+aliceID+"/prekey?device_id=phone", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch prekey: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/keys/"+aliceID+"/prekey?device_id=phone", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch 2nd prekey: %d %s", rec.Code, rec.Body.String())
	}
	// Both pre-keys are now used up: the next fetch is 404.
	rec = doJSON(t, h, http.MethodGet, "/api/keys/"+aliceID+"/prekey?device_id=phone", nil, aliceTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("fetch exhausted prekey: status %d, want 404", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/keys/"+aliceID+"/phone", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get identity key: %d %s", rec.Code, rec.Body.String())
	}
}

// TestChatListPreviewAndUnread: ListChats carries the latest message
// preview and a per-user unread count (WhatsApp-style list data).
// The count drops to zero after the other user marks it read.
func TestChatListPreviewAndUnread(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	chatID := makeDM(t, h, aliceTok, bobID)

	m := sendMsg(t, h, aliceTok, chatID, map[string]interface{}{"content": "سلام Bob"})

	list := func(tok string) []models.Chat {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/chats", nil, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("list chats: %d %s", rec.Code, rec.Body.String())
		}
		var chats []models.Chat
		mustDecode(t, rec, &chats)
		return chats
	}
	bobChats := list(bobTok)
	if len(bobChats) != 1 {
		t.Fatalf("bob chats = %d, want 1", len(bobChats))
	}
	c := bobChats[0]
	if c.LastMessage == nil || c.LastMessage.Content != "سلام Bob" {
		t.Fatalf("last_message = %+v, want content 'سلام Bob'", c.LastMessage)
	}
	if c.LastMessage.SenderID == "" {
		t.Fatalf("last_message sender is empty")
	}
	if c.UnreadCount != 1 {
		t.Fatalf("bob unread = %d, want 1", c.UnreadCount)
	}
	// Alice sent it: nothing unread for her.
	if aliceChats := list(aliceTok); aliceChats[0].UnreadCount != 0 {
		t.Fatalf("alice unread = %d, want 0", aliceChats[0].UnreadCount)
	}
	// Bob marks it read: his unread count drops to zero.
	rec := doJSON(t, h, http.MethodPost, "/api/messages/"+m.ID+"/read", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("mark read: %d %s", rec.Code, rec.Body.String())
	}
	if bobChats := list(bobTok); bobChats[0].UnreadCount != 0 {
		t.Fatalf("after read, bob unread = %d, want 0", bobChats[0].UnreadCount)
	}
	// A chat with no messages has no preview and no unread count.
	_, carolID := register(t, h, "carol@example.com", "carol")
	makeDM(t, h, aliceTok, carolID)
	if aliceChats := list(aliceTok); len(aliceChats) != 2 {
		t.Fatalf("alice chats = %d, want 2", len(aliceChats))
	} else {
		for _, ch := range aliceChats {
			if ch.ID == chatID {
				continue
			}
			if ch.LastMessage != nil || ch.UnreadCount != 0 {
				t.Fatalf("empty chat preview = %+v unread = %d, want nil/0", ch.LastMessage, ch.UnreadCount)
			}
		}
	}
}
