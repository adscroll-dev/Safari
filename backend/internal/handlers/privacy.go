package handlers

import (
	"net/http"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/store"
)

// Last-seen privacy (Telegram/WhatsApp-style, user direction
// 2026-11-14: the app must be exactly like Telegram and WhatsApp).
// A user chooses who may see their online state and last-seen time:
// "everyone" (default) or "nobody". "nobody" hides both from everyone
// except the user themself — enforced in Presence and in the
// WebSocket presence broadcast.
//
// Endpoints (see cmd/server/main.go):
//   GET /api/users/me/privacy  {"last_seen_visibility": ...}
//   PUT /api/users/me/privacy  {"last_seen_visibility": "everyone"|"nobody"}

type privacyReq struct {
	LastSeenVisibility string `json:"last_seen_visibility"`
}

// GetPrivacy serves GET /api/users/me/privacy.
func (h *Handler) GetPrivacy(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"last_seen_visibility": h.Store.GetLastSeenVisibility(me),
	})
}

// SetPrivacy serves PUT /api/users/me/privacy.
func (h *Handler) SetPrivacy(w http.ResponseWriter, r *http.Request) {
	var req privacyReq
	if !decode(w, r, &req) {
		return
	}
	if !store.ValidLastSeenVisibility(req.LastSeenVisibility) {
		middleware.WriteError(w, http.StatusBadRequest,
			"last_seen_visibility must be \"everyone\" or \"nobody\"")
		return
	}
	me := middleware.UserID(r)
	if err := h.Store.SetLastSeenVisibility(me, req.LastSeenVisibility); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError,
			"could not save privacy setting")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"last_seen_visibility": req.LastSeenVisibility,
	})
}

// presenceVisible reports whether userID's online state may be shown
// to other users at all (WebSocket broadcast gate).
func (h *Handler) presenceVisible(userID string) bool {
	return h.Store.GetLastSeenVisibility(userID) != store.LastSeenNobody
}
