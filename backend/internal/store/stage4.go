package store

import (
	"errors"
	"sort"
	"time"

	"github.com/vyro/backend/internal/models"
)

// Stage 4 — Polls, MemoryStore implementation.
// Postgres versions live in postgres_stage4.go.
// Schema: migrations/014_stage4_polls.sql.

// CreatePoll stores a new poll; the id and timestamp are assigned
// here (Postgres generates them with gen_random_uuid() instead).
func (s *MemoryStore) CreatePoll(p *models.Poll) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[p.ChatID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[p.CreatorID]; !ok {
		return ErrNotFound
	}
	if p.ID == "" {
		p.ID = NewID()
	}
	p.CreatedAt = time.Now().UTC()
	cp := *p
	cp.Options = append([]string{}, p.Options...)
	s.polls[p.ID] = &cp
	return nil
}

// PollByID returns the poll with the given id, or nil.
func (s *MemoryStore) PollByID(id string) *models.Poll {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.polls[id]
	if !ok {
		return nil
	}
	cp := *p
	cp.Options = append([]string{}, p.Options...)
	return &cp
}

// ListPollsForChat returns the chat's polls, newest first.
func (s *MemoryStore) ListPollsForChat(chatID string) []*models.Poll {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Poll{}
	for _, p := range s.polls {
		if p.ChatID != chatID {
			continue
		}
		cp := *p
		cp.Options = append([]string{}, p.Options...)
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// VotePoll replaces the user's votes on a poll (re-voting is allowed;
// an empty list clears the vote).
func (s *MemoryStore) VotePoll(pollID, userID string, indexes []int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.polls[pollID]
	if !ok {
		return ErrNotFound
	}
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	for _, i := range indexes {
		if i < 0 || i >= len(p.Options) {
			return errors.New("option index out of range")
		}
	}
	if s.pollVotes[pollID] == nil {
		s.pollVotes[pollID] = map[string][]int{}
	}
	s.pollVotes[pollID][userID] = append([]int{}, indexes...)
	return nil
}

// PollVotes returns per-option vote counts and the total number of
// votes cast on the poll.
func (s *MemoryStore) PollVotes(pollID string) (map[int]int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := map[int]int{}
	total := 0
	for _, indexes := range s.pollVotes[pollID] {
		for _, i := range indexes {
			counts[i]++
			total++
		}
	}
	return counts, total
}

// MyPollVotes returns the option indexes the user voted for.
func (s *MemoryStore) MyPollVotes(pollID, userID string) []int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	votes := s.pollVotes[pollID][userID]
	if votes == nil {
		return []int{}
	}
	return append([]int{}, votes...)
}

// DeletePoll removes a poll and its votes. The poll message in the
// chat is deleted separately by the caller (it may already be gone).
func (s *MemoryStore) DeletePoll(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.polls[id]; !ok {
		return ErrNotFound
	}
	delete(s.polls, id)
	delete(s.pollVotes, id)
	return nil
}

// SetPollMessageID links a poll to the message that renders it in
// the chat flow.
func (s *MemoryStore) SetPollMessageID(pollID, messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.polls[pollID]
	if !ok {
		return ErrNotFound
	}
	p.MessageID = messageID
	return nil
}
