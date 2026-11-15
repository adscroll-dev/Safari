package handlers

import (
	"net/http"
	"testing"
)

// Stage 11 (Settings parity): combined privacy settings, read
// receipts, profile-photo / about / groups privacy, delete account.
func TestStage11Settings(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice11@vyro.test", "alice11")
	bobTok, bobID := register(t, h, "bob11@vyro.test", "bob11")
	dm := makeDM(t, h, aliceTok, bobID)

	// Defaults: everything "everyone", read receipts on.
	rec := doJSON(t, h, http.MethodGet, "/api/users/me/privacy-settings", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("privacy-settings GET: status %d body %s", rec.Code, rec.Body.String())
	}
	var settings map[string]interface{}
	mustDecode(t, rec, &settings)
	for _, k := range []string{"last_seen_visibility", "status_privacy", "avatar_privacy", "about_privacy", "groups_privacy"} {
		if settings[k] != "everyone" {
			t.Fatalf("default %s = %v, want everyone", k, settings[k])
		}
	}
	if settings["read_receipts"] != true {
		t.Fatalf("default read_receipts = %v, want true", settings["read_receipts"])
	}

	// Partial update + invalid value rejected.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"read_receipts": false, "avatar_privacy": "nobody", "groups_privacy": "nobody"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("privacy-settings PUT: status %d body %s", rec.Code, rec.Body.String())
	}
	mustDecode(t, rec, &settings)
	if settings["read_receipts"] != false || settings["avatar_privacy"] != "nobody" || settings["groups_privacy"] != "nobody" {
		t.Fatalf("PUT did not apply: %v", settings)
	}
	if settings["last_seen_visibility"] != "everyone" {
		t.Fatalf("PUT changed an untouched field: %v", settings)
	}
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"about_privacy": "friends"}, bobTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad privacy value: status %d, want 400", rec.Code)
	}

	// Read receipts: Bob reads with receipts off — Alice must not
	// see "read" (only "delivered"), and Bob turning them back on
	// and re-reading makes the blue tick appear.
	msg := sendMsg(t, h, aliceTok, dm, map[string]interface{}{"kind": "text", "content": "receipts?"})
	// Bob fetches the chat first (marks delivered), then reads.
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+dm+"/messages", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob list messages: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/messages/"+msg.ID+"/read", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob read (receipts off): status %d body %s", rec.Code, rec.Body.String())
	}
	var listed []map[string]interface{}
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+dm+"/messages", nil, aliceTok)
	mustDecode(t, rec, &listed)
	if len(listed) != 1 || listed[0]["status"] != "delivered" {
		t.Fatalf("alice sees status %v with bob receipts off, want delivered", listed)
	}
	// Bob turns receipts on and reads again: blue tick appears.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"read_receipts": true}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob receipts on: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/messages/"+msg.ID+"/read", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob read (receipts on): status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+dm+"/messages", nil, aliceTok)
	mustDecode(t, rec, &listed)
	if len(listed) != 1 || listed[0]["status"] != "read" {
		t.Fatalf("alice sees status %v with bob receipts on, want read", listed)
	}
	// Viewer-side: Alice with receipts off sees no blue ticks on
	// her own messages even when Bob read them.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"read_receipts": false}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice receipts off: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+dm+"/messages", nil, aliceTok)
	mustDecode(t, rec, &listed)
	if len(listed) != 1 || listed[0]["status"] != "delivered" {
		t.Fatalf("alice (receipts off) sees status %v, want delivered", listed)
	}
	// And Alice's own read is not recorded while her receipts are
	// off: Bob sends a message, Alice reads it, Bob must not see
	// "read".
	msg2 := sendMsg(t, h, bobTok, dm, map[string]interface{}{"kind": "text", "content": "for alice"})
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+dm+"/messages", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice list: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/messages/"+msg2.ID+"/read", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice read (receipts off): status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+dm+"/messages", nil, bobTok)
	mustDecode(t, rec, &listed)
	if len(listed) != 2 {
		t.Fatalf("bob lists %d messages, want 2", len(listed))
	}
	if listed[1]["status"] != "delivered" {
		t.Fatalf("bob sees status %v for a message alice read with receipts off, want delivered", listed[1]["status"])
	}

	// Avatar / about privacy: Bob hides both; Alice's user list
	// must show neither. (Bob still has avatar_privacy=nobody
	// from the earlier PUT.)
	rec = doJSON(t, h, http.MethodPatch, "/api/users/me",
		map[string]interface{}{"bio": "Bob here", "avatar_url": "/uploads/bob.png"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob update profile: status %d body %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"about_privacy": "nobody"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob about privacy: status %d", rec.Code)
	}
	var users []map[string]interface{}
	rec = doJSON(t, h, http.MethodGet, "/api/users?search=bob11", nil, aliceTok)
	mustDecode(t, rec, &users)
	if len(users) != 1 {
		t.Fatalf("alice user search: %d results, want 1", len(users))
	}
	if users[0]["avatar_url"] != nil && users[0]["avatar_url"] != "" {
		t.Fatalf("bob avatar visible despite nobody: %v", users[0]["avatar_url"])
	}
	if users[0]["bio"] != nil && users[0]["bio"] != "" {
		t.Fatalf("bob bio visible despite nobody: %v", users[0]["bio"])
	}
	// Back to everyone: both visible again.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"avatar_privacy": "everyone", "about_privacy": "everyone"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob privacy back: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/users?search=bob11", nil, aliceTok)
	mustDecode(t, rec, &users)
	if users[0]["avatar_url"] != "/uploads/bob.png" || users[0]["bio"] != "Bob here" {
		t.Fatalf("bob avatar/bio not visible after everyone: %v", users[0])
	}

	// Groups privacy: Bob (nobody) cannot be added to a group, at
	// creation or by an admin, but can join after switching back.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"groups_privacy": "nobody"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob groups privacy: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats",
		map[string]interface{}{"type": "group", "title": "No Bob", "member_ids": []string{bobID}}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create group with nobody member: status %d, want 400", rec.Code)
	}
	group := makeGroup(t, h, aliceTok, []string{})
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+group+"/members",
		map[string]interface{}{"user_id": bobID, "role": "member"}, aliceTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("add nobody member: status %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy-settings",
		map[string]interface{}{"groups_privacy": "everyone"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob groups back: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+group+"/members",
		map[string]interface{}{"user_id": bobID, "role": "member"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("add bob after everyone: status %d body %s", rec.Code, rec.Body.String())
	}

	// Delete account: Carol registers, chats with Alice, then
	// deletes herself. Her token yields nothing afterwards (the
	// auth middleware only checks the signature, so the lookup
	// answers 404 "user not found"), she disappears from the
	// user list, her messages are gone, and her email can
	// register again.
	carolTok, _ := register(t, h, "carol11@vyro.test", "carol11")
	carolDM := makeDM(t, h, carolTok, aliceID)
	sendMsg(t, h, carolTok, carolDM, map[string]interface{}{"kind": "text", "content": "goodbye"})
	rec = doJSON(t, h, http.MethodDelete, "/api/users/me", nil, carolTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete account: status %d body %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/users/me", nil, carolTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("deleted user still readable: status %d, want 404", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/users?search=carol11", nil, aliceTok)
	mustDecode(t, rec, &users)
	if len(users) != 0 {
		t.Fatalf("deleted user still listed: %v", users)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+carolDM+"/messages", nil, aliceTok)
	mustDecode(t, rec, &listed)
	if len(listed) != 0 {
		t.Fatalf("deleted user's messages still listed: %d", len(listed))
	}
	_, _ = register(t, h, "carol11@vyro.test", "carol11")
}
