package store

import (
	"sort"

	"github.com/vyro/backend/internal/models"
)

// CreateCallRecord stores one finished (or missed) call. The caller
// writes it when the call ends; the record is visible to the caller,
// the callee, and (for group calls, where CalleeID is empty) every
// member of the chat.
func (s *MemoryStore) CreateCallRecord(rec *models.CallRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[rec.ChatID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[rec.CallerID]; !ok {
		return ErrNotFound
	}
	if rec.CalleeID != "" {
		if _, ok := s.users[rec.CalleeID]; !ok {
			return ErrNotFound
		}
	}
	cp := *rec
	s.callRecords = append(s.callRecords, &cp)
	return nil
}

// ListCallRecordsForUser returns the calls the user took part in,
// newest first (at most 100). A record is visible when the user is
// the caller, the callee, or — for group calls — a member of the chat.
func (s *MemoryStore) ListCallRecordsForUser(userID string) []*models.CallRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.CallRecord{}
	for _, rec := range s.callRecords {
		if rec.CallerID != userID && rec.CalleeID != userID {
			if rec.CalleeID != "" {
				continue
			}
			if !s.isMemberLocked(rec.ChatID, userID) {
				continue
			}
		}
		cp := *rec
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	if len(out) > 100 {
		out = out[:100]
	}
	return out
}

// isMemberLocked reports whether userID is a member of chatID.
// Callers must hold at least a read lock.
func (s *MemoryStore) isMemberLocked(chatID, userID string) bool {
	roles, ok := s.memberRoles[chatID]
	if !ok {
		return false
	}
	_, ok = roles[userID]
	return ok
}
