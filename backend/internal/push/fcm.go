// FCM (Firebase Cloud Messaging) push for the mobile app.
//
// The web client keeps using Web Push (VAPID) — see push.go. FCM is the
// second, parallel path: it wakes the Android/iOS app when it is closed
// or backgrounded, which Web Push cannot do on a phone.
//
// How it works: the mobile app gets an FCM registration token from
// Firebase (google-services.json is bundled in the app; no server key
// is involved on the client) and POSTs it to /api/push/fcm-token. The
// server stores one token per (user, device) and, when a message
// arrives for a member who did not receive it over WebSocket, sends a
// notification to each of their tokens through the Firebase Admin SDK
// (FCM HTTP v1 under the hood).
//
// Credentials: the service-account JSON is read ONLY from the
// environment — FIREBASE_SERVICE_ACCOUNT_JSON (the raw JSON) or
// FIREBASE_SERVICE_ACCOUNT_PATH (a file path, e.g. a Render secret
// file). It is never hardcoded, never committed, and never logged.
// When neither variable is set, FCM stays disabled and the server
// behaves exactly as before (Web Push only); endpoints answer 503.
//
// Privacy (same rule as Web Push): payloads never contain message
// content — only "New message from <name>" (or "New encrypted message"
// for E2EE chats), plus a data field with the chat id so the app can
// open the right chat on tap. FCM never sees plaintext.
//
// Honest limits: real delivery to a phone was NOT tested here — the
// tests prove token storage, the send path against a fake client, and
// dead-token pruning. Testing real delivery needs the app installed on
// a real phone with a registered token, which this environment cannot
// do. A token that FCM reports as unregistered (app uninstalled, token
// revoked) is deleted automatically so the server stops sending to it.
package push

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMToken is one mobile device's Firebase Cloud Messaging token.
type FCMToken struct {
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"` // "android" or "ios"
}

// FCMTokenStore persists FCM tokens. Both store.StoreIface
// implementations (MemoryStore, PostgresStore) satisfy it, so tokens
// live in the store and survive a restart; when no store is given the
// registry keeps them in memory.
type FCMTokenStore interface {
	SaveFCMToken(userID string, tok FCMToken) error
	ListFCMTokens(userID string) []FCMToken
	DeleteFCMToken(userID, token string) error
}

// FCMClient sends one notification to one FCM token. The real
// implementation wraps the Firebase Admin SDK messaging client; tests
// use a fake.
type FCMClient interface {
	Send(ctx context.Context, token, title, body string, data map[string]string) error
}

// ErrTokenUnregistered is returned by Send when FCM says the token is
// dead (app uninstalled, token revoked). The registry deletes such
// tokens instead of retrying them forever.
var ErrTokenUnregistered = errors.New("fcm token is not registered")

type adminFCMClient struct {
	client *messaging.Client
}

// Send delivers one notification via FCM HTTP v1. The notification
// payload is what Android/iOS shows when the app is backgrounded or
// killed; the data payload is what the app reads on tap (chat_id) and
// in the foreground.
func (c *adminFCMClient) Send(ctx context.Context, token, title, body string, data map[string]string) error {
	_, err := c.client.Send(ctx, &messaging.Message{
		Token:        token,
		Notification: &messaging.Notification{Title: title, Body: body},
		Data:         data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{ContentAvailable: true},
			},
		},
	})
	if err == nil {
		return nil
	}
	if messaging.IsUnregistered(err) {
		return ErrTokenUnregistered
	}
	return err
}

// NewFCMClient builds a client from service-account JSON bytes (the
// contents of the key file, not a path).
func NewFCMClient(ctx context.Context, jsonCreds []byte) (FCMClient, error) {
	if len(strings.TrimSpace(string(jsonCreds))) == 0 {
		return nil, errors.New("empty Firebase service-account JSON")
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON(jsonCreds))
	if err != nil {
		return nil, fmt.Errorf("firebase app: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase messaging client: %w", err)
	}
	return &adminFCMClient{client: client}, nil
}

// NewFCMClientFromEnv builds a client from the environment:
// FIREBASE_SERVICE_ACCOUNT_JSON (raw JSON) wins, otherwise
// FIREBASE_SERVICE_ACCOUNT_PATH (path to the JSON file). It returns
// (nil, nil) when neither is set — FCM simply stays disabled. A set
// but unreadable/invalid credential is an error, not a silent skip:
// misconfiguration must be loud at startup.
func NewFCMClientFromEnv(ctx context.Context) (FCMClient, error) {
	jsonCreds := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
	if jsonCreds == "" {
		if p := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"); p != "" {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("read FIREBASE_SERVICE_ACCOUNT_PATH: %w", err)
			}
			jsonCreds = string(b)
		}
	}
	if strings.TrimSpace(jsonCreds) == "" {
		return nil, nil
	}
	return NewFCMClient(ctx, []byte(jsonCreds))
}

// FCMRegistry holds FCM tokens per user id and sends through an
// FCMClient. When the client is nil (no credentials configured) sends
// are no-ops and Available reports false.
type FCMRegistry struct {
	mu     sync.Mutex
	tokens map[string][]FCMToken // in-memory fallback when store is nil
	store  FCMTokenStore
	client FCMClient
}

// NewFCMRegistry returns a registry backed by st and sending through
// client. Either may be nil: a nil store means tokens live in memory
// only; a nil client means FCM is disabled (Available() is false and
// SendToUser sends nothing).
func NewFCMRegistry(st FCMTokenStore, client FCMClient) *FCMRegistry {
	return &FCMRegistry{tokens: map[string][]FCMToken{}, store: st, client: client}
}

// Available reports whether FCM can actually send (credentials were
// configured and the Admin SDK client was built).
func (r *FCMRegistry) Available() bool { return r.client != nil }

// Add stores a token (idempotent by token value). A token re-registered
// for another device id replaces the old row for that token.
func (r *FCMRegistry) Add(userID string, tok FCMToken) {
	if r.store != nil {
		_ = r.store.SaveFCMToken(userID, tok)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, t := range r.tokens[userID] {
		if t.Token == tok.Token {
			r.tokens[userID][i] = tok
			return
		}
	}
	r.tokens[userID] = append(r.tokens[userID], tok)
}

// Remove deletes a token. Returns true when one existed (in-memory
// registry) or the store accepted the delete.
func (r *FCMRegistry) Remove(userID, token string) bool {
	if r.store != nil {
		return r.store.DeleteFCMToken(userID, token) == nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.tokens[userID][:0]
	removed := false
	for _, t := range r.tokens[userID] {
		if t.Token == token {
			removed = true
			continue
		}
		out = append(out, t)
	}
	r.tokens[userID] = out
	return removed
}

// List returns a copy of a user's tokens.
func (r *FCMRegistry) List(userID string) []FCMToken {
	if r.store != nil {
		return r.store.ListFCMTokens(userID)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]FCMToken, len(r.tokens[userID]))
	copy(out, r.tokens[userID])
	return out
}

// Count returns how many tokens a user has.
func (r *FCMRegistry) Count(userID string) int {
	return len(r.List(userID))
}

// SendToUser sends a content-free notification to every token of a
// user, returning (sent, failed) counts. Failures are per-token and
// never fatal to the caller. Tokens FCM reports as unregistered are
// deleted so the server stops sending to dead devices.
func (r *FCMRegistry) SendToUser(ctx context.Context, userID, title, body string, data map[string]string) (sent, failed int) {
	if r.client == nil {
		return 0, 0
	}
	for _, tok := range r.List(userID) {
		err := r.client.Send(ctx, tok.Token, title, body, data)
		if err == nil {
			sent++
			continue
		}
		if errors.Is(err, ErrTokenUnregistered) {
			r.Remove(userID, tok.Token)
		}
		failed++
	}
	return sent, failed
}
