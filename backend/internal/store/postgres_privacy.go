// Package store — last-seen privacy (Telegram/WhatsApp-style),
// PostgresStore implementation. MemoryStore implementations of the
// same StoreIface methods live in privacy.go.
// Schema: migrations/010_privacy.sql.
package store

import (
	"context"
	"time"
)

func (s *PostgresStore) GetLastSeenVisibility(userID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT last_seen_visibility FROM user_settings WHERE user_id = $1`, userID).
		Scan(&v)
	if err != nil {
		return LastSeenEveryone
	}
	return v
}

func (s *PostgresStore) SetLastSeenVisibility(userID, visibility string) error {
	if !ValidLastSeenVisibility(visibility) {
		return ErrForbidden
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, last_seen_visibility)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET last_seen_visibility = EXCLUDED.last_seen_visibility`,
		userID, visibility)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
