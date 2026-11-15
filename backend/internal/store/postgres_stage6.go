package store

// Stage 6 — Communities, PostgresStore implementation.
// In-memory versions live in stage6.go.
// Schema: migrations/016_stage6_communities.sql.

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/vyro/backend/internal/models"
)

func scanCommunity(row pgx.Row, memberCount, subgroupCount int) (*models.Community, error) {
	var c models.Community
	var announcementID *string
	err := row.Scan(&c.ID, &c.Name, &c.Description, &c.OwnerID,
		&announcementID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if announcementID != nil {
		c.AnnouncementChatID = *announcementID
	}
	c.MemberCount = memberCount
	c.SubgroupCount = subgroupCount
	return &c, nil
}

// CreateCommunity stores a new community and adds the owner as a
// member. The caller creates the announcement chat first.
func (s *PostgresStore) CreateCommunity(c *models.Community) error {
	if !isUUID(c.OwnerID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	announcement := pgtype.UUID{Valid: false}
	if isUUID(c.AnnouncementChatID) {
		_ = announcement.Scan(c.AnnouncementChatID)
	}
	var id string
	var createdAt, updatedAt time.Time
	if isUUID(c.ID) {
		err = tx.QueryRow(ctx,
			`INSERT INTO communities (id, name, description, owner_id, announcement_chat_id)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id::text, created_at, updated_at`,
			c.ID, c.Name, c.Description, c.OwnerID, announcement).Scan(&id, &createdAt, &updatedAt)
	} else {
		err = tx.QueryRow(ctx,
			`INSERT INTO communities (name, description, owner_id, announcement_chat_id)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id::text, created_at, updated_at`,
			c.Name, c.Description, c.OwnerID, announcement).Scan(&id, &createdAt, &updatedAt)
	}
	if err != nil {
		return mapStoreErr(err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO community_members (community_id, user_id, role)
		 VALUES ($1, $2, 'owner') ON CONFLICT DO NOTHING`,
		id, c.OwnerID); err != nil {
		return mapStoreErr(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	c.ID, c.CreatedAt, c.UpdatedAt = id, createdAt, updatedAt
	return nil
}

func (s *PostgresStore) communityCounts(ctx context.Context, communityID string) (int, int) {
	var members, subgroups int
	_ = s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM community_members WHERE community_id = $1`,
		communityID).Scan(&members)
	_ = s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM community_subgroups WHERE community_id = $1`,
		communityID).Scan(&subgroups)
	return members, subgroups
}

func (s *PostgresStore) CommunityByID(id string) *models.Community {
	if !isUUID(id) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	members, subgroups := s.communityCounts(ctx, id)
	c, err := scanCommunity(s.pool.QueryRow(ctx,
		`SELECT id::text, name, description, owner_id::text,
			announcement_chat_id::text, created_at, updated_at
		 FROM communities WHERE id = $1`, id), members, subgroups)
	if err != nil {
		return nil
	}
	return c
}

func (s *PostgresStore) ListCommunitiesForUser(userID string) []*models.Community {
	if !isUUID(userID) {
		return []*models.Community{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT c.id::text, c.name, c.description, c.owner_id::text,
			c.announcement_chat_id::text, c.created_at, c.updated_at
		 FROM communities c JOIN community_members m ON m.community_id = c.id
		 WHERE m.user_id = $1 ORDER BY c.created_at DESC`, userID)
	if err != nil {
		return []*models.Community{}
	}
	defer rows.Close()
	out := []*models.Community{}
	for rows.Next() {
		members, subgroups := 0, 0
		c, err := scanCommunity(rows, members, subgroups)
		if err != nil {
			continue
		}
		m, sg := s.communityCounts(ctx, c.ID)
		c.MemberCount, c.SubgroupCount = m, sg
		c.MyRole = s.CommunityRole(c.ID, userID)
		out = append(out, c)
	}
	return out
}

func (s *PostgresStore) CommunityByAnnouncementChat(chatID string) *models.Community {
	if !isUUID(chatID) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id string
	if err := s.pool.QueryRow(ctx,
		`SELECT id::text FROM communities WHERE announcement_chat_id = $1`,
		chatID).Scan(&id); err != nil {
		return nil
	}
	return s.CommunityByID(id)
}

func (s *PostgresStore) CommunityMembers(communityID string) []*models.CommunityMember {
	if !isUUID(communityID) {
		return []*models.CommunityMember{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT user_id::text, role, joined_at FROM community_members
		 WHERE community_id = $1 ORDER BY joined_at ASC`, communityID)
	if err != nil {
		return []*models.CommunityMember{}
	}
	defer rows.Close()
	out := []*models.CommunityMember{}
	for rows.Next() {
		var m models.CommunityMember
		if err := rows.Scan(&m.UserID, &m.Role, &m.JoinedAt); err == nil {
			out = append(out, &m)
		}
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

func (s *PostgresStore) CommunityRole(communityID, userID string) string {
	if !isUUID(communityID) || !isUUID(userID) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var role string
	if err := s.pool.QueryRow(ctx,
		`SELECT role FROM community_members WHERE community_id = $1 AND user_id = $2`,
		communityID, userID).Scan(&role); err != nil {
		return ""
	}
	return role
}

func (s *PostgresStore) AddCommunityMember(communityID, userID, role string) error {
	if !isUUID(communityID) || !isUUID(userID) {
		return ErrNotFound
	}
	if role != "member" && role != "admin" && role != "owner" {
		role = "member"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO community_members (community_id, user_id, role)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		communityID, userID, role)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrAlreadyExists
	}
	return nil
}

func (s *PostgresStore) RemoveCommunityMember(communityID, userID string) error {
	if !isUUID(communityID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`DELETE FROM community_members WHERE community_id = $1 AND user_id = $2`,
		communityID, userID)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetCommunityMemberRole(communityID, userID, role string) error {
	if !isUUID(communityID) || !isUUID(userID) {
		return ErrNotFound
	}
	if role != "member" && role != "admin" {
		return errors.New("role must be member or admin")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`UPDATE community_members SET role = $3
		 WHERE community_id = $1 AND user_id = $2`,
		communityID, userID, role)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) AddSubgroup(communityID, chatID string) error {
	if !isUUID(communityID) || !isUUID(chatID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO community_subgroups (community_id, chat_id)
		 VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		communityID, chatID)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrAlreadyExists
	}
	return nil
}

func (s *PostgresStore) RemoveSubgroup(communityID, chatID string) error {
	if !isUUID(communityID) || !isUUID(chatID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`DELETE FROM community_subgroups WHERE community_id = $1 AND chat_id = $2`,
		communityID, chatID)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListSubgroups(communityID string) []*models.Chat {
	if !isUUID(communityID) {
		return []*models.Chat{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT c.id::text, c.type, c.title, c.owner_id::text, c.is_public,
			c.created_at, c.updated_at
		 FROM chats c JOIN community_subgroups sg ON sg.chat_id = c.id
		 WHERE sg.community_id = $1 ORDER BY c.title ASC`, communityID)
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

func (s *PostgresStore) UpdateCommunity(communityID string, name, description *string) error {
	if !isUUID(communityID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`UPDATE communities SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			updated_at = now()
		 WHERE id = $1`,
		communityID, name, description)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
