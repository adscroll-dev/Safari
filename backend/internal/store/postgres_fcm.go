// FCM tokens (Firebase Cloud Messaging) for the PostgresStore.
// Same shape as the Web Push subscriptions in postgres_v06.go.
// Schema: migrations/008_fcm.sql.
package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/push"
)

// SaveFCMToken stores a token (idempotent by (user_id, token);
// re-registering updates device id / platform).
func (s *PostgresStore) SaveFCMToken(userID string, tok push.FCMToken) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO fcm_tokens (user_id, token, device_id, platform)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, token) DO UPDATE
		   SET device_id = EXCLUDED.device_id, platform = EXCLUDED.platform`,
		userID, tok.Token, tok.DeviceID, tok.Platform)
	return mapStoreErr(err)
}

// ListFCMTokens returns a user's tokens in registration order.
func (s *PostgresStore) ListFCMTokens(userID string) []push.FCMToken {
	if !isUUID(userID) {
		return []push.FCMToken{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT token, device_id, platform FROM fcm_tokens
		 WHERE user_id = $1 ORDER BY created_at ASC`, userID)
	if err != nil {
		return []push.FCMToken{}
	}
	defer rows.Close()
	out := []push.FCMToken{}
	for rows.Next() {
		var tok push.FCMToken
		if err := rows.Scan(&tok.Token, &tok.DeviceID, &tok.Platform); err == nil {
			out = append(out, tok)
		}
	}
	return out
}

// DeleteFCMToken removes a token by value.
func (s *PostgresStore) DeleteFCMToken(userID, token string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`DELETE FROM fcm_tokens WHERE user_id = $1 AND token = $2`,
		userID, token)
	return mapStoreErr(err)
}
