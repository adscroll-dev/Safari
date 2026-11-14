// PostgresStore implements StoreIface on top of PostgreSQL 16.
//
// It is the v0.2 persistence milestone: with DATABASE_URL set, the server
// uses this store instead of MemoryStore, so users, chats and messages
// survive a restart. Without DATABASE_URL the server keeps using the
// in-memory store for local dev and tests.
//
// Notes on the mapping (schema: backend/migrations/001_init.sql):
//   - IDs are PostgreSQL UUIDs. Handlers never pre-assign IDs, so inserts
//     let the database generate them (gen_random_uuid) and the generated
//     id is written back onto the struct. An ID that is already a valid
//     UUID is honoured; a non-UUID id (e.g. a MemoryStore dev id) is
//     ignored and a UUID is generated instead.
//   - Email / username uniqueness is case-insensitive via CITEXT; a
//     unique-violation (SQLSTATE 23505) maps to ErrAlreadyExists, and a
//     foreign-key violation (23503) to ErrNotFound.
//   - Messages are soft-deleted (deleted_at set, content cleared), exactly
//     like MemoryStore, and hidden from MessagesForChat / search.
//   - Reactions live in the reactions table and are loaded together with
//     their messages, grouped by emoji -> user ids.
//   - Search uses ILIKE (case-insensitive substring), restricted to chats
//     the user belongs to. The pg_trgm index in the migration makes this
//     fast; search still works if that index is absent.
package store

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vyro/backend/internal/models"
)

//go:embed migrations/001_init.sql
var migrationSQL string

//go:embed migrations/002_uploads.sql
var migration002SQL string

//go:embed migrations/003_v04.sql
var migration003SQL string

//go:embed migrations/004_v05.sql
var migration004SQL string

//go:embed migrations/005_v06.sql
var migration005SQL string

//go:embed migrations/006_verification.sql
var migration006SQL string

//go:embed migrations/007_forward.sql
var migration007SQL string

//go:embed migrations/008_fcm.sql
var migration008SQL string

//go:embed migrations/009_invites.sql
var migration009SQL string

// The embedded migration is a copy of backend/migrations/001_init.sql
// (go:embed cannot reach outside the package directory).
// TestMigrationMatchesFile fails if the two ever drift apart.

// PostgresStore is a StoreIface backed by a pgx connection pool.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore connects to PostgreSQL, applies the (idempotent)
// v0.1 migration, and returns a store ready to use. It fails loudly on
// purpose: a server started with DATABASE_URL must not silently fall back
// to the in-memory store and lose data on restart.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is empty")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	// Supabase's pooler runs in transaction mode (port 6543), which
	// does not support prepared statements. pgx uses them by default
	// (extended protocol), so every parameterized query failed and
	// registration answered 500 while /health still said "postgres".
	// The simple protocol sends each query in full and works with any
	// pooler mode. (Startup ping and migrations use parameter-less
	// statements, which is why the server started fine and the failure
	// only showed up on the first real query.)
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 001_init.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration002SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 002_uploads.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration003SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 003_v04.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration004SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 004_v05.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration005SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 005_v06.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration006SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 006_verification.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration007SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 007_forward.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration008SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 008_fcm.sql: %w", err)
	}
	if _, err := pool.Exec(ctx, migration009SQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migration 009_invites.sql: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

// Close releases the connection pool.
func (s *PostgresStore) Close() { s.pool.Close() }

// Ping reports whether the database is reachable.
func (s *PostgresStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(id string) bool { return uuidRe.MatchString(id) }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// mapStoreErr converts PostgreSQL errors to the store's sentinel errors.
func mapStoreErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return ErrNotFound
	case isUniqueViolation(err):
		return ErrAlreadyExists
	case isFKViolation(err):
		return ErrNotFound
	default:
		return err
	}
}

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	var lastSeen *time.Time
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.Bio,
		&u.AvatarURL, &u.PasswordHash, &u.IsOnline, &lastSeen, &u.CreatedAt, &u.UpdatedAt, &u.IsBot)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if lastSeen != nil {
		u.LastSeenAt = *lastSeen
	}
	return &u, nil
}

const userColumns = `id::text, email, username, display_name, bio, avatar_url,
	password_hash, is_online, last_seen_at, created_at, updated_at, is_bot`

func (s *PostgresStore) CreateUser(u *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id string
	var createdAt, updatedAt time.Time
	if isUUID(u.ID) {
		err := s.pool.QueryRow(ctx,
			`INSERT INTO users (id, email, username, display_name, bio, avatar_url, password_hash)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id::text, created_at, updated_at`,
			u.ID, strings.TrimSpace(u.Email), strings.TrimSpace(u.Username),
			u.DisplayName, u.Bio, u.AvatarURL, u.PasswordHash).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			return mapStoreErr(err)
		}
	} else {
		err := s.pool.QueryRow(ctx,
			`INSERT INTO users (email, username, display_name, bio, avatar_url, password_hash)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, created_at, updated_at`,
			strings.TrimSpace(u.Email), strings.TrimSpace(u.Username),
			u.DisplayName, u.Bio, u.AvatarURL, u.PasswordHash).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			return mapStoreErr(err)
		}
	}
	u.ID, u.CreatedAt, u.UpdatedAt = id, createdAt, updatedAt
	return nil
}

func (s *PostgresStore) UserByEmail(email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, strings.TrimSpace(email)))
}

func (s *PostgresStore) UserByID(id string) (*models.User, error) {
	if !isUUID(id) {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (s *PostgresStore) UserByUsername(username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE username = $1`, strings.TrimSpace(username)))
}

func (s *PostgresStore) ListUsers() []*models.User {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY created_at ASC`)
	if err != nil {
		return []*models.User{}
	}
	defer rows.Close()
	out := []*models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err == nil {
			out = append(out, u)
		}
	}
	return out
}

// --- Chats ---

func scanChat(row pgx.Row, memberIDs []string) (*models.Chat, error) {
	var c models.Chat
	var ownerID *string
	err := row.Scan(&c.ID, &c.Type, &c.Title, &ownerID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if ownerID != nil {
		c.OwnerID = *ownerID
	}
	c.MemberIDs = memberIDs
	return &c, nil
}

func (s *PostgresStore) chatMembers(ctx context.Context, chatID string) []string {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id::text FROM chat_members WHERE chat_id = $1 ORDER BY joined_at ASC, user_id ASC`, chatID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *PostgresStore) CreateChat(c *models.Chat) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Deduplicate members, preserving order (same rule as MemoryStore).
	seen := map[string]bool{}
	members := []string{}
	for _, id := range c.MemberIDs {
		if id != "" && isUUID(id) && !seen[id] {
			seen[id] = true
			members = append(members, id)
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	var createdAt, updatedAt time.Time
	owner := pgtype.UUID{Valid: false}
	if isUUID(c.OwnerID) {
		_ = owner.Scan(c.OwnerID)
	}
	if isUUID(c.ID) {
		err = tx.QueryRow(ctx,
			`INSERT INTO chats (id, type, title, owner_id) VALUES ($1, $2, $3, $4)
			 RETURNING id::text, created_at, updated_at`,
			c.ID, string(c.Type), c.Title, owner).Scan(&id, &createdAt, &updatedAt)
	} else {
		err = tx.QueryRow(ctx,
			`INSERT INTO chats (type, title, owner_id) VALUES ($1, $2, $3)
			 RETURNING id::text, created_at, updated_at`,
			string(c.Type), c.Title, owner).Scan(&id, &createdAt, &updatedAt)
	}
	if err != nil {
		return mapStoreErr(err)
	}
	for _, memberID := range members {
		role := "member"
		if memberID == c.OwnerID {
			role = "owner"
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO chat_members (chat_id, user_id, role) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`, id, memberID, role); err != nil {
			return mapStoreErr(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	c.ID, c.CreatedAt, c.UpdatedAt, c.MemberIDs = id, createdAt, updatedAt, members
	return nil
}

func (s *PostgresStore) ChatByID(id string) (*models.Chat, error) {
	if !isUUID(id) {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := scanChat(s.pool.QueryRow(ctx,
		`SELECT id::text, type, title, owner_id::text, created_at, updated_at
		 FROM chats WHERE id = $1`, id), nil)
	if err != nil {
		return nil, err
	}
	c.MemberIDs = s.chatMembers(ctx, id)
	return c, nil
}

func (s *PostgresStore) ChatsForUser(userID string) []*models.Chat {
	if !isUUID(userID) {
		return []*models.Chat{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT c.id::text, c.type, c.title, c.owner_id::text, c.created_at, c.updated_at
		 FROM chats c JOIN chat_members m ON m.chat_id = c.id
		 WHERE m.user_id = $1 ORDER BY c.updated_at DESC`, userID)
	if err != nil {
		return []*models.Chat{}
	}
	defer rows.Close()
	out := []*models.Chat{}
	for rows.Next() {
		c, err := scanChat(rows, nil)
		if err != nil {
			continue
		}
		c.MemberIDs = s.chatMembers(ctx, c.ID)
		out = append(out, c)
	}
	return out
}

func (s *PostgresStore) IsMember(chatID, userID string) bool {
	if !isUUID(chatID) || !isUUID(userID) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = $1 AND user_id = $2)`,
		chatID, userID).Scan(&exists)
	return err == nil && exists
}

// --- Messages ---

func scanMessage(row pgx.Row) (*models.Message, error) {
	var m models.Message
	var replyTo *string
	var lat, lng *float64
	var locLabel *string
	var fwdFromID *string
	err := row.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Kind, &m.Content,
		&m.FileURL, &m.FileName, &m.FileSize, &m.MimeType,
		&replyTo, &m.EditedAt, &m.DeletedAt, &m.CreatedAt, &m.UpdatedAt,
		&lat, &lng, &locLabel, &m.ExpiresAt, &m.ForwardedFrom, &fwdFromID)
	m.Lat, m.Lng = lat, lng
	if locLabel != nil {
		m.LocationLabel = *locLabel
	}
	if fwdFromID != nil {
		m.ForwardedFromID = *fwdFromID
	}
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if replyTo != nil {
		m.ReplyToID = *replyTo
	}
	m.Reactions = map[string][]string{}
	return &m, nil
}

const messageColumns = `id::text, chat_id::text, sender_id::text, kind, content,
	file_url, file_name, file_size, mime_type,
	reply_to_id::text, edited_at, deleted_at, created_at, updated_at,
	lat, lng, location_label, expires_at, forwarded_from,
	forwarded_from_id::text`

// loadReactions fills Reactions for the given messages in one query.
func (s *PostgresStore) loadReactions(ctx context.Context, msgs []*models.Message) {
	if len(msgs) == 0 {
		return
	}
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT message_id::text, emoji, user_id::text FROM reactions
		 WHERE message_id::text = ANY($1) ORDER BY created_at ASC`, ids)
	if err != nil {
		return
	}
	defer rows.Close()
	byID := map[string]*models.Message{}
	for _, m := range msgs {
		byID[m.ID] = m
	}
	for rows.Next() {
		var msgID, emoji, userID string
		if err := rows.Scan(&msgID, &emoji, &userID); err != nil {
			continue
		}
		if m, ok := byID[msgID]; ok {
			if m.Reactions == nil {
				m.Reactions = map[string][]string{}
			}
			m.Reactions[emoji] = append(m.Reactions[emoji], userID)
		}
	}
}

func (s *PostgresStore) CreateMessage(m *models.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !isUUID(m.ChatID) || !isUUID(m.SenderID) {
		return ErrNotFound
	}
	kind := m.Kind
	if kind == "" {
		kind = "text"
	}
	var replyTo any
	if isUUID(m.ReplyToID) {
		replyTo = m.ReplyToID
	}
	var fwdFromID any
	if isUUID(m.ForwardedFromID) {
		fwdFromID = m.ForwardedFromID
	}
	var id string
	var createdAt, updatedAt time.Time
	if isUUID(m.ID) {
		err := s.pool.QueryRow(ctx,
			`INSERT INTO messages (id, chat_id, sender_id, kind, content, file_url, file_name, file_size, mime_type, reply_to_id, lat, lng, location_label, expires_at, forwarded_from, forwarded_from_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			 RETURNING id::text, created_at, updated_at`,
			m.ID, m.ChatID, m.SenderID, kind, m.Content, m.FileURL, m.FileName, m.FileSize, m.MimeType, replyTo, m.Lat, m.Lng, m.LocationLabel, m.ExpiresAt, m.ForwardedFrom, fwdFromID).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			return mapStoreErr(err)
		}
	} else {
		err := s.pool.QueryRow(ctx,
			`INSERT INTO messages (chat_id, sender_id, kind, content, file_url, file_name, file_size, mime_type, reply_to_id, lat, lng, location_label, expires_at, forwarded_from, forwarded_from_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			 RETURNING id::text, created_at, updated_at`,
			m.ChatID, m.SenderID, kind, m.Content, m.FileURL, m.FileName, m.FileSize, m.MimeType, replyTo, m.Lat, m.Lng, m.LocationLabel, m.ExpiresAt, m.ForwardedFrom, fwdFromID).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			return mapStoreErr(err)
		}
	}
	// Keep the chat list ordered by recent activity, like MemoryStore.
	_, _ = s.pool.Exec(ctx, `UPDATE chats SET updated_at = now() WHERE id = $1`, m.ChatID)
	m.ID, m.Kind, m.CreatedAt, m.UpdatedAt = id, kind, createdAt, updatedAt
	if m.Reactions == nil {
		m.Reactions = map[string][]string{}
	}
	return nil
}

func (s *PostgresStore) MessageByID(id string) (*models.Message, error) {
	if !isUUID(id) {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m, err := scanMessage(s.pool.QueryRow(ctx,
		`SELECT `+messageColumns+` FROM messages WHERE id = $1`, id))
	if err != nil {
		return nil, err
	}
	s.loadReactions(ctx, []*models.Message{m})
	s.loadReceipts(ctx, []*models.Message{m})
	return m, nil
}

func (s *PostgresStore) MessagesForChat(chatID string, limit, offset int) []*models.Message {
	if !isUUID(chatID) {
		return []*models.Message{}
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+messageColumns+` FROM messages
		 WHERE chat_id = $1 AND deleted_at IS NULL
		   AND (expires_at IS NULL OR expires_at > now())
		 ORDER BY created_at ASC, id ASC LIMIT $2 OFFSET $3`, chatID, limit, offset)
	if err != nil {
		return []*models.Message{}
	}
	defer rows.Close()
	out := []*models.Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err == nil {
			out = append(out, m)
		}
	}
	s.loadReactions(ctx, out)
	s.loadReceipts(ctx, out)
	return out
}

func escapeLike(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(q)
}

func (s *PostgresStore) SearchMessages(userID, query string, limit int) []*models.Message {
	if !isUUID(userID) || strings.TrimSpace(query) == "" {
		return []*models.Message{}
	}
	if limit <= 0 {
		limit = 50
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+messageColumns+` FROM messages m
		 WHERE m.deleted_at IS NULL AND (m.expires_at IS NULL OR m.expires_at > now())
		   AND m.content ILIKE $1
		   AND EXISTS (SELECT 1 FROM chat_members cm
		               WHERE cm.chat_id = m.chat_id AND cm.user_id = $2)
		 ORDER BY m.created_at DESC LIMIT $3`,
		"%"+escapeLike(query)+"%", userID, limit)
	if err != nil {
		return []*models.Message{}
	}
	defer rows.Close()
	out := []*models.Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err == nil {
			out = append(out, m)
		}
	}
	s.loadReactions(ctx, out)
	s.loadReceipts(ctx, out)
	return out
}

func (s *PostgresStore) UpdateMessage(m *models.Message) error {
	if !isUUID(m.ID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var editedAt time.Time
	var updatedAt time.Time
	err := s.pool.QueryRow(ctx,
		`UPDATE messages SET content = $2, edited_at = now(), updated_at = now()
		 WHERE id = $1 RETURNING edited_at, updated_at`,
		m.ID, m.Content).Scan(&editedAt, &updatedAt)
	if err != nil {
		return mapStoreErr(err)
	}
	m.EditedAt, m.UpdatedAt = &editedAt, updatedAt
	return nil
}

func (s *PostgresStore) DeleteMessage(id string) error {
	if !isUUID(id) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE messages SET deleted_at = now(), content = '', updated_at = now()
		 WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) AddReaction(messageID, userID, emoji string) error {
	if !isUUID(messageID) || !isUUID(userID) || strings.TrimSpace(emoji) == "" {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO reactions (message_id, user_id, emoji) VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`, messageID, userID, emoji)
	return mapStoreErr(err)
}

func (s *PostgresStore) RemoveReaction(messageID, userID, emoji string) error {
	if !isUUID(messageID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.MessageByID(messageID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`,
		messageID, userID, emoji)
	return err
}

// --- Blocks ---

func (s *PostgresStore) BlockUser(blockerID, blockedID string) error {
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
		`INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`, blockerID, blockedID)
	return mapStoreErr(err)
}

func (s *PostgresStore) IsBlocked(a, b string) bool {
	if !isUUID(a) || !isUUID(b) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM blocks
		 WHERE (blocker_id = $1 AND blocked_id = $2)
		    OR (blocker_id = $2 AND blocked_id = $1))`, a, b).Scan(&exists)
	return err == nil && exists
}
