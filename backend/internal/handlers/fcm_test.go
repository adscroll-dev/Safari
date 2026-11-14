package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/vyro/backend/internal/push"
)

// fakeFCMClient records sends for the FCM handler tests and can be told
// to fail per token. It implements push.FCMClient.
type fakeFCMClient struct {
	sent []fakeFCMSend
	fail map[string]error
}

type fakeFCMSend struct {
	token string
	title string
	body  string
	data  map[string]string
}

func (f *fakeFCMClient) Send(_ context.Context, token, title, body string, data map[string]string) error {
	f.sent = append(f.sent, fakeFCMSend{token: token, title: title, body: body, data: data})
	if err, ok := f.fail[token]; ok {
		return err
	}
	return nil
}

// TestFCMTokenEndpoints: registration is 503 while FCM is unconfigured;
// once configured, tokens register (idempotently), unregister, and the
// test-push endpoint reports FCM counts.
func TestFCMTokenEndpoints(t *testing.T) {
	h := newTestHandler()
	tok, _ := register(t, h, "fcm@example.com", "fcmuser")

	// FCM not configured: registration is refused, never silently kept.
	rec := doJSON(t, h, http.MethodPost, "/api/push/fcm-token", map[string]interface{}{
		"token": "tok-1", "device_id": "phone", "platform": "android",
	}, tok)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("register without FCM: status %d, want 503", rec.Code)
	}

	client := &fakeFCMClient{fail: map[string]error{}}
	h.FCM = push.NewFCMRegistry(h.Store, client)

	// Empty token is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/push/fcm-token", map[string]interface{}{
		"token": "", "device_id": "phone",
	}, tok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("register empty token: status %d, want 400", rec.Code)
	}
	// A token registers; re-registering it stays at one token.
	for i := 0; i < 2; i++ {
		rec = doJSON(t, h, http.MethodPost, "/api/push/fcm-token", map[string]interface{}{
			"token": "tok-1", "device_id": "phone", "platform": "android",
		}, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("register token: %d %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Tokens int `json:"tokens"`
		}
		mustDecode(t, rec, &resp)
		if resp.Tokens != 1 {
			t.Fatalf("tokens after register = %d, want 1", resp.Tokens)
		}
	}
	// A second device adds a second token.
	rec = doJSON(t, h, http.MethodPost, "/api/push/fcm-token", map[string]interface{}{
		"token": "tok-2", "device_id": "tablet", "platform": "android",
	}, tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("register second token: %d %s", rec.Code, rec.Body.String())
	}
	// Test push reaches both tokens (Web Push is unconfigured here, so
	// the FCM-only branch of TestPush answers).
	rec = doJSON(t, h, http.MethodPost, "/api/push/test", nil, tok)
	var testResp struct {
		FCMSent   int `json:"fcm_sent"`
		FCMFailed int `json:"fcm_failed"`
	}
	mustDecode(t, rec, &testResp)
	if testResp.FCMSent != 2 || testResp.FCMFailed != 0 {
		t.Fatalf("test push = %+v, want fcm_sent=2 fcm_failed=0", testResp)
	}
	// Unregister one token; the next test push reaches only one.
	rec = doJSON(t, h, http.MethodDelete, "/api/push/fcm-token", map[string]interface{}{
		"token": "tok-1",
	}, tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unregister token: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/push/test", nil, tok)
	mustDecode(t, rec, &testResp)
	if testResp.FCMSent != 1 || testResp.FCMFailed != 0 {
		t.Fatalf("test push after unregister = %+v, want fcm_sent=1 fcm_failed=0", testResp)
	}
}

// TestFCMOfflineMessage: a message sent to an offline member (no
// WebSocket) produces an FCM notification with the chat id and no
// message content; a token FCM reports as unregistered is pruned.
func TestFCMOfflineMessage(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "fcm-alice@example.com", "fcmalice")
	bobTok, bobID := register(t, h, "fcm-bob@example.com", "fcmbob")

	client := &fakeFCMClient{fail: map[string]error{}}
	h.FCM = push.NewFCMRegistry(h.Store, client)

	// Alice registers her phone token.
	rec := doJSON(t, h, http.MethodPost, "/api/push/fcm-token", map[string]interface{}{
		"token": "alice-phone", "device_id": "phone", "platform": "android",
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice register token: %d %s", rec.Code, rec.Body.String())
	}
	// Bob opens a DM with Alice and sends a message. Alice has no
	// WebSocket connection, so she is offline for this message.
	rec = doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{aliceID},
	}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create chat: %d %s", rec.Code, rec.Body.String())
	}
	var chat struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&chat)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]string{
		"content": "secret hello",
	}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send message: %d %s", rec.Code, rec.Body.String())
	}
	if len(client.sent) != 1 {
		t.Fatalf("FCM sends = %d, want 1", len(client.sent))
	}
	sent := client.sent[0]
	if sent.token != "alice-phone" {
		t.Fatalf("FCM sent to %q, want alice-phone", sent.token)
	}
	if sent.data["chat_id"] != chat.ID {
		t.Fatalf("FCM data chat_id = %q, want %q", sent.data["chat_id"], chat.ID)
	}
	// The payload must never contain the message text.
	if sent.body == "secret hello" || sent.title == "secret hello" {
		t.Fatalf("FCM payload leaked message content: %+v", sent)
	}
	// The sender's name (not the content) is in the body.
	if sent.body != "New message from fcmbob" {
		t.Fatalf("FCM body = %q, want %q", sent.body, "New message from fcmbob")
	}
	_ = bobID

	// FCM reports the token dead: the next message prunes it.
	client.fail["alice-phone"] = push.ErrTokenUnregistered
	client.sent = nil
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]string{
		"content": "second hello",
	}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send second message: %d %s", rec.Code, rec.Body.String())
	}
	if got := h.FCM.Count(aliceID); got != 0 {
		t.Fatalf("unregistered token must be pruned (count %d)", got)
	}
}
