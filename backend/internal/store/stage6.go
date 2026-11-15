package store

import (
	"errors"
	"sort"
	"time"

	"github.com/vyro/backend/internal/models"
)

// Stage 6 — Communities, MemoryStore implementation.
// Postgres versions live in postgres_stage6.go.
// Schema: migrations/016_stage6_communities.sql.

// CreateCommunity stores a new community. The caller sets the
// announcement chat first; the owner is added as a member here.
func (s *MemoryStore) CreateCommunity(c *models.Community) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = NewID()
	}
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	s.communities[c.ID] = c
	if s.communityMembers[c.ID] == nil {
		s.communityMembers[c.ID] = map[string]*models.CommunityMember{}
	}
	s.communityMembers[c.ID][c.OwnerID] = &models.CommunityMember{
		UserID: c.OwnerID, Role: "owner", JoinedAt: now,
	}
	if s.communitySubgroups[c.ID] == nil {
		s.communitySubgroups[c.ID] = map[string]bool{}
	}
	return nil
}

func (s *MemoryStore) CommunityByID(id string) *models.Community {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.communities[id]
	if !ok {
		return nil
	}
	cp := *c
	return &cp
}

// ListCommunitiesForUser returns the communities the user belongs
// to, newest first, with member/subgroup counts filled in.
func (s *MemoryStore) ListCommunitiesForUser(userID string) []*models.Community {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Community{}
	for _, c := range s.communities {
		if _, ok := s.communityMembers[c.ID][userID]; !ok {
			continue
		}
		cp := *c
		cp.MemberCount = len(s.communityMembers[c.ID])
		cp.SubgroupCount = len(s.communitySubgroups[c.ID])
		if m, ok := s.communityMembers[c.ID][userID]; ok {
			cp.MyRole = m.Role
		}
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

func (s *MemoryStore) CommunityByAnnouncementChat(chatID string) *models.Community {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.communities {
		if c.AnnouncementChatID == chatID {
			cp := *c
			return &cp
		}
	}
	return nil
}

func (s *MemoryStore) CommunityMembers(communityID string) []*models.CommunityMember {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.CommunityMember{}
	for _, m := range s.communityMembers[communityID] {
		cp := *m
		out = append(out, &cp)
	}
	rank := map[string]int{"owner": 0, "admin": 1, "member": 2}
	sort.Slice(out, func(i, j int) bool {
		if rank[out[i].Role] != rank[out[j].Role] {
			return rank[out[i].Role] < rank[out[j].Role]
		}
		return out[i].JoinedAt.Before(out[j].JoinedAt)
	})
	return out
}

func (s *MemoryStore) CommunityRole(communityID, userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.communityMembers[communityID][userID]; ok {
		return m.Role
	}
	return ""
}

func (s *MemoryStore) AddCommunityMember(communityID, userID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.communities[communityID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	if role != "member" && role != "admin" && role != "owner" {
		role = "member"
	}
	if s.communityMembers[communityID] == nil {
		s.communityMembers[communityID] = map[string]*models.CommunityMember{}
	}
	if _, ok := s.communityMembers[communityID][userID]; ok {
		return ErrAlreadyExists
	}
	s.communityMembers[communityID][userID] = &models.CommunityMember{
		UserID: userID, Role: role, JoinedAt: time.Now().UTC(),
	}
	return nil
}

func (s *MemoryStore) RemoveCommunityMember(communityID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.communities[communityID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.communityMembers[communityID][userID]; !ok {
		return ErrNotFound
	}
	delete(s.communityMembers[communityID], userID)
	return nil
}

func (s *MemoryStore) SetCommunityMemberRole(communityID, userID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.communityMembers[communityID][userID]
	if !ok {
		return ErrNotFound
	}
	if role != "member" && role != "admin" {
		return errors.New("role must be member or admin")
	}
	m.Role = role
	return nil
}

func (s *MemoryStore) AddSubgroup(communityID, chatID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.communities[communityID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.chats[chatID]; !ok {
		return ErrNotFound
	}
	if s.communitySubgroups[communityID] == nil {
		s.communitySubgroups[communityID] = map[string]bool{}
	}
	if s.communitySubgroups[communityID][chatID] {
		return ErrAlreadyExists
	}
	s.communitySubgroups[communityID][chatID] = true
	return nil
}

func (s *MemoryStore) RemoveSubgroup(communityID, chatID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.communities[communityID]; !ok {
		return ErrNotFound
	}
	if !s.communitySubgroups[communityID][chatID] {
		return ErrNotFound
	}
	delete(s.communitySubgroups[communityID], chatID)
	return nil
}

func (s *MemoryStore) ListSubgroups(communityID string) []*models.Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Chat{}
	for chatID := range s.communitySubgroups[communityID] {
		if c, ok := s.chats[chatID]; ok {
			cp := *c
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func (s *MemoryStore) UpdateCommunity(communityID string, name, description *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.communities[communityID]
	if !ok {
		return ErrNotFound
	}
	if name != nil {
		c.Name = *name
	}
	if description != nil {
		c.Description = *description
	}
	c.UpdatedAt = time.Now().UTC()
	return nil
}
