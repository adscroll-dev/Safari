package store

import (
	"sort"
	"strings"
	"time"

	"github.com/vyro/backend/internal/models"
)

// Stage 5 — Channels, MemoryStore implementation.
// Postgres versions live in postgres_stage5.go.
// Schema: migrations/015_stage5_channels.sql.

// RecordMessageView records that a user viewed a message.
// Idempotent: viewing again changes nothing.
func (s *MemoryStore) RecordMessageView(messageID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.messages[messageID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	if s.messageViews[messageID] == nil {
		s.messageViews[messageID] = map[string]bool{}
	}
	s.messageViews[messageID][userID] = true
	return nil
}

// MessageViewCount returns the number of distinct viewers.
func (s *MemoryStore) MessageViewCount(messageID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messageViews[messageID])
}

// MessageViewCounts returns per-message viewer counts for a chat.
func (s *MemoryStore) MessageViewCounts(chatID string) map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]int{}
	for _, id := range s.chatMsgs[chatID] {
		if n := len(s.messageViews[id]); n > 0 {
			out[id] = n
		}
	}
	return out
}

// SearchPublicChannels finds public channels whose title contains
// the query (case-insensitive), most-followed first, max 20.
func (s *MemoryStore) SearchPublicChannels(query string) []*models.Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return []*models.Chat{}
	}
	out := []*models.Chat{}
	for _, c := range s.chats {
		if c.Type != models.ChatChannel || !c.IsPublic {
			continue
		}
		if !strings.Contains(strings.ToLower(c.Title), q) {
			continue
		}
		cp := *c
		cp.FollowerCount = len(c.MemberIDs)
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FollowerCount != out[j].FollowerCount {
			return out[i].FollowerCount > out[j].FollowerCount
		}
		return out[i].Title < out[j].Title
	})
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

// UpdateChat changes a chat's title and/or public flag (channels
// only for the flag). A nil pointer leaves that field unchanged.
func (s *MemoryStore) UpdateChat(chatID string, title *string, isPublic *bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chats[chatID]
	if !ok {
		return ErrNotFound
	}
	if title != nil {
		c.Title = *title
	}
	if isPublic != nil && c.Type == models.ChatChannel {
		c.IsPublic = *isPublic
	}
	c.UpdatedAt = time.Now().UTC()
	return nil
}
