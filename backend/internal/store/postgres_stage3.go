package store

// Stage 3 — Calls, PostgresStore implementation.
// In-memory versions live in stage3.go.
// Schema: migrations/013_stage3_calls.sql.

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vyro/backend/internal/models"
)

const callRecordColumns = `id, chat_id, caller_id, callee_id, call_type,
	status, started_at, ended_at, duration_seconds`

func scanCallRecord(row pgx.Row) (*models.CallRecord, error) {
	var rec models.CallRecord
	var calleeID *string
	var endedAt *time.Time
	err := row.Scan(&rec.ID, &rec.ChatID, &rec.CallerID, &calleeID,
		&rec.CallType, &rec.Status, &rec.StartedAt, &endedAt,
		&rec.DurationSeconds)
	if err != nil {
		return nil, err
	}
	if calleeID != nil {
		rec.CalleeID = *calleeID
	}
	rec.EndedAt = endedAt
	return &rec, nil
}

// CreateCallRecord stores one finished (or missed) call.
func (s *PostgresStore) CreateCallRecord(rec *models.CallRecord) error {
	if !isUUID(rec.ChatID) || !isUUID(rec.CallerID) ||
		(rec.CalleeID != "" && !isUUID(rec.CalleeID)) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var calleeID *string
	if rec.CalleeID != "" {
		calleeID = &rec.CalleeID
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO call_records (id, chat_id, caller_id, callee_id,
		 call_type, status, started_at, ended_at, duration_seconds)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		rec.ID, rec.ChatID, rec.CallerID, calleeID, rec.CallType,
		rec.Status, rec.StartedAt, rec.EndedAt, rec.DurationSeconds)
	return mapStoreErr(err)
}

// ListCallRecordsForUser returns the calls the user took part in,
// newest first (at most 100): where they are the caller, the callee,
// or — for group calls (NULL callee) — a member of the chat.
func (s *PostgresStore) ListCallRecordsForUser(userID string) []*models.CallRecord {
	if !isUUID(userID) {
		return []*models.CallRecord{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx,
		`SELECT `+callRecordColumns+` FROM call_records r
		 JOIN chat_members cm ON cm.chat_id = r.chat_id AND cm.user_id = $1
		 WHERE r.caller_id = $1 OR r.callee_id = $1 OR r.callee_id IS NULL
		 ORDER BY r.started_at DESC, r.id DESC LIMIT 100`, userID)
	if err != nil {
		return []*models.CallRecord{}
	}
	defer rows.Close()
	out := []*models.CallRecord{}
	for rows.Next() {
		rec, err := scanCallRecord(rows)
		if err == nil {
			out = append(out, rec)
		}
	}
	return out
}
