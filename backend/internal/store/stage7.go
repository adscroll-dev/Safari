package store

import (
	"time"

	"github.com/vyro/backend/internal/models"
)

// Stage 7 (accounts): profile editing, unblocking, blocked list,
// user reports — in-memory implementations.

// UpdateProfile applies the non-nil fields to the user and bumps
// UpdatedAt. Unknown user -> ErrNotFound.
func (s *MemoryStore) UpdateProfile(userID string, displayName, bio, avatarURL *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok {
		return ErrNotFound
	}
	if displayName != nil {
		u.DisplayName = *displayName
	}
	if bio != nil {
		u.Bio = *bio
	}
	if avatarURL != nil {
		u.AvatarURL = *avatarURL
	}
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// UnblockUser removes a block. No error when no block existed.
func (s *MemoryStore) UnblockUser(blockerID, blockedID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[blockerID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[blockedID]; !ok {
		return ErrNotFound
	}
	delete(s.blocks[blockerID], blockedID)
	return nil
}

// ListBlocked returns the user IDs this user blocked.
func (s *MemoryStore) ListBlocked(userID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []string{}
	for id := range s.blocks[userID] {
		out = append(out, id)
	}
	return out
}

// CreateReport stores a user report (status starts "open").
func (s *MemoryStore) CreateReport(r *models.Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[r.ReporterID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[r.TargetID]; !ok {
		return ErrNotFound
	}
	if r.ID == "" {
		r.ID = NewID()
	}
	r.Status = "open"
	r.CreatedAt = time.Now().UTC()
	s.reports = append(s.reports, r)
	return nil
}
