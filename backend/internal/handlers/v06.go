// VYRO v0.6 handlers: group E2EE sender-key distribution, multi-device
// registry, and per-device key bundles.
//
// Group E2EE (Sender Keys): the server stores one distribution envelope
// per (sender device -> recipient device) and relays it. The envelope
// is a 1:1 E2EE envelope (the sender key encrypted under the existing
// 1:1 session), so the server validates only its shape — it never
// decrypts it and cannot read group messages. Device binding is
// enforced cryptographically by the 1:1 session (X3DH authenticates
// the sender's identity key), not by this endpoint.
//
// Multi-device: a client registers each device (POST /api/devices),
// uploads keys per device, and peers fetch a key bundle per device
// (GET /api/keys/{userID}/{deviceID}/bundle). Messages are fanned out
// to every connected device of every member over WebSocket; the
// sender's own other devices receive a "message_sync" copy.
package handlers

import (
	"net/http"
	"strings"

	"github.com/vyro/backend/internal/e2ee"
	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
)

// --- Group E2EE sender keys ---

type uploadSenderKeyReq struct {
	SenderDeviceID string `json:"sender_device_id"`
	KeyID          string `json:"key_id"`
	ForUserID      string `json:"for_user_id"`
	ForDeviceID    string `json:"for_device_id"`
	Ciphertext     string `json:"ciphertext"`
}

// UploadGroupSenderKey serves POST /api/chats/{id}/sender-keys: store one
// sender-key distribution envelope addressed to one recipient device.
// The ciphertext must be a well-formed 1:1 envelope (the sender key
// encrypted under the 1:1 session to that device); the server never
// decrypts it.
func (h *Handler) UploadGroupSenderKey(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	var req uploadSenderKeyReq
	if !decode(w, r, &req) {
		return
	}
	if len(req.SenderDeviceID) == 0 || len(req.SenderDeviceID) > 64 ||
		len(req.KeyID) == 0 || len(req.KeyID) > 64 ||
		len(req.ForDeviceID) == 0 || len(req.ForDeviceID) > 64 ||
		req.ForUserID == "" {
		middleware.WriteError(w, http.StatusBadRequest, "sender_device_id, key_id, for_user_id and for_device_id are required (max 64 chars)")
		return
	}
	if !h.Store.IsMember(chatID, req.ForUserID) {
		middleware.WriteError(w, http.StatusBadRequest, "recipient is not a member of this chat")
		return
	}
	// The distribution is a 1:1 envelope: validate its shape only.
	if _, err := e2ee.ValidateEnvelope(req.Ciphertext); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid distribution envelope: "+err.Error())
		return
	}
	d := &models.GroupSenderKey{
		ChatID: chatID, SenderUserID: me, SenderDeviceID: req.SenderDeviceID,
		KeyID: req.KeyID, ForUserID: req.ForUserID, ForDeviceID: req.ForDeviceID,
		Ciphertext: req.Ciphertext,
	}
	if err := h.Store.PutGroupSenderKey(d); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not store sender key")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, d)
}

// ListGroupSenderKeys serves GET /api/chats/{id}/sender-keys?device_id=:
// every distribution envelope addressed to the caller's device in this
// chat (all senders, all key generations), so the device can pick up
// the sender keys it needs. device_id defaults to "default".
func (h *Handler) ListGroupSenderKeys(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = "default"
	}
	out := h.Store.GroupSenderKeysFor(chatID, me, deviceID)
	if out == nil {
		out = []*models.GroupSenderKey{}
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

// --- Multi-device registry ---

var validPlatforms = map[string]bool{
	"": true, "web": true, "android": true, "ios": true, "desktop": true,
}

func validateAuthDevice(w http.ResponseWriter, deviceID, name, platform string) bool {
	if len(deviceID) > 64 || len(name) > 64 || !validPlatforms[strings.ToLower(platform)] {
		middleware.WriteError(w, http.StatusBadRequest, "invalid device_id, device_name or platform")
		return false
	}
	return true
}

type registerDeviceReq struct {
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

// RegisterDevice serves POST /api/devices: register (or re-register)
// one device of the caller's account.
func (h *Handler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req registerDeviceReq
	if !decode(w, r, &req) {
		return
	}
	if len(req.DeviceID) == 0 || len(req.DeviceID) > 64 {
		middleware.WriteError(w, http.StatusBadRequest, "device_id is required (max 64 chars)")
		return
	}
	if len(req.Name) > 64 || !validPlatforms[strings.ToLower(req.Platform)] {
		middleware.WriteError(w, http.StatusBadRequest, "name too long or platform must be web, android, ios or desktop")
		return
	}
	d := &models.Device{
		UserID: middleware.UserID(r), DeviceID: req.DeviceID,
		Name: req.Name, Platform: strings.ToLower(req.Platform),
	}
	if err := h.Store.RegisterDevice(d); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not register device")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, d)
}

// ListMyDevices serves GET /api/devices: the caller's registered devices.
func (h *Handler) ListMyDevices(w http.ResponseWriter, r *http.Request) {
	out := h.Store.ListDevices(middleware.UserID(r))
	if out == nil {
		out = []*models.Device{}
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

// RemoveDevice serves DELETE /api/devices/{device_id}: unregister one of
// the caller's devices.
func (h *Handler) RemoveDevice(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	deviceID := strings.Split(rest, "/")[0]
	if deviceID == "" {
		middleware.WriteError(w, http.StatusBadRequest, "invalid device id")
		return
	}
	if err := h.Store.RevokeDevice(middleware.UserID(r), deviceID); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not revoke device")
		return
	}
	if err := h.Store.RemoveDevice(middleware.UserID(r), deviceID); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "device not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// ListUserDevices serves GET /api/users/{id}/devices: another user's
// registered devices (public view, no secrets), so a peer client can
// fetch a key bundle for each of the user's devices.
func (h *Handler) ListUserDevices(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/users/")
	userID := strings.Split(rest, "/")[0]
	if _, err := h.Store.UserByID(userID); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	out := []models.PublicDevice{}
	for _, d := range h.Store.ListDevices(userID) {
		out = append(out, d.Public())
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}
