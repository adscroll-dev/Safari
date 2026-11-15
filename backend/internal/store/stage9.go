package store

import (
	"sort"

	"github.com/vyro/backend/internal/models"
)

// Stage 9 (extras): storage report — in-memory implementation.
// Only chats the user belongs to; deleted messages are excluded.
func (s *MemoryStore) StorageStats(userID string) ([]models.ChatStorage, int, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := map[string]*models.ChatStorage{}
	for _, chat := range s.chats {
		if !s.isMemberLocked(chat.ID, userID) {
			continue
		}
		stats[chat.ID] = &models.ChatStorage{
			ChatID: chat.ID, Title: chat.Title, Type: string(chat.Type),
		}
	}
	totalMessages := 0
	var totalBytes int64
	for _, m := range s.messages {
		if m.DeletedAt != nil {
			continue
		}
		st, ok := stats[m.ChatID]
		if !ok {
			continue
		}
		st.MessageCount++
		st.Bytes += m.FileSize
		totalMessages++
		totalBytes += m.FileSize
	}
	out := make([]models.ChatStorage, 0, len(stats))
	for _, st := range stats {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	return out, totalMessages, totalBytes
}
