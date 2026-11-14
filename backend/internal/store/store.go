// Package store provides the data layer.
//
// The current implementation is Store, a thread-safe in-memory store.
// It is the store used by the dev build and by tests, and it is the
// reference implementation of the StoreIface interface.
//
// PostgreSQL: the schema lives in migrations/001_init.sql and
// PostgresStore (postgres.go) implements StoreIface on top of it. When
// DATABASE_URL is set the server uses PostgresStore; otherwise it uses
// the in-memory store. See docs/ARCHITECTURE.md.
package store

import (
	"crypto/rand"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/push"
)

func readRandom(b []byte) (int, error) { return rand.Read(b) }

// NOTE: uuid is avoided as a dependency — IDs below use a local generator.
// (Kept import-free on purpose; see NewID.)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrForbidden     = errors.New("forbidden")
)

// StoreIface is the interface handlers depend on.
type StoreIface interface {
	CreateUser(u *models.User) error
	UserByEmail(email string) (*models.User, error)
	UserByID(id string) (*models.User, error)
	UserByUsername(username string) (*models.User, error)
	ListUsers() []*models.User

	CreateChat(c *models.Chat) error
	ChatByID(id string) (*models.Chat, error)
	ChatsForUser(userID string) []*models.Chat
	IsMember(chatID, userID string) bool

	CreateMessage(m *models.Message) error
	MessageByID(id string) (*models.Message, error)
	MessagesForChat(chatID string, limit, offset int) []*models.Message
	SearchMessages(userID, query string, limit int) []*models.Message
	UpdateMessage(m *models.Message) error
	DeleteMessage(id string) error
	AddReaction(messageID, userID, emoji string) error
	RemoveReaction(messageID, userID, emoji string) error

	BlockUser(blockerID, blockedID string) error
	IsBlocked(a, b string) bool

	// v0.4: admin roles, receipts, presence, stories, scheduled, bots, keys, settings.
	MemberRole(chatID, userID string) string
	ChatMembers(chatID string) []*models.ChatMember
	AddMember(chatID, userID, role string) error
	RemoveMember(chatID, userID string) error
	SetMemberRole(chatID, userID, role string) error
	MarkDelivered(messageID, userID string) error
	MarkRead(messageID, userID string) error
	SetPresence(userID string, online bool, lastSeen time.Time) error
	GetPresence(userID string) (bool, time.Time)
	CreateStory(st *models.Story) error
	ListStories() []*models.Story
	ListStoriesByUser(userID string) []*models.Story
	CreateScheduled(sm *models.ScheduledMessage) error
	DueScheduled(now time.Time) []*models.ScheduledMessage
	MarkScheduledSent(id, messageID string) error
	ListScheduled(chatID, userID string) []*models.ScheduledMessage
	CancelScheduled(id, userID string) error
	PurgeExpired(now time.Time) int
	CreateBot(b *models.Bot) error
	BotByToken(token string) (*models.Bot, error)
	BotByUsername(username string) (*models.Bot, error)
	BotByID(id string) (*models.Bot, error)
	ListBots(ownerID string) []*models.Bot
	DeleteBot(id, ownerID string) error
	SetBotWebhook(id, ownerID, url string) error
	UploadIdentityKey(k *models.IdentityKey) error
	GetIdentityKey(userID, deviceID string) (*models.IdentityKey, error)
	UploadOneTimePreKeys(userID, deviceID string, keys []*models.OneTimePreKey) error
	FetchOneTimePreKey(userID string) (*models.OneTimePreKey, error)
	GetChatSettings(chatID, userID string) models.ChatSettings
	SetChatSettings(cs models.ChatSettings) error

	// v0.6: group E2EE sender-key distribution (ciphertext only),
	// multi-device registry, per-device pre-key claim, push subscriptions.
	PutGroupSenderKey(d *models.GroupSenderKey) error
	GroupSenderKeysFor(chatID, forUserID, forDeviceID string) []*models.GroupSenderKey
	DeleteGroupSenderKeysForUser(chatID, userID string) error
	RegisterDevice(d *models.Device) error
	ListDevices(userID string) []*models.Device
	RemoveDevice(userID, deviceID string) error
	TouchDevice(userID, deviceID string) error
	FetchOneTimePreKeyForDevice(userID, deviceID string) (*models.OneTimePreKey, error)
	SavePushSubscription(userID string, sub push.Subscription) error
	ListPushSubscriptions(userID string) []push.Subscription
	DeletePushSubscription(userID, endpoint string) error

	// FCM tokens (mobile push, Firebase Cloud Messaging): one token
	// per (user, device). Same shape as the Web Push subscriptions
	// above, but the token is an FCM registration token, not a
	// browser endpoint.
	SaveFCMToken(userID string, tok push.FCMToken) error
	ListFCMTokens(userID string) []push.FCMToken
	DeleteFCMToken(userID, token string) error

	// v0.7: email verification with 6-digit codes. Only the SHA-256
	// hash of a code is stored, never the code itself. A code row is
	// keyed by (email, purpose); saving a new code replaces the old
	// one, which is how resending invalidates the previous code.
	SaveEmailCode(c *models.EmailCode) error
	GetEmailCode(email, purpose string) (*models.EmailCode, error)
	IncrementCodeAttempts(email, purpose string) (int, error)
	ConsumeEmailCode(email, purpose string) error
	InvalidateEmailCode(email, purpose string) error
	SavePendingUser(p *models.PendingUser) error
	GetPendingUser(email string) (*models.PendingUser, error)
	DeletePendingUser(email string) error

	// Invite links for groups/channels (Telegram-style). A token is
	// unique; revoking keeps the row so old links fail closed.
	CreateInvite(inv *models.ChatInvite) error
	InviteByToken(token string) (*models.ChatInvite, error)
	ListInvites(chatID string) []*models.ChatInvite
	RevokeInvite(token string) error

	// Last-seen privacy (Telegram/WhatsApp-style): "everyone"
	// (default) or "nobody". When "nobody", other users see neither
	// the online state nor the last-seen time.
	GetLastSeenVisibility(userID string) string
	SetLastSeenVisibility(userID, visibility string) error
}

// MemoryStore is a thread-safe in-memory StoreIface.
type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]*models.User // id -> user
	byEmail  map[string]string       // lower(email) -> id
	byName   map[string]string       // lower(username) -> id
	chats    map[string]*models.Chat
	messages map[string]*models.Message
	chatMsgs map[string][]string // chat id -> message ids in order
	blocks   map[string]map[string]bool
	// v0.4 state
	memberRoles  map[string]map[string]string   // chat id -> user id -> role
	receipts     map[string]map[string]*receipt // message id -> user id -> receipt
	presence     map[string]presenceInfo
	stories      map[string]*models.Story
	scheduled    map[string]*models.ScheduledMessage
	bots         map[string]*models.Bot
	botsByToken  map[string]string
	identityKeys map[string]*models.IdentityKey     // userID + "/" + deviceID
	prekeys      map[string][]*models.OneTimePreKey // userID -> keys
	chatSettings map[string]models.ChatSettings     // chatID + "/" + userID
	// v0.6 state
	groupSenderKeys map[string]*models.GroupSenderKey // distribution key, see v06.go
	devices         map[string]*models.Device         // userID + "/" + deviceID
	pushSubs        map[string][]push.Subscription    // userID -> subscriptions
	fcmTokens       map[string][]push.FCMToken        // userID -> FCM tokens
	// v0.7 state: email verification codes and pending registrations.
	emailCodes   map[string]*models.EmailCode   // lower(email) + "|" + purpose
	pendingUsers map[string]*models.PendingUser // lower(email) -> pending registration
	// Invite links for groups/channels (Telegram-style).
	invites map[string]*models.ChatInvite // token -> invite
	// Last-seen privacy: user id -> "everyone" | "nobody" (default
	// "everyone" when absent).
	lastSeenVisibility map[string]string
}

type receipt struct {
	delivered bool
	read      bool
}

type presenceInfo struct {
	online   bool
	lastSeen time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users: map[string]*models.User{}, byEmail: map[string]string{},
		byName: map[string]string{}, chats: map[string]*models.Chat{},
		messages: map[string]*models.Message{}, chatMsgs: map[string][]string{},
		blocks:             map[string]map[string]bool{},
		memberRoles:        map[string]map[string]string{},
		receipts:           map[string]map[string]*receipt{},
		presence:           map[string]presenceInfo{},
		stories:            map[string]*models.Story{},
		scheduled:          map[string]*models.ScheduledMessage{},
		bots:               map[string]*models.Bot{},
		botsByToken:        map[string]string{},
		identityKeys:       map[string]*models.IdentityKey{},
		prekeys:            map[string][]*models.OneTimePreKey{},
		chatSettings:       map[string]models.ChatSettings{},
		groupSenderKeys:    map[string]*models.GroupSenderKey{},
		devices:            map[string]*models.Device{},
		pushSubs:           map[string][]push.Subscription{},
		fcmTokens:          map[string][]push.FCMToken{},
		emailCodes:         map[string]*models.EmailCode{},
		pendingUsers:       map[string]*models.PendingUser{},
		invites:            map[string]*models.ChatInvite{},
		lastSeenVisibility: map[string]string{},
	}
}

// NewID returns a unique id without an external dependency.
func NewID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + uuidString()
}

// uuidString is a tiny random hex suffix (not a full UUID implementation,
// just uniqueness for dev ids). Production ids come from PostgreSQL gen_random_uuid().
func uuidString() string {
	b := make([]byte, 8)
	// crypto/rand read; ignore error — fall back to time-based bytes.
	_, _ = readRandom(b)
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 16)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}

func (s *MemoryStore) CreateUser(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	email := strings.ToLower(strings.TrimSpace(u.Email))
	name := strings.ToLower(strings.TrimSpace(u.Username))
	if _, ok := s.byEmail[email]; ok {
		return ErrAlreadyExists
	}
	if _, ok := s.byName[name]; ok {
		return ErrAlreadyExists
	}
	if u.ID == "" {
		u.ID = NewID()
	}
	now := time.Now().UTC()
	u.CreatedAt, u.UpdatedAt = now, now
	s.users[u.ID] = u
	s.byEmail[email] = u.ID
	s.byName[name] = u.ID
	return nil
}

func (s *MemoryStore) UserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, ErrNotFound
	}
	return s.users[id], nil
}

func (s *MemoryStore) UserByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) UserByUsername(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byName[strings.ToLower(strings.TrimSpace(username))]
	if !ok {
		return nil, ErrNotFound
	}
	return s.users[id], nil
}

func (s *MemoryStore) ListUsers() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *MemoryStore) CreateChat(c *models.Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = NewID()
	}
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	// Deduplicate members.
	seen := map[string]bool{}
	members := c.MemberIDs[:0]
	for _, id := range c.MemberIDs {
		if id != "" && !seen[id] {
			seen[id] = true
			members = append(members, id)
		}
	}
	c.MemberIDs = members
	s.chats[c.ID] = c
	if s.memberRoles[c.ID] == nil {
		s.memberRoles[c.ID] = map[string]string{}
	}
	for _, id := range members {
		role := "member"
		if id == c.OwnerID {
			role = "owner"
		}
		s.memberRoles[c.ID][id] = role
	}
	return nil
}

func (s *MemoryStore) ChatByID(id string) (*models.Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.chats[id]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *MemoryStore) ChatsForUser(userID string) []*models.Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Chat{}
	for _, c := range s.chats {
		for _, m := range c.MemberIDs {
			if m == userID {
				out = append(out, c)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

func (s *MemoryStore) IsMember(chatID, userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.chats[chatID]
	if !ok {
		return false
	}
	for _, m := range c.MemberIDs {
		if m == userID {
			return true
		}
	}
	return false
}

func (s *MemoryStore) CreateMessage(m *models.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[m.ChatID]; !ok {
		return ErrNotFound
	}
	if m.ID == "" {
		m.ID = NewID()
	}
	now := time.Now().UTC()
	m.CreatedAt, m.UpdatedAt = now, now
	if m.Reactions == nil {
		m.Reactions = map[string][]string{}
	}
	s.messages[m.ID] = m
	s.chatMsgs[m.ChatID] = append(s.chatMsgs[m.ChatID], m.ID)
	if c, ok := s.chats[m.ChatID]; ok {
		c.UpdatedAt = now
	}
	return nil
}

func (s *MemoryStore) MessageByID(id string) (*models.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.messages[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s.withReceipts(m), nil
}

// withReceipts returns a copy of m with DeliveredTo/ReadBy/Status filled
// from the receipt table. Callers must hold at least a read lock.
func (s *MemoryStore) withReceipts(m *models.Message) *models.Message {
	cp := *m
	recs := s.receipts[m.ID]
	delivered, read := []string{}, []string{}
	for uid, r := range recs {
		if r.delivered {
			delivered = append(delivered, uid)
		}
		if r.read {
			read = append(read, uid)
		}
	}
	sort.Strings(delivered)
	sort.Strings(read)
	cp.DeliveredTo, cp.ReadBy = delivered, read
	others := []string{}
	if c, ok := s.chats[m.ChatID]; ok {
		for _, id := range c.MemberIDs {
			if id != m.SenderID {
				others = append(others, id)
			}
		}
	}
	cp.Status = cp.StatusFor(others)
	return &cp
}

func (s *MemoryStore) MessagesForChat(chatID string, limit, offset int) []*models.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.chatMsgs[chatID]
	if offset < 0 {
		offset = 0
	}
	if offset >= len(ids) {
		return []*models.Message{}
	}
	end := len(ids)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	out := make([]*models.Message, 0, end-offset)
	now := time.Now().UTC()
	for _, id := range ids[offset:end] {
		if m, ok := s.messages[id]; ok && m.DeletedAt == nil &&
			(m.ExpiresAt == nil || m.ExpiresAt.After(now)) {
			out = append(out, s.withReceipts(m))
		}
	}
	return out
}

func (s *MemoryStore) SearchMessages(userID, query string, limit int) []*models.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	out := []*models.Message{}
	if q == "" {
		return out
	}
	now := time.Now().UTC()
	for _, m := range s.messages {
		if m.DeletedAt != nil || m.Kind == "e2ee" || (m.ExpiresAt != nil && !m.ExpiresAt.After(now)) ||
			!strings.Contains(strings.ToLower(m.Content), q) {
			continue
		}
		// Only messages in chats the user belongs to.
		c, ok := s.chats[m.ChatID]
		if !ok {
			continue
		}
		for _, member := range c.MemberIDs {
			if member == userID {
				out = append(out, s.withReceipts(m))
				break
			}
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *MemoryStore) UpdateMessage(m *models.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.messages[m.ID]
	if !ok {
		return ErrNotFound
	}
	existing.Content = m.Content
	existing.UpdatedAt = time.Now().UTC()
	now := time.Now().UTC()
	existing.EditedAt = &now
	return nil
}

func (s *MemoryStore) DeleteMessage(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.messages[id]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	m.DeletedAt = &now
	m.Content = ""
	return nil
}

func (s *MemoryStore) AddReaction(messageID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.messages[messageID]
	if !ok {
		return ErrNotFound
	}
	if m.Reactions == nil {
		m.Reactions = map[string][]string{}
	}
	for _, u := range m.Reactions[emoji] {
		if u == userID {
			return nil // already reacted
		}
	}
	m.Reactions[emoji] = append(m.Reactions[emoji], userID)
	return nil
}

func (s *MemoryStore) RemoveReaction(messageID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.messages[messageID]
	if !ok {
		return ErrNotFound
	}
	users := m.Reactions[emoji]
	kept := users[:0]
	for _, u := range users {
		if u != userID {
			kept = append(kept, u)
		}
	}
	if len(kept) == 0 {
		delete(m.Reactions, emoji)
	} else {
		m.Reactions[emoji] = kept
	}
	return nil
}

func (s *MemoryStore) BlockUser(blockerID, blockedID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[blockerID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[blockedID]; !ok {
		return ErrNotFound
	}
	if s.blocks[blockerID] == nil {
		s.blocks[blockerID] = map[string]bool{}
	}
	s.blocks[blockerID][blockedID] = true
	return nil
}

func (s *MemoryStore) IsBlocked(a, b string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blocks[a][b] || s.blocks[b][a]
}
