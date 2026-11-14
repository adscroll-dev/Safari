// Package store — invite links for groups/channels (Telegram-style),
// PostgresStore implementation. MemoryStore implementations of the
// same StoreIface methods live in invites.go.
// Schema: migrations/009_invites.sql.
package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
)

func (s *PostgresStore) CreateInvite(inv *models.ChatInvite) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO chat_invites (token, chat_id, created_by, created_at, revoked)
		 VALUES ($1, $2, $3, $4, $5)`,
		inv.Token, inv.ChatID, inv.CreatedBy, inv.CreatedAt, inv.Revoked)
	return mapStoreErr(err)
}

func (s *PostgresStore) InviteByToken(token string) (*models.ChatInvite, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var inv models.ChatInvite
	err := s.pool.QueryRow(ctx,
		`SELECT token, chat_id, created_by, created_at, revoked
		 FROM chat_invites WHERE token = $1`, token).
		Scan(&inv.Token, &inv.ChatID, &inv.CreatedBy, &inv.CreatedAt, &inv.Revoked)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return &inv, nil
}

func (s *PostgresStore) ListInvites(chatID string) []*models.ChatInvite {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT token, chat_id, created_by, created_at, revoked
		 FROM chat_invites WHERE chat_id = $1 ORDER BY created_at ASC`, chatID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*models.ChatInvite
	for rows.Next() {
		var inv models.ChatInvite
		if err := rows.Scan(&inv.Token, &inv.ChatID, &inv.CreatedBy, &inv.CreatedAt, &inv.Revoked); err != nil {
			continue
		}
		cp := inv
		out = append(out, &cp)
	}
	return out
}

func (s *PostgresStore) RevokeInvite(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`UPDATE chat_invites SET revoked = TRUE WHERE token = $1`, token)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
