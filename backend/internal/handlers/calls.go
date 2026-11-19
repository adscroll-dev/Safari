package handlers

import (
	"net/http"
	"time"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
)

// Stage 3 — Calls: call history.
// Endpoints (see cmd/server/main.go):
//   POST /api/calls   record a finished/missed call (caller only)
//   GET  /api/calls   my call history, newest first

// CreateCallRecord records one finished (or missed) call. The caller
// writes it when the call ends; the callee (or, for a group call,
// every member) sees the same record in their history.
func (h *Handler) CreateCallRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	me := middleware.UserID(r)
	var req struct {
		ChatID          string     `json:"chat_id"`
		CalleeID        string     `json:"callee_id,omitempty"`
		CallType        string     `json:"call_type"`
		Status          string     `json:"status"`
		DurationSeconds int        `json:"duration_seconds"`
		StartedAt       *time.Time `json:"started_at,omitempty"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.ChatID == "" {
		middleware.WriteError(w, http.StatusBadRequest, "chat_id is required")
		return
	}
	if req.CallType != "voice" && req.CallType != "video" {
		middleware.WriteError(w, http.StatusBadRequest, "call_type must be voice or video")
		return
	}
	if req.Status != "missed" && req.Status != "ended" {
		middleware.WriteError(w, http.StatusBadRequest, "status must be missed or ended")
		return
	}
	if req.DurationSeconds < 0 || req.DurationSeconds > 86400 {
		middleware.WriteError(w, http.StatusBadRequest, "duration_seconds out of range")
		return
	}
	chat, err := h.Store.ChatByID(req.ChatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	if !h.Store.IsMember(req.ChatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	// A private call always names its callee; a group call never does.
	if chat.Type == models.ChatDM && req.CalleeID == "" {
		middleware.WriteError(w, http.StatusBadRequest, "callee_id is required for a private call")
		return
	}
	if chat.Type != models.ChatDM && req.CalleeID != "" {
		middleware.WriteError(w, http.StatusBadRequest, "callee_id is only for private calls")
		return
	}
	if req.CalleeID != "" {
		if req.CalleeID == me {
			middleware.WriteError(w, http.StatusBadRequest, "cannot call yourself")
			return
		}
		if !h.Store.IsMember(req.ChatID, req.CalleeID) {
			middleware.WriteError(w, http.StatusBadRequest, "callee is not a member of this chat")
			return
		}
	}
	started := time.Now().UTC()
	if req.StartedAt != nil {
		started = req.StartedAt.UTC()
	}
	var ended *time.Time
	if req.Status == "ended" {
		e := started.Add(time.Duration(req.DurationSeconds) * time.Second)
		ended = &e
	}
	rec := &models.CallRecord{
		ID:              store.NewID(),
		ChatID:          req.ChatID,
		CallerID:        me,
		CalleeID:        req.CalleeID,
		CallType:        req.CallType,
		Status:          req.Status,
		StartedAt:       started,
		EndedAt:         ended,
		DurationSeconds: req.DurationSeconds,
	}
	if err := h.Store.CreateCallRecord(rec); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not record call")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, rec)
}

// StartCallCheck validates and reserves a private call slot atomically.
func (h *Handler) StartCallCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	me := middleware.UserID(r)
	var req struct {
		ChatID   string `json:"chat_id"`
		CalleeID string `json:"callee_id"`
		CallType string `json:"call_type"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.ChatID == "" {
		middleware.WriteError(w, http.StatusBadRequest, "chat_id is required")
		return
	}
	if req.CalleeID == "" {
		middleware.WriteError(w, http.StatusBadRequest, "callee_id is required")
		return
	}
	if req.CallType != "voice" && req.CallType != "video" {
		middleware.WriteError(w, http.StatusBadRequest, "call_type must be voice or video")
		return
	}
	if req.CalleeID == me {
		middleware.WriteError(w, http.StatusBadRequest, "cannot call yourself")
		return
	}
	chat, err := h.Store.ChatByID(req.ChatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	if chat.Type != models.ChatDM {
		middleware.WriteError(w, http.StatusBadRequest, "callee_id is only for private calls")
		return
	}
	if !h.Store.IsMember(req.ChatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	if !h.Store.IsMember(req.ChatID, req.CalleeID) {
		middleware.WriteError(w, http.StatusBadRequest, "callee is not a member of this chat")
		return
	}
	if _, err := h.Store.UserByID(req.CalleeID); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "callee not found")
		return
	}
	busyUser, ok := h.Hub.ReserveCall(me, req.CalleeID, req.ChatID)
	if !ok {
		middleware.WriteError(w, http.StatusConflict, "user is busy: "+busyUser)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListCallRecords returns my call history, newest first, enriched
// with direction, peer name and the group flag.
func (h *Handler) ListCallRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	me := middleware.UserID(r)
	recs := h.Store.ListCallRecordsForUser(me)
	views := make([]*models.CallRecordView, 0, len(recs))
	for _, rec := range recs {
		v := &models.CallRecordView{CallRecord: *rec}
		if rec.CallerID == me {
			v.Direction = "outgoing"
		} else {
			v.Direction = "incoming"
		}
		if chat, err := h.Store.ChatByID(rec.ChatID); err == nil {
			v.IsGroup = chat.Type == models.ChatGroup
			v.ChatTitle = chat.Title
		}
		if v.IsGroup {
			v.PeerName = v.ChatTitle
			if v.PeerName == "" {
				v.PeerName = "Group call"
			}
		} else {
			other := rec.CalleeID
			if other == me || other == "" {
				other = rec.CallerID
			}
			if u, err := h.Store.UserByID(other); err == nil {
				v.PeerName = u.DisplayName
				v.PeerUsername = u.Username
			} else {
				v.PeerName = "Unknown"
			}
		}
		views = append(views, v)
	}
	middleware.WriteJSON(w, http.StatusOK, views)
}
