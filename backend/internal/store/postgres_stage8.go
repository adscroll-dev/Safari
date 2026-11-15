package store

import (
	"context"
	"time"
)

// Stage 8 (extras): change the account email — PostgreSQL
// implementation. Unknown user -> ErrNotFound.
func (s *PostgresStore) ChangeEmail(userID, newEmail string) error {
	if !isUUID(userID) {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET email = $2, updated_at = now() WHERE id = $1`,
		userID, newEmail)
	if err != nil {
		return mapStoreErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
