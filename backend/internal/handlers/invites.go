package handlers

import (
	"crypto/rand"
	"net/http"
	"strings"
	"time"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
)

// Invite links for groups and channels (Telegram-style, user direction
// 2026-11-14: the app must be exactly like Telegram and WhatsApp).
// An admin/owner creates a link; anyone with it can preview the chat
// and join. Revoking a link makes its token stop working.
//
// Endpoints (see cmd/server/main.go):
//   POST   /api/chats/{id}/invites          create (owner/admin)
//   GET    /api/chats/{id}/invites          list (owner/admin)
//   DELETE /api/chats/{id}/invites/{token}  revoke (owner/admin)
//   GET    /api/invites/{token}             preview (any user)
//   POST   /api/invites/{token}/join        join (any user)

// inviteLinkBase is the shareable link prefix. The download site serves
// /join/{token} with the chat preview and a download button.
const inviteLinkBase = "https://vyro-web-8p4l.onrender.com/join/"

func inviteLink(token string) string { return inviteLinkBase + token }

func newInviteToken() string {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand essentially never fails; time-based fallback
		// rather than failing the request.
		for i := range b {
			b[i] = byte(time.Now().UnixNano() >> (uint(i%8) * 8))
		}
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func inviteFromPath(r *http.Request) string {
	// /api/invites/{token}[/join]
	rest := strings.TrimPrefix(r.URL.Path, "/api/invites/")
	parts := strings.Split(rest, "/")
	return parts[0]
}

// CreateInvite serves POST /api/chats/{id}/invites.
func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.requireRole(w, r, chatID, "owner", "admin") {
		return
	}
	chat, err := h.Store.ChatByID(chatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	if chat.Type == models.ChatDM {
		middleware.WriteError(w, http.StatusBadRequest, "private chats have no invite links")
		return
	}
	inv := &models.ChatInvite{
		Token:     newInviteToken(),
		ChatID:    chatID,
		CreatedBy: middleware.UserID(r),
		CreatedAt: time.Now().UTC(),
	}
	// A random collision is near-impossible; retry once if it happens.
	if err := h.Store.CreateInvite(inv); err != nil {
		if err == store.ErrAlreadyExists {
			inv.Token = newInviteToken()
			if err := h.Store.CreateInvite(inv); err != nil {
				middleware.WriteError(w, http.StatusInternalServerError, "could not create invite link")
				return
			}
		} else {
			middleware.WriteError(w, http.StatusInternalServerError, "could not create invite link")
			return
		}
	}
	middleware.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"token": inv.Token, "link": inviteLink(inv.Token),
		"chat_id": inv.ChatID, "created_at": inv.CreatedAt,
	})
}

// ListInvites serves GET /api/chats/{id}/invites.
func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.requireRole(w, r, chatID, "owner", "admin") {
		return
	}
	if _, err := h.Store.ChatByID(chatID); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	invites := h.Store.ListInvites(chatID)
	if invites == nil {
		invites = []*models.ChatInvite{}
	}
	out := make([]map[string]interface{}, 0, len(invites))
	for _, inv := range invites {
		out = append(out, map[string]interface{}{
			"token": inv.Token, "link": inviteLink(inv.Token),
			"chat_id": inv.ChatID, "created_by": inv.CreatedBy,
			"created_at": inv.CreatedAt, "revoked": inv.Revoked,
		})
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

// RevokeInvite serves DELETE /api/chats/{id}/invites/{token}.
func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.requireRole(w, r, chatID, "owner", "admin") {
		return
	}
	// /api/chats/{chatID}/invites/{token}
	rest := strings.TrimPrefix(r.URL.Path, "/api/chats/")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid invite token")
		return
	}
	token := parts[2]
	inv, err := h.Store.InviteByToken(token)
	if err != nil || inv.ChatID != chatID {
		middleware.WriteError(w, http.StatusNotFound, "invite link not found")
		return
	}
	if err := h.Store.RevokeInvite(token); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not revoke invite link")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// PreviewInvite serves GET /api/invites/{token}: what the chat is,
// without joining it.
func (h *Handler) PreviewInvite(w http.ResponseWriter, r *http.Request) {
	token := inviteFromPath(r)
	inv, err := h.Store.InviteByToken(token)
	if err != nil || inv.Revoked {
		middleware.WriteError(w, http.StatusNotFound, "invite link not found")
		return
	}
	chat, err := h.Store.ChatByID(inv.ChatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "invite link not found")
		return
	}
	inviter := ""
	if u, err := h.Store.UserByID(inv.CreatedBy); err == nil {
		inviter = u.Username
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"token": inv.Token, "chat_id": chat.ID, "title": chat.Title,
		"type": string(chat.Type), "member_count": len(chat.MemberIDs),
		"invited_by": inviter,
	})
}

// JoinViaInvite serves POST /api/invites/{token}/join.
func (h *Handler) JoinViaInvite(w http.ResponseWriter, r *http.Request) {
	token := inviteFromPath(r)
	inv, err := h.Store.InviteByToken(token)
	if err != nil || inv.Revoked {
		middleware.WriteError(w, http.StatusNotFound, "invite link not found")
		return
	}
	chat, err := h.Store.ChatByID(inv.ChatID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "invite link not found")
		return
	}
	if chat.Type == models.ChatDM {
		middleware.WriteError(w, http.StatusBadRequest, "private chats have no invite links")
		return
	}
	me := middleware.UserID(r)
	if h.Store.IsMember(chat.ID, me) {
		middleware.WriteError(w, http.StatusConflict, "already a member")
		return
	}
	if err := h.Store.AddMember(chat.ID, me, "member"); err != nil {
		if err == store.ErrAlreadyExists {
			middleware.WriteError(w, http.StatusConflict, "already a member")
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "could not join")
		return
	}
	// Stage 6: joining an announcement group via its invite link
	// also joins the community.
	if community := h.Store.CommunityByAnnouncementChat(chat.ID); community != nil {
		_ = h.Store.AddCommunityMember(community.ID, me, "member")
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "joined", "chat_id": chat.ID})
}
