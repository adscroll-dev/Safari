package store

// Stage 4 — Polls, PostgresStore implementation.
// In-memory versions live in stage4.go.
// Schema: migrations/014_stage4_polls.sql.
// Options are stored as a JSON array in a TEXT column.

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vyro/backend/internal/models"
)

const pollColumns = `id, chat_id, creator_id, message_id, question,
	options, allow_multiple, created_at`

func scanPoll(row pgx.Row) (*models.Poll, error) {
	var p models.Poll
	var messageID *string
	var optionsJSON string
	err := row.Scan(&p.ID, &p.ChatID, &p.CreatorID, &messageID,
		&p.Question, &optionsJSON, &p.AllowMultiple, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	if messageID != nil {
		p.MessageID = *messageID
	}
	if err := json.Unmarshal([]byte(optionsJSON), &p.Options); err != nil {
		return nil, err
	}
	return &p, nil
}

func pollOptionsJSON(options []string) string {
	b, err := json.Marshal(options)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// CreatePoll stores a new poll; the id and timestamp come from the
// database (gen_random_uuid()).
func (s *PostgresStore) CreatePoll(p *models.Poll) error {
	if !isUUID(p.ChatID) || !isUUID(p.CreatorID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.pool.QueryRow(ctx,
		`INSERT INTO polls (chat_id, creator_id, question, options, allow_multiple)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id::text, created_at`,
		p.ChatID, p.CreatorID, p.Question, pollOptionsJSON(p.Options),
		p.AllowMultiple).Scan(&p.ID, &p.CreatedAt)
	return mapStoreErr(err)
}

// PollByID returns the poll with the given id, or nil.
func (s *PostgresStore) PollByID(id string) *models.Poll {
	if !isUUID(id) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := scanPoll(s.pool.QueryRow(ctx,
		`SELECT `+pollColumns+` FROM polls WHERE id = $1`, id))
	if err != nil {
		return nil
	}
	return p
}

// ListPollsForChat returns the chat's polls, newest first.
func (s *PostgresStore) ListPollsForChat(chatID string) []*models.Poll {
	if !isUUID(chatID) {
		return []*models.Poll{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+pollColumns+` FROM polls WHERE chat_id = $1
		 ORDER BY created_at DESC, id DESC`, chatID)
	if err != nil {
		return []*models.Poll{}
	}
	defer rows.Close()
	out := []*models.Poll{}
	for rows.Next() {
		p, err := scanPoll(rows)
		if err == nil {
			out = append(out, p)
		}
	}
	return out
}

// VotePoll replaces the user's votes on a poll (re-voting is allowed;
// an empty list clears the vote).
func (s *PostgresStore) VotePoll(pollID, userID string, indexes []int) error {
	if !isUUID(pollID) || !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return mapStoreErr(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`DELETE FROM poll_votes WHERE poll_id = $1 AND user_id = $2`,
		pollID, userID); err != nil {
		return mapStoreErr(err)
	}
	for _, i := range indexes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO poll_votes (poll_id, user_id, option_index)
			 VALUES ($1, $2, $3)`, pollID, userID, i); err != nil {
			return mapStoreErr(err)
		}
	}
	return mapStoreErr(tx.Commit(ctx))
}

// PollVotes returns per-option vote counts and the total number of
// votes cast on the poll.
func (s *PostgresStore) PollVotes(pollID string) (map[int]int, int) {
	if !isUUID(pollID) {
		return map[int]int{}, 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT option_index, COUNT(*) FROM poll_votes
		 WHERE poll_id = $1 GROUP BY option_index`, pollID)
	if err != nil {
		return map[int]int{}, 0
	}
	defer rows.Close()
	counts := map[int]int{}
	total := 0
	for rows.Next() {
		var idx, n int
		if err := rows.Scan(&idx, &n); err == nil {
			counts[idx] = n
			total += n
		}
	}
	return counts, total
}

// MyPollVotes returns the option indexes the user voted for.
func (s *PostgresStore) MyPollVotes(pollID, userID string) []int {
	if !isUUID(pollID) || !isUUID(userID) {
		return []int{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT option_index FROM poll_votes
		 WHERE poll_id = $1 AND user_id = $2 ORDER BY option_index`,
		pollID, userID)
	if err != nil {
		return []int{}
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var idx int
		if err := rows.Scan(&idx); err == nil {
			out = append(out, idx)
		}
	}
	return out
}

// DeletePoll removes a poll and its votes (ON DELETE CASCADE).
func (s *PostgresStore) DeletePoll(id string) error {
	if !isUUID(id) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx, `DELETE FROM polls WHERE id = $1`, id)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPollMessageID links a poll to the message that renders it in
// the chat flow.
func (s *PostgresStore) SetPollMessageID(pollID, messageID string) error {
	if !isUUID(pollID) || !isUUID(messageID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd, err := s.pool.Exec(ctx,
		`UPDATE polls SET message_id = $1 WHERE id = $2`, messageID, pollID)
	if err != nil {
		return mapStoreErr(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
