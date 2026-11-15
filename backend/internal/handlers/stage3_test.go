package handlers

import (
	"net/http"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// Stage 3 calls (WhatsApp parity): call history — the caller records
// a finished/missed call; the callee (and, for group calls, every
// member) sees it in their history with direction and missed marker.
func TestStage3Calls(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, carolID := register(t, h, "carol@example.com", "carol")
	daveTok, _ := register(t, h, "dave@example.com", "dave")

	dm := makeDM(t, h, aliceTok, bobID)
	group := makeGroup(t, h, aliceTok, []string{bobID, carolID})

	record := func(tok string, body map[string]interface{}) (int, models.CallRecord) {
		t.Helper()
		rec := doJSON(t, h, http.MethodPost, "/api/calls", body, tok)
		if rec.Code != http.StatusCreated {
			return rec.Code, models.CallRecord{}
		}
		var out models.CallRecord
		mustDecode(t, rec, &out)
		return rec.Code, out
	}
	list := func(tok string) []models.CallRecordView {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/calls", nil, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("list calls: %d %s", rec.Code, rec.Body.String())
		}
		var out []models.CallRecordView
		mustDecode(t, rec, &out)
		return out
	}

	// Alice calls Bob: an ended voice call (125 s) and a missed video call.
	code, ended := record(aliceTok, map[string]interface{}{
		"chat_id": dm, "callee_id": bobID,
		"call_type": "voice", "status": "ended", "duration_seconds": 125,
	})
	if code != http.StatusCreated {
		t.Fatalf("record ended call: %d", code)
	}
	if ended.CallerID != aliceID || ended.CalleeID != bobID ||
		ended.DurationSeconds != 125 || ended.EndedAt == nil {
		t.Fatalf("ended call fields wrong: %+v", ended)
	}
	code, missed := record(aliceTok, map[string]interface{}{
		"chat_id": dm, "callee_id": bobID,
		"call_type": "video", "status": "missed", "duration_seconds": 0,
	})
	if code != http.StatusCreated {
		t.Fatalf("record missed call: %d", code)
	}
	if missed.Status != "missed" || missed.EndedAt != nil {
		t.Fatalf("missed call fields wrong: %+v", missed)
	}
	// Alice's group call (no callee): ended, 60 s.
	code, groupCall := record(aliceTok, map[string]interface{}{
		"chat_id": group, "call_type": "voice",
		"status": "ended", "duration_seconds": 60,
	})
	if code != http.StatusCreated {
		t.Fatalf("record group call: %d", code)
	}
	if groupCall.CalleeID != "" {
		t.Fatalf("group call must have no callee: %+v", groupCall)
	}

	// Bob's history: newest first — group call, missed video, ended voice.
	bobCalls := list(bobTok)
	if len(bobCalls) != 3 {
		t.Fatalf("bob history: want 3, got %d", len(bobCalls))
	}
	if bobCalls[0].ID != groupCall.ID || !bobCalls[0].IsGroup ||
		bobCalls[0].PeerName != "Test Group" || bobCalls[0].Direction != "incoming" {
		t.Fatalf("bob newest (group) wrong: %+v", bobCalls[0])
	}
	if bobCalls[1].Status != "missed" || bobCalls[1].CallType != "video" ||
		bobCalls[1].Direction != "incoming" || bobCalls[1].PeerName != "alice" {
		t.Fatalf("bob missed call wrong: %+v", bobCalls[1])
	}
	if bobCalls[2].Status != "ended" || bobCalls[2].DurationSeconds != 125 ||
		bobCalls[2].PeerName != "alice" {
		t.Fatalf("bob ended call wrong: %+v", bobCalls[2])
	}

	// Alice's history: same three, all outgoing.
	aliceCalls := list(aliceTok)
	if len(aliceCalls) != 3 {
		t.Fatalf("alice history: want 3, got %d", len(aliceCalls))
	}
	for _, c := range aliceCalls {
		if c.Direction != "outgoing" {
			t.Fatalf("alice call not outgoing: %+v", c)
		}
	}
	if aliceCalls[1].PeerName != "bob" {
		t.Fatalf("alice peer name wrong: %+v", aliceCalls[1])
	}

	// Carol only sees the group call; Dave sees nothing.
	carolCalls := list(carolTok)
	if len(carolCalls) != 1 || carolCalls[0].ID != groupCall.ID {
		t.Fatalf("carol history: want only the group call, got %+v", carolCalls)
	}
	if len(list(daveTok)) != 0 {
		t.Fatalf("dave history must be empty")
	}

	// Validation errors.
	bad := []map[string]interface{}{
		{"chat_id": dm, "callee_id": bobID, "call_type": "smoke", "status": "ended"},
		{"chat_id": dm, "callee_id": bobID, "call_type": "voice", "status": "ringing"},
		{"chat_id": dm, "callee_id": bobID, "call_type": "voice", "status": "ended", "duration_seconds": -1},
		{"chat_id": dm, "callee_id": aliceID, "call_type": "voice", "status": "ended"},
		{"chat_id": dm, "callee_id": carolID, "call_type": "voice", "status": "ended"},
		{"chat_id": dm, "call_type": "voice", "status": "ended"},
	}
	for i, b := range bad {
		rec := doJSON(t, h, http.MethodPost, "/api/calls", b, aliceTok)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("bad record %d: want 400, got %d %s", i, rec.Code, rec.Body.String())
		}
	}
	// Unknown chat -> 404; non-member -> 403; no token -> 401.
	rec := doJSON(t, h, http.MethodPost, "/api/calls", map[string]interface{}{
		"chat_id": "00000000-0000-0000-0000-000000000000",
		"call_type": "voice", "status": "missed",
	}, aliceTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown chat: want 404, got %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/calls", map[string]interface{}{
		"chat_id": dm, "callee_id": bobID,
		"call_type": "voice", "status": "missed",
	}, daveTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member record: want 403, got %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/calls", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", rec.Code)
	}
}
