package store

import (
	"sort"

	"github.com/vyro/backend/internal/models"
)

// Invite links for groups/channels (Telegram-style), in-memory store.

func (s *MemoryStore) CreateInvite(inv *models.ChatInvite) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[inv.ChatID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[inv.CreatedBy]; !ok {
		return ErrNotFound
	}
	if _, exists := s.invites[inv.Token]; exists {
		return ErrAlreadyExists
	}
	cp := *inv
	s.invites[inv.Token] = &cp
	return nil
}

func (s *MemoryStore) InviteByToken(token string) (*models.ChatInvite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inv, ok := s.invites[token]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *inv
	return &cp, nil
}

func (s *MemoryStore) ListInvites(chatID string) []*models.ChatInvite {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*models.ChatInvite
	for _, inv := range s.invites {
		if inv.ChatID == chatID {
			cp := *inv
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *MemoryStore) RevokeInvite(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.invites[token]
	if !ok {
		return ErrNotFound
	}
	inv.Revoked = true
	return nil
}
