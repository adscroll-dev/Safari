package store

// Last-seen privacy (Telegram/WhatsApp-style), in-memory store.
// "everyone" is the default; "nobody" hides online state and
// last-seen time from everyone except the user themself.

const (
	// LastSeenEveryone is the default: anyone may see online state
	// and last-seen time.
	LastSeenEveryone = "everyone"
	// LastSeenNobody hides online state and last-seen time from
	// everyone except the user themself.
	LastSeenNobody = "nobody"
)

// ValidLastSeenVisibility reports whether v is an accepted value.
func ValidLastSeenVisibility(v string) bool {
	return v == LastSeenEveryone || v == LastSeenNobody
}

func (s *MemoryStore) GetLastSeenVisibility(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.lastSeenVisibility[userID]; ok {
		return v
	}
	return LastSeenEveryone
}

func (s *MemoryStore) SetLastSeenVisibility(userID, visibility string) error {
	if !ValidLastSeenVisibility(visibility) {
		return ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	s.lastSeenVisibility[userID] = visibility
	return nil
}
