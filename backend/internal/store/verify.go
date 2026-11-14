// Package store — v0.7 additions for MemoryStore: email verification
// codes (hash only, never the plain code) and pending registrations.
// PostgreSQL implementations of the same StoreIface methods live in
// postgres_verify.go. Schema: migrations/006_verification.sql.
package store

import (
	"strings"
	"time"

	"github.com/vyro/backend/internal/models"
)

func emailCodeKey(email, purpose string) string {
	return strings.ToLower(strings.TrimSpace(email)) + "|" + purpose
}

// --- Email verification codes ---

// SaveEmailCode stores (or replaces) the code row for (email,
// purpose). Saving a new code is what invalidates the previous one:
// a resend overwrites the old hash, so the old code no longer matches.
func (s *MemoryStore) SaveEmailCode(c *models.EmailCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	cp := *c
	s.emailCodes[emailCodeKey(c.Email, c.Purpose)] = &cp
	return nil
}

func (s *MemoryStore) GetEmailCode(email, purpose string) (*models.EmailCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.emailCodes[emailCodeKey(email, purpose)]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *c
	return &cp, nil
}

// IncrementCodeAttempts bumps the failed-attempt counter and returns
// the new count, so the caller can invalidate the code at the limit.
func (s *MemoryStore) IncrementCodeAttempts(email, purpose string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.emailCodes[emailCodeKey(email, purpose)]
	if !ok {
		return 0, ErrNotFound
	}
	c.Attempts++
	return c.Attempts, nil
}

// ConsumeEmailCode marks the code used. A consumed code never verifies
// again (single-use), but the row stays until replaced or expired so
// the resend cooldown can still be checked against CreatedAt.
func (s *MemoryStore) ConsumeEmailCode(email, purpose string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.emailCodes[emailCodeKey(email, purpose)]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	c.ConsumedAt = &now
	return nil
}

// InvalidateEmailCode deletes the code row outright.
func (s *MemoryStore) InvalidateEmailCode(email, purpose string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.emailCodes, emailCodeKey(email, purpose))
	return nil
}

// --- Pending registrations ---

func (s *MemoryStore) SavePendingUser(p *models.PendingUser) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.Email = strings.ToLower(strings.TrimSpace(p.Email))
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	cp := *p
	s.pendingUsers[p.Email] = &cp
	return nil
}

func (s *MemoryStore) GetPendingUser(email string) (*models.PendingUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pendingUsers[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (s *MemoryStore) DeletePendingUser(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pendingUsers, strings.ToLower(strings.TrimSpace(email)))
	return nil
}
