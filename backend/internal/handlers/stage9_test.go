package handlers

import (
	"net/http"
	"testing"
)

// Stage 9 (extras): backup export, storage report, anti-spam
// send rate limit.
func TestStage9BackupStorageAntiSpam(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice9@vyro.test", "alice9")
	bobTok, bobID := register(t, h, "bob9@vyro.test", "bob9")
	dm := makeDM(t, h, aliceTok, bobID)

	// 3 text messages + 1 file message (file_size 2048) from Alice.
	for i := 0; i < 3; i++ {
		sendMsg(t, h, aliceTok, dm, map[string]interface{}{"kind": "text", "content": "hello 9"})
	}
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+dm+"/messages",
		map[string]interface{}{"kind": "file", "content": "report.pdf",
			"file_url": "/uploads/x", "file_name": "report.pdf",
			"file_size": 2048}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("file message: status %d body %s", rec.Code, rec.Body.String())
	}

	// Storage: 4 messages, 2048 bytes, in the one DM.
	rec = doJSON(t, h, http.MethodGet, "/api/users/me/storage", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("storage: status %d body %s", rec.Code, rec.Body.String())
	}
	var storage map[string]interface{}
	mustDecode(t, rec, &storage)
	if storage["total_messages"] != float64(4) {
		t.Fatalf("total_messages = %v, want 4", storage["total_messages"])
	}
	if storage["total_bytes"] != float64(2048) {
		t.Fatalf("total_bytes = %v, want 2048", storage["total_bytes"])
	}
	chats, ok := storage["chats"].([]interface{})
	if !ok || len(chats) != 1 {
		t.Fatalf("storage chats = %v, want 1 row", storage["chats"])
	}
	row := chats[0].(map[string]interface{})
	if row["message_count"] != float64(4) || row["bytes"] != float64(2048) {
		t.Fatalf("storage row = %v, want 4 messages / 2048 bytes", row)
	}
	// Bob's view: the same DM (he is a member), same totals.
	rec = doJSON(t, h, http.MethodGet, "/api/users/me/storage", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob storage: status %d", rec.Code)
	}

	// Export: profile + the DM with its 4 messages + privacy.
	rec = doJSON(t, h, http.MethodGet, "/api/users/me/export", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("export: status %d body %s", rec.Code, rec.Body.String())
	}
	var exp map[string]interface{}
	mustDecode(t, rec, &exp)
	user, ok := exp["user"].(map[string]interface{})
	if !ok || user["email"] != "alice9@vyro.test" {
		t.Fatalf("export user = %v", exp["user"])
	}
	expChats, ok := exp["chats"].([]interface{})
	if !ok || len(expChats) != 1 {
		t.Fatalf("export chats = %v, want 1", exp["chats"])
	}
	expMsgs, ok := expChats[0].(map[string]interface{})["messages"].([]interface{})
	if !ok || len(expMsgs) != 4 {
		t.Fatalf("export messages = %d, want 4", len(expMsgs))
	}
	if exp["last_seen"] != "everyone" || exp["status_privacy"] != "everyone" {
		t.Fatalf("export privacy = %v / %v", exp["last_seen"], exp["status_privacy"])
	}

	// Anti-spam: the 60/min limit. Alice already sent 4 messages
	// through the HTTP path in this test (3 text + 1 file); send
	// 56 more (total 60) — all must succeed — then the 61st must
	// be 429.
	for i := 0; i < 56; i++ {
		sendMsg(t, h, aliceTok, dm, map[string]interface{}{"kind": "text", "content": "flood check"})
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm+"/messages",
		map[string]interface{}{"kind": "text", "content": "one too many"}, aliceTok)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("61st message: status %d, want 429", rec.Code)
	}
	// Bob is a different account: his own budget is untouched.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm+"/messages",
		map[string]interface{}{"kind": "text", "content": "bob is fine"}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bob message after alice flood: status %d, want 201", rec.Code)
	}
	// No token at all: 401, not 429.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm+"/messages",
		map[string]interface{}{"kind": "text", "content": "anon"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon message: status %d, want 401", rec.Code)
	}
}
