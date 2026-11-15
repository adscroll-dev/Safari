// Package store — Stage 1 chat gaps (WhatsApp parity): mute chat,
// pin message, delete-for-me. PostgresStore implementation.
// MemoryStore implementations of the same StoreIface methods live
// in store.go. Schema: migrations/011_stage1.sql.
package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
)

// LastMessageForChatForUser is LastMessageForChat, but it also skips
// messages the user deleted for themself.
func (s *PostgresStore) LastMessageForChatForUser(chatID, userID string) *models.Message {
	if !isUUID(chatID) || !isUUID(userID) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m, err := scanMessage(s.pool.QueryRow(ctx,
		`SELECT `+messageColumns+` FROM messages m
		 WHERE m.chat_id = $1 AND m.deleted_at IS NULL
		   AND (m.expires_at IS NULL OR m.expires_at > now())
		   AND NOT EXISTS (
		     SELECT 1 FROM message_deletions d
		     WHERE d.message_id = m.id AND d.user_id = $2)
		 ORDER BY m.created_at DESC, m.id DESC LIMIT 1`, chatID, userID))
	if err != nil {
		return nil
	}
	s.loadReceipts(ctx, []*models.Message{m})
	return m
}

// SetPinnedMessage pins one message in a chat; an empty messageID
// clears the pin. The message must exist and belong to the chat.
func (s *PostgresStore) SetPinnedMessage(chatID, messageID string) error {
	if !isUUID(chatID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if messageID == "" {
		_, err := s.pool.Exec(ctx,
			`UPDATE chats SET pinned_message_id = NULL WHERE id = $1`, chatID)
		return mapStoreErr(err)
	}
	if !isUUID(messageID) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE chats SET pinned_message_id = $2 WHERE id = $1
		   AND EXISTS (SELECT 1 FROM messages
		               WHERE id = $2 AND chat_id = $1 AND deleted_at IS NULL)`,
		chatID, messageID)
	if err != nil {
		return mapStoreErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) PinnedMessageForChat(chatID string) *models.Message {
	if !isUUID(chatID) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m, err := scanMessage(s.pool.QueryRow(ctx,
		`SELECT `+messageColumns+` FROM messages m
		 WHERE m.id = (SELECT pinned_message_id FROM chats WHERE id = $1)
		   AND m.deleted_at IS NULL`, chatID))
	if err != nil {
		return nil
	}
	s.loadReceipts(ctx, []*models.Message{m})
	return m
}

// DeleteMessageForUser hides a message only for one user
// (delete-for-me). The message stays for everyone else.
func (s *PostgresStore) DeleteMessageForUser(messageID, userID string) error {
	if !isUUID(messageID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Check the message exists first: the foreign key would reject
	// an insert for a missing message with a constraint error, and
	// the caller wants ErrNotFound instead.
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM messages WHERE id = $1)`,
		messageID).Scan(&exists); err != nil {
		return mapStoreErr(err)
	}
	if !exists {
		return ErrNotFound
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO message_deletions (message_id, user_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		messageID, userID)
	return mapStoreErr(err)
}

func (s *PostgresStore) IsMessageDeletedForUser(messageID, userID string) bool {
	if !isUUID(messageID) || !isUUID(userID) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM message_deletions
		 WHERE message_id = $1 AND user_id = $2)`,
		messageID, userID).Scan(&exists); err != nil {
		return false
	}
	return exists
}
