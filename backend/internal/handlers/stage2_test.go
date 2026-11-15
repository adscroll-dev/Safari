package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// Stage 2 status gaps (WhatsApp parity): text/audio stories, delete
// own story, viewers, reactions, replies, status privacy.
func TestStage2Status(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, _ := register(t, h, "carol@example.com", "carol")

	createStory := func(tok string, body map[string]interface{}) (int, models.Story) {
		t.Helper()
		rec := doJSON(t, h, http.MethodPost, "/api/stories", body, tok)
		if rec.Code != http.StatusCreated {
			return rec.Code, models.Story{}
		}
		var st models.Story
		mustDecode(t, rec, &st)
		return rec.Code, st
	}
	listStories := func(tok string) []models.Story {
		t.Helper()
		rec := doJSON(t, h, http.MethodGet, "/api/stories", nil, tok)
		if rec.Code != http.StatusOK {
			t.Fatalf("list stories: %d %s", rec.Code, rec.Body.String())
		}
		var out []models.Story
		mustDecode(t, rec, &out)
		return out
	}
	findStory := func(stories []models.Story, id string) (models.Story, bool) {
		t.Helper()
		for _, st := range stories {
			if st.ID == id {
				return st, true
			}
		}
		return models.Story{}, false
	}

	// --- Text story + validation ---
	code, textStory := createStory(aliceTok, map[string]interface{}{
		"kind": "text", "text_content": "Hello status", "background_color": "#008069",
	})
	if code != http.StatusCreated {
		t.Fatalf("create text story: %d", code)
	}
	if textStory.Kind != "text" || textStory.TextContent != "Hello status" || textStory.BackgroundColor != "#008069" {
		t.Fatalf("text story = %+v", textStory)
	}
	for name, body := range map[string]map[string]interface{}{
		"empty text":      {"kind": "text", "text_content": "  "},
		"text too long":   {"kind": "text", "text_content": strings.Repeat("x", 701)},
		"bad colour":      {"kind": "text", "text_content": "hi", "background_color": "#zzzzzz"},
		"text with media": {"kind": "text", "text_content": "hi", "media_url": "/uploads/x.jpg"},
		"bad kind":        {"kind": "gif", "media_url": "/uploads/x.jpg"},
		"photo no media":  {"kind": "photo"},
	} {
		rec := doJSON(t, h, http.MethodPost, "/api/stories", body, aliceTok)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d, want 400", name, rec.Code)
		}
	}

	// --- Audio + photo stories ---
	code, audioStory := createStory(aliceTok, map[string]interface{}{
		"kind": "audio", "media_url": "/uploads/voice.ogg",
	})
	if code != http.StatusCreated {
		t.Fatalf("create audio story: %d", code)
	}
	if audioStory.Kind != "audio" {
		t.Fatalf("audio story kind = %q, want audio", audioStory.Kind)
	}
	code, photoStory := createStory(bobTok, map[string]interface{}{
		"media_url": "/uploads/pic.jpg", "caption": "hi",
	})
	if code != http.StatusCreated {
		t.Fatalf("create photo story: %d", code)
	}
	if photoStory.Kind != "photo" {
		t.Fatalf("photo story kind = %q, want photo (default)", photoStory.Kind)
	}

	// Bob sees Alice's stories; view counts stay hidden from others.
	if st, ok := findStory(listStories(bobTok), textStory.ID); !ok {
		t.Fatalf("bob cannot see alice's text story")
	} else if st.ViewCount != 0 || st.MyReaction != "" {
		t.Fatalf("bob sees view_count=%d my_reaction=%q, want 0/empty", st.ViewCount, st.MyReaction)
	}

	// --- Viewers ---
	rec := doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/view", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("view: %d %s", rec.Code, rec.Body.String())
	}
	var viewResp map[string]int
	mustDecode(t, rec, &viewResp)
	if viewResp["view_count"] != 1 {
		t.Fatalf("view_count = %d, want 1", viewResp["view_count"])
	}
	// Second view is idempotent.
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/view", nil, bobTok)
	mustDecode(t, rec, &viewResp)
	if viewResp["view_count"] != 1 {
		t.Fatalf("second view view_count = %d, want 1", viewResp["view_count"])
	}
	// Owner sees the viewer; others get 403.
	rec = doJSON(t, h, http.MethodGet, "/api/stories/"+textStory.ID+"/views", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list views: %d %s", rec.Code, rec.Body.String())
	}
	var views []models.StoryView
	mustDecode(t, rec, &views)
	if len(views) != 1 || views[0].UserID != bobID {
		t.Fatalf("views = %+v, want exactly bob", views)
	}
	// Owner's own list shows the count.
	if st, ok := findStory(listStories(aliceTok), textStory.ID); !ok || st.ViewCount != 1 {
		t.Fatalf("alice view_count = %d, want 1", st.ViewCount)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/stories/"+textStory.ID+"/views", nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner list views: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories/00000000-0000-0000-0000-000000000000/view", nil, bobTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("view unknown story: %d, want 404", rec.Code)
	}

	// --- Reactions ---
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reactions", map[string]string{"emoji": "xyz"}, bobTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad emoji: %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reactions", map[string]string{"emoji": "❤️"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("react: %d %s", rec.Code, rec.Body.String())
	}
	if st, ok := findStory(listStories(bobTok), textStory.ID); !ok || st.MyReaction != "❤️" {
		t.Fatalf("my_reaction = %q, want ❤️", st.MyReaction)
	}
	// Changing the reaction replaces it (still one reaction).
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reactions", map[string]string{"emoji": "👍"}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("re-react: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/stories/"+textStory.ID+"/reactions", nil, aliceTok)
	var reactions []models.StoryReaction
	mustDecode(t, rec, &reactions)
	if len(reactions) != 1 || reactions[0].Emoji != "👍" || reactions[0].UserID != bobID {
		t.Fatalf("reactions = %+v, want exactly bob 👍", reactions)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/stories/"+textStory.ID+"/reactions", nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner list reactions: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodDelete, "/api/stories/"+textStory.ID+"/reactions", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("remove reaction: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/stories/"+textStory.ID+"/reactions", nil, aliceTok)
	mustDecode(t, rec, &reactions)
	if len(reactions) != 0 {
		t.Fatalf("reactions after remove = %+v, want empty", reactions)
	}

	// --- Replies ---
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reply", map[string]string{"content": "Nice!"}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reply: %d %s", rec.Code, rec.Body.String())
	}
	var replyMsg models.Message
	mustDecode(t, rec, &replyMsg)
	if replyMsg.Content != "Nice!" || replyMsg.Kind != "text" {
		t.Fatalf("reply message = %+v", replyMsg)
	}
	// The reply is a real DM message to the owner.
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+replyMsg.ChatID+"/messages", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list dm messages: %d %s", rec.Code, rec.Body.String())
	}
	var msgs []models.Message
	mustDecode(t, rec, &msgs)
	if len(msgs) != 1 || msgs[0].Content != "Nice!" {
		t.Fatalf("dm messages = %+v, want the reply", msgs)
	}
	// And it is listed on the owner's status.
	rec = doJSON(t, h, http.MethodGet, "/api/stories/"+textStory.ID+"/replies", nil, aliceTok)
	var replies []models.StoryReply
	mustDecode(t, rec, &replies)
	if len(replies) != 1 || replies[0].Content != "Nice!" || replies[0].UserID != bobID {
		t.Fatalf("replies = %+v, want bob's reply", replies)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/stories/"+textStory.ID+"/replies", nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner list replies: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reply", map[string]string{"content": "mine"}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reply to own story: %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reply", map[string]string{"content": "  "}, bobTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty reply: %d, want 400", rec.Code)
	}

	// --- Delete own story ---
	rec = doJSON(t, h, http.MethodDelete, "/api/stories/"+audioStory.ID, nil, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner delete: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodDelete, "/api/stories/"+audioStory.ID, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete own story: %d %s", rec.Code, rec.Body.String())
	}
	if _, ok := findStory(listStories(aliceTok), audioStory.ID); ok {
		t.Fatalf("deleted story still listed")
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+audioStory.ID+"/view", nil, bobTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("view deleted story: %d, want 404", rec.Code)
	}

	// --- Status privacy ---
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/status-privacy", map[string]string{"status_privacy": "contacts"}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad status privacy: %d, want 400", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/status-privacy", map[string]string{"status_privacy": "nobody"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("set status privacy: %d %s", rec.Code, rec.Body.String())
	}
	// Bob no longer sees Alice's stories at all.
	if _, ok := findStory(listStories(bobTok), textStory.ID); ok {
		t.Fatalf("bob still sees alice's story after nobody")
	}
	// Carol cannot view, react or reply either.
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/view", nil, carolTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("view private story: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reactions", map[string]string{"emoji": "❤️"}, carolTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("react to private story: %d, want 403", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/stories/"+textStory.ID+"/reply", map[string]string{"content": "hi"}, carolTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("reply to private story: %d, want 403", rec.Code)
	}
	// Alice still sees her own stories.
	if _, ok := findStory(listStories(aliceTok), textStory.ID); !ok {
		t.Fatalf("alice cannot see her own story after nobody")
	}
	// Back to everyone: Bob sees it again.
	rec = doJSON(t, h, http.MethodPut, "/api/users/me/status-privacy", map[string]string{"status_privacy": "everyone"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset status privacy: %d %s", rec.Code, rec.Body.String())
	}
	if _, ok := findStory(listStories(bobTok), textStory.ID); !ok {
		t.Fatalf("bob cannot see alice's story after everyone")
	}
	rec = doJSON(t, h, http.MethodGet, "/api/users/me/status-privacy", nil, aliceTok)
	var priv map[string]string
	mustDecode(t, rec, &priv)
	if priv["status_privacy"] != "everyone" {
		t.Fatalf("status_privacy = %q, want everyone", priv["status_privacy"])
	}
}
