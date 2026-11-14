// FCM (Firebase Cloud Messaging) handlers: the mobile app registers its
// device token here after login, so the server can wake the app when a
// message arrives while it is closed. See internal/push/fcm.go for the
// sender and the privacy rules (payloads never contain message content).
package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/push"
)

// maxFCMTokenLen caps a registration token: real FCM tokens are ~150
// characters; anything far longer is not a token.
const maxFCMTokenLen = 4096

// RegisterFCMToken serves POST /api/push/fcm-token with
// {"token": "...", "device_id": "...", "platform": "android"|"ios"}.
// The token is stored per user and used only for that user's own
// notifications. Answers 503 when FCM is not configured on the server.
func (h *Handler) RegisterFCMToken(w http.ResponseWriter, r *http.Request) {
	if h.FCM == nil || !h.FCM.Available() {
		middleware.WriteError(w, http.StatusServiceUnavailable, "FCM push not configured on this server")
		return
	}
	var req struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
		Platform string `json:"platform"`
	}
	if !decode(w, r, &req) {
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.Platform = strings.TrimSpace(strings.ToLower(req.Platform))
	if req.Token == "" || len(req.Token) > maxFCMTokenLen {
		middleware.WriteError(w, http.StatusBadRequest, "a valid FCM token is required")
		return
	}
	if req.Platform != "android" && req.Platform != "ios" {
		req.Platform = "android"
	}
	h.FCM.Add(middleware.UserID(r), push.FCMToken{
		Token:    req.Token,
		DeviceID: req.DeviceID,
		Platform: req.Platform,
	})
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": "registered",
		"tokens": h.FCM.Count(middleware.UserID(r)),
	})
}

// UnregisterFCMToken serves DELETE /api/push/fcm-token with
// {"token": "..."}: removes one token (e.g. on logout).
func (h *Handler) UnregisterFCMToken(w http.ResponseWriter, r *http.Request) {
	if h.FCM == nil || !h.FCM.Available() {
		middleware.WriteError(w, http.StatusServiceUnavailable, "FCM push not configured on this server")
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.FCM.Remove(middleware.UserID(r), strings.TrimSpace(req.Token))
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": "unregistered",
		"tokens": h.FCM.Count(middleware.UserID(r)),
	})
}

// notifyOfflineFCM sends the same content-free notification as the Web
// Push path, through FCM, to the members who did not receive the
// message over WebSocket. data carries the chat id so the app can open
// the right chat on tap. Best effort: failures never block delivery.
func (h *Handler) notifyOfflineFCM(ctx context.Context, chatID, senderID, kind string, delivered []string) {
	if h.FCM == nil || !h.FCM.Available() {
		return
	}
	online := map[string]bool{}
	for _, id := range delivered {
		online[id] = true
	}
	var body string
	if kind == "e2ee" {
		body = "New encrypted message"
	} else {
		senderName := "Someone"
		if u, err := h.Store.UserByID(senderID); err == nil && u.DisplayName != "" {
			senderName = u.DisplayName
		} else if u != nil && u.Username != "" {
			senderName = u.Username
		}
		body = "New message from " + senderName
	}
	for _, memberID := range h.chatMemberIDs(chatID) {
		if memberID == senderID || online[memberID] {
			continue
		}
		h.FCM.SendToUser(ctx, memberID, "VYRO", body, map[string]string{
			"chat_id": chatID,
			"type":    "message",
		})
	}
}
