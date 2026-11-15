package handlers

import (
	"net/http"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// Stage 6 communities (WhatsApp parity): a real community with an
// admin-only announcement group, subgroups, member management and
// admin settings.
func TestStage6Communities(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, carolID := register(t, h, "carol@example.com", "carol")
	daveTok, _ := register(t, h, "dave@example.com", "dave")

	// Create: Alice creates a community; the announcement group
	// is created with it and she is its owner.
	rec := doJSON(t, h, http.MethodPost, "/api/communities",
		map[string]interface{}{"name": "Kabul Tech", "description": "Tech folks"}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create community: %d %s", rec.Code, rec.Body.String())
	}
	var community models.Community
	mustDecode(t, rec, &community)
	if community.AnnouncementChatID == "" || community.MyRole != "owner" {
		t.Fatalf("community fields wrong: %+v", community)
	}
	announcementID := community.AnnouncementChatID

	// Announcement group: only admins/owner may post; a plain
	// member gets 403.
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/members",
		map[string]interface{}{"user_id": bobID}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("add bob: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+announcementID+"/messages",
		map[string]interface{}{"kind": "text", "content": "hello"}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member post in announcement group: %d", rec.Code)
	}
	_ = sendMsg(t, h, aliceTok, announcementID, map[string]interface{}{
		"kind": "text", "content": "Welcome!",
	})

	// Member management: promote Bob to admin, then he can post
	// announcements; demote him again and he cannot.
	rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/members/"+bobID+"/role",
		map[string]interface{}{"role": "admin"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("promote bob: %d", rec.Code)
	}
	_ = sendMsg(t, h, bobTok, announcementID, map[string]interface{}{
		"kind": "text", "content": "Admin announcement",
	})
	rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/members/"+bobID+"/role",
		map[string]interface{}{"role": "member"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("demote bob: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+announcementID+"/messages",
		map[string]interface{}{"kind": "text", "content": "again"}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("demoted member post: %d", rec.Code)
	}
	// Non-members cannot touch the community at all.
	rec = doJSON(t, h, http.MethodGet, "/api/communities/"+community.ID, nil, daveTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("dave read community: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/members",
		map[string]interface{}{"user_id": carolID}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member add member: %d", rec.Code)
	}

	// Subgroups: create a new one, and link an existing group.
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/subgroups",
		map[string]interface{}{"title": "Announcements"}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create subgroup: %d %s", rec.Code, rec.Body.String())
	}
	var subgroup models.Chat
	mustDecode(t, rec, &subgroup)
	group := makeGroup(t, h, aliceTok, []string{bobID})
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/subgroups",
		map[string]interface{}{"chat_id": group}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("link subgroup: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/subgroups",
		map[string]interface{}{"chat_id": group}, aliceTok)
	if rec.Code != http.StatusConflict {
		t.Fatalf("link subgroup twice: %d", rec.Code)
	}
	dm := makeDM(t, h, aliceTok, bobID)
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/subgroups",
		map[string]interface{}{"chat_id": dm}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("link DM as subgroup: %d", rec.Code)
	}

	// Detail: 2 subgroups, 2 members (Alice owner + Bob member).
	rec = doJSON(t, h, http.MethodGet, "/api/communities/"+community.ID, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("get community: %d", rec.Code)
	}
	var detail struct {
		Community models.Community `json:"community"`
		Subgroups []models.Chat    `json:"subgroups"`
	}
	mustDecode(t, rec, &detail)
	if len(detail.Subgroups) != 2 || detail.Community.MemberCount != 2 {
		t.Fatalf("detail wrong: %+v %d subgroups", detail.Community, len(detail.Subgroups))
	}

	// Admin settings: rename (announcement group renames too),
	// edit description; a member cannot edit.
	rec = doJSON(t, h, http.MethodPatch, "/api/communities/"+community.ID,
		map[string]interface{}{"name": "Kabul Tech Hub", "description": "All tech folks"}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename community: %d", rec.Code)
	}
	var renamed models.Community
	mustDecode(t, rec, &renamed)
	if renamed.Name != "Kabul Tech Hub" || renamed.Description != "All tech folks" {
		t.Fatalf("rename did not stick: %+v", renamed)
	}
	annChat, _ := h.Store.ChatByID(announcementID)
	if annChat.Title != "Kabul Tech Hub" {
		t.Fatalf("announcement group not renamed: %q", annChat.Title)
	}
	rec = doJSON(t, h, http.MethodPatch, "/api/communities/"+community.ID,
		map[string]interface{}{"name": "Hacked"}, bobTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member rename: %d", rec.Code)
	}

	// Invite link: joining the announcement group via its link
	// also joins the community.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+announcementID+"/invites", nil, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create invite: %d", rec.Code)
	}
	var invite struct {
		Token string `json:"token"`
	}
	mustDecode(t, rec, &invite)
	rec = doJSON(t, h, http.MethodPost, "/api/invites/"+invite.Token+"/join", nil, carolTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("carol join via invite: %d", rec.Code)
	}
	if h.Store.CommunityRole(community.ID, carolID) != "member" {
		t.Fatalf("invite join did not add carol to the community")
	}

	// Leave: Carol leaves the community (and the announcement
	// group); the owner cannot leave.
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/leave", nil, carolTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("carol leave: %d", rec.Code)
	}
	if h.Store.CommunityRole(community.ID, carolID) != "" {
		t.Fatalf("carol still a member after leaving")
	}
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+community.ID+"/leave", nil, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("owner leave: %d", rec.Code)
	}
	// Remove member: Alice removes Bob (also from the
	// announcement group).
	rec = doJSON(t, h, http.MethodDelete, "/api/communities/"+community.ID+"/members/"+bobID, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("remove bob: %d", rec.Code)
	}
	if h.Store.IsMember(announcementID, bobID) {
		t.Fatalf("bob still in announcement group after removal")
	}
	// Unlink a subgroup: the group chat itself survives.
	rec = doJSON(t, h, http.MethodDelete, "/api/communities/"+community.ID+"/subgroups/"+group, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unlink subgroup: %d", rec.Code)
	}
	if _, err := h.Store.ChatByID(group); err != nil {
		t.Fatalf("unlink deleted the group chat")
	}
}
