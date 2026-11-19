package handlers

import (
	"net/http"
	"testing"

	"github.com/vyro/backend/internal/models"
)

func createCommunityForDiscover(t *testing.T, h *Handler, token, name, visibility, policy string) models.Community {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/communities",
		map[string]interface{}{
			"name": name, "description": name + " description",
			"visibility": visibility, "join_policy": policy,
		}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", name, rec.Code, rec.Body.String())
	}
	var community models.Community
	mustDecode(t, rec, &community)
	return community
}

func TestCommunityDiscoveryAndJoin(t *testing.T) {
	h := newTestHandler()
	aliceToken, _ := register(t, h, "discover-alice@example.com", "discoveralice")
	bobToken, bobID := register(t, h, "discover-bob@example.com", "discoverbob")

	openCommunity := createCommunityForDiscover(t, h, aliceToken, "Open Community", "public", "open")
	approvalCommunity := createCommunityForDiscover(t, h, aliceToken, "Approval Community", "public", "approval")
	inviteCommunity := createCommunityForDiscover(t, h, aliceToken, "Invite Community", "public", "invite_only")
	privateCommunity := createCommunityForDiscover(t, h, aliceToken, "Private Community", "private", "open")
	_ = privateCommunity

	// Bob has already joined this public community.
	joinedCommunity := createCommunityForDiscover(t, h, aliceToken, "Already Joined", "public", "open")
	if err := h.Store.AddCommunityMember(joinedCommunity.ID, bobID, "member"); err != nil {
		t.Fatalf("add bob to joined community: %v", err)
	}

	archivedCommunity := createCommunityForDiscover(t, h, aliceToken, "Archived Community", "public", "open")
	archivedCommunity.IsArchived = true
	_ = h.Store.CreateCommunity(&archivedCommunity)

	rec := doJSON(t, h, http.MethodGet, "/api/communities/discover?limit=50&offset=0", nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("discover: %d %s", rec.Code, rec.Body.String())
	}
	var discovered []models.Community
	mustDecode(t, rec, &discovered)
	seen := map[string]bool{}
	for _, community := range discovered {
		seen[community.ID] = true
		if community.MemberCount < 1 {
			t.Fatalf("discover community missing member count: %+v", community)
		}
	}
	for _, excluded := range []models.Community{joinedCommunity, privateCommunity, archivedCommunity} {
		if seen[excluded.ID] {
			t.Fatalf("discover included excluded community %s", excluded.ID)
		}
	}
	for _, expected := range []models.Community{openCommunity, approvalCommunity, inviteCommunity} {
		if !seen[expected.ID] {
			t.Fatalf("discover omitted community %s", expected.ID)
		}
	}

	// Pagination uses limit and offset.
	rec = doJSON(t, h, http.MethodGet, "/api/communities/discover?limit=1&offset=1", nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("paginated discover: %d", rec.Code)
	}
	var page []models.Community
	mustDecode(t, rec, &page)
	if len(page) != 1 {
		t.Fatalf("paginated discover returned %d communities", len(page))
	}

	// Open join is idempotent and adds caller to the announcement group.
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+openCommunity.ID+"/join", nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("join open: %d %s", rec.Code, rec.Body.String())
	}
	var join map[string]interface{}
	mustDecode(t, rec, &join)
	if join["status"] != "joined" {
		t.Fatalf("open join status: %v", join["status"])
	}
	if h.Store.CommunityRole(openCommunity.ID, bobID) != "member" {
		t.Fatal("open join did not add member")
	}
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+openCommunity.ID+"/join", nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("repeat open join: %d", rec.Code)
	}

	// Approval join is pending and idempotent.
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+approvalCommunity.ID+"/join", nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("join approval: %d", rec.Code)
	}
	mustDecode(t, rec, &join)
	if join["status"] != "pending" {
		t.Fatalf("approval join status: %v", join["status"])
	}
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+approvalCommunity.ID+"/join", nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("repeat approval join: %d", rec.Code)
	}
	mustDecode(t, rec, &join)
	if join["status"] != "pending" {
		t.Fatalf("repeat approval status: %v", join["status"])
	}

	// Invite-only and nonexistent communities.
	rec = doJSON(t, h, http.MethodPost, "/api/communities/"+inviteCommunity.ID+"/join", nil, bobToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("invite join: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/communities/00000000-0000-0000-0000-000000000000/join", nil, bobToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("nonexistent join: %d", rec.Code)
	}
}

func TestCommunityJoinRequestReview(t *testing.T) {
	h := newTestHandler()
	ownerToken, _ := register(t, h, "review-owner@example.com", "reviewowner")
	requesterToken, requesterID := register(t, h, "review-requester@example.com", "reviewrequester")
	otherToken, otherID := register(t, h, "review-other@example.com", "reviewother")
	community := createCommunityForDiscover(t, h, ownerToken, "Review Community", "public", "approval")

	if rec := doJSON(t, h, http.MethodGet,
		"/api/communities/"+community.ID+"/join-requests", nil, requesterToken); rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner list: %d", rec.Code)
	}
	if rec := doJSON(t, h, http.MethodGet,
		"/api/communities/00000000-0000-0000-0000-000000000000/join-requests", nil, ownerToken); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown community list: %d", rec.Code)
	}
	if rec := doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join", nil, requesterToken); rec.Code != http.StatusOK {
		t.Fatalf("request join: %d", rec.Code)
	}
	rec := doJSON(t, h, http.MethodGet,
		"/api/communities/"+community.ID+"/join-requests", nil, ownerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list requests: %d %s", rec.Code, rec.Body.String())
	}
	var requests []models.CommunityJoinRequest
	mustDecode(t, rec, &requests)
	if len(requests) != 1 || requests[0].UserID != requesterID ||
		requests[0].DisplayName != "reviewrequester" || requests[0].CreatedAt.IsZero() {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join-requests/"+requesterID+"/approve",
		nil, requesterToken); rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner approve: %d %s", rec.Code, rec.Body.String())
	}
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join-requests/"+requesterID+"/approve",
		nil, ownerToken); rec.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", rec.Code, rec.Body.String())
	}
	if h.Store.CommunityRole(community.ID, requesterID) != "member" ||
		!h.Store.IsMember(community.AnnouncementChatID, requesterID) {
		t.Fatal("approval did not mirror membership")
	}
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join-requests/"+requesterID+"/approve",
		nil, ownerToken); rec.Code != http.StatusOK {
		t.Fatalf("idempotent approve: %d", rec.Code)
	}
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join-requests/"+otherID+"/approve",
		nil, ownerToken); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown request approve: %d", rec.Code)
	}
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join", nil, otherToken); rec.Code != http.StatusOK {
		t.Fatalf("second request join: %d", rec.Code)
	}
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join-requests/"+otherID+"/reject",
		nil, ownerToken); rec.Code != http.StatusOK {
		t.Fatalf("reject: %d", rec.Code)
	}
	// Rejected requests require a fresh request before approval.
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join-requests/"+otherID+"/approve",
		nil, ownerToken); rec.Code != http.StatusConflict {
		t.Fatalf("approve rejected request: %d", rec.Code)
	}
	if rec = doJSON(t, h, http.MethodPost,
		"/api/communities/"+community.ID+"/join", nil, otherToken); rec.Code != http.StatusOK {
		t.Fatalf("re-request after reject: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet,
		"/api/communities/"+community.ID+"/join-requests", nil, ownerToken)
	mustDecode(t, rec, &requests)
	if len(requests) != 1 || requests[0].UserID != otherID {
		t.Fatalf("re-request not pending: %+v", requests)
	}
}
