// PostgresStore — v0.4 additions: admin roles, read receipts, presence,
// stories, scheduled messages, bots, E2EE key foundation and per-user
// chat settings. All methods are part of StoreIface (see store.go).
package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vyro/backend/internal/models"
)

func botTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// loadReceipts fills DeliveredTo/ReadBy/Status for the given messages.
func (s *PostgresStore) loadReceipts(ctx context.Context, msgs []*models.Message) {
	if len(msgs) == 0 {
		return
	}
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT message_id::text, user_id::text, delivered_at, read_at
		 FROM message_receipts WHERE message_id::text = ANY($1)`, ids)
	if err != nil {
		return
	}
	defer rows.Close()
	byID := map[string]*models.Message{}
	for _, m := range msgs {
		byID[m.ID] = m
		m.DeliveredTo, m.ReadBy = []string{}, []string{}
	}
	for rows.Next() {
		var msgID, userID string
		var deliveredAt, readAt *time.Time
		if err := rows.Scan(&msgID, &userID, &deliveredAt, &readAt); err != nil {
			continue
		}
		if m, ok := byID[msgID]; ok {
			if deliveredAt != nil {
				m.DeliveredTo = append(m.DeliveredTo, userID)
			}
			if readAt != nil {
				m.ReadBy = append(m.ReadBy, userID)
			}
		}
	}
	// Status needs the other members of each chat.
	for _, m := range msgs {
		others := []string{}
		for _, id := range s.chatMembers(ctx, m.ChatID) {
			if id != m.SenderID {
				others = append(others, id)
			}
		}
		m.Status = m.StatusFor(others)
	}
}

func (s *PostgresStore) withReceipts(ctx context.Context, m *models.Message) *models.Message {
	s.loadReceipts(ctx, []*models.Message{m})
	return m
}

// --- Admin roles ---

func (s *PostgresStore) MemberRole(chatID, userID string) string {
	if !isUUID(chatID) || !isUUID(userID) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var role string
	if err := s.pool.QueryRow(ctx,
		`SELECT role FROM chat_members WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID).Scan(&role); err != nil {
		return ""
	}
	return role
}

func (s *PostgresStore) ChatMembers(chatID string) []*models.ChatMember {
	if !isUUID(chatID) {
		return []*models.ChatMember{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT user_id::text, role FROM chat_members WHERE chat_id = $1 ORDER BY joined_at ASC`, chatID)
	if err != nil {
		return []*models.ChatMember{}
	}
	defer rows.Close()
	out := []*models.ChatMember{}
	for rows.Next() {
		var cm models.ChatMember
		cm.ChatID = chatID
		if err := rows.Scan(&cm.UserID, &cm.Role); err == nil {
			out = append(out, &cm)
		}
	}
	return out
}

func (s *PostgresStore) AddMember(chatID, userID, role string) error {
	if !isUUID(chatID) || !isUUID(userID) {
		return ErrNotFound
	}
	if role != "member" && role != "admin" {
		role = "member"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO chat_members (chat_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`, chatID, userID, role)
	if err != nil {
		return mapStoreErr(err)
	}
	if tag.RowsAffected() == 0 {
		// Either already a member or an unknown user/chat (FK would error).
		if !s.IsMember(chatID, userID) {
			return ErrNotFound
		}
		return ErrAlreadyExists
	}
	return nil
}

func (s *PostgresStore) RemoveMember(chatID, userID string) error {
	if !isUUID(chatID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM chat_members WHERE chat_id = $1 AND user_id = $2`, chatID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetMemberRole(chatID, userID, role string) error {
	if !isUUID(chatID) || !isUUID(userID) {
		return ErrNotFound
	}
	if role != "member" && role != "admin" && role != "owner" {
		return errors.New("invalid role")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE chat_members SET role = $3 WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Receipts ---

func (s *PostgresStore) MarkDelivered(messageID, userID string) error {
	if !isUUID(messageID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO message_receipts (message_id, user_id, delivered_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (message_id, user_id) DO UPDATE SET delivered_at = COALESCE(message_receipts.delivered_at, now())`,
		messageID, userID)
	return mapStoreErr(err)
}

func (s *PostgresStore) MarkRead(messageID, userID string) error {
	if !isUUID(messageID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// WhatsApp semantics: reading a message marks every earlier
	// message from others in the same chat as read too.
	_, err := s.pool.Exec(ctx,
		`INSERT INTO message_receipts (message_id, user_id, delivered_at, read_at)
		 SELECT m.id, $2, now(), now() FROM messages m
		 WHERE m.chat_id = (SELECT chat_id FROM messages WHERE id = $1)
		   AND m.created_at <= (SELECT created_at FROM messages WHERE id = $1)
		   AND m.sender_id != $2 AND m.deleted_at IS NULL
		 ON CONFLICT (message_id, user_id) DO UPDATE
		   SET delivered_at = COALESCE(message_receipts.delivered_at, now()), read_at = now()`,
		messageID, userID)
	return mapStoreErr(err)
}

// --- Presence ---

func (s *PostgresStore) SetPresence(userID string, online bool, lastSeen time.Time) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET is_online = $2, last_seen_at = $3 WHERE id = $1`,
		userID, online, lastSeen)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetPresence(userID string) (bool, time.Time) {
	if !isUUID(userID) {
		return false, time.Time{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var online bool
	var lastSeen *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT is_online, last_seen_at FROM users WHERE id = $1`, userID).Scan(&online, &lastSeen); err != nil {
		return false, time.Time{}
	}
	if lastSeen != nil {
		return online, *lastSeen
	}
	return online, time.Time{}
}

// --- Stories ---

func scanStory(row pgx.Row) (*models.Story, error) {
	var st models.Story
	err := row.Scan(&st.ID, &st.UserID, &st.MediaURL, &st.Caption, &st.ExpiresAt, &st.CreatedAt)
	return &st, err
}

const storyColumns = `id::text, user_id::text, media_url, caption, expires_at, created_at`

func (s *PostgresStore) CreateStory(st *models.Story) error {
	if !isUUID(st.UserID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO stories (user_id, media_url, caption, expires_at)
		 VALUES ($1, $2, $3, $4) RETURNING id::text, created_at`,
		st.UserID, st.MediaURL, st.Caption, st.ExpiresAt).Scan(&st.ID, &st.CreatedAt)
	return mapStoreErr(err)
}

func (s *PostgresStore) ListStories() []*models.Story {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+storyColumns+` FROM stories WHERE expires_at > now() ORDER BY created_at DESC`)
	if err != nil {
		return []*models.Story{}
	}
	defer rows.Close()
	out := []*models.Story{}
	for rows.Next() {
		if st, err := scanStory(rows); err == nil {
			out = append(out, st)
		}
	}
	return out
}

func (s *PostgresStore) ListStoriesByUser(userID string) []*models.Story {
	if !isUUID(userID) {
		return []*models.Story{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+storyColumns+` FROM stories
		 WHERE user_id = $1 AND expires_at > now() ORDER BY created_at DESC`, userID)
	if err != nil {
		return []*models.Story{}
	}
	defer rows.Close()
	out := []*models.Story{}
	for rows.Next() {
		if st, err := scanStory(rows); err == nil {
			out = append(out, st)
		}
	}
	return out
}

// --- Scheduled messages ---

func scanScheduled(row pgx.Row) (*models.ScheduledMessage, error) {
	var sm models.ScheduledMessage
	var replyTo, sentMsg *string
	err := row.Scan(&sm.ID, &sm.ChatID, &sm.SenderID, &sm.Content, &sm.Kind,
		&sm.FileURL, &sm.FileName, &sm.FileSize, &sm.MimeType, &replyTo,
		&sm.SendAt, &sm.Sent, &sentMsg, &sm.CreatedAt)
	if err != nil {
		return nil, err
	}
	if replyTo != nil {
		sm.ReplyToID = *replyTo
	}
	if sentMsg != nil {
		sm.SentMessageID = *sentMsg
	}
	return &sm, nil
}

const scheduledColumns = `id::text, chat_id::text, sender_id::text, content, kind,
	file_url, file_name, file_size, mime_type, reply_to_id::text,
	send_at, sent, sent_message_id::text, created_at`

func (s *PostgresStore) CreateScheduled(sm *models.ScheduledMessage) error {
	if !isUUID(sm.ChatID) || !isUUID(sm.SenderID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var replyTo any
	if isUUID(sm.ReplyToID) {
		replyTo = sm.ReplyToID
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO scheduled_messages (chat_id, sender_id, content, kind, file_url, file_name, file_size, mime_type, reply_to_id, send_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id::text, created_at`,
		sm.ChatID, sm.SenderID, sm.Content, sm.Kind, sm.FileURL, sm.FileName,
		sm.FileSize, sm.MimeType, replyTo, sm.SendAt).Scan(&sm.ID, &sm.CreatedAt)
	return mapStoreErr(err)
}

func (s *PostgresStore) DueScheduled(now time.Time) []*models.ScheduledMessage {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+scheduledColumns+` FROM scheduled_messages
		 WHERE NOT sent AND send_at <= $1 ORDER BY send_at ASC`, now)
	if err != nil {
		return []*models.ScheduledMessage{}
	}
	defer rows.Close()
	out := []*models.ScheduledMessage{}
	for rows.Next() {
		if sm, err := scanScheduled(rows); err == nil {
			out = append(out, sm)
		}
	}
	return out
}

func (s *PostgresStore) MarkScheduledSent(id, messageID string) error {
	if !isUUID(id) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sentMsg any
	if isUUID(messageID) {
		sentMsg = messageID
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE scheduled_messages SET sent = TRUE, sent_message_id = $2 WHERE id = $1`,
		id, sentMsg)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListScheduled(chatID, userID string) []*models.ScheduledMessage {
	if !isUUID(chatID) || !isUUID(userID) {
		return []*models.ScheduledMessage{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+scheduledColumns+` FROM scheduled_messages
		 WHERE chat_id = $1 AND sender_id = $2 AND NOT sent ORDER BY send_at ASC`,
		chatID, userID)
	if err != nil {
		return []*models.ScheduledMessage{}
	}
	defer rows.Close()
	out := []*models.ScheduledMessage{}
	for rows.Next() {
		if sm, err := scanScheduled(rows); err == nil {
			out = append(out, sm)
		}
	}
	return out
}

func (s *PostgresStore) CancelScheduled(id, userID string) error {
	if !isUUID(id) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM scheduled_messages WHERE id = $1 AND sender_id = $2 AND NOT sent`,
		id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) PurgeExpired(now time.Time) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE messages SET deleted_at = $1, content = '', updated_at = now()
		 WHERE deleted_at IS NULL AND expires_at IS NOT NULL AND expires_at <= $1`, now)
	if err != nil {
		return 0
	}
	return int(tag.RowsAffected())
}

// --- Bots ---

func (s *PostgresStore) CreateBot(b *models.Bot) error {
	if !isUUID(b.OwnerID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.pool.QueryRow(ctx,
		`INSERT INTO bots (owner_id, name, username, token_hash, webhook_url)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id::text, created_at`,
		b.OwnerID, b.Name, strings.TrimSpace(b.Username), botTokenHash(b.Token), b.WebhookURL).
		Scan(&b.ID, &b.CreatedAt)
	if err != nil {
		return mapStoreErr(err)
	}
	b.Token = "" // never keep the raw token on the stored struct
	return nil
}

func scanBot(row pgx.Row) (*models.Bot, error) {
	var b models.Bot
	err := row.Scan(&b.ID, &b.OwnerID, &b.Name, &b.Username, &b.WebhookURL, &b.CreatedAt)
	return &b, err
}

const botColumns = `id::text, owner_id::text, name, username, webhook_url, created_at`

func (s *PostgresStore) BotByToken(token string) (*models.Bot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, err := scanBot(s.pool.QueryRow(ctx,
		`SELECT `+botColumns+` FROM bots WHERE token_hash = $1`, botTokenHash(token)))
	return b, mapStoreErr(err)
}

func (s *PostgresStore) BotByUsername(username string) (*models.Bot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, err := scanBot(s.pool.QueryRow(ctx,
		`SELECT `+botColumns+` FROM bots WHERE username = $1`, strings.TrimSpace(username)))
	return b, mapStoreErr(err)
}

func (s *PostgresStore) BotByID(id string) (*models.Bot, error) {
	if !isUUID(id) {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, err := scanBot(s.pool.QueryRow(ctx,
		`SELECT `+botColumns+` FROM bots WHERE id = $1`, id))
	return b, mapStoreErr(err)
}

func (s *PostgresStore) ListBots(ownerID string) []*models.Bot {
	if !isUUID(ownerID) {
		return []*models.Bot{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+botColumns+` FROM bots WHERE owner_id = $1 ORDER BY created_at ASC`, ownerID)
	if err != nil {
		return []*models.Bot{}
	}
	defer rows.Close()
	out := []*models.Bot{}
	for rows.Next() {
		if b, err := scanBot(rows); err == nil {
			out = append(out, b)
		}
	}
	return out
}

func (s *PostgresStore) DeleteBot(id, ownerID string) error {
	if !isUUID(id) || !isUUID(ownerID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM bots WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetBotWebhook(id, ownerID, url string) error {
	if !isUUID(id) || !isUUID(ownerID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE bots SET webhook_url = $3 WHERE id = $1 AND owner_id = $2`, id, ownerID, url)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- E2EE key foundation (public keys only) ---

func (s *PostgresStore) UploadIdentityKey(k *models.IdentityKey) error {
	if !isUUID(k.UserID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO identity_keys (user_id, device_id, identity_key, signed_prekey, signed_prekey_id, sign_key, signed_prekey_sig)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, device_id) DO UPDATE
		   SET identity_key = EXCLUDED.identity_key,
		       signed_prekey = EXCLUDED.signed_prekey,
		       signed_prekey_id = EXCLUDED.signed_prekey_id,
		       sign_key = EXCLUDED.sign_key,
		       signed_prekey_sig = EXCLUDED.signed_prekey_sig`,
		k.UserID, k.DeviceID, []byte(k.IdentityKey), []byte(k.SignedPreKey), k.SignedPreKeyID, []byte(k.SignKey), []byte(k.SignedPreKeySig))
	return mapStoreErr(err)
}

func (s *PostgresStore) GetIdentityKey(userID, deviceID string) (*models.IdentityKey, error) {
	if !isUUID(userID) {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var k models.IdentityKey
	var ik, spk, signKey, spkSig []byte
	err := s.pool.QueryRow(ctx,
		`SELECT user_id::text, device_id, identity_key, signed_prekey, signed_prekey_id,
		        COALESCE(sign_key, ''::bytea), COALESCE(signed_prekey_sig, ''::bytea), created_at
		 FROM identity_keys WHERE user_id = $1 AND device_id = $2`,
		userID, deviceID).Scan(&k.UserID, &k.DeviceID, &ik, &spk, &k.SignedPreKeyID, &signKey, &spkSig, &k.CreatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	k.IdentityKey, k.SignedPreKey = string(ik), string(spk)
	k.SignKey, k.SignedPreKeySig = string(signKey), string(spkSig)
	return &k, nil
}

func (s *PostgresStore) UploadOneTimePreKeys(userID, deviceID string, keys []*models.OneTimePreKey) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Re-publishing a key_id replaces it (see MemoryStore): a device
	// that uploads key_id N again has rotated that slot, and the old
	// public key's private half may be gone — serving it would produce
	// sessions the device cannot decrypt.
	for _, k := range keys {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO one_time_prekeys (user_id, device_id, key_id, public_key)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (user_id, device_id, key_id) DO UPDATE
			 SET public_key = EXCLUDED.public_key, used = FALSE, created_at = now()`,
			userID, deviceID, k.KeyID, []byte(k.PublicKey)); err != nil {
			return mapStoreErr(err)
		}
	}
	return nil
}

// FetchOneTimePreKey atomically claims one unused pre-key for the user.
func (s *PostgresStore) FetchOneTimePreKey(userID string) (*models.OneTimePreKey, error) {
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
		 FROM one_time_prekeys WHERE user_id = $1 AND NOT used
		 ORDER BY created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED`, userID).
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

// --- Per-user chat settings ---

func (s *PostgresStore) GetChatSettings(chatID, userID string) models.ChatSettings {
	cs := models.ChatSettings{ChatID: chatID, UserID: userID}
	if !isUUID(chatID) || !isUUID(userID) {
		return cs
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.pool.QueryRow(ctx,
		`SELECT pinned, archived, folder FROM chat_settings WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID).Scan(&cs.Pinned, &cs.Archived, &cs.Folder)
	return cs
}

func (s *PostgresStore) SetChatSettings(cs models.ChatSettings) error {
	if !isUUID(cs.ChatID) || !isUUID(cs.UserID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO chat_settings (chat_id, user_id, pinned, archived, folder)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (chat_id, user_id) DO UPDATE
		   SET pinned = EXCLUDED.pinned, archived = EXCLUDED.archived, folder = EXCLUDED.folder`,
		cs.ChatID, cs.UserID, cs.Pinned, cs.Archived, cs.Folder)
	return mapStoreErr(err)
}
