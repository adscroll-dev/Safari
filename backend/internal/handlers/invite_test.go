package handlers

import (
	"net/http"
	"testing"
)

// TestInviteLinks covers the Telegram-style invite-link flow: an admin
// creates a link for a group, anyone can preview it and join with it,
// revoking makes the token stop working, and private chats have no
// invite links at all.
func TestInviteLinks(t *testing.T) {
	h := newTestHandler()

	// Three users, registered through the real auth flow.
	ownerTok, _ := register(t, h, "owner@example.com", "Owner One")
	memberTok, memberID := register(t, h, "member@example.com", "Member Two")
	outsiderTok, outsiderID := register(t, h, "outsider@example.com", "Outsider Three")

	// Owner creates a group with the member already in it.
	chatID := makeGroup(t, h, ownerTok, []string{memberID})

	// A plain member cannot create an invite link.
	rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/invites", nil, memberTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member create invite: expected 403, got %d %s", rec.Code, rec.Body.String())
	}

	// Owner creates one.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chatID+"/invites", nil, ownerTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create invite: expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Token  string `json:"token"`
		Link   string `json:"link"`
		ChatID string `json:"chat_id"`
	}
	mustDecode(t, rec, &created)
	if created.Token == "" || created.ChatID != chatID || created.Link == "" {
		t.Fatalf("unexpected invite: %+v", created)
	}

	// Outsider previews the link: title, type, member count.
	rec = doJSON(t, h, http.MethodGet, "/api/invites/"+created.Token, nil, outsiderTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview invite: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var preview struct {
		Title       string `json:"title"`
		Type        string `json:"type"`
		MemberCount int    `json:"member_count"`
		InvitedBy   string `json:"invited_by"`
	}
	mustDecode(t, rec, &preview)
	if preview.Title != "Test Group" || preview.Type != "group" || preview.MemberCount != 2 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.InvitedBy == "" {
		t.Fatalf("preview should name the inviter: %+v", preview)
	}

	// Outsider joins with the link.
	rec = doJSON(t, h, http.MethodPost, "/api/invites/"+created.Token+"/join", nil, outsiderTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("join via invite: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	if !h.Store.IsMember(chatID, outsiderID) {
		t.Fatal("outsider should be a member after joining via invite")
	}

	// Joining again is a conflict.
	rec = doJSON(t, h, http.MethodPost, "/api/invites/"+created.Token+"/join", nil, outsiderTok)
	if rec.Code != http.StatusConflict {
		t.Fatalf("re-join: expected 409, got %d %s", rec.Code, rec.Body.String())
	}

	// Owner lists the chat's invites and sees the one just used.
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chatID+"/invites", nil, ownerTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("list invites: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var listed []struct {
		Token   string `json:"token"`
		Revoked bool   `json:"revoked"`
	}
	mustDecode(t, rec, &listed)
	if len(listed) != 1 || listed[0].Token != created.Token || listed[0].Revoked {
		t.Fatalf("unexpected invite list: %+v", listed)
	}

	// Owner revokes the link; preview and join both fail afterwards.
	rec = doJSON(t, h, http.MethodDelete, "/api/chats/"+chatID+"/invites/"+created.Token, nil, ownerTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke invite: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/invites/"+created.Token, nil, memberTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("preview revoked: expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/invites/"+created.Token+"/join", nil, memberTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("join revoked: expected 404, got %d %s", rec.Code, rec.Body.String())
	}

	// A private (DM) chat has no invite links.
	dmID := makeDM(t, h, ownerTok, memberID)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+dmID+"/invites", nil, ownerTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("dm invite: expected 400, got %d %s", rec.Code, rec.Body.String())
	}

	// A bogus token previews as 404.
	rec = doJSON(t, h, http.MethodGet, "/api/invites/doesnotexist", nil, outsiderTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("preview bogus: expected 404, got %d %s", rec.Code, rec.Body.String())
	}
}
