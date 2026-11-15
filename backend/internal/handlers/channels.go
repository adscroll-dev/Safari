package handlers

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
)

// Stage 5 — Channels (WhatsApp parity): view counter on channel
// posts, public channel search, follow/unfollow, channel info edits.
// Endpoints (see cmd/server/main.go):
//   GET  /api/channels/search?q=     search public channels
//   POST /api/chats/{id}/follow      follow a public channel
//   POST /api/chats/{id}/unfollow    leave a channel or group
//   PATCH /api/chats/{id}            edit title / public (admin/owner)
//   POST /api/messages/{id}/view     record my view of a channel post
//   GET  /api/messages/{id}/views    view count of a channel post

// SearchChannels finds public channels by title. Any signed-in user
// may search; the follower count comes from the store.
func (h *Handler) SearchChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" || utf8.RuneCountInString(q) > 100 {
		middleware.WriteJSON(w, http.StatusOK, []*models.Chat{})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, h.Store.SearchPublicChannels(q))
}

// FollowChannel joins a public channel as a follower. Private
// channels need an invite link instead (403).
func (h *Handler) FollowChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	chat, err := h.Store.ChatByID(chatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	if chat.Type != models.ChatChannel {
		middleware.WriteError(w, http.StatusBadRequest, "only channels can be followed")
		return
	}
	if h.Store.IsMember(chatID, me) {
		middleware.WriteJSON(w, http.StatusOK, chat)
		return
	}
	if !chat.IsPublic {
		middleware.WriteError(w, http.StatusForbidden, "this channel is private — use an invite link to join")
		return
	}
	if err := h.Store.AddMember(chatID, me, "member"); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not follow channel")
		return
	}
	chat, _ = h.Store.ChatByID(chatID)
	middleware.WriteJSON(w, http.StatusOK, chat)
}

// UnfollowChannel leaves a channel or a group (the owner cannot
// leave — ownership must be transferred first, which is not built
// yet). DMs cannot be left.
func (h *Handler) UnfollowChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	chat, err := h.Store.ChatByID(chatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	if chat.Type == models.ChatDM {
		middleware.WriteError(w, http.StatusBadRequest, "a DM cannot be left")
		return
	}
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	if h.Store.MemberRole(chatID, me) == "owner" {
		middleware.WriteError(w, http.StatusBadRequest, "the owner cannot leave — transfer ownership first")
		return
	}
	if err := h.Store.RemoveMember(chatID, me); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not leave")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "left"})
}

// UpdateChat edits a chat's title and (channels only) its public
// flag. Only admins and the owner may edit.
func (h *Handler) UpdateChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	if !h.requireRole(w, r, chatID, "admin", "owner") {
		return
	}
	var req struct {
		Title  *string `json:"title"`
		Public *bool   `json:"public"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" || utf8.RuneCountInString(t) > 100 {
			middleware.WriteError(w, http.StatusBadRequest, "title is required (max 100 characters)")
			return
		}
		req.Title = &t
	}
	if req.Title == nil && req.Public == nil {
		middleware.WriteError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if err := h.Store.UpdateChat(chatID, req.Title, req.Public); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not update chat")
		return
	}
	chat, _ := h.Store.ChatByID(chatID)
	middleware.WriteJSON(w, http.StatusOK, chat)
}

// RecordMessageView records my view of a channel post. One row per
// viewer, so re-views never inflate the count.
func (h *Handler) RecordMessageView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := messageIDFromPath(r)
	me := middleware.UserID(r)
	msg, err := h.Store.MessageByID(id)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	chat, err := h.Store.ChatByID(msg.ChatID)
	if err != nil || chat.Type != models.ChatChannel {
		middleware.WriteError(w, http.StatusBadRequest, "view counts are for channel posts only")
		return
	}
	if !h.Store.IsMember(msg.ChatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	if err := h.Store.RecordMessageView(id, me); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not record view")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]int{"view_count": h.Store.MessageViewCount(id)})
}

// GetMessageViews returns a channel post's view count.
func (h *Handler) GetMessageViews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := messageIDFromPath(r)
	me := middleware.UserID(r)
	msg, err := h.Store.MessageByID(id)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	chat, err := h.Store.ChatByID(msg.ChatID)
	if err != nil || chat.Type != models.ChatChannel {
		middleware.WriteError(w, http.StatusBadRequest, "view counts are for channel posts only")
		return
	}
	if !h.Store.IsMember(msg.ChatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]int{"view_count": h.Store.MessageViewCount(id)})
}
