// Package store — Stage 11 (Settings parity, WhatsApp-style),
// PostgresStore implementation. MemoryStore implementations of the
// same StoreIface methods live in stage11.go.
// Schema: migrations/017_stage11_settings.sql.
package store

import (
	"context"
	"time"
)

func (s *PostgresStore) GetReadReceipts(userID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var v bool
	err := s.pool.QueryRow(ctx,
		`SELECT read_receipts FROM user_settings WHERE user_id = $1`, userID).
		Scan(&v)
	if err != nil {
		return true // WhatsApp default: receipts on.
	}
	return v
}

func (s *PostgresStore) SetReadReceipts(userID string, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, read_receipts)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET read_receipts = EXCLUDED.read_receipts`,
		userID, enabled)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetAvatarPrivacy(userID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT avatar_privacy FROM user_settings WHERE user_id = $1`, userID).
		Scan(&v)
	if err != nil {
		return LastSeenEveryone
	}
	return v
}

func (s *PostgresStore) SetAvatarPrivacy(userID, privacy string) error {
	if !ValidPrivacyValue(privacy) {
		return ErrForbidden
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, avatar_privacy)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET avatar_privacy = EXCLUDED.avatar_privacy`,
		userID, privacy)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetAboutPrivacy(userID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT about_privacy FROM user_settings WHERE user_id = $1`, userID).
		Scan(&v)
	if err != nil {
		return LastSeenEveryone
	}
	return v
}

func (s *PostgresStore) SetAboutPrivacy(userID, privacy string) error {
	if !ValidPrivacyValue(privacy) {
		return ErrForbidden
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, about_privacy)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET about_privacy = EXCLUDED.about_privacy`,
		userID, privacy)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetGroupsPrivacy(userID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT groups_privacy FROM user_settings WHERE user_id = $1`, userID).
		Scan(&v)
	if err != nil {
		return LastSeenEveryone
	}
	return v
}

func (s *PostgresStore) SetGroupsPrivacy(userID, privacy string) error {
	if !ValidPrivacyValue(privacy) {
		return ErrForbidden
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, groups_privacy)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET groups_privacy = EXCLUDED.groups_privacy`,
		userID, privacy)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteUser removes the account. Every REFERENCES users(...) FK in
// the schema is ON DELETE CASCADE, so the user's messages, chats
// memberships, blocks, stories, polls, call records, devices, FCM
// tokens and settings go with it; chats they owned keep existing
// with owner_id SET NULL.
func (s *PostgresStore) DeleteUser(userID string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
