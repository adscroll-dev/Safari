// VYRO v0.6 handler tests: group E2EE (Sender Keys) end-to-end over
// HTTP, per-device key bundles, multi-device registry, push
// subscription persistence, and two-device WebSocket fan-out.
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
	"github.com/vyro/backend/internal/middleware"
	"github.com/vyro/backend/internal/push"
	"github.com/vyro/backend/internal/ws"
)

func b64enc(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// testIdentity bundles one user's test key material.
type testIdentity struct {
	ident *e2ee.Identity
	spk   *e2ee.SignedPreKey
	opks  map[int]*e2ee.OneTimePreKey
}

func newTestIdentity(t *testing.T) *testIdentity {
	t.Helper()
	ident, err := e2ee.GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	spk, err := e2ee.GenerateSignedPreKey(1, ident)
	if err != nil {
		t.Fatal(err)
	}
	opks := map[int]*e2ee.OneTimePreKey{}
	for i := 1; i <= 5; i++ {
		opk, err := e2ee.GenerateOneTimePreKey(i)
		if err != nil {
			t.Fatal(err)
		}
		opks[i] = opk
	}
	return &testIdentity{ident: ident, spk: spk, opks: opks}
}

// uploadTestKeys publishes an identity key + pre-keys for one device.
func uploadTestKeys(t *testing.T, h *Handler, token, deviceID string, ti *testIdentity) {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/keys/identity", map[string]interface{}{
		"device_id": deviceID, "identity_key": b64enc(ti.ident.DHPub),
		"signed_prekey": b64enc(ti.spk.Pub), "signed_prekey_id": ti.spk.ID,
		"sign_key": b64enc(ti.ident.SignPub), "signed_prekey_sig": b64enc(ti.spk.Signature),
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload identity key %s: %d %s", deviceID, rec.Code, rec.Body.String())
	}
	keys := []map[string]interface{}{}
	for id, opk := range ti.opks {
		keys = append(keys, map[string]interface{}{"key_id": id, "public_key": b64enc(opk.Pub)})
	}
	rec = doJSON(t, h, http.MethodPost, "/api/keys/prekeys", map[string]interface{}{
		"device_id": deviceID, "keys": keys,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload prekeys %s: %d %s", deviceID, rec.Code, rec.Body.String())
	}
}

// fetchTestBundle GETs a key bundle and decodes it.
func fetchTestBundle(t *testing.T, h *Handler, token, userID, deviceID string) *e2ee.Bundle {
	t.Helper()
	path := "/api/keys/" + userID + "/bundle"
	if deviceID != "default" {
		path = "/api/keys/" + userID + "/" + deviceID + "/bundle"
	}
	rec := doJSON(t, h, http.MethodGet, path, nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch bundle %s/%s: %d %s", userID, deviceID, rec.Code, rec.Body.String())
	}
	var b e2ee.Bundle
	if err := json.NewDecoder(rec.Body).Decode(&b); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	return &b
}

// receiveTestSession builds the receiver's 1:1 session from the first
// envelope of a session (the X3DH initiation message).
func receiveTestSession(t *testing.T, ti *testIdentity, env *e2ee.Envelope, peerUserID string) *e2ee.Session {
	t.Helper()
	ik, err := base64.StdEncoding.DecodeString(env.IK)
	if err != nil {
		t.Fatal(err)
	}
	ek, err := base64.StdEncoding.DecodeString(env.EK)
	if err != nil {
		t.Fatal(err)
	}
	var opk *e2ee.OneTimePreKey
	if env.OPKID != 0 {
		opk = ti.opks[env.OPKID]
		if opk == nil {
			t.Fatalf("unknown opk id %d in envelope", env.OPKID)
		}
	}
	sess, err := e2ee.ReceiveInit(ti.ident, ti.spk, opk, ik, ek, peerUserID, "default")
	if err != nil {
		t.Fatalf("ReceiveInit: %v", err)
	}
	return sess
}

func mustJSON(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
}

func TestGroupE2EEEndToEnd(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, carolID := register(t, h, "carol@example.com", "carol")

	aliceKeys := newTestIdentity(t)
	bobKeys := newTestIdentity(t)
	carolKeys := newTestIdentity(t)
	uploadTestKeys(t, h, aliceTok, "default", aliceKeys)
	uploadTestKeys(t, h, bobTok, "default", bobKeys)
	uploadTestKeys(t, h, carolTok, "default", carolKeys)

	// Group chat.
	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "group", "title": "group", "member_ids": []string{bobID, carolID},
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: %d %s", rec.Code, rec.Body.String())
	}
	var chat struct {
		ID string `json:"id"`
	}
	mustJSON(t, rec, &chat)

	// Alice distributes her sender key to Bob and Carol over 1:1 sessions.
	senderKey, err := e2ee.NewSenderKey("alice-k1")
	if err != nil {
		t.Fatal(err)
	}
	sender := e2ee.NewGroupSender(aliceID, "default", senderKey)
	receivers := map[string]*e2ee.GroupReceiver{}
	for _, peer := range []struct {
		token string
		id    string
		keys  *testIdentity
	}{{bobTok, bobID, bobKeys}, {carolTok, carolID, carolKeys}} {
		bundle := fetchTestBundle(t, h, aliceTok, peer.id, "default")
		sess, err := e2ee.Initiate(aliceKeys.ident, bundle)
		if err != nil {
			t.Fatalf("initiate to %s: %v", peer.id, err)
		}
		distBytes, err := senderKey.Marshal(0)
		if err != nil {
			t.Fatal(err)
		}
		env, err := sess.Encrypt(distBytes)
		if err != nil {
			t.Fatal(err)
		}
		envJSON, _ := json.Marshal(env)
		rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/sender-keys", map[string]interface{}{
			"sender_device_id": "default", "key_id": "alice-k1",
			"for_user_id": peer.id, "for_device_id": "default",
			"ciphertext": string(envJSON),
		}, aliceTok)
		if rec.Code != http.StatusCreated {
			t.Fatalf("upload sender key to %s: %d %s", peer.id, rec.Code, rec.Body.String())
		}
		// The peer fetches the distribution and decrypts it.
		rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chat.ID+"/sender-keys?device_id=default", nil, peer.token)
		if rec.Code != http.StatusOK {
			t.Fatalf("list sender keys %s: %d %s", peer.id, rec.Code, rec.Body.String())
		}
		var dists []struct {
			Ciphertext string `json:"ciphertext"`
		}
		mustJSON(t, rec, &dists)
		if len(dists) != 1 {
			t.Fatalf("expected 1 distribution for %s, got %d", peer.id, len(dists))
		}
		var storedEnv e2ee.Envelope
		if err := json.Unmarshal([]byte(dists[0].Ciphertext), &storedEnv); err != nil {
			t.Fatal(err)
		}
		peerSess := receiveTestSession(t, peer.keys, &storedEnv, aliceID)
		pt, err := peerSess.Decrypt(&storedEnv)
		if err != nil {
			t.Fatalf("decrypt distribution %s: %v", peer.id, err)
		}
		d, err := e2ee.UnmarshalDistribution(pt)
		if err != nil {
			t.Fatal(err)
		}
		recv, err := e2ee.NewGroupReceiver(aliceID, "default", d)
		if err != nil {
			t.Fatal(err)
		}
		receivers[peer.id] = recv
	}

	// Alice sends a group message. The stored content must not contain
	// the plaintext anywhere.
	plaintext := "سلام گروه — this is secret"
	genv, err := sender.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	genvJSON, _ := json.Marshal(genv)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]interface{}{
		"content": string(genvJSON), "kind": "e2ee",
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send group e2ee message: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), plaintext) {
		t.Fatal("plaintext leaked in send response")
	}
	var sent struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	mustJSON(t, rec, &sent)
	if strings.Contains(sent.Content, plaintext) {
		t.Fatal("plaintext leaked in stored message content")
	}
	// A 1:1 envelope is rejected in a group chat, and vice versa.
	oneToOne, _ := json.Marshal(&e2ee.Envelope{V: 1, DH: b64enc(make([]byte, 32)), Nonce: b64enc(make([]byte, 12)), CT: b64enc([]byte("x"))})
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]interface{}{
		"content": string(oneToOne), "kind": "e2ee",
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("1:1 envelope in group: expected 400, got %d", rec.Code)
	}

	// Bob and Carol read the message and decrypt it.
	for _, peer := range []struct {
		token string
		id    string
	}{{bobTok, bobID}, {carolTok, carolID}} {
		rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chat.ID+"/messages", nil, peer.token)
		if rec.Code != http.StatusOK {
			t.Fatalf("list messages %s: %d", peer.id, rec.Code)
		}
		var msgs []struct {
			Content string `json:"content"`
		}
		mustJSON(t, rec, &msgs)
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message for %s, got %d", peer.id, len(msgs))
		}
		var stored e2ee.GroupEnvelope
		if err := json.Unmarshal([]byte(msgs[0].Content), &stored); err != nil {
			t.Fatal(err)
		}
		pt, err := receivers[peer.id].Decrypt(&stored)
		if err != nil {
			t.Fatalf("decrypt group message %s: %v", peer.id, err)
		}
		if string(pt) != plaintext {
			t.Fatalf("%s got %q want %q", peer.id, pt, plaintext)
		}
	}
	_ = aliceID
}

func TestGroupE2EEUsesRegisteredDeviceIDs(t *testing.T) {
	h := newTestHandler()
	aliceToken, aliceID := register(t, h, "alice-device@example.com", "alice")
	bobToken, bobID := register(t, h, "bob-device@example.com", "bob")
	aliceDeviceID, bobDeviceID := "alice-phone", "bob-tablet"
	aliceKeys, bobKeys := newTestIdentity(t), newTestIdentity(t)

	for _, account := range []struct {
		token, deviceID string
	}{
		{aliceToken, aliceDeviceID},
		{bobToken, bobDeviceID},
	} {
		rec := doJSON(t, h, http.MethodPost, "/api/devices", map[string]interface{}{
			"device_id": account.deviceID, "name": account.deviceID, "platform": "android",
		}, account.token)
		if rec.Code != http.StatusOK {
			t.Fatalf("register device %s: %d %s", account.deviceID, rec.Code, rec.Body.String())
		}
	}
	uploadTestKeys(t, h, aliceToken, aliceDeviceID, aliceKeys)
	uploadTestKeys(t, h, bobToken, bobDeviceID, bobKeys)

	rec := doJSON(t, h, http.MethodGet, "/api/users/"+bobID+"/devices", nil, aliceToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list Bob devices: %d %s", rec.Code, rec.Body.String())
	}
	var bobDevices []struct {
		DeviceID string `json:"device_id"`
	}
	mustJSON(t, rec, &bobDevices)
	if len(bobDevices) != 1 || bobDevices[0].DeviceID != bobDeviceID {
		t.Fatalf("registered recipient devices = %+v, want %q", bobDevices, bobDeviceID)
	}

	bundle := fetchTestBundle(t, h, aliceToken, bobID, bobDeviceID)
	if bundle.DeviceID != bobDeviceID || bundle.DHPub != b64enc(bobKeys.ident.DHPub) {
		t.Fatalf("recipient key bundle did not match registered device %q: %+v", bobDeviceID, bundle)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "group", "title": "device-id test", "member_ids": []string{bobID},
	}, aliceToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: %d %s", rec.Code, rec.Body.String())
	}
	var chat struct {
		ID string `json:"id"`
	}
	mustJSON(t, rec, &chat)

	senderKey, err := e2ee.NewSenderKey("alice-device-key")
	if err != nil {
		t.Fatal(err)
	}
	sender := e2ee.NewGroupSender(aliceID, aliceDeviceID, senderKey)
	session, err := e2ee.Initiate(aliceKeys.ident, bundle)
	if err != nil {
		t.Fatalf("initiate device-bound session: %v", err)
	}
	dist, err := senderKey.Marshal(0)
	if err != nil {
		t.Fatal(err)
	}
	distEnvelope, err := session.Encrypt(dist)
	if err != nil {
		t.Fatal(err)
	}
	distJSON, _ := json.Marshal(distEnvelope)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/sender-keys", map[string]interface{}{
		"sender_device_id": aliceDeviceID, "key_id": "alice-device-key",
		"for_user_id": bobID, "for_device_id": bobDeviceID,
		"ciphertext": string(distJSON),
	}, aliceToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload device-bound sender key: %d %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chat.ID+"/sender-keys?device_id="+bobDeviceID, nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch sender key for registered device: %d %s", rec.Code, rec.Body.String())
	}
	var distributions []struct {
		Ciphertext string `json:"ciphertext"`
	}
	mustJSON(t, rec, &distributions)
	if len(distributions) != 1 {
		t.Fatalf("registered device received %d sender keys, want 1", len(distributions))
	}
	var receivedEnvelope e2ee.Envelope
	if err := json.Unmarshal([]byte(distributions[0].Ciphertext), &receivedEnvelope); err != nil {
		t.Fatal(err)
	}
	receivedOPK := bobKeys.opks[receivedEnvelope.OPKID]
	peerSession, err := e2ee.ReceiveInit(
		bobKeys.ident, bobKeys.spk, receivedOPK,
		mustB64(t, receivedEnvelope.IK), mustB64(t, receivedEnvelope.EK),
		aliceID, aliceDeviceID,
	)
	if err != nil {
		t.Fatalf("receive sender-key distribution: %v", err)
	}
	distPlaintext, err := peerSession.Decrypt(&receivedEnvelope)
	if err != nil {
		t.Fatalf("decrypt sender-key distribution: %v", err)
	}
	decodedDistribution, err := e2ee.UnmarshalDistribution(distPlaintext)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := e2ee.NewGroupReceiver(aliceID, aliceDeviceID, decodedDistribution)
	if err != nil {
		t.Fatal(err)
	}

	const plaintext = "device-bound group message"
	groupEnvelope, err := sender.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	groupJSON, _ := json.Marshal(groupEnvelope)
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]interface{}{
		"content": string(groupJSON), "kind": "e2ee",
	}, aliceToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send device-bound group message: %d %s", rec.Code, rec.Body.String())
	}
	decrypted, err := receiver.Decrypt(groupEnvelope)
	if err != nil {
		t.Fatalf("decrypt device-bound group message: %v", err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("decrypted message = %q, want %q", decrypted, plaintext)
	}
}

func mustB64(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("decode base64 envelope field: %v", err)
	}
	return decoded
}

func TestGroupE2EEValidation(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	eveTok, _ := register(t, h, "eve@example.com", "eve")
	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "group", "title": "g", "member_ids": []string{bobID},
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: %d %s", rec.Code, rec.Body.String())
	}
	var chat struct {
		ID string `json:"id"`
	}
	mustJSON(t, rec, &chat)

	// Garbage is not a group envelope.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]interface{}{
		"content": "not an envelope", "kind": "e2ee",
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("garbage envelope: expected 400, got %d", rec.Code)
	}
	// Non-members cannot upload or list sender keys.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/sender-keys", map[string]interface{}{
		"sender_device_id": "default", "key_id": "k1", "for_user_id": bobID,
		"for_device_id": "default", "ciphertext": "{}",
	}, eveTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member upload: expected 403, got %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/chats/"+chat.ID+"/sender-keys", nil, eveTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member list: expected 403, got %d", rec.Code)
	}
	// Distribution to a non-member is rejected.
	key, _ := e2ee.NewSenderKey("k1")
	sender := e2ee.NewGroupSender(aliceID, "default", key)
	_ = sender
	aliceKeys := newTestIdentity(t)
	bobKeys := newTestIdentity(t)
	uploadTestKeys(t, h, aliceTok, "default", aliceKeys)
	uploadTestKeys(t, h, bobTok, "default", bobKeys)
	bundle := fetchTestBundle(t, h, aliceTok, bobID, "default")
	sess, err := e2ee.Initiate(aliceKeys.ident, bundle)
	if err != nil {
		t.Fatal(err)
	}
	distBytes, _ := key.Marshal(0)
	env, _ := sess.Encrypt(distBytes)
	envJSON, _ := json.Marshal(env)
	// Eve is not a member of the group.
	_, eveID := register(t, h, "eve2@example.com", "eve2")
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/sender-keys", map[string]interface{}{
		"sender_device_id": "default", "key_id": "k1", "for_user_id": eveID,
		"for_device_id": "default", "ciphertext": string(envJSON),
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("distribution to non-member: expected 400, got %d", rec.Code)
	}
	// A malformed distribution envelope is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/sender-keys", map[string]interface{}{
		"sender_device_id": "default", "key_id": "k1", "for_user_id": bobID,
		"for_device_id": "default", "ciphertext": "not-an-envelope",
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed distribution: expected 400, got %d", rec.Code)
	}
}

func TestMemberRemovalDeletesSenderKeys(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	carolTok, carolID := register(t, h, "carol@example.com", "carol")
	aliceKeys := newTestIdentity(t)
	bobKeys := newTestIdentity(t)
	carolKeys := newTestIdentity(t)
	uploadTestKeys(t, h, aliceTok, "default", aliceKeys)
	uploadTestKeys(t, h, bobTok, "default", bobKeys)
	uploadTestKeys(t, h, carolTok, "default", carolKeys)
	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "group", "title": "g", "member_ids": []string{bobID, carolID},
	}, aliceTok)
	var chat struct {
		ID string `json:"id"`
	}
	mustJSON(t, rec, &chat)
	key, _ := e2ee.NewSenderKey("k1")
	distBytes, _ := key.Marshal(0)
	for _, peerID := range []string{bobID, carolID} {
		bundle := fetchTestBundle(t, h, aliceTok, peerID, "default")
		sess, err := e2ee.Initiate(aliceKeys.ident, bundle)
		if err != nil {
			t.Fatal(err)
		}
		env, _ := sess.Encrypt(distBytes)
		envJSON, _ := json.Marshal(env)
		rec := doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/sender-keys", map[string]interface{}{
			"sender_device_id": "default", "key_id": "k1", "for_user_id": peerID,
			"for_device_id": "default", "ciphertext": string(envJSON),
		}, aliceTok)
		if rec.Code != http.StatusCreated {
			t.Fatalf("upload sender key: %d %s", rec.Code, rec.Body.String())
		}
	}
	if got := h.Store.GroupSenderKeysFor(chat.ID, bobID, "default"); len(got) != 1 {
		t.Fatalf("bob should have 1 distribution, got %d", len(got))
	}
	// Alice (owner) removes Bob.
	rec = doJSON(t, h, http.MethodDelete, "/api/chats/"+chat.ID+"/members/"+bobID, nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("remove member: %d %s", rec.Code, rec.Body.String())
	}
	if got := h.Store.GroupSenderKeysFor(chat.ID, bobID, "default"); len(got) != 0 {
		t.Fatalf("bob's distributions should be deleted, got %d", len(got))
	}
	if got := h.Store.GroupSenderKeysFor(chat.ID, carolID, "default"); len(got) != 1 {
		t.Fatalf("carol's distribution should survive, got %d", len(got))
	}
	_ = bobTok
	_ = carolTok
}

func TestPerDeviceBundles(t *testing.T) {
	h := newTestHandler()
	aliceTok, _ := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	bobPhone := newTestIdentity(t)
	bobLaptop := newTestIdentity(t)
	uploadTestKeys(t, h, bobTok, "phone", bobPhone)
	uploadTestKeys(t, h, bobTok, "laptop", bobLaptop)

	// Each device's bundle carries its own identity key and a pre-key
	// belonging to that device.
	phoneBundle := fetchTestBundle(t, h, aliceTok, bobID, "phone")
	if phoneBundle.DHPub != b64enc(bobPhone.ident.DHPub) || phoneBundle.DeviceID != "phone" {
		t.Fatalf("phone bundle mismatch: %+v", phoneBundle)
	}
	if phoneBundle.OneTimePreKey == nil {
		t.Fatal("phone bundle should claim a pre-key")
	}
	if _, ok := bobPhone.opks[phoneBundle.OneTimePreKey.ID]; !ok {
		t.Fatalf("phone bundle pre-key %d is not a phone pre-key", phoneBundle.OneTimePreKey.ID)
	}
	laptopBundle := fetchTestBundle(t, h, aliceTok, bobID, "laptop")
	if laptopBundle.DHPub != b64enc(bobLaptop.ident.DHPub) || laptopBundle.DeviceID != "laptop" {
		t.Fatalf("laptop bundle mismatch: %+v", laptopBundle)
	}
	if _, ok := bobLaptop.opks[laptopBundle.OneTimePreKey.ID]; !ok {
		t.Fatalf("laptop bundle pre-key %d is not a laptop pre-key", laptopBundle.OneTimePreKey.ID)
	}
	// A second fetch claims a different pre-key (each handed out once).
	phoneBundle2 := fetchTestBundle(t, h, aliceTok, bobID, "phone")
	if phoneBundle2.OneTimePreKey != nil && phoneBundle2.OneTimePreKey.ID == phoneBundle.OneTimePreKey.ID {
		t.Fatal("same pre-key handed out twice")
	}
	// Unknown device -> 404; legacy path still serves "default".
	rec := doJSON(t, h, http.MethodGet, "/api/keys/"+bobID+"/ghost/bundle", nil, aliceTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown device bundle: expected 404, got %d", rec.Code)
	}
}

func TestDeviceEndpoints(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")

	rec := doJSON(t, h, http.MethodPost, "/api/devices", map[string]interface{}{
		"device_id": "phone-2", "name": "Pixel", "platform": "android",
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("register device: %d %s", rec.Code, rec.Body.String())
	}
	// Bad platform is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/devices", map[string]interface{}{
		"device_id": "x", "platform": "toaster",
	}, aliceTok)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad platform: expected 400, got %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/devices", nil, aliceTok)
	var mine []struct {
		DeviceID string `json:"device_id"`
		Platform string `json:"platform"`
	}
	mustJSON(t, rec, &mine)
	if len(mine) != 1 || mine[0].DeviceID != "phone-2" {
		t.Fatalf("my devices: %+v", mine)
	}
	// Another user sees the public device list (no secrets).
	rec = doJSON(t, h, http.MethodGet, "/api/users/"+aliceID+"/devices", nil, bobTok)
	var pub []struct {
		DeviceID string `json:"device_id"`
	}
	mustJSON(t, rec, &pub)
	if len(pub) != 1 || pub[0].DeviceID != "phone-2" {
		t.Fatalf("public devices: %+v", pub)
	}
	// Uploading identity keys auto-registers the device.
	aliceKeys := newTestIdentity(t)
	uploadTestKeys(t, h, aliceTok, "laptop", aliceKeys)
	rec = doJSON(t, h, http.MethodGet, "/api/devices", nil, aliceTok)
	mustJSON(t, rec, &mine)
	if len(mine) != 2 {
		t.Fatalf("expected 2 devices after key upload, got %+v", mine)
	}
	// Remove a device.
	rec = doJSON(t, h, http.MethodDelete, "/api/devices/phone-2", nil, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("remove device: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/devices", nil, aliceTok)
	mustJSON(t, rec, &mine)
	if len(mine) != 1 || mine[0].DeviceID != "laptop" {
		t.Fatalf("after remove: %+v", mine)
	}
	// Unknown user -> 404.
	rec = doJSON(t, h, http.MethodGet, "/api/users/nope/devices", nil, bobTok)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown user devices: expected 404, got %d", rec.Code)
	}
	_ = bobID
}

func TestPushSubscriptionSurvivesRestart(t *testing.T) {
	h := newTestHandler()
	h.Push = push.NewRegistryWithStore(h.Store)
	pub, priv, err := push.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	h.VAPIDPublicKey, h.VAPIDPrivateKey = pub, priv
	h.VAPIDSubject = "mailto:vyro@test.invalid"
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	rec := doJSON(t, h, http.MethodPost, "/api/push/subscribe", map[string]interface{}{
		"endpoint": "https://push.example.invalid/alice-1",
		"keys":     map[string]string{"p256dh": "p", "auth": "a"},
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("subscribe: %d %s", rec.Code, rec.Body.String())
	}
	if h.Push.Count(aliceID) != 1 {
		t.Fatal("subscription not stored")
	}
	// "Restart": a new handler and a new registry over the SAME store.
	h2 := newTestHandler()
	h2.Store = h.Store
	h2.Push = push.NewRegistryWithStore(h2.Store)
	h2.VAPIDPublicKey, h2.VAPIDPrivateKey = pub, priv
	if h2.Push.Count(aliceID) != 1 {
		t.Fatalf("subscription lost after restart: count=%d", h2.Push.Count(aliceID))
	}
	rec = doJSON(t, h2, http.MethodGet, "/api/push/vapid-public-key", nil, aliceTok)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"subscribers":1`) {
		t.Fatalf("vapid endpoint after restart: %d %s", rec.Code, rec.Body.String())
	}
	// Unsubscribe still works after restart.
	rec = doJSON(t, h2, http.MethodDelete, "/api/push/subscribe", map[string]string{
		"endpoint": "https://push.example.invalid/alice-1",
	}, aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("unsubscribe: %d %s", rec.Code, rec.Body.String())
	}
	if h2.Push.Count(aliceID) != 0 {
		t.Fatal("subscription should be gone")
	}
}

// --- Multi-device WebSocket fan-out ---

func wsDial(t *testing.T, srvURL, token string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srvURL, "http") + "?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	return conn
}

// readWS waits for an event of the given type (skipping others, e.g.
// presence) and returns its message id, if any.
func readWS(t *testing.T, conn *websocket.Conn, wantType string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		var ev ws.Event
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read ws event (want %s): %v", wantType, err)
		}
		if ev.Type == wantType {
			if ev.Message != nil {
				return ev.Message.ID
			}
			return ""
		}
	}
}

func TestMultiDeviceWebSocketFanOut(t *testing.T) {
	h := newTestHandler()
	aliceTok, aliceID := register(t, h, "alice@example.com", "alice")
	bobTok, bobID := register(t, h, "bob@example.com", "bob")
	rec := doJSON(t, h, http.MethodPost, "/api/chats", map[string]interface{}{
		"type": "dm", "member_ids": []string{bobID},
	}, aliceTok)
	var chat struct {
		ID string `json:"id"`
	}
	mustJSON(t, rec, &chat)

	// Two simulated devices for Bob, one for Alice (her second device;
	// her first device sends over HTTP).
	srv := httptest.NewServer(middleware.Auth("test-secret")(http.HandlerFunc(h.WebSocketHandler)))
	defer srv.Close()
	bobDev1 := wsDial(t, srv.URL, bobTok)
	defer bobDev1.Close()
	bobDev2 := wsDial(t, srv.URL, bobTok)
	defer bobDev2.Close()
	aliceDev2 := wsDial(t, srv.URL, aliceTok)
	defer aliceDev2.Close()

	// Alice's first device sends a message over HTTP.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]string{
		"content": "hello from alice device 1",
	}, aliceTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send: %d %s", rec.Code, rec.Body.String())
	}
	var sent struct {
		ID string `json:"id"`
	}
	mustJSON(t, rec, &sent)
	// BOTH of Bob's devices receive it.
	if got := readWS(t, bobDev1, "message"); got != sent.ID {
		t.Fatalf("bob device 1 got message %q, want %q", got, sent.ID)
	}
	if got := readWS(t, bobDev2, "message"); got != sent.ID {
		t.Fatalf("bob device 2 got message %q, want %q", got, sent.ID)
	}
	// Alice's second device receives the sync copy.
	if got := readWS(t, aliceDev2, "message_sync"); got != sent.ID {
		t.Fatalf("alice device 2 got sync %q, want %q", got, sent.ID)
	}

	// Bob's second device replies over HTTP: Alice's device gets
	// "message"; Bob's FIRST device is his other device, so it gets
	// the "message_sync" copy (BroadcastToChat never echoes to the
	// sender's own connections), as does the sending device.
	rec = doJSON(t, h, http.MethodPost, "/api/chats/"+chat.ID+"/messages", map[string]string{
		"content": "hello from bob device 2",
	}, bobTok)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bob send: %d %s", rec.Code, rec.Body.String())
	}
	mustJSON(t, rec, &sent)
	if got := readWS(t, aliceDev2, "message"); got != sent.ID {
		t.Fatalf("alice device 2 got message %q, want %q", got, sent.ID)
	}
	if got := readWS(t, bobDev1, "message_sync"); got != sent.ID {
		t.Fatalf("bob device 1 got sync %q, want %q", got, sent.ID)
	}
	if got := readWS(t, bobDev2, "message_sync"); got != sent.ID {
		t.Fatalf("bob device 2 got sync %q, want %q", got, sent.ID)
	}

	// Editing syncs to the editor's other device and to the peer.
	rec = doJSON(t, h, http.MethodPut, "/api/messages/"+sent.ID, map[string]string{
		"content": "hello from bob device 2 (edited)",
	}, bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit: %d %s", rec.Code, rec.Body.String())
	}
	if got := readWS(t, aliceDev2, "message_edited"); got != sent.ID {
		t.Fatalf("alice device 2 got edit %q, want %q", got, sent.ID)
	}
	if got := readWS(t, bobDev1, "message_edited"); got != sent.ID {
		t.Fatalf("bob device 1 got edit %q, want %q", got, sent.ID)
	}
	_ = aliceID
}
