package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestForwardMessage: Bob forwards Alice's text message into a DM with
// Carol. The forwarded copy keeps Alice as the original sender, the
// content, and Bob as the sender of the new message.
func TestForwardMessage(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, carolID := register(t, h, "carol@example.com", "carol")
	_ = carolTok

	// Alice -> Bob DM with one text message.
	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create dm: %d %s", rec.Code, rec.Body.String())
	}
	var dm struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&dm)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm.ID+"/messages",
		map[string]string{"content": "hello from alice"}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send original: %d %s", rec.Code, rec.Body.String())
	}

	// Bob -> Carol DM; Bob forwards Alice's message there.
	rec = doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{carolID},
	}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create dm 2: %d %s", rec.Code, rec.Body.String())
	}
	var dm2 struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&dm2)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm2.ID+"/messages",
		map[string]string{
			"content":           "hello from alice",
			"forwarded_from":    "alice",
			"forwarded_from_id": aliceID,
		}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("forward: %d %s", rec.Code, rec.Body.String())
	}
	var fwd struct {
		ID              string `json:"id"`
		SenderID        string `json:"sender_id"`
		Content         string `json:"content"`
		ForwardedFrom   string `json:"forwarded_from"`
		ForwardedFromID string `json:"forwarded_from_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&fwd); err != nil {
		t.Fatalf("decode forwarded message: %v", err)
	}
	if fwd.SenderID != bobID {
		t.Fatalf("forwarded sender = %q, want bob %q", fwd.SenderID, bobID)
	}
	if fwd.Content != "hello from alice" {
		t.Fatalf("forwarded content = %q, want original text", fwd.Content)
	}
	if fwd.ForwardedFrom != "alice" || fwd.ForwardedFromID != aliceID {
		t.Fatalf("forwarded attribution = %q/%q, want alice/%s",
			fwd.ForwardedFrom, fwd.ForwardedFromID, aliceID)
	}

	// Listing the chat shows the same attribution (round-trip through
	// the store, not just the create response).
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+dm2.ID+"/messages", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list messages: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"forwarded_from":"alice"`) {
		t.Fatalf("listed messages lack forwarded_from: %s", rec.Body.String())
	}
}

// TestForwardMediaMessage: forwarding keeps kind + file fields, so a
// photo can be forwarded without re-uploading it.
func TestForwardMediaMessage(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")

	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceTok)
	var dm struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&dm)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm.ID+"/messages",
		map[string]interface{}{
			"content":           "a photo",
			"kind":              "image",
			"file_url":          "/uploads/photo.png",
			"file_name":         "photo.png",
			"file_size":         1234,
			"mime_type":         "image/png",
			"forwarded_from":    "alice",
			"forwarded_from_id": aliceID,
		}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("forward media: %d %s", rec.Code, rec.Body.String())
	}
	var fwd struct {
		Kind            string `json:"kind"`
		FileURL         string `json:"file_url"`
		ForwardedFrom   string `json:"forwarded_from"`
		ForwardedFromID string `json:"forwarded_from_id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&fwd)
	if fwd.Kind != "image" || fwd.FileURL != "/uploads/photo.png" {
		t.Fatalf("forwarded media lost kind/file: %+v", fwd)
	}
	if fwd.ForwardedFrom != "alice" || fwd.ForwardedFromID != aliceID {
		t.Fatalf("forwarded media attribution = %+v", fwd)
	}
}

// TestForwardLocationMessage: a forwarded location keeps its
// coordinates and label in the new chat.
func TestForwardLocationMessage(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")

	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceTok)
	var dm struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&dm)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm.ID+"/messages",
		map[string]interface{}{
			"kind":              "location",
			"lat":               34.5,
			"lng":               69.2,
			"location_label":    "Kabul",
			"forwarded_from":    "alice",
			"forwarded_from_id": aliceID,
		}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("forward location: %d %s", rec.Code, rec.Body.String())
	}
	var fwd struct {
		Lat           *float64 `json:"lat"`
		Lng           *float64 `json:"lng"`
		LocationLabel string   `json:"location_label"`
		ForwardedFrom string   `json:"forwarded_from"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&fwd)
	if fwd.Lat == nil || *fwd.Lat != 34.5 || fwd.Lng == nil || *fwd.Lng != 69.2 {
		t.Fatalf("forwarded location lost coordinates: %+v", fwd)
	}
	if fwd.LocationLabel != "Kabul" || fwd.ForwardedFrom != "alice" {
		t.Fatalf("forwarded location lost label/attribution: %+v", fwd)
	}
}

// TestForwardValidation: unknown original sender, overlong name, and
// E2EE forwarding are all rejected.
func TestForwardValidation(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	_, bobID := register(t, h, "bob@example.com", "bob")

	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceTok)
	var dm struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&dm)

	// Unknown forwarded_from_id (a well-formed UUID that no user has).
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm.ID+"/messages",
		map[string]string{
			"content":           "x",
			"forwarded_from":    "ghost",
			"forwarded_from_id": "00000000-0000-0000-0000-000000000000",
		}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown forwarded_from_id: expected 400, got %d %s", rec.Code, rec.Body.String())
	}
	// Overlong forwarded_from name.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm.ID+"/messages",
		map[string]string{
			"content":           "x",
			"forwarded_from":    strings.Repeat("a", 101),
			"forwarded_from_id": bobID,
		}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("long forwarded_from: expected 400, got %d %s", rec.Code, rec.Body.String())
	}
	// E2EE envelope cannot be forwarded (rejected before envelope
	// validation, so even a garbage envelope gives the forward error).
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm.ID+"/messages",
		map[string]string{
			"content":           "not-a-real-envelope",
			"kind":              "e2ee",
			"forwarded_from":    "alice",
			"forwarded_from_id": aliceID,
		}, aliceTok)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "cannot be forwarded") {
		t.Fatalf("e2ee forward: expected 400 'cannot be forwarded', got %d %s", rec.Code, rec.Body.String())
	}
	// forwarded_from_id without a name is filled in from the user.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm.ID+"/messages",
		map[string]string{
			"content":           "x",
			"forwarded_from_id": bobID,
		}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("forward with id only: %d %s", rec.Code, rec.Body.String())
	}
	var fwd struct {
		ForwardedFrom string `json:"forwarded_from"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&fwd)
	if fwd.ForwardedFrom != "bob" {
		t.Fatalf("forwarded_from filled from user = %q, want bob", fwd.ForwardedFrom)
	}
}
