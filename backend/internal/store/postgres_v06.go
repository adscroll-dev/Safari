// Package store — v0.6 additions for PostgresStore: group E2EE
// sender-key distribution (ciphertext only), multi-device registry,
// per-device one-time pre-key claims, and Web Push subscriptions.
// MemoryStore implementations of the same StoreIface methods live in
// v06.go. Schema: migrations/005_v06.sql.
package store

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/push"
)

// --- Group E2EE sender-key distribution ---

func (s *PostgresStore) PutGroupSenderKey(d *models.GroupSenderKey) error {
	if !isUUID(d.ChatID) || !isUUID(d.SenderUserID) || !isUUID(d.ForUserID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.pool.QueryRow(ctx,
		`INSERT INTO group_sender_keys (chat_id, sender_user_id, sender_device_id, key_id, for_user_id, for_device_id, ciphertext)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (chat_id, sender_user_id, sender_device_id, key_id, for_user_id, for_device_id)
		 DO UPDATE SET ciphertext = EXCLUDED.ciphertext, created_at = now()
		 RETURNING created_at`,
		d.ChatID, d.SenderUserID, d.SenderDeviceID, d.KeyID, d.ForUserID, d.ForDeviceID, d.Ciphertext).Scan(&d.CreatedAt)
	return mapStoreErr(err)
}

func (s *PostgresStore) GroupSenderKeysFor(chatID, forUserID, forDeviceID string) []*models.GroupSenderKey {
	if !isUUID(chatID) || !isUUID(forUserID) {
		return []*models.GroupSenderKey{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT chat_id::text, sender_user_id::text, sender_device_id, key_id,
		        for_user_id::text, for_device_id, ciphertext, created_at
		 FROM group_sender_keys
		 WHERE chat_id = $1 AND for_user_id = $2 AND for_device_id = $3
		 ORDER BY created_at ASC`, chatID, forUserID, forDeviceID)
	if err != nil {
		return []*models.GroupSenderKey{}
	}
	defer rows.Close()
	out := []*models.GroupSenderKey{}
	for rows.Next() {
		var d models.GroupSenderKey
		if err := rows.Scan(&d.ChatID, &d.SenderUserID, &d.SenderDeviceID, &d.KeyID,
			&d.ForUserID, &d.ForDeviceID, &d.Ciphertext, &d.CreatedAt); err == nil {
			out = append(out, &d)
		}
	}
	return out
}

func (s *PostgresStore) DeleteGroupSenderKeysForUser(chatID, userID string) error {
	if !isUUID(chatID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`DELETE FROM group_sender_keys WHERE chat_id = $1 AND for_user_id = $2`,
		chatID, userID)
	return mapStoreErr(err)
}

// --- Multi-device registry ---

func (s *PostgresStore) RegisterDevice(d *models.Device) error {
	if !isUUID(d.UserID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var createdAt, lastSeen time.Time
	err := s.pool.QueryRow(ctx,
		`INSERT INTO user_devices (user_id, device_id, name, platform)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, device_id) DO UPDATE
		   SET name = EXCLUDED.name, platform = EXCLUDED.platform, last_seen_at = now()
		 RETURNING created_at, last_seen_at`,
		d.UserID, d.DeviceID, d.Name, d.Platform).Scan(&createdAt, &lastSeen)
	if err != nil {
		return mapStoreErr(err)
	}
	d.CreatedAt, d.LastSeenAt = createdAt, lastSeen
	return nil
}

func (s *PostgresStore) ListDevices(userID string) []*models.Device {
	if !isUUID(userID) {
		return []*models.Device{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT user_id::text, device_id, name, platform, last_seen_at, created_at
		 FROM user_devices WHERE user_id = $1 ORDER BY created_at ASC`, userID)
	if err != nil {
		return []*models.Device{}
	}
	defer rows.Close()
	out := []*models.Device{}
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.UserID, &d.DeviceID, &d.Name, &d.Platform, &d.LastSeenAt, &d.CreatedAt); err == nil {
			out = append(out, &d)
		}
	}
	return out
}

func (s *PostgresStore) RemoveDevice(userID, deviceID string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM user_devices WHERE user_id = $1 AND device_id = $2`, userID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) TouchDevice(userID, deviceID string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE user_devices SET last_seen_at = now() WHERE user_id = $1 AND device_id = $2`,
		userID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FetchOneTimePreKeyForDevice atomically claims one unused pre-key of
// one device of the user (see MemoryStore: the bundle's one-time
// pre-key must belong to the same device as its identity key).
func (s *PostgresStore) FetchOneTimePreKeyForDevice(userID, deviceID string) (*models.OneTimePreKey, error) {
	if !isUUID(userID) {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var k models.OneTimePreKey
	var pk []byte
	err = tx.QueryRow(ctx,
		`SELECT user_id::text, device_id, key_id, public_key, used, created_at
		 FROM one_time_prekeys WHERE user_id = $1 AND device_id = $2 AND NOT used
		 ORDER BY created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED`, userID, deviceID).
		Scan(&k.UserID, &k.DeviceID, &k.KeyID, &pk, &k.Used, &k.CreatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE one_time_prekeys SET used = TRUE
		 WHERE user_id = $1 AND device_id = $2 AND key_id = $3`,
		k.UserID, k.DeviceID, k.KeyID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	k.PublicKey = string(pk)
	k.Used = true
	return &k, nil
}

// --- Web Push subscriptions ---

func (s *PostgresStore) SavePushSubscription(userID string, sub push.Subscription) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, endpoint) DO UPDATE
		   SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`,
		userID, sub.Endpoint, sub.P256dh, sub.Auth)
	return mapStoreErr(err)
}

func (s *PostgresStore) ListPushSubscriptions(userID string) []push.Subscription {
	if !isUUID(userID) {
		return []push.Subscription{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT endpoint, p256dh, auth FROM push_subscriptions
		 WHERE user_id = $1 ORDER BY created_at ASC`, userID)
	if err != nil {
		return []push.Subscription{}
	}
	defer rows.Close()
	out := []push.Subscription{}
	for rows.Next() {
		var sub push.Subscription
		if err := rows.Scan(&sub.Endpoint, &sub.P256dh, &sub.Auth); err == nil {
			out = append(out, sub)
		}
	}
	return out
}

func (s *PostgresStore) DeletePushSubscription(userID, endpoint string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`,
		userID, endpoint)
	return mapStoreErr(err)
}
