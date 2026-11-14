// Package push implements free Web Push (VAPID, RFC 8292) for VYRO.
//
// No paid FCM/APNs is used: the browser's own push service (which the
// browser vendor provides to the browser, not a service VYRO pays for)
// delivers the notification. The server encrypts the payload with the
// subscription keys (webpush-go, free/open-source) and signs requests
// with a VAPID keypair.
//
// v0.6: subscriptions are persisted through a Store (PostgreSQL when
// the server runs with DATABASE_URL — see NewRegistryWithStore, wired
// in cmd/server/main.go — or the in-memory store in dev). A restart no
// longer drops subscriptions: a fresh Registry over the same store
// still sees them (proven by TestRegistrySurvivesRestart and the
// Postgres integration test). When no store is given the registry
// keeps subscriptions in memory, as in v0.5.
//
// Honest limits (unchanged): real end-to-end delivery through a
// browser push service (e.g. FCM) was NOT tested here: the test in
// push_test.go proves the full crypto path against a local HTTP
// endpoint that holds the subscription private key and decrypts the
// payload. Testing real delivery needs a subscription a real browser
// generated, which this environment cannot produce (no browser run).
// Push bodies never contain message content — only "New message from
// <username>" (and "New encrypted message" for E2EE chats), so a push
// service never sees plaintext.
package push

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/SherClockHolmes/webpush-go"
)

// Subscription is one browser push subscription.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

func (s Subscription) toWebPush() *webpush.Subscription {
	return &webpush.Subscription{
		Endpoint: s.Endpoint,
		Keys: webpush.Keys{
			P256dh: s.P256dh,
			Auth:   s.Auth,
		},
	}
}

// Store persists push subscriptions. Both store.StoreIface
// implementations (MemoryStore, PostgresStore) satisfy it, so the
// registry can be backed by either; the server wires its real store.
type Store interface {
	SavePushSubscription(userID string, sub Subscription) error
	ListPushSubscriptions(userID string) []Subscription
	DeletePushSubscription(userID, endpoint string) error
}

// Registry holds subscriptions per user id. When a Store is set (see
// NewRegistryWithStore) subscriptions live in the store and survive a
// restart; otherwise they live in memory only.
type Registry struct {
	mu    sync.Mutex
	subs  map[string][]Subscription
	store Store
}

// NewRegistry returns an empty in-memory registry.
func NewRegistry() *Registry { return &Registry{subs: map[string][]Subscription{}} }

// NewRegistryWithStore returns a registry backed by st. When st is nil
// it behaves like NewRegistry.
func NewRegistryWithStore(st Store) *Registry {
	return &Registry{subs: map[string][]Subscription{}, store: st}
}

// Add stores a subscription (idempotent by endpoint).
func (r *Registry) Add(userID string, sub Subscription) {
	if r.store != nil {
		_ = r.store.SavePushSubscription(userID, sub)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.subs[userID] {
		if s.Endpoint == sub.Endpoint {
			return
		}
	}
	r.subs[userID] = append(r.subs[userID], sub)
}

// Remove deletes a subscription by endpoint. Returns true if one existed
// (in-memory registry) or the store accepted the delete.
func (r *Registry) Remove(userID, endpoint string) bool {
	if r.store != nil {
		return r.store.DeletePushSubscription(userID, endpoint) == nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.subs[userID][:0]
	removed := false
	for _, s := range r.subs[userID] {
		if s.Endpoint == endpoint {
			removed = true
			continue
		}
		out = append(out, s)
	}
	r.subs[userID] = out
	return removed
}

// List returns a copy of a user's subscriptions.
func (r *Registry) List(userID string) []Subscription {
	if r.store != nil {
		return r.store.ListPushSubscriptions(userID)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Subscription, len(r.subs[userID]))
	copy(out, r.subs[userID])
	return out
}

// Count returns how many subscriptions a user has.
func (r *Registry) Count(userID string) int {
	return len(r.List(userID))
}

// GenerateVAPIDKeys creates a fresh VAPID keypair (P-256). In
// production set VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY in the
// environment so keys survive restarts; the server generates ephemeral
// keys at startup only when none are configured (and logs a warning).
func GenerateVAPIDKeys() (publicKey, privateKey string, err error) {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", err
	}
	return pub, priv, nil
}

// Send delivers payload to one subscription. vapidPublicKey /
// vapidPrivateKey are the server's VAPID keys, subject is the VAPID
// Send delivers one push notification. It reports gone=true when the
// push service answers 410 Gone / 404 Not Found: the subscription is
// dead and must be removed (RFC 8030: the application server MUST stop
// sending to it). Any other failure is transient.
func Send(sub Subscription, payload []byte, vapidPublicKey, vapidPrivateKey, subject string) (gone bool, err error) {
	resp, err := webpush.SendNotification(payload, sub.toWebPush(), &webpush.Options{
		Subscriber:      subject,
		VAPIDPublicKey:  vapidPublicKey,
		VAPIDPrivateKey: vapidPrivateKey,
		TTL:             60,
	})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		return true, nil
	}
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("push service returned %s", resp.Status)
	}
	return false, nil
}

// SendToAll sends payload to every subscription of a user, returning
// (sent, failed) counts. Failures are per-subscription and never fatal
// to the caller (a dead endpoint must not break sending a message).
// Subscriptions the push service reports gone (410/404) are removed.
func (r *Registry) SendToAll(userID string, payload []byte, vapidPublicKey, vapidPrivateKey, subject string) (sent, failed int) {
	for _, sub := range r.List(userID) {
		gone, err := Send(sub, payload, vapidPublicKey, vapidPrivateKey, subject)
		if gone {
			r.Remove(userID, sub.Endpoint)
			failed++
			continue
		}
		if err != nil {
			failed++
		} else {
			sent++
		}
	}
	return sent, failed
}
