package store

// Stage 2 — Status (Updates) gaps: text/audio stories, viewers,
// reactions, replies, per-user status privacy. In-memory
// implementations; the Postgres versions live in postgres_stage2.go.
// Schema: migrations/012_stage2_status.sql.

import (
	"sort"
	"time"

	"github.com/vyro/backend/internal/models"
)

const (
	// StatusEveryone is the default: anyone may see my status.
	StatusEveryone = "everyone"
	// StatusNobody hides my status from everyone except myself.
	StatusNobody = "nobody"
)

// ValidStatusPrivacy reports whether v is an accepted status-privacy value.
func ValidStatusPrivacy(v string) bool {
	return v == StatusEveryone || v == StatusNobody
}

// StoryByID returns the story with the given id, or nil. Expired
// stories are returned too — callers check ExpiresAt.
func (s *MemoryStore) StoryByID(id string) *models.Story {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stories[id]
}

// DeleteStory removes a story and its views, reactions and replies.
func (s *MemoryStore) DeleteStory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stories[id]; !ok {
		return ErrNotFound
	}
	delete(s.stories, id)
	delete(s.storyViews, id)
	delete(s.storyReactions, id)
	delete(s.storyReplies, id)
	return nil
}

// RecordStoryView records that userID viewed storyID. Viewing twice
// keeps the first viewed_at.
func (s *MemoryStore) RecordStoryView(storyID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stories[storyID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	if s.storyViews[storyID] == nil {
		s.storyViews[storyID] = map[string]time.Time{}
	}
	if _, ok := s.storyViews[storyID][userID]; !ok {
		s.storyViews[storyID][userID] = time.Now().UTC()
	}
	return nil
}

// ListStoryViews returns who viewed a story, oldest view first.
func (s *MemoryStore) ListStoryViews(storyID string) []*models.StoryView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.StoryView{}
	for userID, at := range s.storyViews[storyID] {
		v := &models.StoryView{UserID: userID, ViewedAt: at}
		if u, ok := s.users[userID]; ok {
			v.DisplayName = u.DisplayName
			v.Username = u.Username
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ViewedAt.Before(out[j].ViewedAt) })
	return out
}

// StoryViewCount returns how many distinct users viewed a story.
func (s *MemoryStore) StoryViewCount(storyID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.storyViews[storyID])
}

// SetStoryReaction records (or replaces) one user's emoji reaction to
// a story. One reaction per user.
func (s *MemoryStore) SetStoryReaction(storyID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stories[storyID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	if s.storyReactions[storyID] == nil {
		s.storyReactions[storyID] = map[string]storyReaction{}
	}
	s.storyReactions[storyID][userID] = storyReaction{emoji: emoji, createdAt: time.Now().UTC()}
	return nil
}

// DeleteStoryReaction removes one user's reaction, if any.
func (s *MemoryStore) DeleteStoryReaction(storyID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.storyReactions[storyID] != nil {
		delete(s.storyReactions[storyID], userID)
	}
	return nil
}

// ListStoryReactions returns a story's reactions, oldest first.
func (s *MemoryStore) ListStoryReactions(storyID string) []*models.StoryReaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.StoryReaction{}
	for userID, r := range s.storyReactions[storyID] {
		re := &models.StoryReaction{UserID: userID, Emoji: r.emoji, CreatedAt: r.createdAt}
		if u, ok := s.users[userID]; ok {
			re.DisplayName = u.DisplayName
			re.Username = u.Username
		}
		out = append(out, re)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// MyStoryReaction returns the requester's own emoji on a story, or "".
func (s *MemoryStore) MyStoryReaction(storyID, userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.storyReactions[storyID] == nil {
		return ""
	}
	return s.storyReactions[storyID][userID].emoji
}

// AddStoryReply stores a text reply to a story.
func (s *MemoryStore) AddStoryReply(storyID, userID, content string) (*models.StoryReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stories[storyID]; !ok {
		return nil, ErrNotFound
	}
	if _, ok := s.users[userID]; !ok {
		return nil, ErrNotFound
	}
	r := &models.StoryReply{
		ID: NewID(), StoryID: storyID, UserID: userID,
		Content: content, CreatedAt: time.Now().UTC(),
	}
	if u, ok := s.users[userID]; ok {
		r.DisplayName = u.DisplayName
		r.Username = u.Username
	}
	s.storyReplies[storyID] = append(s.storyReplies[storyID], r)
	return r, nil
}

// ListStoryReplies returns a story's replies, oldest first.
func (s *MemoryStore) ListStoryReplies(storyID string) []*models.StoryReply {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.StoryReply{}
	for _, r := range s.storyReplies[storyID] {
		cp := *r
		out = append(out, &cp)
	}
	return out
}

// GetStatusPrivacy returns who may see userID's status
// ("everyone" by default).
func (s *MemoryStore) GetStatusPrivacy(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.statusPrivacy[userID]; ok {
		return v
	}
	return StatusEveryone
}

// SetStatusPrivacy stores a user's status-privacy choice.
func (s *MemoryStore) SetStatusPrivacy(userID, privacy string) error {
	if !ValidStatusPrivacy(privacy) {
		return ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	s.statusPrivacy[userID] = privacy
	return nil
}
