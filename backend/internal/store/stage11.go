package store

import (
	"strings"
)

// Stage 11 (Settings parity, WhatsApp-style): read receipts and
// profile-photo / about / groups privacy, in-memory store.
// "everyone" is the default; "nobody" hides the thing from everyone
// except the user themself. Postgres implementations live in
// postgres_stage11.go; schema in migrations/017_stage11_settings.sql.

// ValidPrivacyValue reports whether v is an accepted privacy value.
func ValidPrivacyValue(v string) bool {
	return v == LastSeenEveryone || v == LastSeenNobody
}

// GetReadReceipts reports whether the user sends read receipts
// (default true — WhatsApp default).
func (s *MemoryStore) GetReadReceipts(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.readReceipts[userID]; ok {
		return v
	}
	return true
}

func (s *MemoryStore) SetReadReceipts(userID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	s.readReceipts[userID] = enabled
	return nil
}

func (s *MemoryStore) GetAvatarPrivacy(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.avatarPrivacy[userID]; ok {
		return v
	}
	return LastSeenEveryone
}

func (s *MemoryStore) SetAvatarPrivacy(userID, privacy string) error {
	if !ValidPrivacyValue(privacy) {
		return ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	s.avatarPrivacy[userID] = privacy
	return nil
}

func (s *MemoryStore) GetAboutPrivacy(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.aboutPrivacy[userID]; ok {
		return v
	}
	return LastSeenEveryone
}

func (s *MemoryStore) SetAboutPrivacy(userID, privacy string) error {
	if !ValidPrivacyValue(privacy) {
		return ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	s.aboutPrivacy[userID] = privacy
	return nil
}

func (s *MemoryStore) GetGroupsPrivacy(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.groupsPrivacy[userID]; ok {
		return v
	}
	return LastSeenEveryone
}

func (s *MemoryStore) SetGroupsPrivacy(userID, privacy string) error {
	if !ValidPrivacyValue(privacy) {
		return ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	s.groupsPrivacy[userID] = privacy
	return nil
}

// DeleteUser removes the account and everything it owns, mirroring
// the Postgres FK cascades (all REFERENCES users(...) are
// ON DELETE CASCADE, so DELETE FROM users does this on the live
// database). After it, the user's tokens stop working and the
// email/username are free for a new account.
func (s *MemoryStore) DeleteUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok {
		return ErrNotFound
	}
	email := strings.ToLower(u.Email)

	// Identity + settings.
	delete(s.users, userID)
	delete(s.byEmail, email)
	delete(s.byName, strings.ToLower(u.Username))
	delete(s.lastSeenVisibility, userID)
	delete(s.statusPrivacy, userID)
	delete(s.statusExclusions, userID)
	for ownerID := range s.statusExclusions {
		delete(s.statusExclusions[ownerID], userID)
	}
	delete(s.readReceipts, userID)
	delete(s.avatarPrivacy, userID)
	delete(s.aboutPrivacy, userID)
	delete(s.groupsPrivacy, userID)
	delete(s.presence, userID)
	delete(s.pushSubs, userID)
	delete(s.fcmTokens, userID)
	delete(s.prekeys, userID)
	delete(s.pendingUsers, email)
	for k := range s.emailCodes {
		if strings.HasPrefix(k, email+"|") {
			delete(s.emailCodes, k)
		}
	}

	// Blocks (both directions), devices, reports.
	delete(s.blocks, userID)
	for blocker := range s.blocks {
		delete(s.blocks[blocker], userID)
	}
	for id, d := range s.devices {
		if d.UserID == userID {
			delete(s.devices, id)
		}
	}
	keptReports := s.reports[:0]
	for _, r := range s.reports {
		if r.ReporterID != userID && r.TargetID != userID {
			keptReports = append(keptReports, r)
		}
	}
	s.reports = keptReports

	// Stories (their own, plus their views/reactions/replies on
	// other people's stories).
	for id, st := range s.stories {
		if st.UserID == userID {
			delete(s.stories, id)
			delete(s.storyViews, id)
			delete(s.storyReactions, id)
			delete(s.storyReplies, id)
		}
	}
	for storyID := range s.storyViews {
		delete(s.storyViews[storyID], userID)
	}
	for storyID := range s.storyReactions {
		delete(s.storyReactions[storyID], userID)
	}
	for storyID, replies := range s.storyReplies {
		kept := replies[:0]
		for _, r := range replies {
			if r.UserID != userID {
				kept = append(kept, r)
			}
		}
		s.storyReplies[storyID] = kept
	}

	// Messages they sent (and their receipts/views on others').
	for id, m := range s.messages {
		if m.SenderID == userID {
			delete(s.messages, id)
			delete(s.receipts, id)
			delete(s.deletedForMe, id)
			delete(s.messageViews, id)
		}
	}
	for chatID, ids := range s.chatMsgs {
		kept := ids[:0]
		for _, id := range ids {
			if _, ok := s.messages[id]; ok {
				kept = append(kept, id)
			}
		}
		s.chatMsgs[chatID] = kept
	}
	for msgID := range s.receipts {
		delete(s.receipts[msgID], userID)
	}
	for msgID := range s.deletedForMe {
		delete(s.deletedForMe[msgID], userID)
	}
	for msgID := range s.messageViews {
		delete(s.messageViews[msgID], userID)
	}

	// Chats: leave groups/DMs, drop roles and per-chat settings;
	// ownership falls away (Postgres: chats.owner_id SET NULL).
	for _, c := range s.chats {
		members := c.MemberIDs[:0]
		for _, id := range c.MemberIDs {
			if id != userID {
				members = append(members, id)
			}
		}
		c.MemberIDs = members
		if c.OwnerID == userID {
			c.OwnerID = ""
		}
		delete(s.memberRoles[c.ID], userID)
		delete(s.chatSettings, settingsKey(c.ID, userID))
		if pinned, ok := s.pinnedMessages[c.ID]; ok {
			if _, gone := s.messages[pinned]; !gone {
				delete(s.pinnedMessages, c.ID)
			}
		}
	}

	// Polls (their own, plus their votes elsewhere), invites,
	// calls, communities, bots, scheduled messages, E2EE keys.
	for id, p := range s.polls {
		if p.CreatorID == userID {
			delete(s.polls, id)
			delete(s.pollVotes, id)
		}
	}
	for pollID := range s.pollVotes {
		delete(s.pollVotes[pollID], userID)
	}
	for token, inv := range s.invites {
		if inv.CreatedBy == userID {
			delete(s.invites, token)
		}
	}
	keptCalls := s.callRecords[:0]
	for _, rec := range s.callRecords {
		if rec.CallerID != userID && rec.CalleeID != userID {
			keptCalls = append(keptCalls, rec)
		}
	}
	s.callRecords = keptCalls
	for id, cm := range s.communities {
		if cm.OwnerID == userID {
			delete(s.communities, id)
			delete(s.communityMembers, id)
			delete(s.communitySubgroups, id)
		}
	}
	for communityID := range s.communityMembers {
		delete(s.communityMembers[communityID], userID)
	}
	for id, b := range s.bots {
		if b.OwnerID == userID {
			delete(s.bots, id)
			delete(s.botsByToken, b.Token)
		}
	}
	for id, sm := range s.scheduled {
		if sm.SenderID == userID {
			delete(s.scheduled, id)
		}
	}
	for k := range s.identityKeys {
		if strings.HasPrefix(k, userID+"/") {
			delete(s.identityKeys, k)
		}
	}
	for k := range s.groupSenderKeys {
		// key: chatID|senderUserID/senderDeviceID|keyID|forUserID/forDeviceID
		parts := strings.Split(k, "|")
		if len(parts) == 4 {
			sender := strings.SplitN(parts[1], "/", 2)[0]
			forUser := strings.SplitN(parts[3], "/", 2)[0]
			if sender == userID || forUser == userID {
				delete(s.groupSenderKeys, k)
			}
		}
	}
	return nil
}
