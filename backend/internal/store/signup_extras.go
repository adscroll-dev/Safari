package store

import (
	"sort"
	"time"

	"github.com/vyro/backend/internal/models"
)

func (s *MemoryStore) CreateSignupInviteCode(code *models.SignupInviteCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[code.InviterID]; !ok {
		return ErrNotFound
	}
	if _, exists := s.signupInviteCodes[code.Code]; exists {
		return ErrAlreadyExists
	}
	now := time.Now().UTC()
	if code.MaxUses <= 0 {
		code.MaxUses = 1
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = now
	}
	if code.ExpiresAt.IsZero() {
		code.ExpiresAt = code.CreatedAt.Add(7 * 24 * time.Hour)
	}
	cp := *code
	s.signupInviteCodes[code.Code] = &cp
	return nil
}

func (s *MemoryStore) SignupInviteCodeByCode(code string) (*models.SignupInviteCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	invite, ok := s.signupInviteCodes[code]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *invite
	return &cp, nil
}

func (s *MemoryStore) UseSignupInviteCode(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	invite, ok := s.signupInviteCodes[code]
	if !ok || invite.Uses >= invite.MaxUses || !time.Now().UTC().Before(invite.ExpiresAt) {
		return ErrInviteCodeUnavailable
	}
	invite.Uses++
	return nil
}

func (s *MemoryStore) ListSignupInviteCodes(userID string) []*models.SignupInviteCode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.SignupInviteCode{}
	for _, code := range s.signupInviteCodes {
		if code.InviterID == userID {
			cp := *code
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}
