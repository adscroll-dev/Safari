package handlers

import (
	"net/http"
	"time"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
)

// Stage 9 (extras): backup & restore + storage page.
//
// Backup: your chats already live on Safari's server (PostgreSQL),
// so signing in on any device restores them — that IS the backup.
// GET /api/users/me/export additionally gives you a personal copy
// as JSON: your profile, your chats, and the latest 200 messages
// of each chat (encrypted messages export in their stored form),
// plus your blocked list and privacy settings. The file is yours
// to keep; it is never sent anywhere by the server.

const exportMessagesPerChat = 200

// ExportMyData serves GET /api/users/me/export.
func (h *Handler) ExportMyData(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	user, err := h.Store.UserByID(me)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	chats := h.Store.ChatsForUser(me)
	type chatExport struct {
		Chat     interface{}   `json:"chat"`
		Messages []interface{} `json:"messages"`
	}
	out := make([]chatExport, 0, len(chats))
	for _, chat := range chats {
		msgs := h.Store.MessagesForChat(chat.ID, exportMessagesPerChat, 0)
		list := make([]interface{}, 0, len(msgs))
		for _, m := range msgs {
			list = append(list, m)
		}
		out = append(out, chatExport{Chat: chat, Messages: list})
	}
	blocked := h.Store.ListBlocked(me)
	if blocked == nil {
		blocked = []string{}
	}
	lastSeen := h.Store.GetLastSeenVisibility(me)
	statusPrivacy := h.Store.GetStatusPrivacy(me)
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"user":           user.Public(),
		"chats":          out,
		"blocked":        blocked,
		"last_seen":      lastSeen,
		"status_privacy": statusPrivacy,
		"exported_at":    time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// StorageStats serves GET /api/users/me/storage: per-chat message
// counts + media bytes for my chats, plus totals.
func (h *Handler) StorageStats(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	perChat, totalMessages, totalBytes := h.Store.StorageStats(me)
	if perChat == nil {
		perChat = []models.ChatStorage{}
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"chats":          perChat,
		"total_messages": totalMessages,
		"total_bytes":    totalBytes,
	})
}
