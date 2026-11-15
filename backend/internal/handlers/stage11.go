package handlers

import (
	"net/http"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/store"
)

// Stage 11 (Settings parity, WhatsApp-style): one combined
// privacy-settings endpoint so the Settings page fetches everything
// at once, plus account deletion. The older single-setting
// endpoints (/api/users/me/privacy, /api/users/me/status-privacy)
// keep working; this endpoint covers all of them together.
//
//   GET /api/users/me/privacy-settings
//   PUT /api/users/me/privacy-settings  (partial update: only the
//     fields present are changed)
//   DELETE /api/users/me  (delete the account and everything it owns)

type privacySettingsResp struct {
	LastSeenVisibility string `json:"last_seen_visibility"`
	StatusPrivacy      string `json:"status_privacy"`
	AvatarPrivacy      string `json:"avatar_privacy"`
	AboutPrivacy       string `json:"about_privacy"`
	GroupsPrivacy      string `json:"groups_privacy"`
	ReadReceipts       bool   `json:"read_receipts"`
}

type privacySettingsReq struct {
	LastSeenVisibility *string `json:"last_seen_visibility"`
	StatusPrivacy      *string `json:"status_privacy"`
	AvatarPrivacy      *string `json:"avatar_privacy"`
	AboutPrivacy       *string `json:"about_privacy"`
	GroupsPrivacy      *string `json:"groups_privacy"`
	ReadReceipts       *bool   `json:"read_receipts"`
}

func (h *Handler) privacySettings(me string) privacySettingsResp {
	return privacySettingsResp{
		LastSeenVisibility: h.Store.GetLastSeenVisibility(me),
		StatusPrivacy:      h.Store.GetStatusPrivacy(me),
		AvatarPrivacy:      h.Store.GetAvatarPrivacy(me),
		AboutPrivacy:       h.Store.GetAboutPrivacy(me),
		GroupsPrivacy:      h.Store.GetGroupsPrivacy(me),
		ReadReceipts:       h.Store.GetReadReceipts(me),
	}
}

// GetPrivacySettings serves GET /api/users/me/privacy-settings.
func (h *Handler) GetPrivacySettings(w http.ResponseWriter, r *http.Request) {
	middleware.WriteJSON(w, http.StatusOK, h.privacySettings(middleware.UserID(r)))
}

// SetPrivacySettings serves PUT /api/users/me/privacy-settings.
func (h *Handler) SetPrivacySettings(w http.ResponseWriter, r *http.Request) {
	var req privacySettingsReq
	if !decode(w, r, &req) {
		return
	}
	if req.LastSeenVisibility != nil && !store.ValidLastSeenVisibility(*req.LastSeenVisibility) {
		middleware.WriteError(w, http.StatusBadRequest, "last_seen_visibility must be \"everyone\" or \"nobody\"")
		return
	}
	if req.StatusPrivacy != nil && !store.ValidStatusPrivacy(*req.StatusPrivacy) {
		middleware.WriteError(w, http.StatusBadRequest, "status_privacy must be \"everyone\" or \"nobody\"")
		return
	}
	for field, v := range map[string]*string{
		"avatar_privacy": req.AvatarPrivacy, "about_privacy": req.AboutPrivacy,
		"groups_privacy": req.GroupsPrivacy,
	} {
		if v != nil && !store.ValidPrivacyValue(*v) {
			middleware.WriteError(w, http.StatusBadRequest, field+" must be \"everyone\" or \"nobody\"")
			return
		}
	}
	me := middleware.UserID(r)
	save := func(set func() error) bool {
		if err := set(); err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, "could not save privacy setting")
			return false
		}
		return true
	}
	if req.LastSeenVisibility != nil && !save(func() error {
		return h.Store.SetLastSeenVisibility(me, *req.LastSeenVisibility)
	}) {
		return
	}
	if req.StatusPrivacy != nil && !save(func() error {
		return h.Store.SetStatusPrivacy(me, *req.StatusPrivacy)
	}) {
		return
	}
	if req.AvatarPrivacy != nil && !save(func() error {
		return h.Store.SetAvatarPrivacy(me, *req.AvatarPrivacy)
	}) {
		return
	}
	if req.AboutPrivacy != nil && !save(func() error {
		return h.Store.SetAboutPrivacy(me, *req.AboutPrivacy)
	}) {
		return
	}
	if req.GroupsPrivacy != nil && !save(func() error {
		return h.Store.SetGroupsPrivacy(me, *req.GroupsPrivacy)
	}) {
		return
	}
	if req.ReadReceipts != nil && !save(func() error {
		return h.Store.SetReadReceipts(me, *req.ReadReceipts)
	}) {
		return
	}
	middleware.WriteJSON(w, http.StatusOK, h.privacySettings(me))
}

// DeleteMe serves DELETE /api/users/me: deletes the account and
// everything it owns (messages, memberships, stories, polls, call
// records, devices, tokens). The token keeps validating (auth
// checks the signature only) but every lookup answers 404, so it
// yields no data and no action.
func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	if err := h.Store.DeleteUser(me); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
