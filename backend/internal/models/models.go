// Package models defines the core VYRO domain types.
// These types are shared by the in-memory store and (later) the PostgreSQL store.
// The JSON shapes are the public API contract for web / mobile clients.
package models

import "time"

// User is a registered account. PasswordHash is never serialised.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Bio          string    `json:"bio,omitempty"`
	PasswordHash string    `json:"-"`
	IsBot        bool      `json:"is_bot,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	IsOnline     bool      `json:"is_online"`
}

// PublicUser is the safe view of a user returned by the API.
type PublicUser struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	IsOnline    bool      `json:"is_online"`
}

func (u *User) Public() PublicUser {
	return PublicUser{
		ID: u.ID, Email: u.Email, Username: u.Username,
		DisplayName: u.DisplayName, AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt, IsOnline: u.IsOnline,
	}
}

// ChatType distinguishes private chats, groups and channels.
type ChatType string

const (
	ChatDM      ChatType = "dm"
	ChatGroup   ChatType = "group"
	ChatChannel ChatType = "channel"
)

type Chat struct {
	ID        string   `json:"id"`
	Type      ChatType `json:"type"`
	Title     string   `json:"title,omitempty"`
	OwnerID   string   `json:"owner_id,omitempty"`
	MemberIDs []string `json:"member_ids"`
	// Roles maps user id -> role (member | admin | owner). Populated by
	// handlers from chat_members; empty in older callers.
	Roles map[string]string `json:"roles,omitempty"`
	// Per-user settings for the requesting user (pin/archive/folder/mute).
	Pinned   bool   `json:"pinned,omitempty"`
	Archived bool   `json:"archived,omitempty"`
	Folder   string `json:"folder,omitempty"`
	Muted    bool   `json:"muted,omitempty"`
	// WhatsApp-style list data for the requesting user, populated by
	// ListChats only: the latest message preview and how many messages
	// from others the user has not read yet.
	LastMessage *LastMessagePreview `json:"last_message,omitempty"`
	UnreadCount int                 `json:"unread_count,omitempty"`
	// The chat's pinned message (Stage 1), if any, populated by
	// ListChats/GetChat. The client renders it as the pinned banner.
	PinnedMessage *Message  `json:"pinned_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LastMessagePreview is the chat-list preview of the latest message.
// Content is empty for E2EE messages (ciphertext is never a preview);
// the client shows "Encrypted message" for kind "e2ee" instead.
type LastMessagePreview struct {
	Content   string    `json:"content,omitempty"`
	Kind      string    `json:"kind"`
	SenderID  string    `json:"sender_id"`
	Status    string    `json:"status,omitempty"` // sent | delivered | read
	CreatedAt time.Time `json:"created_at"`
}

// ChatSettings are per-user, per-chat preferences (Telegram-style pin /
// archive / folders, WhatsApp-style mute). A folder is a free-form label
// such as "work", "family" or "unread"; the empty folder means the
// default list.
type ChatSettings struct {
	ChatID   string `json:"chat_id"`
	UserID   string `json:"user_id"`
	Pinned   bool   `json:"pinned"`
	Archived bool   `json:"archived"`
	Folder   string `json:"folder,omitempty"`
	Muted    bool   `json:"muted"`
}

type ChatMember struct {
	ChatID   string    `json:"chat_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"` // member | admin | owner
	JoinedAt time.Time `json:"joined_at"`
}

// ChatInvite is a Telegram-style invite link for a group or channel.
// Anyone with the token can preview the chat and join it; the owner or
// an admin can revoke the link, after which the token stops working.
// The shareable link is https://vyro-web-8p4l.onrender.com/join/<token>
// (the token is what the app needs; the web page is only a landing
// page until a join page exists there).
type ChatInvite struct {
	Token     string    `json:"token"`
	ChatID    string    `json:"chat_id"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

// Message is a single chat message. Content is the plaintext only in the
// current (non-E2EE) dev build — see docs/SECURITY.md. In the E2EE build
// Content will carry the ciphertext envelope and ContentType will be
// "e2ee" instead of "text".
type Message struct {
	ID              string              `json:"id"`
	ChatID          string              `json:"chat_id"`
	SenderID        string              `json:"sender_id"`
	Content         string              `json:"content"`
	Kind            string              `json:"kind"` // text | image | video | file | voice | location | sticker | contact
	FileURL         string              `json:"file_url,omitempty"`
	FileName        string              `json:"file_name,omitempty"`
	FileSize        int64               `json:"file_size,omitempty"`
	MimeType        string              `json:"mime_type,omitempty"`
	ReplyToID       string              `json:"reply_to_id,omitempty"`
	ForwardedFrom   string              `json:"forwarded_from,omitempty"`    // display name of the original sender (message forwarding)
	ForwardedFromID string              `json:"forwarded_from_id,omitempty"` // user id of the original sender
	Lat             *float64            `json:"lat,omitempty"`
	Lng             *float64            `json:"lng,omitempty"`
	LocationLabel   string              `json:"location_label,omitempty"`
	DeliveredTo     []string            `json:"delivered_to,omitempty"`
	ReadBy          []string            `json:"read_by,omitempty"`
	Status          string              `json:"status"`
	ExpiresAt       *time.Time          `json:"expires_at,omitempty"`
	EditedAt        *time.Time          `json:"edited_at,omitempty"`
	DeletedAt       *time.Time          `json:"deleted_at,omitempty"`
	Reactions       map[string][]string `json:"reactions,omitempty"` // emoji -> user ids
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// StatusFor computes the WhatsApp-style tick status from receipts and the
// other members of the chat (everyone except the sender).
func (m *Message) StatusFor(otherMembers []string) string {
	if len(otherMembers) == 0 {
		return "sent"
	}
	read := map[string]bool{}
	for _, id := range m.ReadBy {
		read[id] = true
	}
	allRead := true
	for _, id := range otherMembers {
		if !read[id] {
			allRead = false
			break
		}
	}
	if allRead {
		return "read"
	}
	for _, id := range m.DeliveredTo {
		for _, other := range otherMembers {
			if id == other {
				return "delivered"
			}
		}
	}
	return "sent"
}

// ScheduledMessage is a message held by the server until SendAt, then
// delivered as a normal Message (see handlers.ProcessDueScheduled).
type ScheduledMessage struct {
	ID            string    `json:"id"`
	ChatID        string    `json:"chat_id"`
	SenderID      string    `json:"sender_id"`
	Content       string    `json:"content"`
	Kind          string    `json:"kind"`
	FileURL       string    `json:"file_url,omitempty"`
	FileName      string    `json:"file_name,omitempty"`
	FileSize      int64     `json:"file_size,omitempty"`
	MimeType      string    `json:"mime_type,omitempty"`
	ReplyToID     string    `json:"reply_to_id,omitempty"`
	SendAt        time.Time `json:"send_at"`
	Sent          bool      `json:"sent"`
	SentMessageID string    `json:"sent_message_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Bot is a simple Telegram-style bot. The Token is returned only once, at
// creation; it authenticates bot requests (X-Bot-Token header). A bot also
// has a normal user account (IsBot) so DMs with it work like any chat.
type Bot struct {
	ID         string    `json:"id"`
	OwnerID    string    `json:"owner_id"`
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	Token      string    `json:"token,omitempty"`
	WebhookURL string    `json:"webhook_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// IdentityKey is a device's public X3DH identity + signed pre-key.
// Public keys only — private keys NEVER leave the device.
type IdentityKey struct {
	UserID         string `json:"user_id"`
	DeviceID       string `json:"device_id"`
	IdentityKey    string `json:"identity_key"`
	SignedPreKey   string `json:"signed_prekey"`
	SignedPreKeyID int    `json:"signed_prekey_id"`
	// v0.5 E2EE: Ed25519 signing public key (base64) + Ed25519 signature
	// over the signed pre-key (base64). Empty for pre-v0.5 uploads.
	SignKey         string    `json:"sign_key,omitempty"`
	SignedPreKeySig string    `json:"signed_prekey_sig,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// OneTimePreKey is a single-use public pre-key for X3DH session setup.
type OneTimePreKey struct {
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	KeyID     int       `json:"key_id"`
	PublicKey string    `json:"public_key"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

type Reaction struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

// GroupSenderKey is one sender-key distribution envelope for group E2EE
// (v0.6, Sender Keys). A group member's device creates a sender key and
// encrypts it, once per recipient device, under the existing 1:1 E2EE
// session with that device; the server stores the resulting ciphertext
// envelope (Ciphertext) and relays it. The server never sees the sender
// key and cannot decrypt group messages. KeyID identifies one sender-key
// generation: senders rotate it when group membership changes.
type GroupSenderKey struct {
	ChatID         string    `json:"chat_id"`
	SenderUserID   string    `json:"sender_user_id"`
	SenderDeviceID string    `json:"sender_device_id"`
	KeyID          string    `json:"key_id"`
	ForUserID      string    `json:"for_user_id"`
	ForDeviceID    string    `json:"for_device_id"`
	Ciphertext     string    `json:"ciphertext"`
	CreatedAt      time.Time `json:"created_at"`
}

// Device is one device of a user account (v0.6 multi-device sync).
// DeviceID "default" is the first device; further devices are e.g.
// "phone-2", "laptop". Each device has its own E2EE identity key
// (see IdentityKey) and its own key bundle.
type Device struct {
	UserID     string    `json:"user_id"`
	DeviceID   string    `json:"device_id"`
	Name       string    `json:"name,omitempty"`
	Platform   string    `json:"platform,omitempty"` // web | android | ios | desktop
	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// PublicDevice is the safe view of a device returned to other users so
// they can fetch a key bundle per device. It carries no secrets.
type PublicDevice struct {
	DeviceID   string    `json:"device_id"`
	Platform   string    `json:"platform,omitempty"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

func (d *Device) Public() PublicDevice {
	return PublicDevice{DeviceID: d.DeviceID, Platform: d.Platform, LastSeenAt: d.LastSeenAt}
}

type Story struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	MediaURL  string    `json:"media_url"`
	Caption   string    `json:"caption,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Email verification (v0.7) ---

// EmailCode is one issued 6-digit verification code. Only the SHA-256
// hash of the code is ever stored — the plain code exists only in the
// email sent to the user and is never logged or returned by the API.
// Purpose is "register" or "login". A code is single-use (ConsumedAt
// set on success) and expires 10 minutes after creation.
type EmailCode struct {
	Email      string
	Purpose    string // "register" | "login"
	CodeHash   string // hex SHA-256 of the 6-digit code
	Attempts   int    // failed verification attempts against this code
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// PendingUser holds a registration that has not been verified yet.
// The real user row is created only when the emailed code is verified
// (see handlers.VerifyRegister). The password is already bcrypt-hashed
// here, exactly as it will be stored on the user row.
type PendingUser struct {
	Email        string
	Username     string
	DisplayName  string
	PasswordHash string
	ExpiresAt    time.Time // same 10-minute window as its code
	CreatedAt    time.Time
}
