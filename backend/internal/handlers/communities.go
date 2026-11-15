package handlers

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
)

// Stage 6 — Communities (WhatsApp parity): a real community model
// with a name and description, an admin-only announcement group
// (a group chat created with the community), subgroups (ordinary
// group chats linked to the community) and its own membership.
// Endpoints (see cmd/server/main.go):
//   POST /api/communities                 create
//   GET  /api/communities                 list mine
//   GET  /api/communities/{id}            detail (with subgroups)
//   PATCH /api/communities/{id}           edit name/description
//   GET  /api/communities/{id}/members    list members
//   POST /api/communities/{id}/members    add a member
//   POST /api/communities/{id}/members/{userID}/role
//   DELETE /api/communities/{id}/members/{userID}
//   POST /api/communities/{id}/leave      leave
//   POST /api/communities/{id}/subgroups  link an existing group
//          or create a new subgroup ({"title": "..."})
//   DELETE /api/communities/{id}/subgroups/{chatID}

// communityIDFromPath extracts the {id} from
// /api/communities/{id}[/...].
func communityIDFromPath(r *http.Request) string {
	rest := strings.TrimPrefix(r.URL.Path, "/api/communities/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// communityForMember loads a community and requires the caller to
// be a member (403 otherwise). Returns nil when it already wrote
// an error.
func (h *Handler) communityForMember(w http.ResponseWriter, r *http.Request) *models.Community {
	id := communityIDFromPath(r)
	me := middleware.UserID(r)
	c := h.Store.CommunityByID(id)
	if c == nil {
		middleware.WriteError(w, http.StatusNotFound, "community not found")
		return nil
	}
	if h.Store.CommunityRole(id, me) == "" {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this community")
		return nil
	}
	return c
}

// communityAdminOrOwner requires the caller to be an admin or the
// owner of the community (403 otherwise). Returns false when it
// already wrote an error.
func (h *Handler) communityAdminOrOwner(w http.ResponseWriter, r *http.Request, id string) bool {
	role := h.Store.CommunityRole(id, middleware.UserID(r))
	if role != "admin" && role != "owner" {
		middleware.WriteError(w, http.StatusForbidden, "only community admins can do this")
		return false
	}
	return true
}

// CreateCommunity creates the community AND its announcement group
// (a group chat titled after the community, members = the creator).
// Only admins and the owner may post in the announcement group
// (enforced in SendMessage).
func (h *Handler) CreateCommunity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	me := middleware.UserID(r)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		middleware.WriteError(w, http.StatusBadRequest, "name is required (max 100 characters)")
		return
	}
	if utf8.RuneCountInString(req.Description) > 500 {
		middleware.WriteError(w, http.StatusBadRequest, "description is too long (max 500 characters)")
		return
	}
	announcement := &models.Chat{
		Type: models.ChatGroup, Title: name, OwnerID: me, MemberIDs: []string{me},
	}
	if err := h.Store.CreateChat(announcement); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create announcement group")
		return
	}
	community := &models.Community{
		Name: name, Description: req.Description, OwnerID: me,
		AnnouncementChatID: announcement.ID,
	}
	if err := h.Store.CreateCommunity(community); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create community")
		return
	}
	community.MemberCount = 1
	community.MyRole = "owner"
	middleware.WriteJSON(w, http.StatusCreated, community)
}

// ListCommunities returns the caller's communities.
func (h *Handler) ListCommunities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	out := h.Store.ListCommunitiesForUser(middleware.UserID(r))
	if out == nil {
		out = []*models.Community{}
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

// GetCommunity returns one community with its subgroups attached
// (subgroups are full chats, so the client can open them).
func (h *Handler) GetCommunity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	subgroups := h.Store.ListSubgroups(c.ID)
	if subgroups == nil {
		subgroups = []*models.Chat{}
	}
	members := h.Store.CommunityMembers(c.ID)
	c.MemberCount = len(members)
	c.SubgroupCount = len(subgroups)
	c.MyRole = h.Store.CommunityRole(c.ID, middleware.UserID(r))
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"community": c,
		"subgroups": subgroups,
	})
}

// UpdateCommunity edits a community's name and/or description
// (admins and the owner). Renaming also renames the announcement
// group so the two stay in sync.
func (h *Handler) UpdateCommunity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	if !h.communityAdminOrOwner(w, r, c.ID) {
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || utf8.RuneCountInString(name) > 100 {
			middleware.WriteError(w, http.StatusBadRequest, "name is required (max 100 characters)")
			return
		}
		req.Name = &name
	}
	if req.Description != nil && utf8.RuneCountInString(*req.Description) > 500 {
		middleware.WriteError(w, http.StatusBadRequest, "description is too long (max 500 characters)")
		return
	}
	if req.Name == nil && req.Description == nil {
		middleware.WriteError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if err := h.Store.UpdateCommunity(c.ID, req.Name, req.Description); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not update community")
		return
	}
	if req.Name != nil && c.AnnouncementChatID != "" {
		// Keep the announcement group's title in sync.
		_ = h.Store.UpdateChat(c.AnnouncementChatID, req.Name, nil)
	}
	updated := h.Store.CommunityByID(c.ID)
	updated.MemberCount = len(h.Store.CommunityMembers(c.ID))
	updated.SubgroupCount = len(h.Store.ListSubgroups(c.ID))
	updated.MyRole = h.Store.CommunityRole(c.ID, middleware.UserID(r))
	middleware.WriteJSON(w, http.StatusOK, updated)
}

// ListCommunityMembers lists a community's members, owner first.
func (h *Handler) ListCommunityMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	members := h.Store.CommunityMembers(c.ID)
	if members == nil {
		members = []*models.CommunityMember{}
	}
	middleware.WriteJSON(w, http.StatusOK, members)
}

// AddCommunityMember adds a user to the community AND to the
// announcement group, so they see announcements (admin/owner only).
func (h *Handler) AddCommunityMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	if !h.communityAdminOrOwner(w, r, c.ID) {
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	if _, err := h.Store.UserByID(req.UserID); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "unknown user")
		return
	}
	if err := h.Store.AddCommunityMember(c.ID, req.UserID, "member"); err != nil {
		if err == store.ErrAlreadyExists {
			middleware.WriteError(w, http.StatusConflict, "already a member")
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "could not add member")
		return
	}
	if c.AnnouncementChatID != "" {
		// Best-effort: the announcement group mirrors membership.
		_ = h.Store.AddMember(c.AnnouncementChatID, req.UserID, "member")
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// RemoveCommunityMember removes a user from the community AND from
// the announcement group (admin/owner only; the owner cannot be
// removed).
func (h *Handler) RemoveCommunityMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	if !h.communityAdminOrOwner(w, r, c.ID) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/communities/")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid member id")
		return
	}
	targetID := parts[2]
	if targetID == c.OwnerID {
		middleware.WriteError(w, http.StatusBadRequest, "the owner cannot be removed")
		return
	}
	if err := h.Store.RemoveCommunityMember(c.ID, targetID); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "not a member")
		return
	}
	if c.AnnouncementChatID != "" {
		_ = h.Store.RemoveMember(c.AnnouncementChatID, targetID)
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// SetCommunityMemberRole promotes/demotes a member to admin or back
// (admin/owner only; the owner role cannot be granted).
func (h *Handler) SetCommunityMemberRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	if !h.communityAdminOrOwner(w, r, c.ID) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/communities/")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid member id")
		return
	}
	targetID := parts[2]
	var req struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Role != "member" && req.Role != "admin" {
		middleware.WriteError(w, http.StatusBadRequest, "role must be member or admin")
		return
	}
	if err := h.Store.SetCommunityMemberRole(c.ID, targetID, req.Role); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "not a member")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// LeaveCommunity removes the caller from the community AND from
// the announcement group (the owner cannot leave).
func (h *Handler) LeaveCommunity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	me := middleware.UserID(r)
	if me == c.OwnerID {
		middleware.WriteError(w, http.StatusBadRequest, "the owner cannot leave — transfer ownership first")
		return
	}
	if err := h.Store.RemoveCommunityMember(c.ID, me); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not leave")
		return
	}
	if c.AnnouncementChatID != "" {
		_ = h.Store.RemoveMember(c.AnnouncementChatID, me)
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "left"})
}

// AddSubgroup links an existing group chat as a subgroup, or
// creates a new group ({"title": "..."}) and links it
// (admin/owner only). The linked chat must be a group, not a DM
// or a channel, and not the announcement group itself.
func (h *Handler) AddSubgroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	if !h.communityAdminOrOwner(w, r, c.ID) {
		return
	}
	me := middleware.UserID(r)
	var req struct {
		ChatID string `json:"chat_id"`
		Title  string `json:"title"`
	}
	if !decode(w, r, &req) {
		return
	}
	chatID := req.ChatID
	if chatID == "" {
		title := strings.TrimSpace(req.Title)
		if title == "" || utf8.RuneCountInString(title) > 100 {
			middleware.WriteError(w, http.StatusBadRequest, "title is required (max 100 characters)")
			return
		}
		chat := &models.Chat{
			Type: models.ChatGroup, Title: title, OwnerID: me, MemberIDs: []string{me},
		}
		if err := h.Store.CreateChat(chat); err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, "could not create subgroup")
			return
		}
		chatID = chat.ID
	}
	chat, err := h.Store.ChatByID(chatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	if chat.Type != models.ChatGroup {
		middleware.WriteError(w, http.StatusBadRequest, "only groups can be subgroups")
		return
	}
	if chatID == c.AnnouncementChatID {
		middleware.WriteError(w, http.StatusBadRequest, "the announcement group is already part of the community")
		return
	}
	if err := h.Store.AddSubgroup(c.ID, chatID); err != nil {
		if err == store.ErrAlreadyExists {
			middleware.WriteError(w, http.StatusConflict, "already a subgroup")
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "could not add subgroup")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, chat)
}

// RemoveSubgroup unlinks a subgroup (admin/owner only). The group
// chat itself is NOT deleted.
func (h *Handler) RemoveSubgroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c := h.communityForMember(w, r)
	if c == nil {
		return
	}
	if !h.communityAdminOrOwner(w, r, c.ID) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/communities/")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid subgroup id")
		return
	}
	if err := h.Store.RemoveSubgroup(c.ID, parts[2]); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "not a subgroup")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
