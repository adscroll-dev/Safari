package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// Stage 4 polls + group pin permissions (WhatsApp parity): polls
// appear in the chat flow as "poll" messages with live vote counts;
// re-voting replaces the old vote; in groups only admins/owners can
// pin, and pinning posts a system notice.
func TestStage4Polls(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, carolID := register(t, h, "carol@example.com", "carol")
	daveTok, _ := register(t, h, "dave@example.com", "dave")

	dm := makeDM(t, h, aliceTok, bobID)
	group := makeGroup(t, h, aliceTok, []string{bobID, carolID})

	create := func(tok, chatID string, body map[string]interface{}) (int, models.Poll) {
		t.Helper()
		rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/polls", body, tok)
		if rec.Code != http.StatusCreated {
			return rec.Code, models.Poll{}
		}
		var out models.Poll
		mustDecode(t, rec, &out)
		return rec.Code, out
	}
	vote := func(tok, pollID string, indexes []int) (int, models.Poll) {
		t.Helper()
		rec := doJSON(t, h, http.MethodPost, "/api/polls/"+pollID+"/vote",
			map[string]interface{}{"option_indexes": indexes}, tok)
		if rec.Code != http.StatusOK {
			return rec.Code, models.Poll{}
		}
		var out models.Poll
		mustDecode(t, rec, &out)
		return rec.Code, out
	}
	messages := func(tok, chatID string) []models.Message {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/chats/"+chatID+"/messages", nil, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("list messages: %d %s", rec.Code, rec.Body.String())
		}
		var out []models.Message
		mustDecode(t, rec, &out)
		return out
	}

	// Alice creates a poll in the group.
	code, poll := create(aliceTok, group, map[string]interface{}{
		"question": "Where do we meet?",
		"options":  []string{"Park", "Cafe", "Beach"},
	})
	if code != http.StatusCreated {
		t.Fatalf("create poll: %d", code)
	}
	if poll.Question != "Where do we meet?" || len(poll.Options) != 3 ||
		poll.TotalVotes != 0 || len(poll.Votes) != 3 {
		t.Fatalf("poll fields wrong: %+v", poll)
	}
	// The poll appears in the chat flow as a "poll" message.
	found := false
	for _, m := range messages(aliceTok, group) {
		if m.Kind == "poll" && strings.Contains(m.Content, poll.ID) {
			found = true
		}
	}
	if !found {
		t.Fatalf("poll message not in chat flow")
	}

	// Bob votes Cafe, then changes his mind to Beach (re-vote replaces).
	code, v := vote(bobTok, poll.ID, []int{1})
	if code != http.StatusOK || v.Votes[1] != 1 || v.TotalVotes != 1 ||
		len(v.MyVotes) != 1 || v.MyVotes[0] != 1 {
		t.Fatalf("bob vote: %d %+v", code, v)
	}
	code, v = vote(bobTok, poll.ID, []int{2})
	if code != http.StatusOK || v.Votes[1] != 0 || v.Votes[2] != 1 || v.TotalVotes != 1 {
		t.Fatalf("bob re-vote: %d %+v", code, v)
	}
	// Carol votes Park.
	code, v = vote(carolTok, poll.ID, []int{0})
	if code != http.StatusOK || v.Votes[0] != 1 || v.TotalVotes != 2 {
		t.Fatalf("carol vote: %d %+v", code, v)
	}
	// Bob's own view shows his vote.
	rec := doJSON(t, h, http.MethodGet, "/api/polls/"+poll.ID, nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get poll: %d", rec.Code)
	}
	var got models.Poll
	mustDecode(t, rec, &got)
	if len(got.MyVotes) != 1 || got.MyVotes[0] != 2 || got.TotalVotes != 2 {
		t.Fatalf("get poll view wrong: %+v", got)
	}

	// Validation: too few / too many options, empty question,
	// out-of-range index, duplicate index, two options on a
	// single-choice poll.
	for _, body := range []map[string]interface{}{
		{"question": "Q?", "options": []string{"Only one"}},
		{"question": "Q?", "options": []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"}},
		{"question": "  ", "options": []string{"A", "B"}},
	} {
		if code, _ := create(aliceTok, group, body); code != http.StatusBadRequest {
			t.Fatalf("bad poll accepted: %v -> %d", body, code)
		}
	}
	if code, _ := vote(bobTok, poll.ID, []int{5}); code != http.StatusBadRequest {
		t.Fatalf("out-of-range vote: %d", code)
	}
	if code, _ := vote(bobTok, poll.ID, []int{0, 0}); code != http.StatusBadRequest {
		t.Fatalf("duplicate vote: %d", code)
	}
	if code, _ := vote(bobTok, poll.ID, []int{0, 1}); code != http.StatusBadRequest {
		t.Fatalf("two options on single-choice poll: %d", code)
	}
	// Non-member cannot create or vote; unknown poll is 404.
	if code, _ := create(daveTok, group, map[string]interface{}{
		"question": "Q?", "options": []string{"A", "B"}}); code != http.StatusForbidden {
		t.Fatalf("non-member create: %d", code)
	}
	if code, _ := vote(daveTok, poll.ID, []int{0}); code != http.StatusForbidden {
		t.Fatalf("non-member vote: %d", code)
	}
	if code, _ := vote(bobTok, "00000000-0000-0000-0000-000000000000", []int{0}); code != http.StatusNotFound {
		t.Fatalf("unknown poll vote: %d", code)
	}

	// Multiple-choice poll: Alice votes two options, then clears.
	code, multi := create(aliceTok, group, map[string]interface{}{
		"question": "What do we bring?", "options": []string{"Food", "Drinks", "Music"},
		"allow_multiple": true,
	})
	if code != http.StatusCreated {
		t.Fatalf("create multi poll: %d", code)
	}
	code, v = vote(aliceTok, multi.ID, []int{0, 2})
	if code != http.StatusOK || v.Votes[0] != 1 || v.Votes[2] != 1 || v.TotalVotes != 2 {
		t.Fatalf("multi vote: %d %+v", code, v)
	}
	code, v = vote(aliceTok, multi.ID, []int{})
	if code != http.StatusOK || v.TotalVotes != 0 {
		t.Fatalf("clear vote: %d %+v", code, v)
	}

	// List: newest first (multi poll first).
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+group+"/polls", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list polls: %d", rec.Code)
	}
	var listed []models.Poll
	mustDecode(t, rec, &listed)
	if len(listed) != 2 || listed[0].ID != multi.ID || listed[1].ID != poll.ID {
		t.Fatalf("list polls wrong: %d polls", len(listed))
	}

	// Delete: Bob (member, not creator/admin) is forbidden; Alice
	// deletes; the poll and its chat message are gone.
	rec = doJSON(t, h, http.MethodDelete, "/api/polls/"+multi.ID, nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-creator delete: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodDelete, "/api/polls/"+multi.ID, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("creator delete: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/polls/"+multi.ID, nil, aliceTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("deleted poll still readable: %d", rec.Code)
	}
	for _, m := range messages(aliceTok, group) {
		if m.Kind == "poll" && strings.Contains(m.Content, multi.ID) {
			t.Fatalf("deleted poll message still in chat flow")
		}
	}

	// --- Group pin permissions ---
	// Send a text message to pin.
	m := sendMsg(t, h, aliceTok, group, map[string]interface{}{
		"kind": "text", "content": "Hello group",
	})
	// Bob (plain member) cannot pin in a group.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+group+"/pin-message",
		map[string]interface{}{"message_id": m.ID}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member pin in group: %d", rec.Code)
	}
	// Alice (owner) pins; a system notice appears in the chat.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+group+"/pin-message",
		map[string]interface{}{"message_id": m.ID}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner pin in group: %d", rec.Code)
	}
	notice := false
	for _, mm := range messages(aliceTok, group) {
		if mm.Kind == "system" && strings.Contains(mm.Content, "pinned a message") {
			notice = true
		}
	}
	if !notice {
		t.Fatalf("no system notice after group pin")
	}
	// Bob cannot unpin in a group either; Alice can.
	rec = doJSON(t, h, http.MethodDelete, "/api/chats/"+group+"/pin-message", nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member unpin in group: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodDelete, "/api/chats/"+group+"/pin-message", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner unpin in group: %d", rec.Code)
	}
	// In a DM either member may pin (unchanged from Stage 1).
	dmMsg := sendMsg(t, h, aliceTok, dm, map[string]interface{}{
		"kind": "text", "content": "Hello DM",
	})
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dm+"/pin-message",
		map[string]interface{}{"message_id": dmMsg.ID}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("DM pin by either member: %d", rec.Code)
	}
	_ = aliceID
	_ = bobID
}
