// Package store — v0.6 additions for MemoryStore: group E2EE sender-key
// distribution (ciphertext only), multi-device registry, per-device
// one-time pre-key claims, and Web Push subscriptions. PostgreSQL
// implementations of the same StoreIface methods live in
// postgres_v06.go.
package store

import (
	"sort"
	"time"

	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/push"
)

func groupSenderKeyKey(chatID, senderUserID, senderDeviceID, keyID, forUserID, forDeviceID string) string {
	return chatID + "|" + senderUserID + "/" + senderDeviceID + "|" + keyID + "|" + forUserID + "/" + forDeviceID
}

func deviceKey(userID, deviceID string) string { return userID + "/" + deviceID }

// --- Group E2EE sender-key distribution ---

// PutGroupSenderKey stores one distribution envelope: the sender's
// sender-key material encrypted under the 1:1 session to the recipient
// device. The server stores the ciphertext as-is and never decrypts
// it. Re-uploading the same (chat, sender, key, recipient) replaces
// the envelope (sender re-distributed, e.g. after rotation).
func (s *MemoryStore) PutGroupSenderKey(d *models.GroupSenderKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[d.ChatID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[d.ForUserID]; !ok {
		return ErrNotFound
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	cp := *d
	s.groupSenderKeys[groupSenderKeyKey(d.ChatID, d.SenderUserID, d.SenderDeviceID, d.KeyID, d.ForUserID, d.ForDeviceID)] = &cp
	return nil
}

// GroupSenderKeysFor returns every distribution envelope addressed to
// one device in a chat (all senders, all key generations), oldest first,
// so a device can pick up the sender keys it needs.
func (s *MemoryStore) GroupSenderKeysFor(chatID, forUserID, forDeviceID string) []*models.GroupSenderKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.GroupSenderKey{}
	for _, d := range s.groupSenderKeys {
		if d.ChatID == chatID && d.ForUserID == forUserID && d.ForDeviceID == forDeviceID {
			cp := *d
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// DeleteGroupSenderKeysForUser deletes every distribution addressed to
// a user in a chat. Called when a member is removed, so their devices
// cannot pick up sender keys afterwards.
func (s *MemoryStore) DeleteGroupSenderKeysForUser(chatID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return ErrNotFound
	}
	for k, d := range s.groupSenderKeys {
		if d.ChatID == chatID && d.ForUserID == userID {
			delete(s.groupSenderKeys, k)
		}
	}
	return nil
}

// --- Multi-device registry ---

// RegisterDevice upserts a device of a user account. The same device_id
// re-registering (new login on the same device) updates its name,
// platform and last-seen time instead of duplicating it.
func (s *MemoryStore) RegisterDevice(d *models.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[d.UserID]; !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	if existing, ok := s.devices[deviceKey(d.UserID, d.DeviceID)]; ok {
		existing.Name = d.Name
		existing.Platform = d.Platform
		existing.LastSeenAt = now
		d.CreatedAt, d.LastSeenAt = existing.CreatedAt, now
		return nil
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.LastSeenAt = now
	cp := *d
	s.devices[deviceKey(d.UserID, d.DeviceID)] = &cp
	return nil
}

// ListDevices returns a user's registered devices, oldest first.
func (s *MemoryStore) ListDevices(userID string) []*models.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Device{}
	for _, d := range s.devices {
		if d.UserID == userID {
			cp := *d
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// RemoveDevice unregisters a device. Its identity keys and pre-keys are
// left in place (they are public keys; the device's private keys never
// left the device), but no new key bundles are served for it once the
// device row is gone — callers check ListDevices first.
func (s *MemoryStore) RemoveDevice(userID, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.devices[deviceKey(userID, deviceID)]; !ok {
		return ErrNotFound
	}
	delete(s.devices, deviceKey(userID, deviceID))
	return nil
}

// TouchDevice records that a device was seen now.
func (s *MemoryStore) TouchDevice(userID, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[deviceKey(userID, deviceID)]
	if !ok {
		return ErrNotFound
	}
	d.LastSeenAt = time.Now().UTC()
	return nil
}

// FetchOneTimePreKeyForDevice claims one unused pre-key of one device
// of the user and marks it used. A key bundle is per-device, so the
// one-time pre-key in it must belong to the same device as the identity
// key and signed pre-key — claiming any device's key would build a
// session the bundle's device cannot open.
func (s *MemoryStore) FetchOneTimePreKeyForDevice(userID, deviceID string) (*models.OneTimePreKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.prekeys[userID] {
		if k.DeviceID == deviceID && !k.Used {
			k.Used = true
			return k, nil
		}
	}
	return nil, ErrNotFound
}

// --- Web Push subscriptions (persisted; see push.Registry) ---

// SavePushSubscription stores a subscription (idempotent by endpoint).
func (s *MemoryStore) SavePushSubscription(userID string, sub push.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	for _, existing := range s.pushSubs[userID] {
		if existing.Endpoint == sub.Endpoint {
			return nil
		}
	}
	s.pushSubs[userID] = append(s.pushSubs[userID], sub)
	return nil
}

// ListPushSubscriptions returns a copy of a user's subscriptions.
func (s *MemoryStore) ListPushSubscriptions(userID string) []push.Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]push.Subscription, len(s.pushSubs[userID]))
	copy(out, s.pushSubs[userID])
	return out
}

// DeletePushSubscription removes a subscription by endpoint.
func (s *MemoryStore) DeletePushSubscription(userID, endpoint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.pushSubs[userID][:0]
	for _, sub := range s.pushSubs[userID] {
		if sub.Endpoint != endpoint {
			out = append(out, sub)
		}
	}
	s.pushSubs[userID] = out
	return nil
}
