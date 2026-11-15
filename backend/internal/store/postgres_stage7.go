package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
)

// Stage 7 (accounts): profile editing, unblocking, blocked list,
// user reports — PostgreSQL implementations.

// UpdateProfile applies the non-nil fields (COALESCE keeps the old
// value when the argument is NULL). Unknown user -> ErrNotFound.
func (s *PostgresStore) UpdateProfile(userID string, displayName, bio, avatarURL *string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET display_name = COALESCE($2, display_name),
		 bio = COALESCE($3, bio), avatar_url = COALESCE($4, avatar_url),
		 updated_at = now() WHERE id = $1`,
		userID, displayName, bio, avatarURL)
	if err != nil {
		return mapStoreErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UnblockUser deletes a block row. Unknown user -> ErrNotFound;
// no error when no block existed.
func (s *PostgresStore) UnblockUser(blockerID, blockedID string) error {
	if !isUUID(blockerID) || !isUUID(blockedID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.UserByID(blockerID); err != nil {
		return err
	}
	if _, err := s.UserByID(blockedID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedID)
	return mapStoreErr(err)
}

// ListBlocked returns the user IDs this user blocked.
func (s *PostgresStore) ListBlocked(userID string) []string {
	if !isUUID(userID) {
		return []string{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT blocked_id::text FROM blocks WHERE blocker_id = $1`,
		userID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// CreateReport inserts a user report. The id and created_at come
// from the database defaults; status starts "open".
func (s *PostgresStore) CreateReport(r *models.Report) error {
	if !isUUID(r.ReporterID) || !isUUID(r.TargetID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.UserByID(r.ReporterID); err != nil {
		return err
	}
	if _, err := s.UserByID(r.TargetID); err != nil {
		return err
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO reports (reporter_id, target_type, target_id, reason)
		 VALUES ($1, $2, $3, $4) RETURNING id::text, created_at`,
		r.ReporterID, r.TargetType, r.TargetID, r.Reason).
		Scan(&r.ID, &r.CreatedAt)
	if err != nil {
		return mapStoreErr(err)
	}
	r.Status = "open"
	return nil
}
