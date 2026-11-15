package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
)

// Stage 9 (extras): storage report — PostgreSQL implementation.
// One aggregate query over the user's chats; deleted messages
// (deleted_at NOT NULL) are excluded.
func (s *PostgresStore) StorageStats(userID string) ([]models.ChatStorage, int, int64) {
	if !isUUID(userID) {
		return []models.ChatStorage{}, 0, 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text, COALESCE(c.title, ''), c.type,
		       COUNT(m.id), COALESCE(SUM(m.file_size), 0)
		FROM chats c
		JOIN chat_members cm ON cm.chat_id = c.id AND cm.user_id = $1
		LEFT JOIN messages m ON m.chat_id = c.id AND m.deleted_at IS NULL
		GROUP BY c.id, c.title, c.type
		ORDER BY COALESCE(SUM(m.file_size), 0) DESC`, userID)
	if err != nil {
		return []models.ChatStorage{}, 0, 0
	}
	defer rows.Close()
	out := []models.ChatStorage{}
	totalMessages := 0
	var totalBytes int64
	for rows.Next() {
		var st models.ChatStorage
		if err := rows.Scan(&st.ChatID, &st.Title, &st.Type, &st.MessageCount, &st.Bytes); err != nil {
			continue
		}
		totalMessages += st.MessageCount
		totalBytes += st.Bytes
		out = append(out, st)
	}
	return out, totalMessages, totalBytes
}
