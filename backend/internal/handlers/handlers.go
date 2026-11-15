// Package handlers implements the VYRO HTTP API.
//
// Implemented and tested in v0.1 (in-memory store):
//
//	GET  /health
//	POST /api/auth/register   {email, username, password, display_name} -> 201 code_sent (no token yet)
//	POST /api/auth/verify-register {email, code} -> 201 {user, token}
//	POST /api/auth/login      {email, password}  (rate-limited) -> 200 code_sent (no token yet)
//	POST /api/auth/verify-login {email, code} -> 200 {user, token}
//	POST /api/auth/resend-code {email, purpose} -> 200 code_sent
//	GET  /api/users/me
//	GET  /api/users?search=
//	GET  /api/chats
//	POST /api/chats           {type, title, member_ids}
//	GET  /api/chats/{id}/messages
//	POST /api/uploads           multipart "file" -> {url, file_name, file_size, mime_type, kind}
//	GET  /uploads/{name}         download a file uploaded above (auth required)
//	POST /api/chats/{id}/messages  {content, kind?, reply_to_id?, file_url?, file_name?, file_size?, mime_type?}
//	                               kind: text | image | video | file | voice | location | sticker | contact (file kinds need file_url from /api/uploads; contact content is JSON {"user_id"})
//	PUT  /api/messages/{id}   {content}            (sender only)
//	DELETE /api/messages/{id}                       (sender only, delete-for-everyone)
//	POST /api/messages/{id}/delete-for-me           (any member, hides it only for the caller)
//	POST /api/messages/{id}/reactions {emoji}
//	DELETE /api/messages/{id}/reactions?emoji=
//	GET  /api/messages/search?q=
//	POST /api/users/{id}/block
//	GET  /ws?token=           WebSocket live feed
//
// Not yet implemented (documented, not faked): voice
// messages (recording/playback in clients), calls (WebRTC signalling), stories, bots, AI assistant,
// translation. See README.
package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vyro/backend/internal/auth"
	"github.com/vyro/backend/internal/e2ee"
	"github.com/vyro/backend/internal/mail"
	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/push"
	"github.com/vyro/backend/internal/store"
	"github.com/vyro/backend/internal/ws"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type Handler struct {
	Store       store.StoreIface
	Hub         *ws.Hub
	JWTSecret   string
	JWTExpiry   time.Duration
	LoginLimits auth.Limiter
	// StoreName is reported by /health ("memory" or "postgres").
	StoreName string
	// UploadDir is where /api/uploads stores files (default "./uploads").
	UploadDir string
	// MaxUploadMB caps a single upload (default 50).
	MaxUploadMB int64
	// Push is the Web Push subscription registry (v0.6: backed by the
	// store, so subscriptions survive a restart). Nil when push is not
	// configured.
	Push *push.Registry
	// VAPID keys for Web Push (RFC 8292). Empty = push disabled.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
	// FCM is the Firebase Cloud Messaging registry for the mobile app
	// (see internal/push/fcm.go). Nil = FCM not configured; Web Push
	// still works without it.
	FCM *push.FCMRegistry
	// Mailer sends verification-code emails (v0.7). Nil = email is not
	// configured: register/login return 503 instead of faking a code.
	Mailer mail.Sender
}

func New(s store.StoreIface, hub *ws.Hub, secret string, expiry time.Duration) *Handler {
	return &Handler{
		Store: s, Hub: hub, JWTSecret: secret, JWTExpiry: expiry,
		LoginLimits: auth.NewRateLimiter(5, 15*time.Minute),
		StoreName:   "memory",
		UploadDir:   "./uploads",
		MaxUploadMB: 50,
	}
}

func (h *Handler) maxUploadBytes() int64 {
	if h.MaxUploadMB <= 0 {
		return 50 << 20
	}
	return h.MaxUploadMB << 20
}

func (h *Handler) uploadDir() string {
	if h.UploadDir == "" {
		return "./uploads"
	}
	return h.UploadDir
}

func decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// --- Health ---

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	storeName := h.StoreName
	if storeName == "" {
		storeName = "memory"
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok", "service": "vyro-backend", "version": "0.6.0",
		"store": storeName, "fcm": h.fcmStatus(),
		"time": time.Now().UTC().Format(time.RFC3339),
	})
}

// fcmStatus reports whether FCM push is available for the health check.
func (h *Handler) fcmStatus() string {
	if h.FCM != nil && h.FCM.Available() {
		return "enabled"
	}
	return "disabled"
}

// --- Auth ---

type registerReq struct {
	Email    string `json:"email"`
	Username string `json:"username"` // Ignored: the system always
	// generates the username from the display name (user direction
	// 2026-11-14). Kept so older clients can still send it.
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// Register starts a registration (v0.7): it validates the input,
// checks the email is free, generates the username from the user's
// name (the user never picks a custom username), stores a pending
// registration, and emails a 6-digit verification code. No user is
// created and no token is issued until POST /api/auth/verify-register
// confirms the code. Returns 503 when email sending is not configured.
// The response includes the generated "username" so the client can
// show it to the user.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req registerReq
	if !decode(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if !emailRe.MatchString(req.Email) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if req.DisplayName == "" {
		middleware.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	// The username is chosen by the system from the user's name.
	username := generateUsername(req.DisplayName, req.Email, func(u string) bool {
		_, err := h.Store.UserByUsername(u)
		return err == nil
	})
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.Store.UserByEmail(req.Email); err == nil {
		middleware.WriteError(w, http.StatusConflict, "email or username already registered")
		return
	}
	if h.Mailer == nil {
		h.mailerUnavailable(w)
		return
	}
	now := time.Now().UTC()
	if err := h.Store.SavePendingUser(&models.PendingUser{
		Email: req.Email, Username: username, DisplayName: req.DisplayName,
		PasswordHash: hash, ExpiresAt: now.Add(codeTTL), CreatedAt: now,
	}); err != nil {
		log.Printf("register: save pending user failed: %v", err)
		middleware.WriteError(w, http.StatusInternalServerError, "could not start registration")
		return
	}
	if err := h.issueCode(req.Email, purposeRegister); err != nil {
		log.Printf("register: send verification code failed: %v", err)
		_ = h.Store.DeletePendingUser(req.Email)
		middleware.WriteError(w, http.StatusInternalServerError, "could not send verification email")
		return
	}
	codeSentResponseWithUsername(w, http.StatusCreated, req.Email, purposeRegister, username)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login checks the password (v0.7) and, on success, emails a 6-digit
// verification code instead of issuing a token. The token is issued
// only by POST /api/auth/verify-login. The login rate limit (5
// attempts / 15 min) still applies to this step, and a successful
// password check resets it. Returns 503 when email sending is not
// configured.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req loginReq
	if !decode(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	key := r.RemoteAddr + "|" + email
	if !h.LoginLimits.Allow(key) {
		middleware.WriteError(w, http.StatusTooManyRequests, "too many login attempts, try again in 15 minutes")
		return
	}
	user, err := h.Store.UserByEmail(email)
	if err != nil || !auth.CheckPassword(req.Password, user.PasswordHash) {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	h.LoginLimits.Reset(key)
	if h.Mailer == nil {
		h.mailerUnavailable(w)
		return
	}
	if err := h.issueCode(email, purposeLogin); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not send verification email")
		return
	}
	codeSentResponse(w, http.StatusOK, email, purposeLogin)
}

// --- Users ---

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.Store.UserByID(middleware.UserID(r))
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, user.Public())
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	out := []models.PublicUser{}
	for _, u := range h.Store.ListUsers() {
		if u.ID == middleware.UserID(r) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(u.Username), q) &&
			!strings.Contains(strings.ToLower(u.DisplayName), q) &&
			!strings.Contains(strings.ToLower(u.Email), q) {
			continue
		}
		out = append(out, u.Public())
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	targetID := strings.TrimPrefix(r.URL.Path, "/api/users/")
	targetID = strings.TrimSuffix(targetID, "/block")
	if targetID == "" || targetID == middleware.UserID(r) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.Store.BlockUser(middleware.UserID(r), targetID); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
}

// --- Chats ---

type createChatReq struct {
	Type      string   `json:"type"`
	Title     string   `json:"title"`
	MemberIDs []string `json:"member_ids"`
	// Public marks a channel as discoverable (Stage 5); nil means
	// public. Ignored for DMs and groups.
	Public *bool `json:"public"`
}

func (h *Handler) ListChats(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	chats := h.Store.ChatsForUser(me)
	if chats == nil {
		chats = []*models.Chat{}
	}
	h.applySettings(w, r, chats)
	// WhatsApp-style list data: latest message preview + unread
	// count for the requesting user. E2EE previews never carry the
	// ciphertext — the client shows "Encrypted message" instead.
	// The preview skips messages the user deleted for themself.
	for _, c := range chats {
		if m := h.Store.LastMessageForChatForUser(c.ID, me); m != nil {
			content := m.Content
			if m.Kind == "e2ee" {
				content = ""
			}
			c.LastMessage = &models.LastMessagePreview{
				Content:   content,
				Kind:      m.Kind,
				SenderID:  m.SenderID,
				Status:    m.Status,
				CreatedAt: m.CreatedAt,
			}
		}
		c.UnreadCount = h.Store.UnreadCountForChat(c.ID, me)
		c.PinnedMessage = h.Store.PinnedMessageForChat(c.ID)
	}
	// Pinned chats first, archived last (Telegram-style ordering).
	sortChats(chats)
	middleware.WriteJSON(w, http.StatusOK, chats)
}

// sortChats orders pinned first, then by recent activity, archived last.
func sortChats(chats []*models.Chat) {
	for i := 0; i < len(chats); i++ {
		for j := i + 1; j < len(chats); j++ {
			if chatLess(chats[j], chats[i]) {
				chats[i], chats[j] = chats[j], chats[i]
			}
		}
	}
}

func chatLess(a, b *models.Chat) bool {
	if a.Archived != b.Archived {
		return !a.Archived
	}
	if a.Pinned != b.Pinned {
		return a.Pinned
	}
	return a.UpdatedAt.After(b.UpdatedAt)
}

func (h *Handler) CreateChat(w http.ResponseWriter, r *http.Request) {
	var req createChatReq
	if !decode(w, r, &req) {
		return
	}
	typ := models.ChatType(req.Type)
	if typ != models.ChatDM && typ != models.ChatGroup && typ != models.ChatChannel {
		middleware.WriteError(w, http.StatusBadRequest, "type must be dm, group or channel")
		return
	}
	if typ != models.ChatDM && strings.TrimSpace(req.Title) == "" {
		middleware.WriteError(w, http.StatusBadRequest, "title is required for groups and channels")
		return
	}
	me := middleware.UserID(r)
	members := append([]string{me}, req.MemberIDs...)
	// Every member must exist; DM members must not be blocked.
	for _, id := range members {
		if _, err := h.Store.UserByID(id); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "unknown member: "+id)
			return
		}
	}
	if typ == models.ChatDM {
		if len(members) != 2 {
			middleware.WriteError(w, http.StatusBadRequest, "dm requires exactly one other member")
			return
		}
		if h.Store.IsBlocked(me, members[1]) {
			middleware.WriteError(w, http.StatusForbidden, "user is blocked")
			return
		}
	}
	chat := &models.Chat{Type: typ, Title: req.Title, OwnerID: me, MemberIDs: members}
	if typ == models.ChatChannel {
		// Channels are public (discoverable) unless the creator
		// marks them private (Stage 5).
		chat.IsPublic = req.Public == nil || *req.Public
	}
	if err := h.Store.CreateChat(chat); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create chat")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, chat)
}

// --- Messages ---

func chatIDFromPath(r *http.Request) string {
	// /api/chats/{id}/messages
	rest := strings.TrimPrefix(r.URL.Path, "/api/chats/")
	parts := strings.Split(rest, "/")
	return parts[0]
}

func messageIDFromPath(r *http.Request) string {
	// /api/messages/{id}[/reactions]
	rest := strings.TrimPrefix(r.URL.Path, "/api/messages/")
	parts := strings.Split(rest, "/")
	return parts[0]
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	h.Store.PurgeExpired(time.Now().UTC())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	msgs := h.Store.MessagesForChat(chatID, limit, offset)
	if msgs == nil {
		msgs = []*models.Message{}
	}
	// Delete-for-me: hide messages the reader hid only for themself.
	visible := msgs[:0]
	for _, m := range msgs {
		if !h.Store.IsMessageDeletedForUser(m.ID, me) {
			visible = append(visible, m)
		}
	}
	msgs = visible
	// Stage 5: channel posts carry their view count (distinct
	// viewers) so the client can show it like WhatsApp.
	if chat, err := h.Store.ChatByID(chatID); err == nil && chat.Type == models.ChatChannel {
		counts := h.Store.MessageViewCounts(chatID)
		for _, m := range msgs {
			m.ViewCount = counts[m.ID]
		}
	}
	// Fetching a chat delivers its messages to the reader (WhatsApp-style
	// second tick) — except the reader's own messages.
	for _, m := range msgs {
		if m.SenderID != me {
			_ = h.Store.MarkDelivered(m.ID, me)
			if !contains(m.DeliveredTo, me) {
				m.DeliveredTo = append(m.DeliveredTo, me)
			}
			m.Status = m.StatusFor(otherMembers(h.Store, m))
		}
	}
	middleware.WriteJSON(w, http.StatusOK, msgs)
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func otherMembers(st store.StoreIface, m *models.Message) []string {
	chat, err := st.ChatByID(m.ChatID)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, id := range chat.MemberIDs {
		if id != m.SenderID {
			out = append(out, id)
		}
	}
	return out
}

type sendMessageReq struct {
	Content         string   `json:"content"`
	Kind            string   `json:"kind"`
	ReplyToID       string   `json:"reply_to_id"`
	FileURL         string   `json:"file_url"`
	FileName        string   `json:"file_name"`
	FileSize        int64    `json:"file_size"`
	MimeType        string   `json:"mime_type"`
	Lat             *float64 `json:"lat"`
	Lng             *float64 `json:"lng"`
	LocationLabel   string   `json:"location_label"`
	TTLSeconds      int64    `json:"ttl_seconds"`       // disappearing message, 60..604800
	ForwardedFrom   string   `json:"forwarded_from"`    // original sender display name (forwarding)
	ForwardedFromID string   `json:"forwarded_from_id"` // original sender user id (forwarding)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	// Stage 5: in a channel only admins and the owner may post —
	// followers are read-only, like WhatsApp.
	if chat, err := h.Store.ChatByID(chatID); err == nil && chat.Type == models.ChatChannel {
		role := h.Store.MemberRole(chatID, me)
		if role != "admin" && role != "owner" {
			middleware.WriteError(w, http.StatusForbidden, "only channel admins can post")
			return
		}
	}
	// Stage 6: in a community announcement group only community
	// admins and the owner may post.
	if community := h.Store.CommunityByAnnouncementChat(chatID); community != nil {
		role := h.Store.CommunityRole(community.ID, me)
		if role != "admin" && role != "owner" {
			middleware.WriteError(w, http.StatusForbidden, "only community admins can post announcements")
			return
		}
	}
	var req sendMessageReq
	if !decode(w, r, &req) {
		return
	}
	kind := req.Kind
	if kind == "" {
		kind = "text"
	}
	switch kind {
	case "text", "image", "video", "file", "voice", "location", "sticker", "contact", "e2ee":
	default:
		middleware.WriteError(w, http.StatusBadRequest, "kind '"+kind+"' is not implemented yet")
		return
	}
	// An E2EE envelope cannot be forwarded: its ciphertext is only
	// decryptable by the original chat's members, so it would be
	// unreadable in the new chat. Checked before envelope validation
	// so the error is always about forwarding, never the envelope.
	if kind == "e2ee" && (req.ForwardedFromID != "" || req.ForwardedFrom != "") {
		middleware.WriteError(w, http.StatusBadRequest, "e2ee messages cannot be forwarded")
		return
	}
	if kind == "e2ee" {
		// E2EE v1: the server stores the ciphertext envelope only.
		// Validate its shape here; the server never decrypts it and
		// must never log or search its contents. Group/channel chats
		// use the Sender Keys group envelope (v0.6); DMs use the 1:1
		// Double Ratchet envelope (v0.5).
		chat, chatErr := h.Store.ChatByID(chatID)
		if chatErr == nil && (chat.Type == models.ChatGroup || chat.Type == models.ChatChannel) {
			if _, err := e2ee.ValidateGroupEnvelope(req.Content); err != nil {
				middleware.WriteError(w, http.StatusBadRequest, "invalid group e2ee envelope: "+err.Error())
				return
			}
		} else if _, err := e2ee.ValidateEnvelope(req.Content); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "invalid e2ee envelope: "+err.Error())
			return
		}
	}
	if kind == "location" {
		if req.Lat == nil || req.Lng == nil || *req.Lat < -90 || *req.Lat > 90 || *req.Lng < -180 || *req.Lng > 180 {
			middleware.WriteError(w, http.StatusBadRequest, "location requires lat (-90..90) and lng (-180..180)")
			return
		}
	}
	if kind == "contact" {
		// A shared contact is JSON {"user_id": "<uuid>"} naming a
		// Safari user; the client renders it as a contact card.
		var c struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal([]byte(req.Content), &c); err != nil || c.UserID == "" {
			middleware.WriteError(w, http.StatusBadRequest, "contact requires a JSON body with user_id")
			return
		}
		if _, err := h.Store.UserByID(c.UserID); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "contact user not found")
			return
		}
	}
	if kind == "e2ee" {
		// No further content checks: the envelope was validated above.
	} else if kind == "text" {
		if strings.TrimSpace(req.Content) == "" {
			middleware.WriteError(w, http.StatusBadRequest, "content is required")
			return
		}
		if len(req.Content) > 10000 {
			middleware.WriteError(w, http.StatusBadRequest, "message too long (max 10000 characters)")
			return
		}
	} else if kind != "location" && kind != "contact" {
		// A media message must point at a file this server stored.
		if !strings.HasPrefix(req.FileURL, "/uploads/") || strings.Contains(req.FileURL, "..") {
			middleware.WriteError(w, http.StatusBadRequest, "file_url from /api/uploads is required for kind '"+kind+"'")
			return
		}
		if req.FileSize < 0 || req.FileSize > h.maxUploadBytes() {
			middleware.WriteError(w, http.StatusBadRequest, "file_size exceeds the upload limit")
			return
		}
		if len(req.Content) > 10000 {
			middleware.WriteError(w, http.StatusBadRequest, "caption too long (max 10000 characters)")
			return
		}
		if req.FileName == "" {
			req.FileName = filepath.Base(req.FileURL)
		}
	}
	if req.ReplyToID != "" {
		if _, err := h.Store.MessageByID(req.ReplyToID); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, "reply_to message not found")
			return
		}
	}
	// Forwarding attribution: the original sender must be a real
	// user, and the name is capped. (The e2ee-forward rejection is
	// checked earlier, before envelope validation.)
	if req.ForwardedFromID != "" || req.ForwardedFrom != "" {
		req.ForwardedFrom = strings.TrimSpace(req.ForwardedFrom)
		if len(req.ForwardedFrom) > 100 {
			middleware.WriteError(w, http.StatusBadRequest, "forwarded_from too long (max 100 characters)")
			return
		}
		if req.ForwardedFromID != "" {
			orig, err := h.Store.UserByID(req.ForwardedFromID)
			if err != nil {
				middleware.WriteError(w, http.StatusBadRequest, "forwarded_from_id user not found")
				return
			}
			if req.ForwardedFrom == "" {
				req.ForwardedFrom = orig.DisplayName
				if req.ForwardedFrom == "" {
					req.ForwardedFrom = orig.Username
				}
			}
		}
	}
	msg := &models.Message{
		ChatID: chatID, SenderID: me, Content: req.Content,
		Kind: kind, ReplyToID: req.ReplyToID,
		FileURL: req.FileURL, FileName: req.FileName,
		FileSize: req.FileSize, MimeType: req.MimeType,
		Lat: req.Lat, Lng: req.Lng, LocationLabel: req.LocationLabel,
		ForwardedFrom: req.ForwardedFrom, ForwardedFromID: req.ForwardedFromID,
	}
	if req.TTLSeconds != 0 {
		if req.TTLSeconds < 60 || req.TTLSeconds > 604800 {
			middleware.WriteError(w, http.StatusBadRequest, "ttl_seconds must be 60..604800 (1 minute..7 days)")
			return
		}
		exp := time.Now().UTC().Add(time.Duration(req.TTLSeconds) * time.Second)
		msg.ExpiresAt = &exp
	}
	if kind == "text" {
		msg.FileURL, msg.FileName, msg.FileSize, msg.MimeType = "", "", 0, ""
	}
	if err := h.Store.CreateMessage(msg); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	msg.Status = "sent"
	delivered := h.broadcastToChatAs(chatID, me, ws.Event{Type: "message", ChatID: chatID, Message: msg})
	for _, uid := range delivered {
		_ = h.Store.MarkDelivered(msg.ID, uid)
	}
	if len(delivered) > 0 {
		msg.DeliveredTo = delivered
		msg.Status = "delivered"
	}
	// Multi-device sync (v0.6): the sender's other devices receive a
	// sync copy of their own message, so it shows up on every device.
	h.Hub.BroadcastToUser(me, ws.Event{Type: "message_sync", ChatID: chatID, Message: msg})
	// Best-effort Web Push to members who are offline (not on the
	// WebSocket). The payload never contains message content.
	h.notifyOfflinePush(r.Context(), chatID, me, kind, delivered)
	// If the other side of a DM is a bot, notify its webhook (best effort).
	for _, memberID := range h.chatMemberIDs(chatID) {
		if memberID == me {
			continue
		}
		if u, err := h.Store.UserByID(memberID); err == nil && u.IsBot {
			if bot, err := h.Store.BotByUsername(u.Username); err == nil {
				h.notifyBotWebhook(bot, msg)
			}
		}
	}
	middleware.WriteJSON(w, http.StatusCreated, msg)
}

func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	id := messageIDFromPath(r)
	msg, err := h.Store.MessageByID(id)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	if msg.SenderID != middleware.UserID(r) {
		middleware.WriteError(w, http.StatusForbidden, "only the sender can edit this message")
		return
	}
	if msg.Kind == "e2ee" {
		// Ciphertext cannot be edited meaningfully; delete + resend.
		middleware.WriteError(w, http.StatusBadRequest, "encrypted messages cannot be edited")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		middleware.WriteError(w, http.StatusBadRequest, "content is required")
		return
	}
	msg.Content = req.Content
	if err := h.Store.UpdateMessage(msg); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not edit message")
		return
	}
	updated, _ := h.Store.MessageByID(id)
	h.broadcastToChatAs(msg.ChatID, middleware.UserID(r), ws.Event{Type: "message_edited", ChatID: msg.ChatID, Message: updated})
	h.Hub.BroadcastToUser(middleware.UserID(r), ws.Event{Type: "message_edited", ChatID: msg.ChatID, Message: updated})
	middleware.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	id := messageIDFromPath(r)
	msg, err := h.Store.MessageByID(id)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	if msg.SenderID != middleware.UserID(r) {
		middleware.WriteError(w, http.StatusForbidden, "only the sender can delete this message")
		return
	}
	if err := h.Store.DeleteMessage(id); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not delete message")
		return
	}
	h.broadcastToChatAs(msg.ChatID, middleware.UserID(r), ws.Event{Type: "message_deleted", ChatID: msg.ChatID, Message: msg})
	h.Hub.BroadcastToUser(middleware.UserID(r), ws.Event{Type: "message_deleted", ChatID: msg.ChatID, Message: msg})
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) AddReaction(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		Emoji string `json:"emoji"`
	}
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Emoji) == "" {
		middleware.WriteError(w, http.StatusBadRequest, "emoji is required")
		return
	}
	if err := h.Store.AddReaction(id, middleware.UserID(r), req.Emoji); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not add reaction")
		return
	}
	updated, _ := h.Store.MessageByID(id)
	middleware.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	id := messageIDFromPath(r)
	emoji := r.URL.Query().Get("emoji")
	if emoji == "" {
		middleware.WriteError(w, http.StatusBadRequest, "emoji query param is required")
		return
	}
	if err := h.Store.RemoveReaction(id, middleware.UserID(r), emoji); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	updated, _ := h.Store.MessageByID(id)
	middleware.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	msgs := h.Store.SearchMessages(middleware.UserID(r), q, 50)
	if msgs == nil {
		msgs = []*models.Message{}
	}
	middleware.WriteJSON(w, http.StatusOK, msgs)
}

// --- File uploads (v0.3) ---

// allowedUploadExts are the only file extensions accepted, by lower-case
// extension. Executables, scripts and HTML are never stored or served.
var allowedUploadExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".bmp": true,
	".mp4": true, ".mov": true, ".webm": true,
	".mp3": true, ".wav": true, ".ogg": true, ".m4a": true,
	".pdf": true, ".txt": true, ".md": true, ".zip": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
}

func kindForMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "voice"
	default:
		return "file"
	}
}

// Upload stores one multipart file ("file") under a random name in
// UploadDir and returns its URL. The file is NOT scanned for malware —
// this is a dev build; do not expose it to untrusted users without a
// scanner in front of it.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// ParseMultipartForm also caps the in-memory part; the +1MiB slack
	// lets us reject an over-limit file with 413 instead of a parse error.
	if err := r.ParseMultipartForm(h.maxUploadBytes() + (1 << 20)); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "form field 'file' is required")
		return
	}
	defer f.Close()
	if hdr.Size <= 0 || hdr.Size > h.maxUploadBytes() {
		middleware.WriteError(w, http.StatusRequestEntityTooLarge, "file is empty or exceeds the upload limit")
		return
	}
	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	if !allowedUploadExts[ext] {
		middleware.WriteError(w, http.StatusBadRequest, "file type not allowed: "+ext)
		return
	}
	// Sniff the real content type; a renamed executable must not pass.
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	mime := http.DetectContentType(head[:n])
	if strings.HasPrefix(mime, "text/html") || strings.HasPrefix(mime, "application/javascript") ||
		strings.HasPrefix(mime, "application/x-executable") || strings.HasPrefix(mime, "application/x-msdownload") {
		middleware.WriteError(w, http.StatusBadRequest, "file content type not allowed: "+mime)
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not read file")
		return
	}
	if err := os.MkdirAll(h.uploadDir(), 0o755); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create upload directory")
		return
	}
	stored := store.NewID() + ext
	dstPath := filepath.Join(h.uploadDir(), stored)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not store file")
		return
	}
	written, err := io.Copy(dst, io.LimitReader(f, h.maxUploadBytes()+1))
	_ = dst.Close()
	if err != nil || written != hdr.Size {
		_ = os.Remove(dstPath)
		middleware.WriteError(w, http.StatusInternalServerError, "could not store file")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"url": "/uploads/" + stored, "file_name": hdr.Filename,
		"file_size": hdr.Size, "mime_type": mime, "kind": kindForMime(mime),
	})
}

// ServeUpload serves a file previously stored by Upload. The URL carries
// a random id, and the endpoint requires a valid login, so files are not
// publicly listed — but anyone logged in who learns a URL can fetch it.
func (h *Handler) ServeUpload(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		middleware.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	absDir, err := filepath.Abs(h.uploadDir())
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not resolve upload directory")
		return
	}
	absPath := filepath.Join(absDir, name)
	if !strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) {
		middleware.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	http.ServeFile(w, r, absPath)
}

// WebSocketHandler upgrades an authenticated connection to the live feed.
// The connection joins the rooms of the user's chats and reports presence.
func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	me := middleware.UserID(r)
	chatIDs := []string{}
	for _, c := range h.Store.ChatsForUser(me) {
		chatIDs = append(chatIDs, c.ID)
	}
	h.Hub.Serve(w, r, me, chatIDs, func(userID string, online bool) {
		_ = h.Store.SetPresence(userID, online, time.Now().UTC())
	}, h.presenceVisible)
}
