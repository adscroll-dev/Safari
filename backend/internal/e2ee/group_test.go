package e2ee

import (
	"encoding/json"
	"strings"
	"testing"
)

// distribute sends a sender key from alice to bob over their 1:1 session
// and returns the envelope the server would store (ciphertext only) plus
// Bob's receiver state built from the decrypted distribution.
func distribute(t *testing.T, aliceSess, bobSess *Session, sender *GroupSender) (*Envelope, *GroupReceiver) {
	t.Helper()
	distBytes, err := sender.Key.Marshal(sender.N)
	if err != nil {
		t.Fatalf("marshal distribution: %v", err)
	}
	env, err := aliceSess.Encrypt(distBytes)
	if err != nil {
		t.Fatalf("encrypt distribution: %v", err)
	}
	pt, err := bobSess.Decrypt(env)
	if err != nil {
		t.Fatalf("bob decrypt distribution: %v", err)
	}
	d, err := UnmarshalDistribution(pt)
	if err != nil {
		t.Fatalf("unmarshal distribution: %v", err)
	}
	recv, err := NewGroupReceiver(sender.SenderUserID, sender.SenderDeviceID, d)
	if err != nil {
		t.Fatalf("new receiver: %v", err)
	}
	return env, recv
}

func mustGroupEncrypt(t *testing.T, s *GroupSender, msg string) *GroupEnvelope {
	t.Helper()
	env, err := s.Encrypt([]byte(msg))
	if err != nil {
		t.Fatalf("group encrypt: %v", err)
	}
	return env
}

func TestGroupRoundTripThreeMembers(t *testing.T) {
	// Alice distributes her sender key to Bob and Carol over 1:1 sessions.
	_, bob, bobSPK, bobOPK, aliceToBob := setupPair(t, true)
	_, carol, carolSPK, carolOPK, aliceToCarol := setupPair(t, true)
	env0 := mustEncrypt(t, aliceToBob, "init-bob")
	bobSess := receiveInit(t, bob, bobSPK, bobOPK, env0)
	env0c := mustEncrypt(t, aliceToCarol, "init-carol")
	carolSess := receiveInit(t, carol, carolSPK, carolOPK, env0c)

	key, err := NewSenderKey("alice-key-1")
	if err != nil {
		t.Fatal(err)
	}
	sender := NewGroupSender("alice", "default", key)
	_, bobRecv := distribute(t, aliceToBob, bobSess, sender)
	_, carolRecv := distribute(t, aliceToCarol, carolSess, sender)

	msgs := []string{"سلام گروه!", "second message", "third message — with emoji 🎉"}
	for _, want := range msgs {
		env := mustGroupEncrypt(t, sender, want)
		for name, recv := range map[string]*GroupReceiver{"bob": bobRecv, "carol": carolRecv} {
			pt, err := recv.Decrypt(env)
			if err != nil {
				t.Fatalf("%s decrypt: %v", name, err)
			}
			if string(pt) != want {
				t.Fatalf("%s got %q want %q", name, pt, want)
			}
		}
	}
}

func TestGroupOutOfOrder(t *testing.T) {
	_, bob, spk, opk, aliceSess := setupPair(t, true)
	env0 := mustEncrypt(t, aliceSess, "init")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	key, _ := NewSenderKey("k1")
	sender := NewGroupSender("alice", "default", key)
	_, recv := distribute(t, aliceSess, bobSess, sender)

	e1 := mustGroupEncrypt(t, sender, "one")
	e2 := mustGroupEncrypt(t, sender, "two")
	e3 := mustGroupEncrypt(t, sender, "three")
	// Deliver 3, then 1, then 2.
	if pt, err := recv.Decrypt(e3); err != nil || string(pt) != "three" {
		t.Fatalf("decrypt 3 first: pt=%q err=%v", pt, err)
	}
	if pt, err := recv.Decrypt(e1); err != nil || string(pt) != "one" {
		t.Fatalf("decrypt 1 late: pt=%q err=%v", pt, err)
	}
	if pt, err := recv.Decrypt(e2); err != nil || string(pt) != "two" {
		t.Fatalf("decrypt 2 late: pt=%q err=%v", pt, err)
	}
	// Replay of an already-decrypted message is rejected.
	if _, err := recv.Decrypt(e1); err == nil {
		t.Fatal("replay of message 1 should fail")
	}
}

func TestGroupTamper(t *testing.T) {
	_, bob, spk, opk, aliceSess := setupPair(t, true)
	env0 := mustEncrypt(t, aliceSess, "init")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	key, _ := NewSenderKey("k1")
	sender := NewGroupSender("alice", "default", key)
	_, recv := distribute(t, aliceSess, bobSess, sender)

	// Tampered ciphertext.
	env := mustGroupEncrypt(t, sender, "secret")
	ct, _ := unb64(env.CT)
	ct[0] ^= 0x01
	env.CT = b64(ct)
	if _, err := recv.Decrypt(env); err == nil {
		t.Fatal("tampered ciphertext should fail to decrypt")
	}
	// Tampered signature.
	env = mustGroupEncrypt(t, sender, "secret")
	sig, _ := unb64(env.Sig)
	sig[0] ^= 0x01
	env.Sig = b64(sig)
	if _, err := recv.Decrypt(env); err == nil {
		t.Fatal("tampered signature should fail verification")
	}
	// Tampered header (n).
	env = mustGroupEncrypt(t, sender, "secret")
	env.N = env.N + 100
	if _, err := recv.Decrypt(env); err == nil {
		t.Fatal("tampered header should fail")
	}
	// Message under a different sender key generation.
	key2, _ := NewSenderKey("k2")
	sender2 := NewGroupSender("alice", "default", key2)
	env2 := mustGroupEncrypt(t, sender2, "other key")
	if _, err := recv.Decrypt(env2); err == nil {
		t.Fatal("message under a different key_id should fail")
	}
	// A forged envelope from an attacker who does not hold the signing key.
	env3 := mustGroupEncrypt(t, sender, "real")
	attackerKey, _ := NewSenderKey("k1")
	attacker := NewGroupSender("alice", "default", attackerKey)
	forged := mustGroupEncrypt(t, attacker, "forged")
	forged.N = env3.N
	if _, err := recv.Decrypt(forged); err == nil {
		t.Fatal("forged envelope with attacker's signing key should fail")
	}
}

func TestGroupRotation(t *testing.T) {
	_, bob, spk, opk, aliceSess := setupPair(t, true)
	env0 := mustEncrypt(t, aliceSess, "init")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	key1, _ := NewSenderKey("k1")
	sender := NewGroupSender("alice", "default", key1)
	_, recv := distribute(t, aliceSess, bobSess, sender)
	if pt, err := recv.Decrypt(mustGroupEncrypt(t, sender, "before rotation")); err != nil || string(pt) != "before rotation" {
		t.Fatalf("pre-rotation decrypt: %q %v", pt, err)
	}
	// Alice rotates (e.g. a member left): new sender key, new distribution.
	key2, _ := NewSenderKey("k2")
	sender2 := NewGroupSender("alice", "default", key2)
	_, recv2 := distribute(t, aliceSess, bobSess, sender2)
	after := mustGroupEncrypt(t, sender2, "after rotation")
	// The OLD receiver state cannot read post-rotation messages.
	if _, err := recv.Decrypt(after); err == nil {
		t.Fatal("old sender key must not decrypt post-rotation messages")
	}
	// The NEW receiver state can.
	if pt, err := recv2.Decrypt(after); err != nil || string(pt) != "after rotation" {
		t.Fatalf("post-rotation decrypt: %q %v", pt, err)
	}
}

func TestValidateGroupEnvelope(t *testing.T) {
	key, _ := NewSenderKey("k1")
	sender := NewGroupSender("alice", "default", key)
	env := mustGroupEncrypt(t, sender, "hi")
	raw, _ := json.Marshal(env)
	if _, err := ValidateGroupEnvelope(string(raw)); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
	// Bad version.
	var bad GroupEnvelope
	_ = json.Unmarshal(raw, &bad)
	bad.V = 99
	rawBad, _ := json.Marshal(bad)
	if _, err := ValidateGroupEnvelope(string(rawBad)); err == nil {
		t.Fatal("version 99 should be rejected")
	}
	// Missing signature.
	_ = json.Unmarshal(raw, &bad)
	bad.Sig = ""
	rawBad, _ = json.Marshal(bad)
	if _, err := ValidateGroupEnvelope(string(rawBad)); err == nil {
		t.Fatal("missing sig should be rejected")
	}
	// Not JSON / empty.
	if _, err := ValidateGroupEnvelope("not json"); err == nil {
		t.Fatal("non-JSON should be rejected")
	}
	if _, err := ValidateGroupEnvelope(""); err == nil {
		t.Fatal("empty envelope should be rejected")
	}
}

// TestServerCannotReadGroup proves the server — which holds exactly the
// stored group envelope and the stored sender-key distribution envelopes
// — cannot read group plaintext and cannot recover the sender key.
//
// The "server" here is given every byte the real server stores, plus a
// fresh 1:1 session (an attacker's), and must fail at each step.
func TestServerCannotReadGroup(t *testing.T) {
	_, bob, spk, opk, aliceSess := setupPair(t, true)
	env0 := mustEncrypt(t, aliceSess, "init")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	key, _ := NewSenderKey("k1")
	sender := NewGroupSender("alice", "default", key)
	distEnv, recv := distribute(t, aliceSess, bobSess, sender)

	plaintext := "group secret — do not read this, server"
	groupEnv := mustGroupEncrypt(t, sender, plaintext)

	// Exactly what the server stores: the group envelope JSON and the
	// distribution envelope JSON (opaque 1:1 ciphertext).
	groupRaw, _ := json.Marshal(groupEnv)
	distRaw, _ := json.Marshal(distEnv)
	serverBytes := string(groupRaw) + string(distRaw)

	// (a) The plaintext never appears in anything the server holds.
	if strings.Contains(serverBytes, plaintext) {
		t.Fatal("plaintext appears in server-held bytes")
	}
	if strings.Contains(serverBytes, "group secret") {
		t.Fatal("plaintext fragment appears in server-held bytes")
	}
	// (b) A receiver built from a random chain key cannot decrypt.
	attackerKey, _ := NewSenderKey("k1")
	attackerRecv := &GroupReceiver{
		SenderUserID: "alice", SenderDeviceID: "default", KeyID: "k1",
		ChainKey: attackerKey.ChainKey, SignPub: attackerKey.SignPub,
		N: 0, Skipped: map[int]string{},
	}
	if _, err := attackerRecv.Decrypt(groupEnv); err == nil {
		t.Fatal("random chain key must not decrypt the group message")
	}
	// (c) The distribution envelope cannot be opened without Bob's 1:1
	// session: an attacker's fresh session fails to decrypt it, so the
	// server cannot recover the sender key from the stored distribution.
	_, eve, eveSPK, eveOPK, aliceToEve := setupPair(t, true)
	eveInit := mustEncrypt(t, aliceToEve, "init-eve")
	eveSess := receiveInit(t, eve, eveSPK, eveOPK, eveInit)
	if _, err := eveSess.Decrypt(distEnv); err == nil {
		t.Fatal("distribution envelope decrypted with the wrong 1:1 session")
	}
	// (d) Sanity: the real recipient CAN decrypt both (the test would
	// be vacuous if nothing could).
	if pt, err := recv.Decrypt(groupEnv); err != nil || string(pt) != plaintext {
		t.Fatalf("legitimate recipient failed: %q %v", pt, err)
	}
}

func TestDistributionValidation(t *testing.T) {
	key, _ := NewSenderKey("k1")
	sender := NewGroupSender("alice", "default", key)
	raw, err := sender.Key.Marshal(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UnmarshalDistribution(raw); err != nil {
		t.Fatalf("valid distribution rejected: %v", err)
	}
	if _, err := UnmarshalDistribution([]byte(`{"key_id":"k1","chain_key":"!!!","sign_pub":"x","n":0}`)); err == nil {
		t.Fatal("bad chain_key should be rejected")
	}
	if _, err := UnmarshalDistribution([]byte(`not json`)); err == nil {
		t.Fatal("non-JSON distribution should be rejected")
	}
}
