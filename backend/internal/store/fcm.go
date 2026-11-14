// FCM tokens (Firebase Cloud Messaging) for the MemoryStore.
// Same shape as the Web Push subscriptions in v06.go: one token per
// (user, device), idempotent by token value. Schema:
// migrations/008_fcm.sql.
package store

import (
	"github.com/vyro/backend/internal/push"
)

// SaveFCMToken stores a token (idempotent by token value; re-registering
// the same token updates its device id / platform).
func (s *MemoryStore) SaveFCMToken(userID string, tok push.FCMToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	for i, existing := range s.fcmTokens[userID] {
		if existing.Token == tok.Token {
			s.fcmTokens[userID][i] = tok
			return nil
		}
	}
	s.fcmTokens[userID] = append(s.fcmTokens[userID], tok)
	return nil
}

// ListFCMTokens returns a copy of a user's tokens.
func (s *MemoryStore) ListFCMTokens(userID string) []push.FCMToken {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]push.FCMToken, len(s.fcmTokens[userID]))
	copy(out, s.fcmTokens[userID])
	return out
}

// DeleteFCMToken removes a token by value.
func (s *MemoryStore) DeleteFCMToken(userID, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.fcmTokens[userID][:0]
	for _, tok := range s.fcmTokens[userID] {
		if tok.Token != token {
			out = append(out, tok)
		}
	}
	s.fcmTokens[userID] = out
	return nil
}
