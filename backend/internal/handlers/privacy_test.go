package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestLastSeenPrivacy covers the Telegram/WhatsApp-style last-seen
// setting: the default is "everyone"; a user who sets "nobody" is
// invisible (offline, no last-seen) to everyone else, while still
// seeing their own presence; an invalid value is rejected.
func TestLastSeenPrivacy(t *testing.T) {
	h := newTestHandler()

	aliceTok, aliceID := register(t, h, "alice@example.com", "Alice")
	bobTok, _ := register(t, h, "bob@example.com", "Bob")

	// Default is "everyone".
	rec := doJSON(t, h, http.MethodGet, "/api/users/me/privacy", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get privacy: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var got struct {
		LastSeenVisibility string `json:"last_seen_visibility"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode privacy: %v", err)
	}
	if got.LastSeenVisibility != "everyone" {
		t.Fatalf("default privacy = %q, want \"everyone\"", got.LastSeenVisibility)
	}

	// Alice sets her presence online, then Bob sees her as online.
	if err := h.Store.SetPresence(aliceID, true, time.Now().UTC()); err != nil {
		t.Fatalf("set presence: %v", err)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/users/"+aliceID+"/presence", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("presence: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var pres struct {
		IsOnline   bool    `json:"is_online"`
		LastSeenAt *string `json:"last_seen_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&pres); err != nil {
		t.Fatalf("decode presence: %v", err)
	}
	if !pres.IsOnline || pres.LastSeenAt == nil {
		t.Fatalf("presence with everyone = %+v, want online with a last-seen time", pres)
	}

	// An invalid value is rejected.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy",
		map[string]string{"last_seen_visibility": "contacts"}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid privacy: expected 400, got %d %s", rec.Code, rec.Body.String())
	}

	// Alice sets "nobody": Bob now sees her offline with no last-seen.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy",
		map[string]string{"last_seen_visibility": "nobody"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("set privacy: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/users/"+aliceID+"/presence", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("presence hidden: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var hidden struct {
		IsOnline   bool    `json:"is_online"`
		LastSeenAt *string `json:"last_seen_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&hidden); err != nil {
		t.Fatalf("decode hidden presence: %v", err)
	}
	if hidden.IsOnline || hidden.LastSeenAt != nil {
		t.Fatalf("presence with nobody = %+v, want offline with null last-seen", hidden)
	}

	// Alice herself still sees her own presence.
	rec = doJSON(t, h, http.MethodGet, "/api/users/"+aliceID+"/presence", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("self presence: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var self struct {
		IsOnline   bool    `json:"is_online"`
		LastSeenAt *string `json:"last_seen_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&self); err != nil {
		t.Fatalf("decode self presence: %v", err)
	}
	if !self.IsOnline || self.LastSeenAt == nil {
		t.Fatalf("self presence = %+v, want online with a last-seen time", self)
	}

	// Setting it back to "everyone" restores visibility for Bob.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/privacy",
		map[string]string{"last_seen_visibility": "everyone"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset privacy: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}
