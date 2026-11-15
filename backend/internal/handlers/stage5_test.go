package handlers

import (
	"net/http"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// Stage 5 channels (WhatsApp parity): view counter on channel
// posts, public channel search, follow/unfollow, channel info edits,
// and admin-only posting in channels.
func TestStage5Channels(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, _ := register(t, h, "carol@example.com", "carol")

	makeChannel := func(tok, title string, public *bool) string {
		t.Helper()
		body := map[string]interface{}{
			"type": "channel", "title": title, "member_ids": []string{},
		}
		if public != nil {
			body["public"] = *public
		}
		rec := doJSON(t, h, http.MethodPost, "/api/chats", body, tok)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create channel %q: %d %s", title, rec.Code, rec.Body.String())
		}
		var chat models.Chat
		mustDecode(t, rec, &chat)
		return chat.ID
	}
	pub := true
	priv := false
	publicCh := makeChannel(aliceTok, "Tech News", &pub)
	privateCh := makeChannel(aliceTok, "Secret Club", &priv)
	defaultCh := makeChannel(aliceTok, "Daily Tech", nil)
	group := makeGroup(t, h, aliceTok, []string{bobID})

	search := func(tok, q string) []models.Chat {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/channels/search?q="+q, nil, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("search %q: %d %s", q, rec.Code, rec.Body.String())
		}
		var out []models.Chat
		mustDecode(t, rec, &out)
		return out
	}

	// Search finds public channels by title (case-insensitive),
	// most-followed first; private channels never appear.
	found := search(bobTok, "tech")
	if len(found) != 2 {
		t.Fatalf("search tech: want 2, got %d", len(found))
	}
	titles := map[string]int{}
	for _, c := range found {
		titles[c.Title] = c.FollowerCount
	}
	if titles["Tech News"] != 1 || titles["Daily Tech"] != 1 {
		t.Fatalf("search results wrong: %v", titles)
	}
	if len(search(bobTok, "club")) != 0 {
		t.Fatalf("private channel leaked into search")
	}
	if len(search(bobTok, "TECH")) != 2 {
		t.Fatalf("search is not case-insensitive")
	}
	if len(search(bobTok, "")) != 0 {
		t.Fatalf("empty query should return nothing")
	}

	// Follow: public works, private is forbidden, groups are not
	// channels.
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+publicCh+"/follow", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("follow public channel: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+privateCh+"/follow", nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("follow private channel: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+group+"/follow", nil, bobTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("follow group: %d", rec.Code)
	}

	// Posting: followers are read-only in a channel; only admins
	// and the owner may post.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+publicCh+"/messages",
		map[string]interface{}{"kind": "text", "content": "follower post"}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("follower post in channel: %d", rec.Code)
	}
	post := sendMsg(t, h, aliceTok, publicCh, map[string]interface{}{
		"kind": "text", "content": "Welcome to Tech News!",
	})

	// View counter: one row per viewer, re-views never inflate it.
	view := func(tok, msgID string) (int, int) {
		t.Helper()
		rec := doJSON(t, h, http.MethodPost, "/api/messages/"+msgID+"/view", nil, tok)
		if rec.Code != http.StatusOK {
			return rec.Code, -1
		}
		var out map[string]int
		mustDecode(t, rec, &out)
		return rec.Code, out["view_count"]
	}
	if code, n := view(bobTok, post.ID); code != http.StatusOK || n != 1 {
		t.Fatalf("bob view: %d %d", code, n)
	}
	if code, n := view(bobTok, post.ID); code != http.StatusOK || n != 1 {
		t.Fatalf("bob re-view inflated the count: %d %d", code, n)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+publicCh+"/follow", nil, carolTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("carol follow: %d", rec.Code)
	}
	if code, n := view(carolTok, post.ID); code != http.StatusOK || n != 2 {
		t.Fatalf("carol view: %d %d", code, n)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/messages/"+post.ID+"/views", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get views: %d", rec.Code)
	}
	var views map[string]int
	mustDecode(t, rec, &views)
	if views["view_count"] != 2 {
		t.Fatalf("view count: want 2, got %v", views)
	}
	// ListMessages carries the count on channel posts.
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+publicCh+"/messages", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list messages: %d", rec.Code)
	}
	var msgs []models.Message
	mustDecode(t, rec, &msgs)
	if len(msgs) != 1 || msgs[0].ViewCount != 2 {
		t.Fatalf("message view_count: %+v", msgs)
	}
	// View counts are for channel posts only.
	dm := makeDM(t, h, aliceTok, bobID)
	dmMsg := sendMsg(t, h, aliceTok, dm, map[string]interface{}{
		"kind": "text", "content": "hi",
	})
	rec = doJSON(t, h, http.MethodPost, "/api/messages/"+dmMsg.ID+"/view", nil, bobTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DM view: %d", rec.Code)
	}

	// Channel info: followers cannot edit; the owner can rename
	// and flip the public flag (which hides it from search).
	rec = doJSON(t, h, http.MethodPatch, "/api/chats/"+publicCh,
		map[string]interface{}{"title": "Hacked"}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("follower edit: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPatch, "/api/chats/"+publicCh,
		map[string]interface{}{"title": "Tech Daily"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner rename: %d", rec.Code)
	}
	var updated models.Chat
	mustDecode(t, rec, &updated)
	if updated.Title != "Tech Daily" {
		t.Fatalf("rename did not stick: %q", updated.Title)
	}
	rec = doJSON(t, h, http.MethodPatch, "/api/chats/"+defaultCh,
		map[string]interface{}{"public": false}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner make-private: %d", rec.Code)
	}
	if len(search(bobTok, "daily")) != 1 {
		// "Tech Daily" (renamed public) still matches; "Daily Tech"
		// (now private) must not.
		t.Fatalf("private channel still searchable")
	}

	// Unfollow: Bob leaves; the owner cannot leave.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+publicCh+"/unfollow", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unfollow: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+publicCh+"/messages", nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("left channel still readable: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+publicCh+"/unfollow", nil, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("owner leave: %d", rec.Code)
	}
	_ = privateCh
}
