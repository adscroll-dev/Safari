package handlers

import (
	"net/http"
	"strings"
	"testing"
)

// Stage 7 (accounts): profile editing, user reports, block/unblock
// with DM send enforcement, blocked list, linked devices.
func TestStage7Accounts(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice7@vyro.test", "alice7")
	bobTok, bobID := register(t, h, "bob7@vyro.test", "bob7")

	// --- Profile editing ---
	rec := doJSON(t, h, http.MethodPatch, "/api/users/me", map[string]interface{}{
		"display_name": "Alice Seven", "bio": "Kabul, Afghanistan",
		"avatar_url":   "/uploads/avatar7.png",
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("update profile: status %d body %s", rec.Code, rec.Body.String())
	}
	var me map[string]interface{}
	mustDecode(t, rec, &me)
	if me["display_name"] != "Alice Seven" || me["bio"] != "Kabul, Afghanistan" ||
		me["avatar_url"] != "/uploads/avatar7.png" {
		t.Fatalf("profile fields not updated: %v", me)
	}
	// GET /api/users/me now carries the bio.
	rec = doJSON(t, h, http.MethodGet, "/api/users/me", nil, aliceTok)
	mustDecode(t, rec, &me)
	if me["bio"] != "Kabul, Afghanistan" {
		t.Fatalf("GET /me bio = %v", me["bio"])
	}
	// Partial update: only the bio changes.
	rec = doJSON(t, h, http.MethodPatch, "/api/users/me",
		map[string]interface{}{"bio": "New bio"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("partial update: status %d", rec.Code)
	}
	mustDecode(t, rec, &me)
	if me["bio"] != "New bio" || me["display_name"] != "Alice Seven" {
		t.Fatalf("partial update wrong: %v", me)
	}
	// Validation: empty name, oversized bio, remote avatar, empty body.
	for _, body := range []map[string]interface{}{
		{"display_name": "  "},
		{"bio": strings.Repeat("x", 501)},
		{"avatar_url": "https://example.com/a.png"},
		{},
	} {
		rec = doJSON(t, h, http.MethodPatch, "/api/users/me", body, aliceTok)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("profile validation %v: status %d, want 400", body, rec.Code)
		}
	}

	// --- Reports ---
	rec = doJSON(t, h, http.MethodPost, "/api/users/"+bobID+"/report",
		map[string]interface{}{"reason": "spam messages"}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("report: status %d body %s", rec.Code, rec.Body.String())
	}
	var rep map[string]interface{}
	mustDecode(t, rec, &rep)
	if rep["status"] != "open" || rep["target_id"] != bobID {
		t.Fatalf("report fields wrong: %v", rep)
	}
	// Self-report, unknown user, empty reason -> 400/404.
	rec = doJSON(t, h, http.MethodPost, "/api/users/"+aliceID+"/report",
		map[string]interface{}{"reason": "me"}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("self report: status %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/users/00000000-0000-0000-0000-000000000000/report",
		map[string]interface{}{"reason": "spam"}, aliceTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown report target: status %d, want 404", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/users/"+bobID+"/report",
		map[string]interface{}{"reason": "  "}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty report reason: status %d, want 400", rec.Code)
	}

	// --- Block / unblock + DM send enforcement ---
	dm := makeDM(t, h, aliceTok, bobID)
	sendMsg(t, h, aliceTok, dm, map[string]interface{}{"kind": "text", "content": "hi"})
	rec = doJSON(t, h, http.MethodPost, "/api/users/"+bobID+"/block", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("block: status %d body %s", rec.Code, rec.Body.String())
	}
	// Blocked list shows Bob for Alice.
	rec = doJSON(t, h, http.MethodGet, "/api/users/me/blocked", nil, aliceTok)
	var blocked []string
	mustDecode(t, rec, &blocked)
	if len(blocked) != 1 || blocked[0] != bobID {
		t.Fatalf("blocked list = %v, want [%s]", blocked, bobID)
	}
	// Neither side can send in the DM while blocked.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm+"/messages",
		map[string]interface{}{"kind": "text", "content": "blocked?"}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("blocked user send: status %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm+"/messages",
		map[string]interface{}{"kind": "text", "content": "blocked?"}, aliceTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("blocker send: status %d, want 403", rec.Code)
	}
	// Unblock: sending works again, list is empty.
	rec = doJSON(t, h, http.MethodDelete, "/api/users/"+bobID+"/block", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unblock: status %d body %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/users/me/blocked", nil, aliceTok)
	mustDecode(t, rec, &blocked)
	if len(blocked) != 0 {
		t.Fatalf("blocked list after unblock = %v", blocked)
	}
	sendMsg(t, h, bobTok, dm, map[string]interface{}{"kind": "text", "content": "back"})

	// --- Linked devices ---
	rec = doJSON(t, h, http.MethodPost, "/api/devices", map[string]interface{}{
		"device_id": "phone-1", "name": "Galaxy S21", "platform": "android",
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("register device: status %d body %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/devices", nil, aliceTok)
	var devices []map[string]interface{}
	mustDecode(t, rec, &devices)
	if len(devices) != 1 || devices[0]["device_id"] != "phone-1" {
		t.Fatalf("devices = %v", devices)
	}
	rec = doJSON(t, h, http.MethodDelete, "/api/devices/phone-1", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("remove device: status %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/devices", nil, aliceTok)
	mustDecode(t, rec, &devices)
	if len(devices) != 0 {
		t.Fatalf("devices after remove = %v", devices)
	}
}
