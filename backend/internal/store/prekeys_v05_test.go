package store

import (
	"context"
	"os"
	"testing"

	"github.com/vyro/backend/internal/models"
)

// TestPreKeyRepublish: re-uploading a one-time pre-key id replaces the
// old public key instead of silently keeping it (v0.5 fix). A device
// that reinstalls and re-publishes key_id N must never have the old,
// now undecryptable public key handed out to peers.
func TestPreKeyRepublish(t *testing.T) {
	stores := map[string]StoreIface{"memory": NewMemoryStore()}
	if dsn := os.Getenv("VYRO_TEST_DATABASE_URL"); dsn != "" {
		pg, err := NewPostgresStore(context.Background(), dsn)
		if err != nil {
			t.Fatalf("NewPostgresStore: %v", err)
		}
		defer pg.Close()
		stores["postgres"] = pg
	}
	for name, s := range stores {
		u := &models.User{Email: name + "-repub@example.com", Username: name + "-repub", DisplayName: "Repub", PasswordHash: "hash"}
		if err := s.CreateUser(u); err != nil {
			t.Fatalf("[%s] CreateUser: %v", name, err)
		}
		old := &models.OneTimePreKey{KeyID: 1, PublicKey: "b2xkLXB1Yg=="} // "old-pub"
		if err := s.UploadOneTimePreKeys(u.ID, "default", []*models.OneTimePreKey{old}); err != nil {
			t.Fatalf("[%s] upload old: %v", name, err)
		}
		// The device reinstalls and publishes a different key under
		// the same id; the old public key must be gone.
		fresh := &models.OneTimePreKey{KeyID: 1, PublicKey: "ZnJlc2gtcHVi"} // "fresh-pub"
		if err := s.UploadOneTimePreKeys(u.ID, "default", []*models.OneTimePreKey{fresh}); err != nil {
			t.Fatalf("[%s] upload fresh: %v", name, err)
		}
		got, err := s.FetchOneTimePreKey(u.ID)
		if err != nil {
			t.Fatalf("[%s] fetch: %v", name, err)
		}
		if got.KeyID != 1 || got.PublicKey != "ZnJlc2gtcHVi" {
			t.Fatalf("[%s] fetched key = id %d pub %q, want id 1 pub fresh-pub", name, got.KeyID, got.PublicKey)
		}
		// A re-published key that was already claimed becomes
		// claimable again (it is a new key, not the used one).
		if err := s.UploadOneTimePreKeys(u.ID, "default", []*models.OneTimePreKey{{KeyID: 1, PublicKey: "dGhpcmQtcHVi"}}); err != nil {
			t.Fatalf("[%s] upload third: %v", name, err)
		}
		got, err = s.FetchOneTimePreKey(u.ID)
		if err != nil || got.PublicKey != "dGhpcmQtcHVi" {
			t.Fatalf("[%s] re-claim after republish: got %+v, err %v", name, got, err)
		}
	}
}
