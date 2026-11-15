package store

// Stage 5 — Channels, PostgresStore implementation.
// In-memory versions live in stage5.go.
// Schema: migrations/015_stage5_channels.sql.

import (
	"context"
	"strings"
	"time"

	"github.com/vyro/backend/internal/models"
)

// RecordMessageView records that a user viewed a message.
// Idempotent: viewing again changes nothing.
func (s *PostgresStore) RecordMessageView(messageID, userID string) error {
	if !isUUID(messageID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO message_views (message_id, user_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`, messageID, userID)
	return mapStoreErr(err)
}

// MessageViewCount returns the number of distinct viewers.
func (s *PostgresStore) MessageViewCount(messageID string) int {
	if !isUUID(messageID) {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM message_views WHERE message_id = $1`,
		messageID).Scan(&n); err != nil {
		return 0
	}
	return n
}

// MessageViewCounts returns per-message viewer counts for a chat.
func (s *PostgresStore) MessageViewCounts(chatID string) map[string]int {
	if !isUUID(chatID) {
		return map[string]int{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT v.message_id::text, COUNT(*) FROM message_views v
		 JOIN messages m ON m.id = v.message_id
		 WHERE m.chat_id = $1 GROUP BY v.message_id`, chatID)
	if err != nil {
		return map[string]int{}
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err == nil {
			out[id] = n
		}
	}
	return out
}

// SearchPublicChannels finds public channels whose title contains
// the query (case-insensitive), most-followed first, max 20.
func (s *PostgresStore) SearchPublicChannels(query string) []*models.Chat {
	q := strings.TrimSpace(query)
	if q == "" {
		return []*models.Chat{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT c.id::text, c.type, c.title, c.owner_id::text, c.is_public,
			c.created_at, c.updated_at, COUNT(m.user_id) AS followers
		 FROM chats c JOIN chat_members m ON m.chat_id = c.id
		 WHERE c.type = 'channel' AND c.is_public AND c.title ILIKE $1
		 GROUP BY c.id ORDER BY followers DESC, c.title ASC LIMIT 20`,
		"%"+q+"%")
	if err != nil {
		return []*models.Chat{}
	}
	defer rows.Close()
	out := []*models.Chat{}
	for rows.Next() {
		var c models.Chat
		var ownerID *string
		var followers int
		if err := rows.Scan(&c.ID, &c.Type, &c.Title, &ownerID,
			&c.IsPublic, &c.CreatedAt, &c.UpdatedAt, &followers); err != nil {
			continue
		}
		if ownerID != nil {
			c.OwnerID = *ownerID
		}
		c.FollowerCount = followers
		c.MemberIDs = s.chatMembers(ctx, c.ID)
		out = append(out, &c)
	}
	return out
}

// UpdateChat changes a chat's title and/or public flag (channels
// only for the flag). A nil pointer leaves that field unchanged.
func (s *PostgresStore) UpdateChat(chatID string, title *string, isPublic *bool) error {
	if !isUUID(chatID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`UPDATE chats SET
			title = COALESCE($2, title),
			is_public = CASE WHEN $3 IS NULL OR type <> 'channel' THEN is_public ELSE $3 END,
			updated_at = now()
		 WHERE id = $1`,
		chatID, title, isPublic)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
