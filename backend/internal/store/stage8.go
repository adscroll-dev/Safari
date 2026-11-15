package store

import (
	"strings"
	"time"
)

// Stage 8 (extras): change the account email — in-memory
// implementation. The caller checks the address is free first.
func (s *MemoryStore) ChangeEmail(userID, newEmail string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok {
		return ErrNotFound
	}
	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	if other, taken := s.byEmail[newEmail]; taken && other != userID {
		return ErrAlreadyExists
	}
	delete(s.byEmail, strings.ToLower(strings.TrimSpace(u.Email)))
	u.Email = newEmail
	s.byEmail[newEmail] = userID
	u.UpdatedAt = time.Now().UTC()
	return nil
}
