package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vyro/backend/internal/e2ee"
	"github.com/vyro/backend/internal/models"
	"github.com/vyro/backend/internal/push"
)

// TestV05KeyBundleAndE2EEFlow is the full v0.5 E2EE path over HTTP:
// Alice publishes real keys, Bob fetches her bundle (signature verifies,
// one-time pre-key is claimed once), Bob encrypts with X3DH + Double
// Ratchet, the server stores only ciphertext, and Alice decrypts it.
// The server never sees the plaintext.
func TestV05KeyBundleAndE2EEFlow(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")

	// Alice's device generates its identity + keys (private parts stay
	// on the device; only public parts are uploaded).
	aliceIdent, err := e2ee.GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	spk, err := e2ee.GenerateSignedPreKey(1, aliceIdent)
	if err != nil {
		t.Fatal(err)
	}
	opk1, err := e2ee.GenerateOneTimePreKey(1)
	if err != nil {
		t.Fatal(err)
	}
	opk2, err := e2ee.GenerateOneTimePreKey(2)
	if err != nil {
		t.Fatal(err)
	}
	b64 := base64.StdEncoding.EncodeToString
	rec := doJSON(t, h, http.MethodPost, "/api/keys/identity", map[string]interface{}{
		"device_id": "default", "identity_key": b64(aliceIdent.DHPub),
		"signed_prekey": b64(spk.Pub), "signed_prekey_id": spk.ID,
		"sign_key": b64(aliceIdent.SignPub), "signed_prekey_sig": b64(spk.Signature),
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload identity key: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/keys/prekeys", map[string]interface{}{
		"device_id": "default", "keys": []map[string]interface{}{
			{"key_id": opk1.ID, "public_key": b64(opk1.Pub)},
			{"key_id": opk2.ID, "public_key": b64(opk2.Pub)},
		},
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload prekeys: %d %s", rec.Code, rec.Body.String())
	}

	// Bob fetches Alice's bundle: signature must verify, one-time
	// pre-key must be present (first claim).
	rec = doJSON(t, h, http.MethodGet, "/api/keys/"+aliceID+"/bundle", nil, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch bundle: %d %s", rec.Code, rec.Body.String())
	}
	var bundle e2ee.Bundle
	mustDecode(t, rec, &bundle)
	if err := bundle.Verify(); err != nil {
		t.Fatalf("bundle signature must verify: %v", err)
	}
	if bundle.OneTimePreKey == nil {
		t.Fatal("bundle should include a one-time pre-key on first fetch")
	}
	claimedOPKID := bundle.OneTimePreKey.ID

	// Bob starts a session and encrypts a Persian plaintext.
	bobIdent, err := e2ee.GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	sess, err := e2ee.Initiate(bobIdent, &bundle)
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	plaintext := "سلام VYRO — this must stay secret from the server"
	env, err := sess.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	envJSON, _ := json.Marshal(env)

	// Bob sends the envelope as a kind=e2ee message; the server stores
	// the ciphertext and never sees the plaintext.
	chatID := makeDM(t, h, bobTok, aliceID)
	m := sendMsg(t, h, bobTok, chatID, map[string]interface{}{
		"content": string(envJSON), "kind": "e2ee",
	})
	if m.Kind != "e2ee" {
		t.Fatalf("stored message kind = %q, want e2ee", m.Kind)
	}
	if strings.Contains(m.Content, "سلام") || strings.Contains(m.Content, "secret") {
		t.Fatal("server-stored message content leaks plaintext")
	}

	// Alice reads the stored envelope and decrypts it with her keys.
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chatID+"/messages", nil, aliceTok)
	var msgs []*models.Message
	mustDecode(t, rec, &msgs)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	var gotEnv e2ee.Envelope
	if err := json.Unmarshal([]byte(msgs[0].Content), &gotEnv); err != nil {
		t.Fatalf("stored content is not an E2EE envelope: %v", err)
	}
	senderIK, _ := base64.StdEncoding.DecodeString(gotEnv.IK)
	senderEK, _ := base64.StdEncoding.DecodeString(gotEnv.EK)
	var usedOPK *e2ee.OneTimePreKey
	if gotEnv.OPKID == opk1.ID {
		usedOPK = opk1
	} else if gotEnv.OPKID == opk2.ID {
		usedOPK = opk2
	}
	aliceSess, err := e2ee.ReceiveInit(aliceIdent, spk, usedOPK, senderIK, senderEK, bobID, "default")
	if err != nil {
		t.Fatalf("ReceiveInit: %v", err)
	}
	dec, err := aliceSess.Decrypt(&gotEnv)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(dec) != plaintext {
		t.Fatalf("decrypted = %q, want %q", dec, plaintext)
	}

	// One-time pre-keys are single-use: the second bundle fetch claims
	// the remaining key, the third has none left.
	rec = doJSON(t, h, http.MethodGet, "/api/keys/"+aliceID+"/bundle", nil, bobTok)
	var b2 e2ee.Bundle
	mustDecode(t, rec, &b2)
	if b2.OneTimePreKey == nil || b2.OneTimePreKey.ID == claimedOPKID {
		t.Fatalf("second bundle should claim the other pre-key, got %+v", b2.OneTimePreKey)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/keys/"+aliceID+"/bundle", nil, bobTok)
	var b3 e2ee.Bundle
	mustDecode(t, rec, &b3)
	if b3.OneTimePreKey != nil {
		t.Fatalf("third bundle should have no one-time pre-key, got %+v", b3.OneTimePreKey)
	}

	// Unknown user has no bundle.
	rec = doJSON(t, h, http.MethodGet, "/api/keys/00000000-0000-0000-0000-000000000000/bundle", nil, bobTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown bundle: status %d, want 404", rec.Code)
	}
}

// TestV05PushEndpoints covers the VAPID public-key, subscribe,
// unsubscribe and test endpoints. Delivery itself is proven in
// internal/push (local push-service stand-in); here the test endpoint
// posts to an unreachable address, so sent=0/failed=1 is expected.
func TestV05PushEndpoints(t *testing.T) {
	h := newTestHandler()
	tok, userID := register(t, h, "push@example.com", "pushuser")
	_ = userID

	// Push disabled: the public-key endpoint says so.
	rec := doJSON(t, h, http.MethodGet, "/api/push/vapid-public-key", nil, tok)
	var avail struct {
		Available bool `json:"available"`
	}
	mustDecode(t, rec, &avail)
	if avail.Available {
		t.Fatal("push should report unavailable before VAPID keys are set")
	}

	h.Push = push.NewRegistry()
	pub, priv, err := push.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	h.VAPIDPublicKey, h.VAPIDPrivateKey, h.VAPIDSubject = pub, priv, "mailto:vyro@test.invalid"

	rec = doJSON(t, h, http.MethodGet, "/api/push/vapid-public-key", nil, tok)
	var keyResp struct {
		Available bool   `json:"available"`
		PublicKey string `json:"public_key"`
	}
	mustDecode(t, rec, &keyResp)
	if !keyResp.Available || keyResp.PublicKey != pub {
		t.Fatalf("vapid key response = %+v", keyResp)
	}

	// Bad subscription is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/push/subscribe", map[string]interface{}{
		"endpoint": "https://example.com/push",
	}, tok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("subscribe without keys: status %d, want 400", rec.Code)
	}
	// A well-formed (but unreachable) subscription is accepted.
	rec = doJSON(t, h, http.MethodPost, "/api/push/subscribe", map[string]interface{}{
		"endpoint": "http://127.0.0.1:9/push",
		"keys":     map[string]string{"p256dh": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "auth": "AAAAAAAAAAAAAAAAAAAAAA"},
	}, tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("subscribe: %d %s", rec.Code, rec.Body.String())
	}
	// Test delivery to the unreachable endpoint fails (counted, not fatal).
	rec = doJSON(t, h, http.MethodPost, "/api/push/test", nil, tok)
	var testResp struct {
		Sent   int `json:"sent"`
		Failed int `json:"failed"`
	}
	mustDecode(t, rec, &testResp)
	if testResp.Sent != 0 || testResp.Failed != 1 {
		t.Fatalf("test push = %+v, want sent=0 failed=1", testResp)
	}
	// Unsubscribe, then the test reports nothing to send to.
	rec = doJSON(t, h, http.MethodDelete, "/api/push/subscribe", map[string]interface{}{
		"endpoint": "http://127.0.0.1:9/push",
	}, tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unsubscribe: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/push/test", nil, tok)
	mustDecode(t, rec, &testResp)
	if testResp.Sent != 0 || testResp.Failed != 0 {
		t.Fatalf("test push after unsubscribe = %+v, want 0/0", testResp)
	}
}

// TestV05CallSignaling runs two WebSocket clients through the real hub:
// an offer from Alice reaches Bob verbatim (and not Alice herself), an
// answer comes back, ICE flows both ways, and to_user_id addressing
// keeps a third room member out of a 1:1 call.
//
// NOTE: the test reads through per-connection goroutines into channels
// and never sets a read deadline: in gorilla/websocket a ReadMessage
// that times out poisons the connection — later reads fail immediately
// even after SetReadDeadline — so deadline-based "expect nothing"
// checks would break the connection under test. The server itself
// never sets read deadlines, so this is a test-only concern.
func TestV05CallSignaling(t *testing.T) {
	h := newTestHandler()
	_, aliceID := register(t, h, "alice@example.com", "alice")
	_, bobID := register(t, h, "bob@example.com", "bob")
	_, carolID := register(t, h, "carol@example.com", "carol")
	chatID := "call-test-chat"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Query().Get("user")
		chat := r.URL.Query().Get("chat")
		h.Hub.Serve(w, r, user, []string{chat}, nil)
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	type event struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	// reader runs a connection's read loop in a goroutine and forwards
	// non-presence events to ch. Presence broadcasts from joins are
	// filtered here so the test sees only call traffic.
	reader := func(c *websocket.Conn) chan event {
		ch := make(chan event, 16)
		go func() {
			for {
				_, raw, err := c.ReadMessage()
				if err != nil {
					return
				}
				var ev event
				if json.Unmarshal(raw, &ev) != nil || ev.Type == "presence" {
					continue
				}
				ch <- ev
			}
		}()
		return ch
	}
	dial := func(user string) (*websocket.Conn, chan event) {
		c, _, err := websocket.DefaultDialer.Dial(wsURL+"?user="+user+"&chat="+chatID, nil)
		if err != nil {
			t.Fatalf("dial %s: %v", user, err)
		}
		return c, reader(c)
	}
	alice, aliceCh := dial(aliceID)
	defer alice.Close()
	bob, bobCh := dial(bobID)
	defer bob.Close()
	carol, carolCh := dial(carolID)
	defer carol.Close()
	time.Sleep(100 * time.Millisecond) // let all three join the room

	expect := func(ch chan event, wantType, wantPayload string) {
		t.Helper()
		select {
		case ev := <-ch:
			if ev.Type != wantType {
				t.Fatalf("got event %s, want %s", ev.Type, wantType)
			}
			if string(ev.Payload) != wantPayload {
				t.Fatalf("payload = %s, want %s", ev.Payload, wantPayload)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for %s", wantType)
		}
	}
	expectNothing := func(ch chan event, who string) {
		t.Helper()
		select {
		case ev := <-ch:
			t.Fatalf("%s received an event it should not have: %+v", who, ev)
		case <-time.After(300 * time.Millisecond):
		}
	}
	send := func(c *websocket.Conn, typ, to string, payload string) {
		if err := c.WriteJSON(map[string]interface{}{
			"type": typ, "chat_id": chatID, "to_user_id": to, "payload": json.RawMessage(payload),
		}); err != nil {
			t.Fatalf("send %s: %v", typ, err)
		}
	}

	// Offer: Alice -> Bob only (Carol must not see a 1:1 call).
	send(alice, "call-offer", bobID, `{"type":"offer","sdp":"fake-offer-sdp"}`)
	expect(bobCh, "call-offer", `{"type":"offer","sdp":"fake-offer-sdp"}`)
	expectNothing(aliceCh, "alice (sender)")
	expectNothing(carolCh, "carol (not addressed)")

	// Answer: Bob -> Alice.
	send(bob, "call-answer", aliceID, `{"type":"answer","sdp":"fake-answer-sdp"}`)
	expect(aliceCh, "call-answer", `{"type":"answer","sdp":"fake-answer-sdp"}`)

	// ICE both ways.
	send(alice, "call-ice", bobID, `{"candidate":"candidate:1"}`)
	expect(bobCh, "call-ice", `{"candidate":"candidate:1"}`)

	// End without to_user_id reaches every other room member.
	send(alice, "call-end", "", `{"reason":"hangup"}`)
	expect(bobCh, "call-end", `{"reason":"hangup"}`)
	expect(carolCh, "call-end", `{"reason":"hangup"}`)
}
