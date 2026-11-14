package push

import (
	"context"
	"errors"
	"testing"
)

// fakeFCMClient records sends and can be told to fail per token.
type fakeFCMClient struct {
	sent   []fakeFCMSend
	fail   map[string]error
}

type fakeFCMSend struct {
	token string
	title string
	body  string
	data  map[string]string
}

func (f *fakeFCMClient) Send(_ context.Context, token, title, body string, data map[string]string) error {
	f.sent = append(f.sent, fakeFCMSend{token: token, title: title, body: body, data: data})
	if err, ok := f.fail[token]; ok {
		return err
	}
	return nil
}

func TestFCMRegistryAddListRemove(t *testing.T) {
	r := NewFCMRegistry(nil, nil)
	if r.Available() {
		t.Fatal("registry without a client must report Available() == false")
	}
	r.Add("u1", FCMToken{Token: "tok-a", DeviceID: "d1", Platform: "android"})
	r.Add("u1", FCMToken{Token: "tok-b", DeviceID: "d2", Platform: "android"})
	// Re-registering the same token updates it instead of duplicating.
	r.Add("u1", FCMToken{Token: "tok-a", DeviceID: "d1", Platform: "ios"})
	if got := r.Count("u1"); got != 2 {
		t.Fatalf("Count = %d, want 2", got)
	}
	toks := r.List("u1")
	if toks[0].Token != "tok-a" || toks[0].Platform != "ios" {
		t.Fatalf("re-register did not update token: %+v", toks[0])
	}
	if !r.Remove("u1", "tok-a") {
		t.Fatal("Remove of an existing token must return true")
	}
	if got := r.Count("u1"); got != 1 {
		t.Fatalf("Count after Remove = %d, want 1", got)
	}
	if r.Remove("u1", "tok-nope") {
		t.Fatal("Remove of an unknown token must return false")
	}
}

func TestFCMRegistrySendToUser(t *testing.T) {
	client := &fakeFCMClient{fail: map[string]error{}}
	r := NewFCMRegistry(nil, client)
	r.Add("u1", FCMToken{Token: "tok-a", DeviceID: "d1", Platform: "android"})
	r.Add("u1", FCMToken{Token: "tok-b", DeviceID: "d2", Platform: "android"})
	sent, failed := r.SendToUser(context.Background(), "u1", "VYRO", "New message from Ana", map[string]string{"chat_id": "c1"})
	if sent != 2 || failed != 0 {
		t.Fatalf("SendToUser = (%d, %d), want (2, 0)", sent, failed)
	}
	if len(client.sent) != 2 || client.sent[0].data["chat_id"] != "c1" {
		t.Fatalf("sent payloads wrong: %+v", client.sent)
	}
	// A transient failure counts as failed but keeps the token.
	client.fail["tok-b"] = errors.New("boom")
	sent, failed = r.SendToUser(context.Background(), "u1", "VYRO", "New message from Ana", map[string]string{"chat_id": "c1"})
	if sent != 1 || failed != 1 {
		t.Fatalf("SendToUser with one failure = (%d, %d), want (1, 1)", sent, failed)
	}
	if got := r.Count("u1"); got != 2 {
		t.Fatalf("transient failure must not delete the token (count %d)", got)
	}
	// An unregistered token is deleted so the server stops sending to it.
	client.fail["tok-b"] = ErrTokenUnregistered
	sent, failed = r.SendToUser(context.Background(), "u1", "VYRO", "New message from Ana", map[string]string{"chat_id": "c1"})
	if sent != 1 || failed != 1 {
		t.Fatalf("SendToUser with unregistered token = (%d, %d), want (1, 1)", sent, failed)
	}
	if got := r.Count("u1"); got != 1 {
		t.Fatalf("unregistered token must be deleted (count %d)", got)
	}
	if r.List("u1")[0].Token != "tok-a" {
		t.Fatalf("wrong token survived: %+v", r.List("u1"))
	}
}

func TestFCMRegistrySendToUserDisabled(t *testing.T) {
	// No client: sends are a silent no-op, never an error.
	r := NewFCMRegistry(nil, nil)
	r.Add("u1", FCMToken{Token: "tok-a", DeviceID: "d1", Platform: "android"})
	if sent, failed := r.SendToUser(context.Background(), "u1", "VYRO", "hi", nil); sent != 0 || failed != 0 {
		t.Fatalf("disabled SendToUser = (%d, %d), want (0, 0)", sent, failed)
	}
}

func TestNewFCMClientFromEnvDisabled(t *testing.T) {
	t.Setenv("FIREBASE_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("FIREBASE_SERVICE_ACCOUNT_PATH", "")
	client, err := NewFCMClientFromEnv(context.Background())
	if err != nil {
		t.Fatalf("no env must not error: %v", err)
	}
	if client != nil {
		t.Fatal("no env must return a nil client (FCM disabled)")
	}
}

func TestNewFCMClientFromEnvBadPath(t *testing.T) {
	t.Setenv("FIREBASE_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("FIREBASE_SERVICE_ACCOUNT_PATH", "/nonexistent/nope.json")
	if _, err := NewFCMClientFromEnv(context.Background()); err == nil {
		t.Fatal("unreadable FIREBASE_SERVICE_ACCOUNT_PATH must error, not silently disable FCM")
	}
}

func TestNewFCMClientFromEnvBadJSON(t *testing.T) {
	t.Setenv("FIREBASE_SERVICE_ACCOUNT_JSON", "{not json")
	t.Setenv("FIREBASE_SERVICE_ACCOUNT_PATH", "")
	if _, err := NewFCMClientFromEnv(context.Background()); err == nil {
		t.Fatal("invalid FIREBASE_SERVICE_ACCOUNT_JSON must error, not silently disable FCM")
	}
}
