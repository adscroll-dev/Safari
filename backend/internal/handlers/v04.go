// Package handlers — v0.4 additions: read receipts, presence, stories,
// scheduled + disappearing messages, group/channel admin, per-user chat
// settings (pin/archive/folders), bots, and the E2EE key foundation.
// Messages stay plaintext on the server in v0.4 — see docs/SECURITY.md.
package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vyro/backend/internal/auth"
	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/store"
	"github.com/vyro/backend/internal/ws"
)

// --- helpers ---

func (h *Handler) chatMemberIDs(chatID string) []string {
	chat, err := h.Store.ChatByID(chatID)
	if err != nil {
		return []string{}
	}
	return chat.MemberIDs
}

// broadcastToChatAs sends ev to the members of chatID (per-chat rooms)
// and returns the user ids that received it live. The sender never
// receives their own event back.
func (h *Handler) broadcastToChatAs(chatID, senderID string, ev ws.Event) []string {
	return h.Hub.BroadcastToChat(chatID, h.chatMemberIDs(chatID), senderID, ev)
}

func (h *Handler) requireRole(w http.ResponseWriter, r *http.Request, chatID string, roles ...string) bool {
	role := h.Store.MemberRole(chatID, middleware.UserID(r))
	for _, want := range roles {
		if role == want {
			return true
		}
	}
	middleware.WriteError(w, http.StatusForbidden, "requires role "+strings.Join(roles, " or "))
	return false
}

func (h *Handler) applySettings(w http.ResponseWriter, r *http.Request, chats []*models.Chat) {
	me := middleware.UserID(r)
	for _, c := range chats {
		cs := h.Store.GetChatSettings(c.ID, me)
		c.Pinned, c.Archived, c.Folder = cs.Pinned, cs.Archived, cs.Folder
		c.Muted = cs.Muted
		c.Roles = map[string]string{}
		for _, m := range h.Store.ChatMembers(c.ID) {
			c.Roles[m.UserID] = m.Role
		}
	}
}

// --- Read receipts (ticks) ---

// MarkRead marks a message read by the caller and notifies the chat.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
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
	me := middleware.UserID(r)
	if !h.Store.IsMember(msg.ChatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	if me != msg.SenderID {
		// Stage 11: with read receipts off, the reader sends no
		// read receipts at all — the sender never sees blue ticks
		// from them, and no message_read event is broadcast.
		if !h.Store.GetReadReceipts(me) {
			updated, _ := h.Store.MessageByID(id)
			middleware.WriteJSON(w, http.StatusOK, updated)
			return
		}
		if err := h.Store.MarkRead(id, me); err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, "could not mark read")
			return
		}
		h.broadcastToChatAs(msg.ChatID, me, ws.Event{
			Type: "message_read", ChatID: msg.ChatID, MessageID: id, UserID: me,
		})
		// Multi-device sync: the reader's other devices learn the
		// message was read here, so ticks stay consistent.
		h.Hub.BroadcastToUser(me, ws.Event{
			Type: "message_read", ChatID: msg.ChatID, MessageID: id, UserID: me,
		})
	}
	updated, _ := h.Store.MessageByID(id)
	middleware.WriteJSON(w, http.StatusOK, updated)
}

// --- Presence ---

// Presence returns a user's online state and last-seen time.
func (h *Handler) Presence(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/users/")
	id = strings.TrimSuffix(id, "/presence")
	if id == "" {
		middleware.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if _, err := h.Store.UserByID(id); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	// Last-seen privacy: "nobody" hides online state and last-seen
	// time from everyone except the user themself.
	if id != middleware.UserID(r) &&
		h.Store.GetLastSeenVisibility(id) == store.LastSeenNobody {
		middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"user_id": id, "is_online": false, "last_seen_at": nil,
		})
		return
	}
	online, lastSeen := h.Store.GetPresence(id)
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": id, "is_online": online, "last_seen_at": lastSeen,
	})
}

// --- Stories (24h) ---

type createStoryReq struct {
	MediaURL        string `json:"media_url"`
	Caption         string `json:"caption"`
	Kind            string `json:"kind"`
	TextContent     string `json:"text_content"`
	BackgroundColor string `json:"background_color"`
}

func (h *Handler) CreateStory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req createStoryReq
	if !decode(w, r, &req) {
		return
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		kind = "photo"
	}
	switch kind {
	case "photo", "video", "text", "audio":
	default:
		middleware.WriteError(w, http.StatusBadRequest, "kind must be photo, video, text or audio")
		return
	}
	caption := strings.TrimSpace(req.Caption)
	text := strings.TrimSpace(req.TextContent)
	bg := strings.TrimSpace(req.BackgroundColor)
	switch kind {
	case "text":
		// A text status has no media: text + a background colour
		// from the fixed palette.
		if req.MediaURL != "" {
			middleware.WriteError(w, http.StatusBadRequest, "text status must not have media_url")
			return
		}
		if text == "" {
			middleware.WriteError(w, http.StatusBadRequest, "text_content is required for a text status")
			return
		}
		if utf8.RuneCountInString(text) > 700 {
			middleware.WriteError(w, http.StatusBadRequest, "text status too long (max 700 characters)")
			return
		}
		if bg != "" && !validStoryBackgroundColor(bg) {
			middleware.WriteError(w, http.StatusBadRequest, "background_color is not in the allowed palette")
			return
		}
		if bg == "" {
			bg = storyBackgroundPalette[0]
		}
		caption = ""
	default:
		// photo / video / audio: media must come from /api/uploads.
		if !strings.HasPrefix(req.MediaURL, "/uploads/") || strings.Contains(req.MediaURL, "..") {
			middleware.WriteError(w, http.StatusBadRequest, "media_url from /api/uploads is required")
			return
		}
	}
	st := &models.Story{UserID: middleware.UserID(r), MediaURL: req.MediaURL, Caption: caption, Kind: kind, TextContent: text, BackgroundColor: bg}
	if err := h.Store.CreateStory(st); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create story")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, st)
}

func (h *Handler) ListStories(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	me := middleware.UserID(r)
	var out []*models.Story
	if userID != "" {
		out = h.Store.ListStoriesByUser(userID)
	} else {
		out = h.Store.ListStories()
	}
	if out == nil {
		out = []*models.Story{}
	}
	middleware.WriteJSON(w, http.StatusOK, h.visibleStories(out, me))
}

// visibleStories filters out stories the requester may not see (the
// owner set status privacy to "nobody") and populates ViewCount
// (owner only) and MyReaction on COPIES — the stored structs are
// never mutated.
func (h *Handler) visibleStories(in []*models.Story, me string) []*models.Story {
	out := []*models.Story{}
	for _, st := range in {
		if h.statusHiddenFrom(st.UserID, me) {
			continue
		}
		cp := *st
		if st.UserID == me {
			cp.ViewCount = h.Store.StoryViewCount(st.ID)
		}
		cp.MyReaction = h.Store.MyStoryReaction(st.ID, me)
		out = append(out, &cp)
	}
	return out
}

// --- Scheduled messages ---

type scheduleReq struct {
	Content   string `json:"content"`
	Kind      string `json:"kind"`
	ReplyToID string `json:"reply_to_id"`
	FileURL   string `json:"file_url"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	MimeType  string `json:"mime_type"`
	SendAt    string `json:"send_at"` // RFC3339
}

func (h *Handler) ScheduleMessage(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	var req scheduleReq
	if !decode(w, r, &req) {
		return
	}
	sendAt, err := time.Parse(time.RFC3339, req.SendAt)
	if err != nil || sendAt.Before(time.Now().Add(5*time.Second)) {
		middleware.WriteError(w, http.StatusBadRequest, "send_at must be an RFC3339 time at least 5 seconds in the future")
		return
	}
	if sendAt.After(time.Now().Add(365 * 24 * time.Hour)) {
		middleware.WriteError(w, http.StatusBadRequest, "send_at is too far in the future (max 1 year)")
		return
	}
	kind := req.Kind
	if kind == "" {
		kind = "text"
	}
	if kind != "text" && kind != "image" && kind != "video" && kind != "file" && kind != "voice" && kind != "sticker" {
		middleware.WriteError(w, http.StatusBadRequest, "kind '"+kind+"' cannot be scheduled")
		return
	}
	if kind == "e2ee" {
		// Encrypted messages cannot be scheduled: the client holds the
		// session, the server must never see plaintext to re-encrypt.
		middleware.WriteError(w, http.StatusBadRequest, "e2ee messages cannot be scheduled")
		return
	}
	if kind == "text" && strings.TrimSpace(req.Content) == "" {
		middleware.WriteError(w, http.StatusBadRequest, "content is required")
		return
	}
	if kind != "text" && (!strings.HasPrefix(req.FileURL, "/uploads/") || strings.Contains(req.FileURL, "..")) {
		middleware.WriteError(w, http.StatusBadRequest, "file_url from /api/uploads is required")
		return
	}
	sm := &models.ScheduledMessage{
		ChatID: chatID, SenderID: me, Content: req.Content, Kind: kind,
		ReplyToID: req.ReplyToID, FileURL: req.FileURL, FileName: req.FileName,
		FileSize: req.FileSize, MimeType: req.MimeType, SendAt: sendAt.UTC(),
	}
	if err := h.Store.CreateScheduled(sm); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, sm)
}

func (h *Handler) ListScheduled(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.Store.IsMember(chatID, middleware.UserID(r)) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	out := h.Store.ListScheduled(chatID, middleware.UserID(r))
	if out == nil {
		out = []*models.ScheduledMessage{}
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) CancelScheduled(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/scheduled/")
	id := strings.Split(rest, "/")[0]
	if err := h.Store.CancelScheduled(id, middleware.UserID(r)); err != nil {
		if err == store.ErrForbidden {
			middleware.WriteError(w, http.StatusForbidden, "only the sender can cancel this message")
			return
		}
		middleware.WriteError(w, http.StatusNotFound, "scheduled message not found or already sent")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ProcessDueScheduled sends every due scheduled message as a real message
// (called by the server ticker and by tests). It returns the messages sent.
func (h *Handler) ProcessDueScheduled() []*models.Message {
	now := time.Now().UTC()
	sent := []*models.Message{}
	for _, sm := range h.Store.DueScheduled(now) {
		msg := &models.Message{
			ChatID: sm.ChatID, SenderID: sm.SenderID, Content: sm.Content, Kind: sm.Kind,
			ReplyToID: sm.ReplyToID, FileURL: sm.FileURL, FileName: sm.FileName,
			FileSize: sm.FileSize, MimeType: sm.MimeType,
		}
		if err := h.Store.CreateMessage(msg); err != nil {
			continue
		}
		_ = h.Store.MarkScheduledSent(sm.ID, msg.ID)
		h.broadcastToChatAs(sm.ChatID, sm.SenderID, ws.Event{Type: "message", ChatID: sm.ChatID, Message: msg})
		sent = append(sent, msg)
	}
	return sent
}

// --- Group / channel admin ---

type addMemberReq struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // member | admin (owner cannot be granted here)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.requireRole(w, r, chatID, "owner", "admin") {
		return
	}
	var req addMemberReq
	if !decode(w, r, &req) {
		return
	}
	if req.Role == "owner" {
		middleware.WriteError(w, http.StatusBadRequest, "owner role cannot be granted")
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	// Admins may only add plain members; only the owner may promote admins.
	if req.Role == "admin" && h.Store.MemberRole(chatID, middleware.UserID(r)) != "owner" {
		middleware.WriteError(w, http.StatusForbidden, "only the owner can add admins")
		return
	}
	// Stage 11: a user who set groups privacy to "nobody" cannot
	// be added to a group by anyone else (they can still join via
	// an invite link — that path does not go through here).
	if req.UserID != middleware.UserID(r) && h.Store.GetGroupsPrivacy(req.UserID) == store.LastSeenNobody {
		middleware.WriteError(w, http.StatusForbidden, "this user does not allow being added to groups")
		return
	}
	if err := h.Store.AddMember(chatID, req.UserID, req.Role); err != nil {
		if err == store.ErrAlreadyExists {
			middleware.WriteError(w, http.StatusConflict, "user is already a member")
			return
		}
		middleware.WriteError(w, http.StatusNotFound, "chat or user not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "added", "role": req.Role})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.requireRole(w, r, chatID, "owner", "admin") {
		return
	}
	// /api/chats/{chatID}/members/{userID}
	rest := strings.TrimPrefix(r.URL.Path, "/api/chats/")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid member id")
		return
	}
	targetID := parts[2]
	targetRole := h.Store.MemberRole(chatID, targetID)
	me := middleware.UserID(r)
	myRole := h.Store.MemberRole(chatID, me)
	if targetRole == "owner" {
		middleware.WriteError(w, http.StatusForbidden, "the owner cannot be removed")
		return
	}
	if targetRole == "admin" && myRole != "owner" {
		middleware.WriteError(w, http.StatusForbidden, "only the owner can remove an admin")
		return
	}
	if err := h.Store.RemoveMember(chatID, targetID); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "member not found")
		return
	}
	// Group E2EE (v0.6): a removed member's devices must not pick up
	// sender keys afterwards. Senders rotate their sender key on
	// membership change (client-side); the server deletes the stored
	// distributions addressed to the removed member.
	_ = h.Store.DeleteGroupSenderKeysForUser(chatID, targetID)
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Handler) SetMemberRole(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.requireRole(w, r, chatID, "owner") {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/chats/")
	parts := strings.Split(rest, "/")
	if len(parts) < 4 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid member id")
		return
	}
	targetID := parts[2]
	if targetID == middleware.UserID(r) {
		middleware.WriteError(w, http.StatusForbidden, "the owner cannot change their own role")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Role == "owner" {
		middleware.WriteError(w, http.StatusBadRequest, "ownership cannot be transferred here")
		return
	}
	if err := h.Store.SetMemberRole(chatID, targetID, req.Role); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "member not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "role updated", "role": req.Role})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	chatID := chatIDFromPath(r)
	if !h.Store.IsMember(chatID, middleware.UserID(r)) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	out := h.Store.ChatMembers(chatID)
	if out == nil {
		out = []*models.ChatMember{}
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

// --- Per-user chat settings: pin / archive / folders ---

func (h *Handler) SetChatSetting(w http.ResponseWriter, r *http.Request, kind string) {
	chatID := chatIDFromPath(r)
	if !h.Store.IsMember(chatID, middleware.UserID(r)) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	cs := h.Store.GetChatSettings(chatID, middleware.UserID(r))
	switch kind {
	case "pin":
		cs.Pinned = true
	case "unpin":
		cs.Pinned = false
	case "archive":
		cs.Archived = true
	case "unarchive":
		cs.Archived = false
	case "mute":
		cs.Muted = true
	case "unmute":
		cs.Muted = false
	case "folder":
		var req struct {
			Folder string `json:"folder"`
		}
		if !decode(w, r, &req) {
			return
		}
		if len(req.Folder) > 32 {
			middleware.WriteError(w, http.StatusBadRequest, "folder name too long (max 32 characters)")
			return
		}
		cs.Folder = strings.TrimSpace(req.Folder)
	}
	if err := h.Store.SetChatSettings(cs); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not save setting")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, cs)
}

// --- Bots ---

func newBotToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type createBotReq struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

func (h *Handler) CreateBot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req createBotReq
	if !decode(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	username := strings.ToLower(strings.TrimSpace(req.Username))
	if len(username) < 3 || len(username) > 32 || strings.Contains(username, " ") {
		middleware.WriteError(w, http.StatusBadRequest, "username must be 3-32 characters without spaces")
		return
	}
	if !strings.HasPrefix(username, "bot_") {
		username = "bot_" + username
	}
	me := middleware.UserID(r)
	// The bot is a real user account (IsBot) so DMs with it work normally.
	hash, err := auth.HashPassword("bot-" + username + "-no-login")
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create bot")
		return
	}
	botUser := &models.User{
		Email: username + "@bot.vyro", Username: username,
		DisplayName: name, PasswordHash: hash, IsBot: true,
	}
	if err := h.Store.CreateUser(botUser); err != nil {
		middleware.WriteError(w, http.StatusConflict, "bot username already taken")
		return
	}
	token, err := newBotToken()
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create bot token")
		return
	}
	bot := &models.Bot{OwnerID: me, Name: name, Username: username, Token: token}
	if err := h.Store.CreateBot(bot); err != nil {
		middleware.WriteError(w, http.StatusConflict, "bot could not be created")
		return
	}
	// Return the raw token exactly once; the store keeps only its hash
	// in PostgreSQL (in-memory dev store keeps the token for lookup).
	middleware.WriteJSON(w, http.StatusCreated, bot)
}

func (h *Handler) ListBots(w http.ResponseWriter, r *http.Request) {
	out := h.Store.ListBots(middleware.UserID(r))
	if out == nil {
		out = []*models.Bot{}
	}
	for _, b := range out {
		b.Token = "" // tokens are shown once, at creation
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) DeleteBot(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/bots/")
	id = strings.Split(id, "/")[0]
	if err := h.Store.DeleteBot(id, middleware.UserID(r)); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "bot not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SetBotWebhook(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/bots/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid bot id")
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.URL != "" {
		u, err := url.Parse(req.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			middleware.WriteError(w, http.StatusBadRequest, "webhook url must be http(s)")
			return
		}
	}
	if err := h.Store.SetBotWebhook(parts[0], middleware.UserID(r), req.URL); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "bot not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "webhook updated"})
}

// botFromToken authenticates a bot request via the X-Bot-Token header.
func (h *Handler) botFromToken(r *http.Request) (*models.Bot, bool) {
	token := r.Header.Get("X-Bot-Token")
	if token == "" {
		return nil, false
	}
	bot, err := h.Store.BotByToken(token)
	if err != nil {
		return nil, false
	}
	return bot, true
}

// BotUpdates returns recent messages in the bot's chats (polling), newest
// last, excluding the bot's own messages.
func (h *Handler) BotUpdates(w http.ResponseWriter, r *http.Request) {
	bot, ok := h.botFromToken(r)
	if !ok {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid bot token")
		return
	}
	botUser, err := h.Store.UserByUsername(bot.Username)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "bot user not found")
		return
	}
	out := []*models.Message{}
	for _, chat := range h.Store.ChatsForUser(botUser.ID) {
		for _, m := range h.Store.MessagesForChat(chat.ID, 20, 0) {
			if m.SenderID != botUser.ID {
				out = append(out, m)
			}
		}
	}
	if out == nil {
		out = []*models.Message{}
	}
	middleware.WriteJSON(w, http.StatusOK, out)
}

// BotSend sends a message as the bot (bot token auth).
// Route: POST /api/bot/send {chat_id, content} with X-Bot-Token header.
func (h *Handler) BotSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	bot, ok := h.botFromToken(r)
	if !ok {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid bot token")
		return
	}
	botUser, err := h.Store.UserByUsername(bot.Username)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "bot user not found")
		return
	}
	var req struct {
		ChatID  string `json:"chat_id"`
		Content string `json:"content"`
	}
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		middleware.WriteError(w, http.StatusBadRequest, "content is required")
		return
	}
	if !h.Store.IsMember(req.ChatID, botUser.ID) {
		middleware.WriteError(w, http.StatusForbidden, "bot is not a member of this chat")
		return
	}
	msg := &models.Message{ChatID: req.ChatID, SenderID: botUser.ID, Content: req.Content, Kind: "text"}
	if err := h.Store.CreateMessage(msg); err != nil {
		middleware.WriteError(w, http.StatusNotFound, "chat not found")
		return
	}
	delivered := h.broadcastToChatAs(req.ChatID, botUser.ID, ws.Event{Type: "message", ChatID: req.ChatID, Message: msg})
	for _, uid := range delivered {
		_ = h.Store.MarkDelivered(msg.ID, uid)
	}
	middleware.WriteJSON(w, http.StatusCreated, msg)
}

// notifyBotWebhook POSTs a message to a bot's webhook (best effort, 2s
// timeout, errors ignored) when someone messages the bot in a DM.
func (h *Handler) notifyBotWebhook(bot *models.Bot, msg *models.Message) {
	if bot == nil || bot.WebhookURL == "" {
		return
	}
	body, _ := json.Marshal(map[string]interface{}{"bot_id": bot.ID, "message": msg})
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		req, err := http.NewRequest(http.MethodPost, bot.WebhookURL, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}

// --- E2EE key foundation (public keys only) ---

func validB64(s string, maxLen int) bool {
	if s == "" || len(s) > maxLen {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func (h *Handler) UploadIdentityKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID        string `json:"device_id"`
		IdentityKey     string `json:"identity_key"`
		SignedPreKey    string `json:"signed_prekey"`
		SignedPreKeyID  int    `json:"signed_prekey_id"`
		SignKey         string `json:"sign_key"`
		SignedPreKeySig string `json:"signed_prekey_sig"`
	}
	if !decode(w, r, &req) {
		return
	}
	if len(req.DeviceID) == 0 || len(req.DeviceID) > 64 ||
		!validB64(req.IdentityKey, 512) || !validB64(req.SignedPreKey, 512) {
		middleware.WriteError(w, http.StatusBadRequest, "device_id and base64 identity_key / signed_prekey are required")
		return
	}
	// Optional v0.5 fields: Ed25519 signing key (32 bytes) + signature
	// over the signed pre-key (64 bytes). If one is present both must
	// be present and well-formed; the signature itself is verified by
	// the *peer* client (e2ee.Bundle.Verify), not the server.
	if (req.SignKey != "" || req.SignedPreKeySig != "") &&
		(!validB64(req.SignKey, 512) || !validB64(req.SignedPreKeySig, 512)) {
		middleware.WriteError(w, http.StatusBadRequest, "sign_key / signed_prekey_sig must be base64")
		return
	}
	k := &models.IdentityKey{
		UserID: middleware.UserID(r), DeviceID: req.DeviceID,
		IdentityKey: req.IdentityKey, SignedPreKey: req.SignedPreKey,
		SignedPreKeyID: req.SignedPreKeyID,
		SignKey:        req.SignKey, SignedPreKeySig: req.SignedPreKeySig,
	}
	if err := h.Store.UploadIdentityKey(k); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not store identity key")
		return
	}
	// Multi-device (v0.6): uploading keys proves the device exists, so
	// the device is registered (upsert) even if the client never called
	// POST /api/devices. Name/platform stay empty until registered.
	_ = h.Store.RegisterDevice(&models.Device{UserID: k.UserID, DeviceID: k.DeviceID})
	middleware.WriteJSON(w, http.StatusOK, k)
}

func (h *Handler) GetIdentityKey(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/keys/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid key path")
		return
	}
	k, err := h.Store.GetIdentityKey(parts[0], parts[1])
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "identity key not found")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, k)
}

func (h *Handler) UploadPreKeys(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID string `json:"device_id"`
		Keys     []struct {
			KeyID     int    `json:"key_id"`
			PublicKey string `json:"public_key"`
		} `json:"keys"`
	}
	if !decode(w, r, &req) {
		return
	}
	if len(req.DeviceID) == 0 || len(req.DeviceID) > 64 || len(req.Keys) == 0 || len(req.Keys) > 200 {
		middleware.WriteError(w, http.StatusBadRequest, "device_id and 1-200 keys are required")
		return
	}
	keys := make([]*models.OneTimePreKey, 0, len(req.Keys))
	for _, k := range req.Keys {
		if !validB64(k.PublicKey, 512) {
			middleware.WriteError(w, http.StatusBadRequest, "public_key must be base64")
			return
		}
		keys = append(keys, &models.OneTimePreKey{KeyID: k.KeyID, PublicKey: k.PublicKey})
	}
	if err := h.Store.UploadOneTimePreKeys(middleware.UserID(r), req.DeviceID, keys); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not store pre-keys")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{"status": "stored", "count": len(keys)})
}

func (h *Handler) FetchPreKey(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/keys/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[1] != "prekey" {
		middleware.WriteError(w, http.StatusBadRequest, "invalid key path")
		return
	}
	k, err := h.Store.FetchOneTimePreKey(parts[0])
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "no unused pre-key available")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, k)
}
