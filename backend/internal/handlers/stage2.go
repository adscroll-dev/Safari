package handlers

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
	"github.com/vyro/backend/internal/ws"
)

// Stage 2 — Status (Updates) gaps: text/audio stories (CreateStory in
// v04.go), delete own story, viewers, reactions, replies, per-user
// status privacy.
//
// Endpoints (see cmd/server/main.go):
//   DELETE /api/stories/{id}                 delete my own story
//   POST   /api/stories/{id}/view            record a view (idempotent)
//   GET    /api/stories/{id}/views           viewers list (owner only)
//   GET    /api/stories/{id}/reactions       reactions (owner only)
//   POST   /api/stories/{id}/reactions       react (one emoji per user)
//   DELETE /api/stories/{id}/reactions       remove my reaction
//   GET    /api/stories/{id}/replies         replies (owner only)
//   POST   /api/stories/{id}/reply           reply (also a DM message)
//   GET    /api/users/me/status-privacy      {"status_privacy": ...}
//   PUT    /api/users/me/status-privacy      {"status_privacy": "everyone"|"nobody"|"except"}

// storyBackgroundPalette is the fixed set of background colours a
// text status may use (WhatsApp-style). The client offers the same list.
var storyBackgroundPalette = []string{
	"#008069", "#25D366", "#7C3AED", "#DC2626", "#EA580C",
	"#CA8A04", "#0EA5E9", "#EC4899", "#111B21",
}

func validStoryBackgroundColor(c string) bool {
	for _, p := range storyBackgroundPalette {
		if c == p {
			return true
		}
	}
	return false
}

// storyReactionEmojis is the fixed set of reactions a status may get.
var storyReactionEmojis = []string{"❤️", "😂", "😮", "😢", "🙏", "👍", "🔥", "😍"}

func validStoryReactionEmoji(e string) bool {
	for _, p := range storyReactionEmojis {
		if e == p {
			return true
		}
	}
	return false
}

func storyIDFromPath(r *http.Request) string {
	// /api/stories/{id}[/suffix]
	rest := strings.TrimPrefix(r.URL.Path, "/api/stories/")
	parts := strings.SplitN(rest, "/", 2)
	return parts[0]
}

func StorySuffix(r *http.Request) string {
	rest := strings.TrimPrefix(r.URL.Path, "/api/stories/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// storyForViewer loads a story for an interaction (view/react/reply):
// 404 when missing or expired, 403 when the owner hid their status
// from the requester.
func (h *Handler) storyForViewer(w http.ResponseWriter, r *http.Request, id string) (*models.Story, bool) {
	st := h.Store.StoryByID(id)
	if st == nil || st.ExpiresAt.Before(time.Now().UTC()) {
		middleware.WriteError(w, http.StatusNotFound, "story not found")
		return nil, false
	}
	me := middleware.UserID(r)
	if h.statusHiddenFrom(st.UserID, me) {
		middleware.WriteError(w, http.StatusForbidden, "this status is private")
		return nil, false
	}
	return st, true
}

func (h *Handler) statusHiddenFrom(ownerID, viewerID string) bool {
	if ownerID == viewerID {
		return false
	}
	switch h.Store.GetStatusPrivacy(ownerID) {
	case store.StatusNobody:
		return true
	case store.StatusExcept:
		return h.Store.IsStatusExcluded(ownerID, viewerID)
	default:
		return false
	}
}

// DeleteStory deletes the requester's own story (and its views,
// reactions and replies). Anyone else gets 403.
func (h *Handler) DeleteStory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := storyIDFromPath(r)
	st := h.Store.StoryByID(id)
	if st == nil {
		middleware.WriteError(w, http.StatusNotFound, "story not found")
		return
	}
	if st.UserID != middleware.UserID(r) {
		middleware.WriteError(w, http.StatusForbidden, "only the owner can delete a story")
		return
	}
	if err := h.Store.DeleteStory(id); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not delete story")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// RecordStoryView records that the requester viewed a story.
// Viewing twice keeps the first viewed_at.
func (h *Handler) RecordStoryView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := storyIDFromPath(r)
	if _, ok := h.storyForViewer(w, r, id); !ok {
		return
	}
	if err := h.Store.RecordStoryView(id, middleware.UserID(r)); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not record view")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]int{"view_count": h.Store.StoryViewCount(id)})
}

// ListStoryViews returns who viewed a story — owner only.
func (h *Handler) ListStoryViews(w http.ResponseWriter, r *http.Request) {
	id := storyIDFromPath(r)
	st := h.Store.StoryByID(id)
	if st == nil || st.ExpiresAt.Before(time.Now().UTC()) {
		middleware.WriteError(w, http.StatusNotFound, "story not found")
		return
	}
	if st.UserID != middleware.UserID(r) {
		middleware.WriteError(w, http.StatusForbidden, "only the owner can see viewers")
		return
	}
	views := h.Store.ListStoryViews(id)
	if views == nil {
		views = []*models.StoryView{}
	}
	middleware.WriteJSON(w, http.StatusOK, views)
}

// ListStoryReactions returns a story's reactions — owner only.
func (h *Handler) ListStoryReactions(w http.ResponseWriter, r *http.Request) {
	id := storyIDFromPath(r)
	st := h.Store.StoryByID(id)
	if st == nil || st.ExpiresAt.Before(time.Now().UTC()) {
		middleware.WriteError(w, http.StatusNotFound, "story not found")
		return
	}
	if st.UserID != middleware.UserID(r) {
		middleware.WriteError(w, http.StatusForbidden, "only the owner can see reactions")
		return
	}
	reactions := h.Store.ListStoryReactions(id)
	if reactions == nil {
		reactions = []*models.StoryReaction{}
	}
	middleware.WriteJSON(w, http.StatusOK, reactions)
}

type storyReactionReq struct {
	Emoji string `json:"emoji"`
}

// SetStoryReaction records (or replaces) the requester's emoji
// reaction to a story.
func (h *Handler) SetStoryReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := storyIDFromPath(r)
	if _, ok := h.storyForViewer(w, r, id); !ok {
		return
	}
	var req storyReactionReq
	if !decode(w, r, &req) {
		return
	}
	if !validStoryReactionEmoji(req.Emoji) {
		middleware.WriteError(w, http.StatusBadRequest, "emoji is not an allowed story reaction")
		return
	}
	if err := h.Store.SetStoryReaction(id, middleware.UserID(r), req.Emoji); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not save reaction")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"emoji": req.Emoji})
}

// DeleteStoryReaction removes the requester's reaction, if any.
func (h *Handler) DeleteStoryReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := storyIDFromPath(r)
	if _, ok := h.storyForViewer(w, r, id); !ok {
		return
	}
	if err := h.Store.DeleteStoryReaction(id, middleware.UserID(r)); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not remove reaction")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// ListStoryReplies returns a story's text replies — owner only.
func (h *Handler) ListStoryReplies(w http.ResponseWriter, r *http.Request) {
	id := storyIDFromPath(r)
	st := h.Store.StoryByID(id)
	if st == nil || st.ExpiresAt.Before(time.Now().UTC()) {
		middleware.WriteError(w, http.StatusNotFound, "story not found")
		return
	}
	if st.UserID != middleware.UserID(r) {
		middleware.WriteError(w, http.StatusForbidden, "only the owner can see replies")
		return
	}
	replies := h.Store.ListStoryReplies(id)
	if replies == nil {
		replies = []*models.StoryReply{}
	}
	middleware.WriteJSON(w, http.StatusOK, replies)
}

type storyReplyReq struct {
	Content string `json:"content"`
}

// ReplyToStory replies to a status. Like WhatsApp, the reply is
// delivered as a normal DM message to the story owner (so they get
// notified in chat) and is also listed on the owner's status.
func (h *Handler) ReplyToStory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := storyIDFromPath(r)
	st, ok := h.storyForViewer(w, r, id)
	if !ok {
		return
	}
	me := middleware.UserID(r)
	if st.UserID == me {
		middleware.WriteError(w, http.StatusBadRequest, "you cannot reply to your own status")
		return
	}
	var req storyReplyReq
	if !decode(w, r, &req) {
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		middleware.WriteError(w, http.StatusBadRequest, "content is required")
		return
	}
	if utf8.RuneCountInString(content) > 1000 {
		middleware.WriteError(w, http.StatusBadRequest, "reply too long (max 1000 characters)")
		return
	}
	if h.Store.IsBlocked(st.UserID, me) || h.Store.IsBlocked(me, st.UserID) {
		middleware.WriteError(w, http.StatusForbidden, "user is blocked")
		return
	}
	// Find the existing DM with the owner, or create one.
	dmID := ""
	for _, c := range h.Store.ChatsForUser(me) {
		if c.Type != models.ChatDM || len(c.MemberIDs) != 2 {
			continue
		}
		if (c.MemberIDs[0] == me && c.MemberIDs[1] == st.UserID) ||
			(c.MemberIDs[0] == st.UserID && c.MemberIDs[1] == me) {
			dmID = c.ID
			break
		}
	}
	if dmID == "" {
		chat := &models.Chat{Type: models.ChatDM, OwnerID: me, MemberIDs: []string{me, st.UserID}}
		if err := h.Store.CreateChat(chat); err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, "could not open chat")
			return
		}
		dmID = chat.ID
	}
	msg := &models.Message{ChatID: dmID, SenderID: me, Content: content, Kind: "text"}
	if err := h.Store.CreateMessage(msg); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not send reply")
		return
	}
	msg.Status = "sent"
	delivered := h.broadcastToChatAs(dmID, me, ws.Event{Type: "message", ChatID: dmID, Message: msg})
	for _, uid := range delivered {
		_ = h.Store.MarkDelivered(msg.ID, uid)
	}
	if len(delivered) > 0 {
		msg.DeliveredTo = delivered
		msg.Status = "delivered"
	}
	h.Hub.BroadcastToUser(me, ws.Event{Type: "message_sync", ChatID: dmID, Message: msg})
	h.notifyOfflinePush(r.Context(), dmID, me, "text", delivered)
	if _, err := h.Store.AddStoryReply(id, me, content); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not save reply")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, msg)
}

type statusPrivacyReq struct {
	StatusPrivacy string `json:"status_privacy"`
}

// GetStatusPrivacy serves GET /api/users/me/status-privacy.
func (h *Handler) GetStatusPrivacy(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"status_privacy": h.Store.GetStatusPrivacy(me),
	})
}

// SetStatusPrivacy serves PUT /api/users/me/status-privacy.
func (h *Handler) SetStatusPrivacy(w http.ResponseWriter, r *http.Request) {
	var req statusPrivacyReq
	if !decode(w, r, &req) {
		return
	}
	if !store.ValidStatusPrivacy(req.StatusPrivacy) {
		middleware.WriteError(w, http.StatusBadRequest,
			"status_privacy must be \"everyone\", \"nobody\", or \"except\"")
		return
	}
	me := middleware.UserID(r)
	if err := h.Store.SetStatusPrivacy(me, req.StatusPrivacy); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError,
			"could not save status privacy setting")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"status_privacy": req.StatusPrivacy,
	})
}

func (h *Handler) StatusExclusions(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	switch r.Method {
	case http.MethodGet:
		exclusions, err := h.Store.ListStatusExclusions(me)
		if err != nil {
			if err == store.ErrNotFound {
				middleware.WriteError(w, http.StatusNotFound, "user not found")
				return
			}
			middleware.WriteError(w, http.StatusInternalServerError, "could not load status exclusions")
			return
		}
		middleware.WriteJSON(w, http.StatusOK, exclusions)
	case http.MethodPost:
		var req struct {
			UserID string `json:"user_id"`
		}
		if !decode(w, r, &req) {
			return
		}
		if req.UserID == "" {
			middleware.WriteError(w, http.StatusBadRequest, "user_id is required")
			return
		}
		before, err := h.Store.ListStatusExclusions(me)
		if err != nil {
			if err == store.ErrNotFound {
				middleware.WriteError(w, http.StatusNotFound, "user not found")
				return
			}
			middleware.WriteError(w, http.StatusInternalServerError, "could not load status exclusions")
			return
		}
		alreadyExcluded := false
		for _, exclusion := range before {
			if exclusion.UserID == req.UserID {
				alreadyExcluded = true
				break
			}
		}
		if err := h.Store.AddStatusExclusion(me, req.UserID); err != nil {
			switch err {
			case store.ErrNotFound:
				middleware.WriteError(w, http.StatusNotFound, "user not found")
			case store.ErrForbidden:
				middleware.WriteError(w, http.StatusBadRequest, "you cannot exclude yourself")
			default:
				middleware.WriteError(w, http.StatusInternalServerError, "could not add status exclusion")
			}
			return
		}
		after, err := h.Store.ListStatusExclusions(me)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, "could not load status exclusions")
			return
		}
		for _, exclusion := range after {
			if exclusion.UserID == req.UserID {
				code := http.StatusCreated
				if alreadyExcluded {
					code = http.StatusOK
				}
				middleware.WriteJSON(w, code, exclusion)
				return
			}
		}
		middleware.WriteError(w, http.StatusInternalServerError, "could not load added status exclusion")
	case http.MethodDelete:
		userID := strings.TrimPrefix(r.URL.Path, "/api/users/me/status-exclusions/")
		if userID == r.URL.Path || userID == "" || strings.Contains(userID, "/") {
			middleware.WriteError(w, http.StatusBadRequest, "user_id is required")
			return
		}
		if err := h.Store.RemoveStatusExclusion(me, userID); err != nil {
			if err == store.ErrNotFound {
				middleware.WriteError(w, http.StatusNotFound, "user not found")
				return
			}
			middleware.WriteError(w, http.StatusInternalServerError, "could not remove status exclusion")
			return
		}
		middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
