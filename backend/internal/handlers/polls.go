package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/ws"
)

// Stage 4 — Polls (WhatsApp parity): a poll lives in a chat and
// appears in the chat flow as a message of kind "poll" whose content
// is JSON {"poll_id": "...", "question": "..."}.
// Endpoints (see cmd/server/main.go):
//   POST /api/chats/{id}/polls   create a poll (any member)
//   GET  /api/chats/{id}/polls   list the chat's polls
//   GET  /api/polls/{id}         one poll (member of its chat)
//   POST /api/polls/{id}/vote    vote / re-vote / clear my vote
//   DELETE /api/polls/{id}       delete (creator or admin/owner)

const (
	maxPollQuestionRunes = 300
	maxPollOptions       = 12
	minPollOptions       = 2
	maxPollOptionRunes   = 100
)

// pollView enriches a stored poll with live vote counts and the
// requester's own votes.
func (h *Handler) pollView(p *models.Poll, me string) *models.Poll {
	cp := *p
	counts, total := h.Store.PollVotes(p.ID)
	cp.Votes = make([]int, len(p.Options))
	for i := range cp.Votes {
		cp.Votes[i] = counts[i]
	}
	cp.TotalVotes = total
	cp.MyVotes = h.Store.MyPollVotes(p.ID, me)
	return &cp
}

func pollIDFromPath(r *http.Request) string {
	// /api/polls/{id}[/suffix]
	rest := strings.TrimPrefix(r.URL.Path, "/api/polls/")
	parts := strings.SplitN(rest, "/", 2)
	return parts[0]
}

// PollSuffix returns the part after /api/polls/{id}/ ("" when none).
func PollSuffix(r *http.Request) string {
	rest := strings.TrimPrefix(r.URL.Path, "/api/polls/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// CreatePoll creates a poll and posts it into the chat as a "poll"
// message, broadcast like any other message.
func (h *Handler) CreatePoll(w http.ResponseWriter, r *http.Request) {
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
		Question      string   `json:"question"`
		Options       []string `json:"options"`
		AllowMultiple bool     `json:"allow_multiple"`
	}
	if !decode(w, r, &req) {
		return
	}
	question := strings.TrimSpace(req.Question)
	if question == "" || utf8.RuneCountInString(question) > maxPollQuestionRunes {
		middleware.WriteError(w, http.StatusBadRequest, "question is required (max 300 characters)")
		return
	}
	options := []string{}
	for _, o := range req.Options {
		o = strings.TrimSpace(o)
		if o == "" || utf8.RuneCountInString(o) > maxPollOptionRunes {
			continue
		}
		options = append(options, o)
	}
	if len(options) < minPollOptions || len(options) > maxPollOptions {
		middleware.WriteError(w, http.StatusBadRequest, "a poll needs 2-12 options")
		return
	}
	poll := &models.Poll{
		ChatID: chatID, CreatorID: me,
		Question: question, Options: options,
		AllowMultiple: req.AllowMultiple,
	}
	if err := h.Store.CreatePoll(poll); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not create poll")
		return
	}
	// Post the poll into the chat flow (same broadcast as SendMessage).
	content, _ := json.Marshal(map[string]string{
		"poll_id": poll.ID, "question": question,
	})
	msg := &models.Message{
		ChatID: chatID, SenderID: me,
		Kind: "poll", Content: string(content),
	}
	if err := h.Store.CreateMessage(msg); err != nil {
		// The poll exists even if its message could not be posted;
		// the chat's poll list still shows it.
		middleware.WriteJSON(w, http.StatusCreated, h.pollView(poll, me))
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
	h.Hub.BroadcastToUser(me, ws.Event{Type: "message_sync", ChatID: chatID, Message: msg})
	h.notifyOfflinePush(r.Context(), chatID, me, "poll", delivered)
	_ = h.Store.SetPollMessageID(poll.ID, msg.ID)
	poll.MessageID = msg.ID
	middleware.WriteJSON(w, http.StatusCreated, h.pollView(poll, me))
}

// ListPolls returns the chat's polls, newest first.
func (h *Handler) ListPolls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	chatID := chatIDFromPath(r)
	me := middleware.UserID(r)
	if !h.Store.IsMember(chatID, me) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return
	}
	polls := h.Store.ListPollsForChat(chatID)
	views := make([]*models.Poll, 0, len(polls))
	for _, p := range polls {
		views = append(views, h.pollView(p, me))
	}
	middleware.WriteJSON(w, http.StatusOK, views)
}

// pollForMember loads a poll and checks the requester is a member
// of its chat (404 unknown poll, 403 non-member).
func (h *Handler) pollForMember(w http.ResponseWriter, r *http.Request, id string) (*models.Poll, bool) {
	p := h.Store.PollByID(id)
	if p == nil {
		middleware.WriteError(w, http.StatusNotFound, "poll not found")
		return nil, false
	}
	if !h.Store.IsMember(p.ChatID, middleware.UserID(r)) {
		middleware.WriteError(w, http.StatusForbidden, "not a member of this chat")
		return nil, false
	}
	return p, true
}

// GetPoll returns one poll with live vote counts.
func (h *Handler) GetPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	p, ok := h.pollForMember(w, r, pollIDFromPath(r))
	if !ok {
		return
	}
	middleware.WriteJSON(w, http.StatusOK, h.pollView(p, middleware.UserID(r)))
}

// VotePoll votes (or re-votes, or clears with an empty list).
// Single-choice polls accept 0 or 1 option; multiple-choice polls
// accept any number of distinct options.
func (h *Handler) VotePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	me := middleware.UserID(r)
	p, ok := h.pollForMember(w, r, pollIDFromPath(r))
	if !ok {
		return
	}
	var req struct {
		OptionIndexes []int `json:"option_indexes"`
	}
	if !decode(w, r, &req) {
		return
	}
	seen := map[int]bool{}
	for _, i := range req.OptionIndexes {
		if i < 0 || i >= len(p.Options) {
			middleware.WriteError(w, http.StatusBadRequest, "option index out of range")
			return
		}
		if seen[i] {
			middleware.WriteError(w, http.StatusBadRequest, "duplicate option index")
			return
		}
		seen[i] = true
	}
	if !p.AllowMultiple && len(req.OptionIndexes) > 1 {
		middleware.WriteError(w, http.StatusBadRequest, "this poll allows only one option")
		return
	}
	indexes := req.OptionIndexes
	if indexes == nil {
		indexes = []int{}
	}
	if err := h.Store.VotePoll(p.ID, me, indexes); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not record vote")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, h.pollView(h.Store.PollByID(p.ID), me))
}

// DeletePoll removes a poll and its chat message. Only the creator
// or a group admin/owner may delete it.
func (h *Handler) DeletePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		middleware.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	me := middleware.UserID(r)
	p, ok := h.pollForMember(w, r, pollIDFromPath(r))
	if !ok {
		return
	}
	role := h.Store.MemberRole(p.ChatID, me)
	if p.CreatorID != me && role != "admin" && role != "owner" {
		middleware.WriteError(w, http.StatusForbidden, "only the creator or an admin can delete this poll")
		return
	}
	if p.MessageID != "" {
		_ = h.Store.DeleteMessage(p.MessageID)
	}
	if err := h.Store.DeletePoll(p.ID); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "could not delete poll")
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// systemNotice posts a server-made notice into a chat (kind
// "system", rendered by clients as a centered grey line, like
// WhatsApp's "X pinned a message"). Clients cannot send this kind.
func (h *Handler) systemNotice(r *http.Request, chatID, senderID, text string) {
	msg := &models.Message{
		ChatID: chatID, SenderID: senderID, Kind: "system", Content: text,
	}
	if err := h.Store.CreateMessage(msg); err != nil {
		return
	}
	msg.Status = "sent"
	delivered := h.broadcastToChatAs(chatID, senderID, ws.Event{Type: "message", ChatID: chatID, Message: msg})
	for _, uid := range delivered {
		_ = h.Store.MarkDelivered(msg.ID, uid)
	}
	if len(delivered) > 0 {
		msg.DeliveredTo = delivered
		msg.Status = "delivered"
	}
	h.Hub.BroadcastToUser(senderID, ws.Event{Type: "message_sync", ChatID: chatID, Message: msg})
	h.notifyOfflinePush(r.Context(), chatID, senderID, "system", delivered)
}
