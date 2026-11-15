package handlers

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
)

// Stage 7 (accounts): profile editing, user reports, unblocking.

// updateMeReq: pointer fields so an absent field means "unchanged".
type updateMeReq struct {
	DisplayName *string `json:"display_name"`
	Bio         *string `json:"bio"`
	AvatarURL   *string `json:"avatar_url"`
}

// UpdateMe serves PATCH /api/users/me: display name (1-100 runes),
// bio/about (max 500 runes), avatar URL (empty, or an /uploads/
// path — avatars are uploaded files, never remote URLs).
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateMeReq
	if !decode(w, r, &req) {
		return
	}
	if req.DisplayName == nil && req.Bio == nil && req.AvatarURL == nil {
		middleware.WriteError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if req.DisplayName != nil {
		*req.DisplayName = strings.TrimSpace(*req.DisplayName)
		if *req.DisplayName == "" || utf8.RuneCountInString(*req.DisplayName) > 100 {
			middleware.WriteError(w, http.StatusBadRequest, "display name must be 1-100 characters")
			return
		}
	}
	if req.Bio != nil && utf8.RuneCountInString(*req.Bio) > 500 {
		middleware.WriteError(w, http.StatusBadRequest, "bio must be at most 500 characters")
		return
	}
	if req.AvatarURL != nil {
		*req.AvatarURL = strings.TrimSpace(*req.AvatarURL)
		if *req.AvatarURL != "" && !strings.HasPrefix(*req.AvatarURL, "/uploads/") {
			middleware.WriteError(w, http.StatusBadRequest, "avatar must be an uploaded file")
			return
		}
	}
	me := middleware.UserID(r)
	if err := h.Store.UpdateProfile(me, req.DisplayName, req.Bio, req.AvatarURL); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	user, err := h.Store.UserByID(me)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, user.Public())
}

type reportUserReq struct {
	Reason string `json:"reason"`
}

// ReportUser serves POST /api/users/{id}/report: file an abuse/spam
// report about another user. The report is stored for review; it
// does not block the user — the client offers "report and block"
// separately.
func (h *Handler) ReportUser(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/users/")
	targetID := strings.TrimSuffix(rest, "/report")
	me := middleware.UserID(r)
	if targetID == "" || targetID == me {
		middleware.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req reportUserReq
	if !decode(w, r, &req) {
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" || utf8.RuneCountInString(req.Reason) > 500 {
		middleware.WriteError(w, http.StatusBadRequest, "reason must be 1-500 characters")
		return
	}
	rep := &models.Report{
		ReporterID: me, TargetType: "user", TargetID: targetID,
		Reason: req.Reason,
	}
	if err := h.Store.CreateReport(rep); err != nil {
		if err == store.ErrNotFound {
			middleware.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "could not save report")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, rep)
}

// UnblockUser serves DELETE /api/users/{id}/block.
func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimPrefix(r.URL.Path, "/api/users/")
	targetID = strings.TrimSuffix(targetID, "/block")
	if targetID == "" || targetID == middleware.UserID(r) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.Store.UnblockUser(middleware.UserID(r), targetID); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "unblocked"})
}

// GetBlockedUsers serves GET /api/users/me/blocked: the user IDs I
// blocked, so the client can label Block/Unblock correctly.
func (h *Handler) GetBlockedUsers(w http.ResponseWriter, r *http.Request) {
	out := h.Store.ListBlocked(middleware.UserID(r))
	if out == nil {
		out = []string{}
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}
