package store

// Stage 2 — Status (Updates) gaps, PostgresStore implementation.
// In-memory versions live in stage2.go.
// Schema: migrations/012_stage2_status.sql.

import (
	"context"
	"time"

	"github.com/vyro/backend/internal/models"
)

// StoryByID returns the story with the given id, or nil. Expired
// stories are returned too — callers check ExpiresAt.
func (s *PostgresStore) StoryByID(id string) *models.Story {
	if !isUUID(id) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := scanStory(s.pool.QueryRow(ctx,
		`SELECT `+storyColumns+` FROM stories WHERE id = $1`, id))
	if err != nil {
		return nil
	}
	return st
}

// DeleteStory removes a story; views, reactions and replies go with
// it via ON DELETE CASCADE.
func (s *PostgresStore) DeleteStory(id string) error {
	if !isUUID(id) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx, `DELETE FROM stories WHERE id = $1`, id)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordStoryView records a view; a second view by the same user is
// a no-op (the first viewed_at is kept).
func (s *PostgresStore) RecordStoryView(storyID, userID string) error {
	if !isUUID(storyID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO story_views (story_id, user_id) VALUES ($1, $2)
		 ON CONFLICT (story_id, user_id) DO NOTHING`, storyID, userID)
	return mapStoreErr(err)
}

// ListStoryViews returns who viewed a story, oldest view first.
func (s *PostgresStore) ListStoryViews(storyID string) []*models.StoryView {
	if !isUUID(storyID) {
		return []*models.StoryView{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT v.user_id::text, u.display_name, u.username, v.viewed_at
		 FROM story_views v JOIN users u ON u.id = v.user_id
		 WHERE v.story_id = $1 ORDER BY v.viewed_at ASC`, storyID)
	if err != nil {
		return []*models.StoryView{}
	}
	defer rows.Close()
	out := []*models.StoryView{}
	for rows.Next() {
		var v models.StoryView
		if err := rows.Scan(&v.UserID, &v.DisplayName, &v.Username, &v.ViewedAt); err == nil {
			out = append(out, &v)
		}
	}
	return out
}

// StoryViewCount returns how many distinct users viewed a story.
func (s *PostgresStore) StoryViewCount(storyID string) int {
	if !isUUID(storyID) {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM story_views WHERE story_id = $1`, storyID).Scan(&n); err != nil {
		return 0
	}
	return n
}

// SetStoryReaction records (or replaces) one user's emoji reaction.
func (s *PostgresStore) SetStoryReaction(storyID, userID, emoji string) error {
	if !isUUID(storyID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO story_reactions (story_id, user_id, emoji) VALUES ($1, $2, $3)
		 ON CONFLICT (story_id, user_id) DO UPDATE SET emoji = EXCLUDED.emoji, created_at = now()`,
		storyID, userID, emoji)
	return mapStoreErr(err)
}

// DeleteStoryReaction removes one user's reaction, if any.
func (s *PostgresStore) DeleteStoryReaction(storyID, userID string) error {
	if !isUUID(storyID) || !isUUID(userID) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx,
		`DELETE FROM story_reactions WHERE story_id = $1 AND user_id = $2`, storyID, userID)
	return mapStoreErr(err)
}

// ListStoryReactions returns a story's reactions, oldest first.
func (s *PostgresStore) ListStoryReactions(storyID string) []*models.StoryReaction {
	if !isUUID(storyID) {
		return []*models.StoryReaction{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT r.user_id::text, u.display_name, u.username, r.emoji, r.created_at
		 FROM story_reactions r JOIN users u ON u.id = r.user_id
		 WHERE r.story_id = $1 ORDER BY r.created_at ASC`, storyID)
	if err != nil {
		return []*models.StoryReaction{}
	}
	defer rows.Close()
	out := []*models.StoryReaction{}
	for rows.Next() {
		var r models.StoryReaction
		if err := rows.Scan(&r.UserID, &r.DisplayName, &r.Username, &r.Emoji, &r.CreatedAt); err == nil {
			out = append(out, &r)
		}
	}
	return out
}

// MyStoryReaction returns the requester's own emoji on a story, or "".
func (s *PostgresStore) MyStoryReaction(storyID, userID string) string {
	if !isUUID(storyID) || !isUUID(userID) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var emoji string
	if err := s.pool.QueryRow(ctx,
		`SELECT emoji FROM story_reactions WHERE story_id = $1 AND user_id = $2`,
		storyID, userID).Scan(&emoji); err != nil {
		return ""
	}
	return emoji
}

// AddStoryReply stores a text reply to a story.
func (s *PostgresStore) AddStoryReply(storyID, userID, content string) (*models.StoryReply, error) {
	if !isUUID(storyID) || !isUUID(userID) {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r := &models.StoryReply{
		ID: NewID(), StoryID: storyID, UserID: userID, Content: content,
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO story_replies (id, story_id, user_id, content)
		 VALUES ($1, $2, $3, $4) RETURNING created_at`,
		r.ID, r.StoryID, r.UserID, r.Content).Scan(&r.CreatedAt)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if u, err := s.UserByID(userID); err == nil {
		r.DisplayName = u.DisplayName
		r.Username = u.Username
	}
	return r, nil
}

// ListStoryReplies returns a story's replies, oldest first.
func (s *PostgresStore) ListStoryReplies(storyID string) []*models.StoryReply {
	if !isUUID(storyID) {
		return []*models.StoryReply{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT r.id::text, r.story_id::text, r.user_id::text, u.display_name, u.username,
		        r.content, r.created_at
		 FROM story_replies r JOIN users u ON u.id = r.user_id
		 WHERE r.story_id = $1 ORDER BY r.created_at ASC`, storyID)
	if err != nil {
		return []*models.StoryReply{}
	}
	defer rows.Close()
	out := []*models.StoryReply{}
	for rows.Next() {
		var r models.StoryReply
		if err := rows.Scan(&r.ID, &r.StoryID, &r.UserID, &r.DisplayName, &r.Username, &r.Content, &r.CreatedAt); err == nil {
			out = append(out, &r)
		}
	}
	return out
}

// GetStatusPrivacy returns who may see userID's status
// ("everyone" by default).
func (s *PostgresStore) GetStatusPrivacy(userID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT status_privacy FROM user_settings WHERE user_id = $1`, userID).Scan(&v)
	if err != nil {
		return StatusEveryone
	}
	return v
}

// SetStatusPrivacy stores a user's status-privacy choice.
func (s *PostgresStore) SetStatusPrivacy(userID, privacy string) error {
	if !ValidStatusPrivacy(privacy) {
		return ErrForbidden
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, status_privacy)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET status_privacy = EXCLUDED.status_privacy`,
		userID, privacy)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
