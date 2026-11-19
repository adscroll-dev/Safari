// Package store — v0.7 additions for PostgresStore: email
// verification codes (hash only) and pending registrations.
// MemoryStore implementations of the same StoreIface methods live in
// verify.go. Schema: migrations/006_verification.sql.
package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
)

// --- Email verification codes ---

func (s *PostgresStore) SaveEmailCode(c *models.EmailCode) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO email_codes (email, purpose, code_hash, attempts, expires_at, consumed_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (email, purpose) DO UPDATE SET
		   code_hash = EXCLUDED.code_hash, attempts = 0,
		   expires_at = EXCLUDED.expires_at, consumed_at = NULL,
		   created_at = EXCLUDED.created_at`,
		c.Email, c.Purpose, c.CodeHash, c.Attempts, c.ExpiresAt, c.ConsumedAt, c.CreatedAt)
	return mapStoreErr(err)
}

func (s *PostgresStore) GetEmailCode(email, purpose string) (*models.EmailCode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var c models.EmailCode
	err := s.pool.QueryRow(ctx,
		`SELECT email, purpose, code_hash, attempts, expires_at, consumed_at, created_at
		 FROM email_codes WHERE email = $1 AND purpose = $2`, email, purpose).
		Scan(&c.Email, &c.Purpose, &c.CodeHash, &c.Attempts, &c.ExpiresAt, &c.ConsumedAt, &c.CreatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return &c, nil
}

func (s *PostgresStore) IncrementCodeAttempts(email, purpose string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var attempts int
	err := s.pool.QueryRow(ctx,
		`UPDATE email_codes SET attempts = attempts + 1
		 WHERE email = $1 AND purpose = $2 RETURNING attempts`, email, purpose).Scan(&attempts)
	if err != nil {
		return 0, mapStoreErr(err)
	}
	return attempts, nil
}

func (s *PostgresStore) ConsumeEmailCode(email, purpose string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE email_codes SET consumed_at = now() WHERE email = $1 AND purpose = $2`,
		email, purpose)
	if err != nil {
		return mapStoreErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) InvalidateEmailCode(email, purpose string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`DELETE FROM email_codes WHERE email = $1 AND purpose = $2`, email, purpose)
	return mapStoreErr(err)
}

// --- Pending registrations ---

func (s *PostgresStore) SavePendingUser(p *models.PendingUser) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO pending_users
		   (email, username, display_name, password_hash, invite_code, device_id, device_name, platform, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (email) DO UPDATE SET
		   username = EXCLUDED.username, display_name = EXCLUDED.display_name,
		   password_hash = EXCLUDED.password_hash,
		   invite_code = EXCLUDED.invite_code, device_id = EXCLUDED.device_id,
		   device_name = EXCLUDED.device_name, platform = EXCLUDED.platform,
		   expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at`,
		p.Email, p.Username, p.DisplayName, p.PasswordHash, p.InviteCode,
		p.DeviceID, p.DeviceName, p.Platform, p.ExpiresAt, p.CreatedAt)
	return mapStoreErr(err)
}

func (s *PostgresStore) GetPendingUser(email string) (*models.PendingUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var p models.PendingUser
	err := s.pool.QueryRow(ctx,
		`SELECT email, username, display_name, password_hash, invite_code,
		        device_id, device_name, platform, expires_at, created_at
		 FROM pending_users WHERE email = $1`, email).
		Scan(&p.Email, &p.Username, &p.DisplayName, &p.PasswordHash, &p.InviteCode,
			&p.DeviceID, &p.DeviceName, &p.Platform, &p.ExpiresAt, &p.CreatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return &p, nil
}

func (s *PostgresStore) DeletePendingUser(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM pending_users WHERE email = $1`, email)
	return mapStoreErr(err)
}
