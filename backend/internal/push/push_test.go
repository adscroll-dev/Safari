package push

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSendDecryptsEndToEnd proves the full Web Push crypto path: the
// payload is encrypted to the subscription keys, the VAPID headers are
// present, and a holder of the subscription private key can decrypt
// the exact payload. The "push service" here is a local HTTP server —
// delivery through a real browser push service (FCM) is NOT tested.
func TestSendDecryptsEndToEnd(t *testing.T) {
	// Subscription keypair (P-256 ECDH) + auth secret, as a browser
	// would generate them.
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey().Bytes() // uncompressed 65 bytes
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	b64url := base64.RawURLEncoding.EncodeToString

	var gotHeaders http.Header
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	vapPub, vapPriv, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	sub := Subscription{Endpoint: srv.URL, P256dh: b64url(pub), Auth: b64url(auth)}
	if _, err := Send(sub, []byte("hello push"), vapPub, vapPriv, "mailto:vyro@test.invalid"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotHeaders.Get("Authorization") == "" {
		t.Fatal("no VAPID Authorization header sent")
	}
	if !strings.HasPrefix(gotHeaders.Get("Authorization"), "vapid ") && !strings.HasPrefix(gotHeaders.Get("Authorization"), "WebPush ") {
		t.Fatalf("unexpected Authorization header: %s", gotHeaders.Get("Authorization"))
	}
	if gotHeaders.Get("Content-Encoding") != "aes128gcm" {
		t.Fatalf("Content-Encoding = %q, want aes128gcm", gotHeaders.Get("Content-Encoding"))
	}
	pt := decryptAES128GCM(t, gotBody, priv, pub, auth)
	if string(pt) != "hello push" {
		t.Fatalf("decrypted payload = %q, want %q", pt, "hello push")
	}
}

// decryptAES128GCM implements the aes128gcm decryption for a captured
// push request body, using the subscription private key and auth secret.
// It follows the scheme webpush-go (and browsers) actually use:
//
//	IKM   = HKDF(salt=auth, ECDH, info="WebPush: info" || 0x00 || subPub || senderPub)
//	CEK   = HKDF(salt=bodySalt, IKM, info="Content-Encoding: aes128gcm" || 0x00)
//	NONCE = HKDF(salt=bodySalt, IKM, info="Content-Encoding: nonce" || 0x00)
//
// The body is: salt (16) || record size (4) || sender key len (1) ||
// sender key (65) || AES-128-GCM ciphertext. The plaintext is the
// message + 0x02 delimiter + zero padding.
func decryptAES128GCM(t *testing.T, body []byte, subPriv *ecdh.PrivateKey, subPub, auth []byte) []byte {
	t.Helper()
	if len(body) < 86 {
		t.Fatalf("push body too short: %d bytes", len(body))
	}
	salt := body[0:16]
	if body[20] != 65 {
		t.Fatalf("sender public key length = %d, want 65", body[20])
	}
	senderPub := body[21:86]
	ct := body[86:]
	senderKey, err := ecdh.P256().NewPublicKey(senderPub)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := subPriv.ECDH(senderKey)
	if err != nil {
		t.Fatal(err)
	}
	var ikmInfo bytes.Buffer
	ikmInfo.WriteString("WebPush: info\x00")
	ikmInfo.Write(subPub)
	ikmInfo.Write(senderPub)
	ikm := hkdfFull(auth, secret, ikmInfo.Bytes(), 32)
	cek := hkdfFull(salt, ikm, []byte("Content-Encoding: aes128gcm\x00"), 16)
	nonce := hkdfFull(salt, ikm, []byte("Content-Encoding: nonce\x00"), 12)
	block, err := aes.NewCipher(cek)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		t.Fatalf("AES-GCM open failed: %v", err)
	}
	// Strip zero padding, then the 0x02 delimiter.
	raw = bytes.TrimRight(raw, "\x00")
	if len(raw) == 0 || raw[len(raw)-1] != 0x02 {
		t.Fatalf("decrypted record has no 0x02 delimiter: %q", raw)
	}
	return raw[:len(raw)-1]
}

// TestSendToAllPrunesGoneSubscriptions: a push service that answers
// 410 Gone means the subscription is dead; SendToAll must remove it
// so later sends do not keep failing on it. A 201 response keeps it.
func TestSendToAllPrunesGoneSubscriptions(t *testing.T) {
	vapPub, vapPriv, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	b64url := base64.RawURLEncoding.EncodeToString
	status := http.StatusCreated
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	defer srv.Close()
	reg := NewRegistry()
	sub := Subscription{Endpoint: srv.URL, P256dh: b64url(priv.PublicKey().Bytes()), Auth: b64url(auth)}
	reg.Add("u1", sub)
	if sent, failed := reg.SendToAll("u1", []byte("hi"), vapPub, vapPriv, "mailto:vyro@test.invalid"); sent != 1 || failed != 0 {
		t.Fatalf("201: sent=%d failed=%d, want 1/0", sent, failed)
	}
	if reg.Count("u1") != 1 {
		t.Fatal("subscription should be kept after 201")
	}
	status = http.StatusGone
	if sent, failed := reg.SendToAll("u1", []byte("hi"), vapPub, vapPriv, "mailto:vyro@test.invalid"); sent != 0 || failed != 1 {
		t.Fatalf("410: sent=%d failed=%d, want 0/1", sent, failed)
	}
	if reg.Count("u1") != 0 {
		t.Fatal("subscription should be removed after 410 Gone")
	}
}
// TestRegistrySurvivesRestart: a registry backed by a store keeps its
// subscriptions when the registry itself is replaced (what a server
// restart does). The fake store stands in for the real MemoryStore /
// PostgresStore implementations, which are tested in internal/store.
func TestRegistrySurvivesRestart(t *testing.T) {
	st := &fakePushStore{subs: map[string][]Subscription{}}
	reg := NewRegistryWithStore(st)
	sub := Subscription{Endpoint: "https://push.example.invalid/abc", P256dh: "p256", Auth: "auth"}
	reg.Add("u1", sub)
	reg.Add("u1", sub) // idempotent
	if reg.Count("u1") != 1 {
		t.Fatalf("count = %d, want 1", reg.Count("u1"))
	}
	// "Restart": a brand-new registry over the same store.
	reg2 := NewRegistryWithStore(st)
	if got := reg2.List("u1"); len(got) != 1 || got[0].Endpoint != sub.Endpoint {
		t.Fatalf("after restart: %v", got)
	}
	if !reg2.Remove("u1", sub.Endpoint) {
		t.Fatal("remove should succeed")
	}
	if reg2.Count("u1") != 0 {
		t.Fatal("subscription should be gone after remove")
	}
	// A registry with no store still works in memory (v0.5 behaviour).
	mem := NewRegistry()
	mem.Add("u1", sub)
	if mem.Count("u1") != 1 {
		t.Fatal("in-memory registry lost the subscription")
	}
}

type fakePushStore struct {
	subs map[string][]Subscription
}

func (f *fakePushStore) SavePushSubscription(userID string, sub Subscription) error {
	for _, s := range f.subs[userID] {
		if s.Endpoint == sub.Endpoint {
			return nil
		}
	}
	f.subs[userID] = append(f.subs[userID], sub)
	return nil
}

func (f *fakePushStore) ListPushSubscriptions(userID string) []Subscription {
	out := make([]Subscription, len(f.subs[userID]))
	copy(out, f.subs[userID])
	return out
}

func (f *fakePushStore) DeletePushSubscription(userID, endpoint string) error {
	out := f.subs[userID][:0]
	for _, s := range f.subs[userID] {
		if s.Endpoint != endpoint {
			out = append(out, s)
		}
	}
	f.subs[userID] = out
	return nil
}

func hkdfFull(salt, ikm, info []byte, length int) []byte {
	if len(salt) == 0 {
		salt = make([]byte, sha256.Size)
	}
	h := hmac.New(sha256.New, salt)
	h.Write(ikm)
	prk := h.Sum(nil)
	h2 := hmac.New(sha256.New, prk)
	h2.Write(info)
	h2.Write([]byte{0x01})
	return h2.Sum(nil)[:length]
}
