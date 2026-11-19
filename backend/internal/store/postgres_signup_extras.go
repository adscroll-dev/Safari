package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
)

func (s *PostgresStore) CreateSignupInviteCode(code *models.SignupInviteCode) error {
	if !isUUID(code.InviterID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if code.MaxUses <= 0 {
		code.MaxUses = 1
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = time.Now().UTC()
	}
	if code.ExpiresAt.IsZero() {
		code.ExpiresAt = code.CreatedAt.Add(7 * 24 * time.Hour)
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO signup_invite_codes (code, inviter_user_id, max_uses, uses, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING uses`, code.Code, code.InviterID, code.MaxUses, code.Uses,
		code.ExpiresAt, code.CreatedAt).Scan(&code.Uses)
	return mapStoreErr(err)
}

func (s *PostgresStore) SignupInviteCodeByCode(code string) (*models.SignupInviteCode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var invite models.SignupInviteCode
	err := s.pool.QueryRow(ctx,
		`SELECT code, inviter_user_id::text, max_uses, uses, expires_at, created_at
		 FROM signup_invite_codes WHERE code = $1`, code).
		Scan(&invite.Code, &invite.InviterID, &invite.MaxUses, &invite.Uses,
			&invite.ExpiresAt, &invite.CreatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return &invite, nil
}

func (s *PostgresStore) UseSignupInviteCode(code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var used int
	err := s.pool.QueryRow(ctx,
		`UPDATE signup_invite_codes SET uses = uses + 1
		 WHERE code = $1 AND uses < max_uses AND expires_at > now()
		 RETURNING uses`, code).Scan(&used)
	if err != nil {
		if mapStoreErr(err) == ErrNotFound {
			return ErrInviteCodeUnavailable
		}
		return err
	}
	return nil
}

func (s *PostgresStore) ListSignupInviteCodes(userID string) []*models.SignupInviteCode {
	if !isUUID(userID) {
		return []*models.SignupInviteCode{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT code, inviter_user_id::text, max_uses, uses, expires_at, created_at
		 FROM signup_invite_codes WHERE inviter_user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return []*models.SignupInviteCode{}
	}
	defer rows.Close()
	out := []*models.SignupInviteCode{}
	for rows.Next() {
		var invite models.SignupInviteCode
		if err := rows.Scan(&invite.Code, &invite.InviterID, &invite.MaxUses,
			&invite.Uses, &invite.ExpiresAt, &invite.CreatedAt); err == nil {
			out = append(out, &invite)
		}
	}
	return out
}

func (s *PostgresStore) RevokeDevice(userID, deviceID string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO revoked_devices (user_id, device_id, revoked_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (user_id, device_id) DO UPDATE SET revoked_at = now()`,
		userID, deviceID)
	return mapStoreErr(err)
}

func (s *PostgresStore) IsDeviceRevoked(userID, deviceID string) bool {
	if !isUUID(userID) {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var revoked bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM revoked_devices WHERE user_id = $1 AND device_id = $2)`,
		userID, deviceID).Scan(&revoked)
	return err != nil || revoked
}
