package handlers

import (
	"net/http"

	"github.com/vyro/backend/internal/middleware"
)

// Stage 1 chat gaps (WhatsApp parity): pin message and
// delete-for-me. Mute chat reuses SetChatSetting ("mute"/"unmute");
// contact share is a message kind handled in SendMessage.

// PinMessage pins one message in a chat (any member can pin; the
// pin replaces any previous pin in that chat). The pinned message
// is returned by ListChats/GetChat as chat.pinned_message.
func (h *Handler) PinMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	var req struct {
		MessageID string `json:"message_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	msg, err := h.Store.MessageByID(req.MessageID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	if msg.ChatID != chatID {
		middleware.WriteError(w, http.StatusBadRequest, "message is not in this chat")
		return
	}
	if err := h.Store.SetPinnedMessage(chatID, req.MessageID); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not pin message")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, h.Store.PinnedMessageForChat(chatID))
}

// UnpinMessage clears the chat's pinned message.
func (h *Handler) UnpinMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	chatID := chatIDFromPath(r)
	if !h.Store.IsMember(chatID, middleware.UserID(r)) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	if err := h.Store.SetPinnedMessage(chatID, ""); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not unpin message")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "unpinned"})
}

// DeleteMessageForMe hides a message only for the requesting user
// (delete-for-me). The message stays visible for everyone else.
// Any member can delete any message for themself; delete-for-everyone
// stays the sender-only hard delete in DeleteMessage.
func (h *Handler) DeleteMessageForMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := messageIDFromPath(r)
	msg, err := h.Store.MessageByID(id)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	if !h.Store.IsMember(msg.ChatID, middleware.UserID(r)) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	if err := h.Store.DeleteMessageForUser(id, middleware.UserID(r)); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not delete message")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted_for_me"})
}
