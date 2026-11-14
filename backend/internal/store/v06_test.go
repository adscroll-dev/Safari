package store

import (
	"context"
	"os"
	"testing"

	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/push"
)

func mustUserV06(t *testing.T, s *MemoryStore, email, username string) *models.User {
	t.Helper()
	u := &models.User{Email: email, Username: username, DisplayName: username, PasswordHash: "hash"}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser %s: %v", username, err)
	}
	return u
}

func TestMemoryGroupSenderKeys(t *testing.T) {
	s := NewMemoryStore()
	alice := mustUserV06(t, s, "a@example.com", "alice")
	bob := mustUserV06(t, s, "b@example.com", "bob")
	carol := mustUserV06(t, s, "c@example.com", "carol")
	chat := &models.Chat{Type: models.ChatGroup, Title: "g", OwnerID: alice.ID,
		MemberIDs: []string{alice.ID, bob.ID, carol.ID}}
	if err := s.CreateChat(chat); err != nil {
		t.Fatal(err)
	}
	// Unknown chat / unknown recipient are rejected.
	if err := s.PutGroupSenderKey(&models.GroupSenderKey{ChatID: "nope", SenderUserID: alice.ID,
		ForUserID: bob.ID, Ciphertext: "{}"}); err != ErrNotFound {
		t.Fatalf("unknown chat: got %v, want ErrNotFound", err)
	}
	dist := &models.GroupSenderKey{
		ChatID: chat.ID, SenderUserID: alice.ID, SenderDeviceID: "default",
		KeyID: "k1", ForUserID: bob.ID, ForDeviceID: "default",
		Ciphertext: `{"v":1,"ct":"opaque"}`,
	}
	if err := s.PutGroupSenderKey(dist); err != nil {
		t.Fatalf("PutGroupSenderKey: %v", err)
	}
	if err := s.PutGroupSenderKey(&models.GroupSenderKey{
		ChatID: chat.ID, SenderUserID: alice.ID, SenderDeviceID: "default",
		KeyID: "k1", ForUserID: carol.ID, ForDeviceID: "phone",
		Ciphertext: `{"v":1,"ct":"opaque2"}`,
	}); err != nil {
		t.Fatalf("PutGroupSenderKey carol: %v", err)
	}
	// Bob's device sees only his envelope; Carol's phone sees only hers.
	got := s.GroupSenderKeysFor(chat.ID, bob.ID, "default")
	if len(got) != 1 || got[0].Ciphertext != `{"v":1,"ct":"opaque"}` {
		t.Fatalf("bob's keys: %+v", got)
	}
	if got := s.GroupSenderKeysFor(chat.ID, carol.ID, "phone"); len(got) != 1 {
		t.Fatalf("carol's phone keys: %d", len(got))
	}
	if got := s.GroupSenderKeysFor(chat.ID, carol.ID, "default"); len(got) != 0 {
		t.Fatalf("carol's default device should have no keys, got %d", len(got))
	}
	// Re-upload replaces the envelope (rotation / re-distribution).
	dist.Ciphertext = `{"v":1,"ct":"rotated"}`
	if err := s.PutGroupSenderKey(dist); err != nil {
		t.Fatal(err)
	}
	if got := s.GroupSenderKeysFor(chat.ID, bob.ID, "default"); len(got) != 1 || got[0].Ciphertext != `{"v":1,"ct":"rotated"}` {
		t.Fatalf("re-upload did not replace: %+v", got)
	}
	// Removing a member deletes only their distributions.
	if err := s.DeleteGroupSenderKeysForUser(chat.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if got := s.GroupSenderKeysFor(chat.ID, bob.ID, "default"); len(got) != 0 {
		t.Fatalf("bob's keys should be gone, got %d", len(got))
	}
	if got := s.GroupSenderKeysFor(chat.ID, carol.ID, "phone"); len(got) != 1 {
		t.Fatalf("carol's keys should survive, got %d", len(got))
	}
}

func TestMemoryDevices(t *testing.T) {
	s := NewMemoryStore()
	alice := mustUserV06(t, s, "a@example.com", "alice")
	if err := s.RegisterDevice(&models.Device{UserID: alice.ID, DeviceID: "default", Platform: "web"}); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterDevice(&models.Device{UserID: alice.ID, DeviceID: "phone-2", Name: "Phone", Platform: "android"}); err != nil {
		t.Fatal(err)
	}
	devs := s.ListDevices(alice.ID)
	if len(devs) != 2 || devs[0].DeviceID != "default" || devs[1].DeviceID != "phone-2" {
		t.Fatalf("devices: %+v", devs)
	}
	// Re-registering updates name/platform instead of duplicating.
	if err := s.RegisterDevice(&models.Device{UserID: alice.ID, DeviceID: "phone-2", Name: "Pixel", Platform: "android"}); err != nil {
		t.Fatal(err)
	}
	if devs := s.ListDevices(alice.ID); len(devs) != 2 || devs[1].Name != "Pixel" {
		t.Fatalf("re-register: %+v", devs)
	}
	if err := s.TouchDevice(alice.ID, "phone-2"); err != nil {
		t.Fatalf("TouchDevice: %v", err)
	}
	if err := s.TouchDevice(alice.ID, "ghost"); err != ErrNotFound {
		t.Fatalf("touch unknown device: got %v, want ErrNotFound", err)
	}
	if err := s.RegisterDevice(&models.Device{UserID: "nobody", DeviceID: "x"}); err != ErrNotFound {
		t.Fatalf("register for unknown user: got %v, want ErrNotFound", err)
	}
	if err := s.RemoveDevice(alice.ID, "phone-2"); err != nil {
		t.Fatal(err)
	}
	if devs := s.ListDevices(alice.ID); len(devs) != 1 {
		t.Fatalf("after remove: %d devices", len(devs))
	}
	if err := s.RemoveDevice(alice.ID, "phone-2"); err != ErrNotFound {
		t.Fatalf("double remove: got %v, want ErrNotFound", err)
	}
}

func TestMemoryPreKeyForDevice(t *testing.T) {
	s := NewMemoryStore()
	alice := mustUserV06(t, s, "a@example.com", "alice")
	keys := []*models.OneTimePreKey{
		{KeyID: 1, PublicKey: "phone-key-1"},
		{KeyID: 2, PublicKey: "phone-key-2"},
		{KeyID: 1, PublicKey: "laptop-key-1"},
	}
	// Upload phone keys under "phone", laptop key under "laptop".
	if err := s.UploadOneTimePreKeys(alice.ID, "phone", keys[:2]); err != nil {
		t.Fatal(err)
	}
	if err := s.UploadOneTimePreKeys(alice.ID, "laptop", keys[2:]); err != nil {
		t.Fatal(err)
	}
	k, err := s.FetchOneTimePreKeyForDevice(alice.ID, "laptop")
	if err != nil || k.PublicKey != "laptop-key-1" || k.DeviceID != "laptop" {
		t.Fatalf("laptop prekey: %+v %v", k, err)
	}
	// The laptop key is now used; the next laptop claim finds nothing,
	// while the phone still has keys.
	if _, err := s.FetchOneTimePreKeyForDevice(alice.ID, "laptop"); err != ErrNotFound {
		t.Fatalf("exhausted laptop keys: got %v, want ErrNotFound", err)
	}
	k, err = s.FetchOneTimePreKeyForDevice(alice.ID, "phone")
	if err != nil || k.PublicKey != "phone-key-1" {
		t.Fatalf("phone prekey: %+v %v", k, err)
	}
}

func TestMemoryPushSubscriptions(t *testing.T) {
	s := NewMemoryStore()
	alice := mustUserV06(t, s, "a@example.com", "alice")
	sub := push.Subscription{Endpoint: "https://push.example.invalid/1", P256dh: "p", Auth: "a"}
	if err := s.SavePushSubscription(alice.ID, sub); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePushSubscription(alice.ID, sub); err != nil {
		t.Fatal(err)
	}
	if got := s.ListPushSubscriptions(alice.ID); len(got) != 1 || got[0].Endpoint != sub.Endpoint {
		t.Fatalf("subs: %+v", got)
	}
	if err := s.SavePushSubscription(alice.ID, push.Subscription{Endpoint: "https://push.example.invalid/2", P256dh: "p", Auth: "a"}); err != nil {
		t.Fatal(err)
	}
	if got := s.ListPushSubscriptions(alice.ID); len(got) != 2 {
		t.Fatalf("expected 2 subs, got %d", len(got))
	}
	if err := s.DeletePushSubscription(alice.ID, sub.Endpoint); err != nil {
		t.Fatal(err)
	}
	if got := s.ListPushSubscriptions(alice.ID); len(got) != 1 {
		t.Fatalf("after delete: %d subs", len(got))
	}
	if err := s.SavePushSubscription("nobody", sub); err != ErrNotFound {
		t.Fatalf("save for unknown user: got %v, want ErrNotFound", err)
	}
}

// TestPostgresV06 is an integration test for the v0.6 store methods. It
// runs only when VYRO_TEST_DATABASE_URL is set (same rule as
// TestPostgresStore) and proves the new tables persist across a store
// reopen (a restart).
func TestPostgresV06(t *testing.T) {
	dsn := os.Getenv("VYRO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VYRO_TEST_DATABASE_URL not set; skipping Postgres v0.6 integration test")
	}
	ctx := context.Background()
	s, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer s.Close()
	truncate := func() {
		_, _ = s.pool.Exec(ctx, `TRUNCATE users, chats, messages, reactions, blocks CASCADE`)
	}
	truncate()
	defer truncate()

	a := &models.User{Email: "v06-a@example.com", Username: "v06alice", DisplayName: "Alice", PasswordHash: "hash"}
	if err := s.CreateUser(a); err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	b := &models.User{Email: "v06-b@example.com", Username: "v06bob", DisplayName: "Bob", PasswordHash: "hash"}
	if err := s.CreateUser(b); err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}
	chat := &models.Chat{Type: models.ChatGroup, Title: "g", OwnerID: a.ID, MemberIDs: []string{a.ID, b.ID}}
	if err := s.CreateChat(chat); err != nil {
		t.Fatalf("CreateChat: %v", err)
	}

	// Group sender keys.
	dist := &models.GroupSenderKey{
		ChatID: chat.ID, SenderUserID: a.ID, SenderDeviceID: "default",
		KeyID: "k1", ForUserID: b.ID, ForDeviceID: "phone",
		Ciphertext: `{"v":1,"ct":"opaque"}`,
	}
	if err := s.PutGroupSenderKey(dist); err != nil {
		t.Fatalf("PutGroupSenderKey: %v", err)
	}
	if got := s.GroupSenderKeysFor(chat.ID, b.ID, "phone"); len(got) != 1 || got[0].Ciphertext != `{"v":1,"ct":"opaque"}` {
		t.Fatalf("GroupSenderKeysFor: %+v", got)
	}

	// Devices.
	if err := s.RegisterDevice(&models.Device{UserID: b.ID, DeviceID: "phone", Name: "Pixel", Platform: "android"}); err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}
	if err := s.RegisterDevice(&models.Device{UserID: b.ID, DeviceID: "laptop", Platform: "desktop"}); err != nil {
		t.Fatalf("RegisterDevice laptop: %v", err)
	}
	if devs := s.ListDevices(b.ID); len(devs) != 2 {
		t.Fatalf("ListDevices: %d", len(devs))
	}
	if err := s.TouchDevice(b.ID, "phone"); err != nil {
		t.Fatalf("TouchDevice: %v", err)
	}

	// Per-device pre-keys: the phone's key must not be claimed for the laptop.
	if err := s.UploadOneTimePreKeys(b.ID, "phone", []*models.OneTimePreKey{{KeyID: 1, PublicKey: "phone-pk"}}); err != nil {
		t.Fatalf("UploadOneTimePreKeys: %v", err)
	}
	if _, err := s.FetchOneTimePreKeyForDevice(b.ID, "laptop"); err != ErrNotFound {
		t.Fatalf("laptop claim: got %v, want ErrNotFound", err)
	}
	k, err := s.FetchOneTimePreKeyForDevice(b.ID, "phone")
	if err != nil || k.PublicKey != "phone-pk" {
		t.Fatalf("phone claim: %+v %v", k, err)
	}

	// Push subscriptions.
	sub := push.Subscription{Endpoint: "https://push.example.invalid/v06", P256dh: "p", Auth: "a"}
	if err := s.SavePushSubscription(b.ID, sub); err != nil {
		t.Fatalf("SavePushSubscription: %v", err)
	}

	// Reopen: everything must still be there (restart persistence).
	s2, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("reopen PostgresStore: %v", err)
	}
	defer s2.Close()
	if got := s2.GroupSenderKeysFor(chat.ID, b.ID, "phone"); len(got) != 1 {
		t.Fatalf("sender key lost after reopen: %d", len(got))
	}
	if devs := s2.ListDevices(b.ID); len(devs) != 2 {
		t.Fatalf("devices lost after reopen: %d", len(devs))
	}
	if got := s2.ListPushSubscriptions(b.ID); len(got) != 1 || got[0].Endpoint != sub.Endpoint {
		t.Fatalf("push subscription lost after reopen: %+v", got)
	}
	// Removal still works after reopen.
	if err := s2.DeleteGroupSenderKeysForUser(chat.ID, b.ID); err != nil {
		t.Fatalf("DeleteGroupSenderKeysForUser: %v", err)
	}
	if got := s2.GroupSenderKeysFor(chat.ID, b.ID, "phone"); len(got) != 0 {
		t.Fatalf("sender keys should be gone: %d", len(got))
	}
	if err := s2.DeletePushSubscription(b.ID, sub.Endpoint); err != nil {
		t.Fatalf("DeletePushSubscription: %v", err)
	}
	if err := s2.RemoveDevice(b.ID, "phone"); err != nil {
		t.Fatalf("RemoveDevice: %v", err)
	}
	if devs := s2.ListDevices(b.ID); len(devs) != 1 {
		t.Fatalf("after remove: %d devices", len(devs))
	}
}
