// Package store — v0.4 additions for MemoryStore: admin roles, read
// receipts, presence, stories, scheduled messages, bots, E2EE key
// foundation and per-user chat settings. PostgreSQL implementations of
// the same StoreIface methods live in postgres_v04.go.
package store

import (
	"errors"
	"sort"
	"time"

	"github.com/vyro/backend/internal/models"
)

func settingsKey(chatID, userID string) string { return chatID + "/" + userID }
func keyKey(userID, deviceID string) string    { return userID + "/" + deviceID }

// --- Admin roles ---

func (s *MemoryStore) MemberRole(chatID, userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if roles, ok := s.memberRoles[chatID]; ok {
		return roles[userID]
	}
	return ""
}

func (s *MemoryStore) ChatMembers(chatID string) []*models.ChatMember {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.chats[chatID]
	if !ok {
		return []*models.ChatMember{}
	}
	out := []*models.ChatMember{}
	for _, id := range c.MemberIDs {
		role := s.memberRoles[chatID][id]
		if role == "" {
			role = "member"
		}
		out = append(out, &models.ChatMember{ChatID: chatID, UserID: id, Role: role})
	}
	return out
}

func (s *MemoryStore) AddMember(chatID, userID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chats[chatID]
	if !ok {
		return ErrNotFound
	}
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	if role != "member" && role != "admin" {
		role = "member"
	}
	for _, id := range c.MemberIDs {
		if id == userID {
			return ErrAlreadyExists
		}
	}
	c.MemberIDs = append(c.MemberIDs, userID)
	if s.memberRoles[chatID] == nil {
		s.memberRoles[chatID] = map[string]string{}
	}
	s.memberRoles[chatID][userID] = role
	return nil
}

func (s *MemoryStore) RemoveMember(chatID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chats[chatID]
	if !ok {
		return ErrNotFound
	}
	kept := []string{}
	found := false
	for _, id := range c.MemberIDs {
		if id == userID {
			found = true
			continue
		}
		kept = append(kept, id)
	}
	if !found {
		return ErrNotFound
	}
	c.MemberIDs = kept
	delete(s.memberRoles[chatID], userID)
	return nil
}

func (s *MemoryStore) SetMemberRole(chatID, userID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chatID]; !ok {
		return ErrNotFound
	}
	if role != "member" && role != "admin" && role != "owner" {
		return errors.New("invalid role")
	}
	if s.memberRoles[chatID] == nil || s.memberRoles[chatID][userID] == "" {
		return ErrNotFound
	}
	s.memberRoles[chatID][userID] = role
	return nil
}

// --- Receipts ---

func (s *MemoryStore) MarkDelivered(messageID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.messages[messageID]; !ok {
		return ErrNotFound
	}
	if s.receipts[messageID] == nil {
		s.receipts[messageID] = map[string]*receipt{}
	}
	r := s.receipts[messageID][userID]
	if r == nil {
		r = &receipt{}
		s.receipts[messageID][userID] = r
	}
	r.delivered = true
	return nil
}

func (s *MemoryStore) MarkRead(messageID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.messages[messageID]
	if !ok {
		return ErrNotFound
	}
	// WhatsApp semantics: reading a message marks every earlier
	// message from others in the same chat as read too.
	for _, id := range s.chatMsgs[m.ChatID] {
		om, ok := s.messages[id]
		if !ok || om.SenderID == userID || om.CreatedAt.After(m.CreatedAt) {
			continue
		}
		if s.receipts[id] == nil {
			s.receipts[id] = map[string]*receipt{}
		}
		r := s.receipts[id][userID]
		if r == nil {
			r = &receipt{}
			s.receipts[id][userID] = r
		}
		r.delivered = true
		r.read = true
	}
	return nil
}

// --- Presence ---

func (s *MemoryStore) SetPresence(userID string, online bool, lastSeen time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	s.presence[userID] = presenceInfo{online: online, lastSeen: lastSeen}
	if u, ok := s.users[userID]; ok {
		u.IsOnline = online
		u.LastSeenAt = lastSeen
	}
	return nil
}

func (s *MemoryStore) GetPresence(userID string) (bool, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.presence[userID]; ok {
		return p.online, p.lastSeen
	}
	if u, ok := s.users[userID]; ok {
		return u.IsOnline, u.LastSeenAt
	}
	return false, time.Time{}
}

// --- Stories (24h) ---

func (s *MemoryStore) CreateStory(st *models.Story) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[st.UserID]; !ok {
		return ErrNotFound
	}
	if st.ID == "" {
		st.ID = NewID()
	}
	now := time.Now().UTC()
	st.CreatedAt = now
	st.ExpiresAt = now.Add(24 * time.Hour)
	s.stories[st.ID] = st
	return nil
}

func (s *MemoryStore) ListStories() []*models.Story {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	out := []*models.Story{}
	for _, st := range s.stories {
		if st.ExpiresAt.After(now) {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *MemoryStore) ListStoriesByUser(userID string) []*models.Story {
	all := s.ListStories()
	out := []*models.Story{}
	for _, st := range all {
		if st.UserID == userID {
			out = append(out, st)
		}
	}
	return out
}

func (s *MemoryStore) StoriesByMediaURL(mediaURL string) []*models.Story {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Story{}
	for _, st := range s.stories {
		if st.MediaURL == mediaURL {
			cp := *st
			out = append(out, &cp)
		}
	}
	return out
}

// --- Scheduled messages ---

func (s *MemoryStore) CreateScheduled(sm *models.ScheduledMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[sm.ChatID]; !ok {
		return ErrNotFound
	}
	if sm.ID == "" {
		sm.ID = NewID()
	}
	sm.CreatedAt = time.Now().UTC()
	s.scheduled[sm.ID] = sm
	return nil
}

func (s *MemoryStore) DueScheduled(now time.Time) []*models.ScheduledMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.ScheduledMessage{}
	for _, sm := range s.scheduled {
		if !sm.Sent && !sm.SendAt.After(now) {
			cp := *sm
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SendAt.Before(out[j].SendAt) })
	return out
}

func (s *MemoryStore) MarkScheduledSent(id, messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sm, ok := s.scheduled[id]
	if !ok {
		return ErrNotFound
	}
	sm.Sent = true
	sm.SentMessageID = messageID
	return nil
}

func (s *MemoryStore) ListScheduled(chatID, userID string) []*models.ScheduledMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.ScheduledMessage{}
	for _, sm := range s.scheduled {
		if sm.ChatID == chatID && sm.SenderID == userID && !sm.Sent {
			cp := *sm
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SendAt.Before(out[j].SendAt) })
	return out
}

func (s *MemoryStore) CancelScheduled(id, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sm, ok := s.scheduled[id]
	if !ok {
		return ErrNotFound
	}
	if sm.SenderID != userID {
		return ErrForbidden
	}
	if sm.Sent {
		return ErrAlreadyExists
	}
	delete(s.scheduled, id)
	return nil
}

// PurgeExpired soft-deletes messages past their disappearing-message
// expiry and returns how many were purged.
func (s *MemoryStore) PurgeExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, m := range s.messages {
		if m.DeletedAt == nil && m.ExpiresAt != nil && !m.ExpiresAt.After(now) {
			m.DeletedAt = &now
			m.Content = ""
			n++
		}
	}
	return n
}

// --- Bots ---

func (s *MemoryStore) CreateBot(b *models.Bot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.ID == "" {
		b.ID = NewID()
	}
	b.CreatedAt = time.Now().UTC()
	s.bots[b.ID] = b
	if b.Token != "" {
		s.botsByToken[b.Token] = b.ID
	}
	return nil
}

func (s *MemoryStore) BotByToken(token string) (*models.Bot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.botsByToken[token]
	if !ok {
		return nil, ErrNotFound
	}
	return s.bots[id], nil
}

func (s *MemoryStore) BotByUsername(username string) (*models.Bot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range s.bots {
		if b.Username == username {
			return b, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) BotByID(id string) (*models.Bot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.bots[id]
	if !ok {
		return nil, ErrNotFound
	}
	return b, nil
}

func (s *MemoryStore) ListBots(ownerID string) []*models.Bot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Bot{}
	for _, b := range s.bots {
		if b.OwnerID == ownerID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *MemoryStore) DeleteBot(id, ownerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.bots[id]
	if !ok {
		return ErrNotFound
	}
	if b.OwnerID != ownerID {
		return ErrForbidden
	}
	delete(s.botsByToken, b.Token)
	delete(s.bots, id)
	return nil
}

func (s *MemoryStore) SetBotWebhook(id, ownerID, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.bots[id]
	if !ok {
		return ErrNotFound
	}
	if b.OwnerID != ownerID {
		return ErrForbidden
	}
	b.WebhookURL = url
	return nil
}

// --- E2EE key foundation (public keys only) ---

func (s *MemoryStore) UploadIdentityKey(k *models.IdentityKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[k.UserID]; !ok {
		return ErrNotFound
	}
	k.CreatedAt = time.Now().UTC()
	s.identityKeys[keyKey(k.UserID, k.DeviceID)] = k
	return nil
}

func (s *MemoryStore) GetIdentityKey(userID, deviceID string) (*models.IdentityKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.identityKeys[keyKey(userID, deviceID)]
	if !ok {
		return nil, ErrNotFound
	}
	return k, nil
}

func (s *MemoryStore) UploadOneTimePreKeys(userID, deviceID string, keys []*models.OneTimePreKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	for _, k := range keys {
		k.UserID, k.DeviceID = userID, deviceID
		k.CreatedAt = time.Now().UTC()
		// Re-publishing a key_id replaces it: a device that uploads
		// key_id N again has rotated that slot, and the old public
		// key's private half may be gone — serving it would produce
		// sessions the device cannot decrypt. Same rule as Postgres.
		replaced := false
		for i, old := range s.prekeys[userID] {
			if old.DeviceID == deviceID && old.KeyID == k.KeyID {
				k.Used = false
				s.prekeys[userID][i] = k
				replaced = true
				break
			}
		}
		if !replaced {
			s.prekeys[userID] = append(s.prekeys[userID], k)
		}
	}
	return nil
}

// FetchOneTimePreKey returns one unused pre-key for the user (any device)
// and marks it used, so it is never handed out twice.
func (s *MemoryStore) FetchOneTimePreKey(userID string) (*models.OneTimePreKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.prekeys[userID] {
		if !k.Used {
			k.Used = true
			return k, nil
		}
	}
	return nil, ErrNotFound
}

// --- Per-user chat settings ---

func (s *MemoryStore) GetChatSettings(chatID, userID string) models.ChatSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if cs, ok := s.chatSettings[settingsKey(chatID, userID)]; ok {
		return cs
	}
	return models.ChatSettings{ChatID: chatID, UserID: userID}
}

func (s *MemoryStore) SetChatSettings(cs models.ChatSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[cs.ChatID]; !ok {
		return ErrNotFound
	}
	s.chatSettings[settingsKey(cs.ChatID, cs.UserID)] = cs
	return nil
}
