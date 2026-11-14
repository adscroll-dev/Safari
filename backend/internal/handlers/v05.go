// VYRO v0.5 handlers: E2EE key bundles and Web Push (VAPID).
//
// Key bundle: a client starting an encrypted DM fetches the peer's bundle
// (identity key + signed pre-key + one claimed one-time pre-key) and runs
// X3DH locally (internal/e2ee). The server never sees any private key.
//
// Web Push: free, standards-based push for the web client (RFC 8030 /
// RFC 8291, VAPID RFC 8292). No FCM/APNs account, no paid service. Push
// bodies NEVER contain message content — only "New message from <name>"
// (or "New encrypted message" for E2EE), so a push service cannot read
// chat content. Subscriptions are persisted in the store (v0.6), so a
// server restart no longer drops them.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vyro/backend/internal/e2ee"
	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/push"
)

// GetKeyBundle serves GET /api/keys/{userID}/bundle (device "default")
// and GET /api/keys/{userID}/{deviceID}/bundle (v0.6 multi-device):
// the full X3DH key bundle for one device of the peer. Claiming a
// one-time pre-key consumes it (the store hands each one out once);
// the claimed pre-key always belongs to the same device as the bundle
// (FetchOneTimePreKeyForDevice), otherwise the session it builds could
// not be opened by that device. When none are left the bundle is
// returned with one_time_prekey: null and the client runs X3DH without DH4.
func (h *Handler) GetKeyBundle(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/keys/")
	parts := strings.Split(rest, "/")
	deviceID := "default"
	var userID string
	switch {
	case len(parts) == 2 && parts[1] == "bundle":
		userID = parts[0]
	case len(parts) == 3 && parts[2] == "bundle":
		userID, deviceID = parts[0], parts[1]
	default:
		middleware.WriteError(w, http.StatusBadRequest, "invalid key path")
		return
	}
	k, err := h.Store.GetIdentityKey(userID, deviceID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "identity key not found for this user")
		return
	}
	b := &e2ee.Bundle{
		UserID:   k.UserID,
		DeviceID: k.DeviceID,
		DHPub:    k.IdentityKey,
		SignPub:  k.SignKey,
		SPKPub:   k.SignedPreKey,
		SPKID:    k.SignedPreKeyID,
		SPKSig:   k.SignedPreKeySig,
	}
	if opk, err := h.Store.FetchOneTimePreKeyForDevice(userID, deviceID); err == nil && opk != nil {
		b.OneTimePreKey = &e2ee.OPK{ID: opk.KeyID, Pub: opk.PublicKey}
	}
	middleware.WriteJSON(w, http.StatusOK, b)
}

// VAPIDPublicKey serves GET /api/push/vapid-public-key so the web client
// can subscribe. When no VAPID key is configured the endpoint reports
// push as unavailable instead of failing the client.
func (h *Handler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	if h.VAPIDPublicKey == "" {
		middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"available": false,
			"error":     "push not configured on this server",
		})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"available":   true,
		"public_key":  h.VAPIDPublicKey,
		"subject":     h.VAPIDSubject,
		"subscribers": h.pushCount(middleware.UserID(r)),
	})
}

// SubscribePush serves POST /api/push/subscribe. The body is the browser
// PushSubscription JSON: {endpoint, keys:{p256dh, auth}} (flat
// p256dh/auth fields are also accepted).
func (h *Handler) SubscribePush(w http.ResponseWriter, r *http.Request) {
	if h.Push == nil {
		middleware.WriteError(w, http.StatusServiceUnavailable, "push not configured on this server")
		return
	}
	var req struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	}
	if !decode(w, r, &req) {
		return
	}
	sub := push.Subscription{Endpoint: req.Endpoint, P256dh: req.Keys.P256dh, Auth: req.Keys.Auth}
	if sub.P256dh == "" {
		sub.P256dh = req.P256dh
	}
	if sub.Auth == "" {
		sub.Auth = req.Auth
	}
	if sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "" {
		middleware.WriteError(w, http.StatusBadRequest, "endpoint and keys.p256dh / keys.auth are required")
		return
	}
	h.Push.Add(middleware.UserID(r), sub)
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "subscribed",
		"subscribers": h.pushCount(middleware.UserID(r)),
	})
}

// UnsubscribePush serves DELETE /api/push/subscribe with
// {"endpoint": "..."}.
func (h *Handler) UnsubscribePush(w http.ResponseWriter, r *http.Request) {
	if h.Push == nil {
		middleware.WriteError(w, http.StatusServiceUnavailable, "push not configured on this server")
		return
	}
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.Push.Remove(middleware.UserID(r), req.Endpoint)
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{"status": "unsubscribed"})
}

// TestPush serves POST /api/push/test: sends a test notification to the
// caller's own subscriptions so the client can verify delivery. When
// FCM is configured it also sends a test notification to the caller's
// FCM tokens and reports fcm_sent/fcm_failed.
func (h *Handler) TestPush(w http.ResponseWriter, r *http.Request) {
	if h.Push == nil || h.VAPIDPublicKey == "" || h.VAPIDPrivateKey == "" {
		// Web Push is unavailable, but FCM alone is still testable.
		if h.FCM != nil && h.FCM.Available() {
			sent, failed := h.FCM.SendToUser(r.Context(), middleware.UserID(r), "VYRO", "Test notification — VYRO push is working.", map[string]string{"type": "test"})
			middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"sent": 0, "failed": 0, "fcm_sent": sent, "fcm_failed": failed,
			})
			return
		}
		middleware.WriteError(w, http.StatusServiceUnavailable, "push not configured on this server")
		return
	}
	body, _ := json.Marshal(map[string]string{
		"title": "VYRO",
		"body":  "Test notification — VYRO push is working.",
	})
	sent, failed := h.Push.SendToAll(middleware.UserID(r), body, h.VAPIDPublicKey, h.VAPIDPrivateKey, h.VAPIDSubject)
	resp := map[string]interface{}{
		"sent": sent, "failed": failed,
	}
	if h.FCM != nil && h.FCM.Available() {
		fcmSent, fcmFailed := h.FCM.SendToUser(r.Context(), middleware.UserID(r), "VYRO", "Test notification — VYRO push is working.", map[string]string{"type": "test"})
		resp["fcm_sent"], resp["fcm_failed"] = fcmSent, fcmFailed
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// notifyOfflinePush sends a content-free push to the chat members who
// did not receive the message over WebSocket (offline). It is best
// effort: push failures are ignored and never block message delivery.
// The payload contains no message text — only the sender's name — so
// neither the push service nor anyone intercepting it learns the
// content. For E2EE messages even the sender's name is omitted
// ("New encrypted message"), so the push leaks nothing at all about
// the conversation. Web Push (VAPID) and FCM are independent paths:
// either may be configured without the other.
func (h *Handler) notifyOfflinePush(ctx context.Context, chatID, senderID string, kind string, delivered []string) {
	// FCM first: it needs no VAPID keys, so it must still fire when
	// Web Push is unconfigured.
	h.notifyOfflineFCM(ctx, chatID, senderID, kind, delivered)
	if h.Push == nil || h.VAPIDPublicKey == "" || h.VAPIDPrivateKey == "" {
		return
	}
	online := map[string]bool{}
	for _, id := range delivered {
		online[id] = true
	}
	senderName := "Someone"
	if u, err := h.Store.UserByID(senderID); err == nil && u.DisplayName != "" {
		senderName = u.DisplayName
	} else if u != nil && u.Username != "" {
		senderName = u.Username
	}
	var body string
	if kind == "e2ee" {
		body = "New encrypted message"
	} else {
		body = "New message from " + senderName
	}
	payload, _ := json.Marshal(map[string]string{"title": "VYRO", "body": body})
	for _, memberID := range h.chatMemberIDs(chatID) {
		if memberID == senderID || online[memberID] {
			continue
		}
		h.Push.SendToAll(memberID, payload, h.VAPIDPublicKey, h.VAPIDPrivateKey, h.VAPIDSubject)
	}
}

func (h *Handler) pushCount(userID string) int {
	if h.Push == nil {
		return 0
	}
	return h.Push.Count(userID)
}
