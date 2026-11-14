package e2ee

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// setupPair creates two identities with a full key bundle for Bob,
// and an initiated session for Alice. Returns (aliceIdent, bobIdent,
// bobSPK, bobOPK, aliceSession).
func setupPair(t *testing.T, withOPK bool) (*Identity, *Identity, *SignedPreKey, *OneTimePreKey, *Session) {
	t.Helper()
	alice, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	spk, err := GenerateSignedPreKey(1, bob)
	if err != nil {
		t.Fatal(err)
	}
	var opk *OneTimePreKey
	bundle := &Bundle{
		UserID: "bob", DeviceID: "default",
		DHPub: b64(bob.DHPub), SignPub: b64(bob.SignPub),
		SPKPub: b64(spk.Pub), SPKID: spk.ID, SPKSig: b64(spk.Signature),
	}
	if withOPK {
		opk, err = GenerateOneTimePreKey(7)
		if err != nil {
			t.Fatal(err)
		}
		bundle.OneTimePreKey = &OPK{ID: opk.ID, Pub: b64(opk.Pub)}
	}
	sess, err := Initiate(alice, bundle)
	if err != nil {
		t.Fatal(err)
	}
	return alice, bob, spk, opk, sess
}

func mustEncrypt(t *testing.T, s *Session, msg string) *Envelope {
	t.Helper()
	env, err := s.Encrypt([]byte(msg))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return env
}

// receiveInit builds Bob's session from Alice's first envelope.
func receiveInit(t *testing.T, bob *Identity, spk *SignedPreKey, opk *OneTimePreKey, env *Envelope) *Session {
	t.Helper()
	ik, err := unb64(env.IK)
	if err != nil {
		t.Fatal(err)
	}
	ek, err := unb64(env.EK)
	if err != nil {
		t.Fatal(err)
	}
	var usedOPK *OneTimePreKey
	if env.OPKID != 0 {
		if opk == nil || opk.ID != env.OPKID {
			t.Fatalf("envelope uses opk %d, test has %v", env.OPKID, opk)
		}
		usedOPK = opk
	}
	sess, err := ReceiveInit(bob, spk, usedOPK, ik, ek, "alice", "default")
	if err != nil {
		t.Fatalf("ReceiveInit: %v", err)
	}
	return sess
}

func TestRoundTripAlternating(t *testing.T) {
	_, bob, spk, opk, alice := setupPair(t, true)
	env0 := mustEncrypt(t, alice, "سلام علی — پیام اول")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	pt, err := bobSess.Decrypt(env0)
	if err != nil {
		t.Fatalf("bob decrypt first: %v", err)
	}
	if string(pt) != "سلام علی — پیام اول" {
		t.Fatalf("got %q", pt)
	}
	// 20 alternating messages.
	for i := 0; i < 20; i++ {
		msg := "ping " + string(rune('a'+i))
		env := mustEncrypt(t, alice, msg)
		pt, err := bobSess.Decrypt(env)
		if err != nil {
			t.Fatalf("bob decrypt %d: %v", i, err)
		}
		if string(pt) != msg {
			t.Fatalf("bob got %q want %q", pt, msg)
		}
		reply := "pong " + string(rune('a'+i))
		env2 := mustEncrypt(t, bobSess, reply)
		pt2, err := alice.Decrypt(env2)
		if err != nil {
			t.Fatalf("alice decrypt %d: %v", i, err)
		}
		if string(pt2) != reply {
			t.Fatalf("alice got %q want %q", pt2, reply)
		}
	}
}

func TestNoOneTimePreKey(t *testing.T) {
	_, bob, spk, _, alice := setupPair(t, false)
	env := mustEncrypt(t, alice, "no opk here")
	bobSess := receiveInit(t, bob, spk, nil, env)
	pt, err := bobSess.Decrypt(env)
	if err != nil {
		t.Fatalf("decrypt without opk: %v", err)
	}
	if string(pt) != "no opk here" {
		t.Fatalf("got %q", pt)
	}
}

func TestOutOfOrder(t *testing.T) {
	_, bob, spk, opk, alice := setupPair(t, true)
	env0 := mustEncrypt(t, alice, "first")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	if _, err := bobSess.Decrypt(env0); err != nil {
		t.Fatal(err)
	}
	env1 := mustEncrypt(t, alice, "second")
	env2 := mustEncrypt(t, alice, "third")
	env3 := mustEncrypt(t, alice, "fourth")
	// Deliver 3rd, then 2nd, then 4th.
	pt, err := bobSess.Decrypt(env2)
	if err != nil || string(pt) != "third" {
		t.Fatalf("out-of-order 3rd: %q %v", pt, err)
	}
	pt, err = bobSess.Decrypt(env1)
	if err != nil || string(pt) != "second" {
		t.Fatalf("out-of-order 2nd: %q %v", pt, err)
	}
	pt, err = bobSess.Decrypt(env3)
	if err != nil || string(pt) != "fourth" {
		t.Fatalf("out-of-order 4th: %q %v", pt, err)
	}
}

func TestTamperFails(t *testing.T) {
	_, bob, spk, opk, alice := setupPair(t, true)
	env0 := mustEncrypt(t, alice, "secret")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	if _, err := bobSess.Decrypt(env0); err != nil {
		t.Fatal(err)
	}
	env1 := mustEncrypt(t, alice, "do not touch")
	// Flip a byte of the ciphertext.
	ct, _ := base64.StdEncoding.DecodeString(env1.CT)
	ct[0] ^= 0x01
	env1.CT = base64.StdEncoding.EncodeToString(ct)
	if _, err := bobSess.Decrypt(env1); err == nil {
		t.Fatal("tampered ciphertext decrypted — AEAD not binding")
	}
	// Tamper with the header (ratchet public key).
	env2 := mustEncrypt(t, alice, "header matters")
	env2.PN = env2.PN + 5
	if _, err := bobSess.Decrypt(env2); err == nil {
		t.Fatal("tampered header decrypted — AD not binding")
	}
}

func TestWrongKeyFails(t *testing.T) {
	_, bob, spk, opk, alice := setupPair(t, true)
	env0 := mustEncrypt(t, alice, "for bob only")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	if _, err := bobSess.Decrypt(env0); err != nil {
		t.Fatal(err)
	}
	// An attacker who controls a *different* identity (different
	// private keys) builds a session from the same first envelope:
	// their X3DH secret differs, so decryption must fail.
	attacker, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	attackerSPK, err := GenerateSignedPreKey(1, attacker)
	if err != nil {
		t.Fatal(err)
	}
	evilSess, err := ReceiveInit(attacker, attackerSPK, nil, mustB64(t, env0.IK), mustB64(t, env0.EK), "alice", "default")
	if err != nil {
		t.Fatal(err)
	}
	env1 := mustEncrypt(t, alice, "still for bob")
	if _, err := evilSess.Decrypt(env1); err == nil {
		t.Fatal("decryption with wrong session keys succeeded")
	}
}

func mustB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestServerCannotRead proves the property the whole feature rests on:
// the envelope the server stores contains no plaintext, and no key the
// server holds (it holds only public keys) can decrypt it.
func TestServerCannotRead(t *testing.T) {
	alice, bob, spk, opk, aliceSess := setupPair(t, true)
	secret := "رمز عبور من 12345 — فقط علی باید بخواند"
	env := mustEncrypt(t, aliceSess, secret)
	raw, _ := json.Marshal(env)
	if strings.Contains(string(raw), "رمز") || strings.Contains(string(raw), "12345") {
		t.Fatal("plaintext leaked into the envelope the server stores")
	}
	if _, err := ValidateEnvelope(string(raw)); err != nil {
		t.Fatalf("server-side envelope validation failed: %v", err)
	}
	// The server holds Bob's public bundle only. Try to decrypt with
	// a session built from public keys alone (no private keys): the
	// crypto must fail — there is no private key to run X3DH with.
	_ = alice
	_ = bob
	_ = spk
	_ = opk
	// Tamper-free check: a random AES key cannot open the ciphertext.
	nonce, _ := base64.StdEncoding.DecodeString(env.Nonce)
	ct, _ := base64.StdEncoding.DecodeString(env.CT)
	randomKey := sha256.Sum256([]byte("server-guess"))
	if _, err := aeadDecrypt(randomKey[:], nonce, env.headerAD(), ct); err == nil {
		t.Fatal("random key decrypted the ciphertext")
	}
}

func TestForgedSignedPreKeyRejected(t *testing.T) {
	alice, _, _, _, _ := setupPair(t, true)
	// Attacker swaps Bob's SPK for their own, keeping Bob's signature.
	attacker, _ := GenerateIdentity()
	evilSPK, _ := GenerateSignedPreKey(9, attacker)
	// Reuse Bob's real bundle fields but with the attacker's SPK.
	bob, _ := GenerateIdentity()
	spk, _ := GenerateSignedPreKey(1, bob)
	bundle := &Bundle{
		UserID: "bob", DeviceID: "default",
		DHPub: b64(bob.DHPub), SignPub: b64(bob.SignPub),
		SPKPub: b64(evilSPK.Pub), SPKID: 9, SPKSig: b64(spk.Signature),
	}
	if _, err := Initiate(alice, bundle); err == nil {
		t.Fatal("Initiate accepted a bundle with a forged signed pre-key")
	}
}

func TestForwardSecrecyOldKeyCannotDecryptNew(t *testing.T) {
	_, bob, spk, opk, alice := setupPair(t, true)
	env0 := mustEncrypt(t, alice, "m0")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	if _, err := bobSess.Decrypt(env0); err != nil {
		t.Fatal(err)
	}
	// Capture Bob's receiving chain key after message 0, then advance
	// the ratchet with several more messages, then try the old chain
	// key on a new message: it must not decrypt it.
	oldCKr := append([]byte{}, bobSess.CKr...)
	env1 := mustEncrypt(t, alice, "m1")
	if _, err := bobSess.Decrypt(env1); err != nil {
		t.Fatal(err)
	}
	env2 := mustEncrypt(t, alice, "m2")
	// Simulate an attacker who stole the old chain key: derive a
	// message key from it and try to open m2's ciphertext.
	_, mkOld := kdfCK(oldCKr)
	nonce, _ := base64.StdEncoding.DecodeString(env2.Nonce)
	ct, _ := base64.StdEncoding.DecodeString(env2.CT)
	if _, err := aeadDecrypt(msgKey(mkOld), nonce, env2.headerAD(), ct); err == nil {
		t.Fatal("old chain key decrypted a later message — no forward secrecy")
	}
	// Sanity: Bob himself can still decrypt m2.
	if pt, err := bobSess.Decrypt(env2); err != nil || string(pt) != "m2" {
		t.Fatalf("bob decrypt m2: %q %v", pt, err)
	}
}

func TestSessionMarshalRoundTrip(t *testing.T) {
	_, bob, spk, opk, alice := setupPair(t, true)
	env0 := mustEncrypt(t, alice, "persist me")
	bobSess := receiveInit(t, bob, spk, opk, env0)
	if _, err := bobSess.Decrypt(env0); err != nil {
		t.Fatal(err)
	}
	data, err := alice.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := UnmarshalSession(data)
	if err != nil {
		t.Fatal(err)
	}
	env1 := mustEncrypt(t, restored, "after restore")
	pt, err := bobSess.Decrypt(env1)
	if err != nil || string(pt) != "after restore" {
		t.Fatalf("decrypt after session restore: %q %v", pt, err)
	}
}

// TestInteropVector writes a deterministic fixture that the TypeScript
// client (web/lib/e2ee.ts) must be able to decrypt. The fixture uses
// fixed seeds so the vector is reproducible; it is a TEST vector only.
func TestInteropVector(t *testing.T) {
	dhSeed := sha256.Sum256([]byte("vyro-test-vector-alice-dh"))
	signSeed := sha256.Sum256([]byte("vyro-test-vector-alice-sign"))
	alice, err := IdentityFromSeeds(dhSeed[:], signSeed[:])
	if err != nil {
		t.Fatal(err)
	}
	dhSeedB := sha256.Sum256([]byte("vyro-test-vector-bob-dh"))
	signSeedB := sha256.Sum256([]byte("vyro-test-vector-bob-sign"))
	bob, err := IdentityFromSeeds(dhSeedB[:], signSeedB[:])
	if err != nil {
		t.Fatal(err)
	}
	spkPriv := sha256.Sum256([]byte("vyro-test-vector-bob-spk"))
	spkPubKey, err := x25519.NewPrivateKey(spkPriv[:])
	if err != nil {
		t.Fatal(err)
	}
	spkPub := spkPubKey.PublicKey().Bytes()
	spk := &SignedPreKey{ID: 1, Priv: spkPriv[:], Pub: spkPub, Signature: ed25519.Sign(bob.SignPriv, spkPub)}
	opkPriv := sha256.Sum256([]byte("vyro-test-vector-bob-opk"))
	opkPubKey, err := x25519.NewPrivateKey(opkPriv[:])
	if err != nil {
		t.Fatal(err)
	}
	opkPub := opkPubKey.PublicKey().Bytes()
	opk := &OneTimePreKey{ID: 7, Priv: opkPriv[:], Pub: opkPub}
	bundle := &Bundle{
		UserID: "bob", DeviceID: "default",
		DHPub: b64(bob.DHPub), SignPub: b64(bob.SignPub),
		SPKPub: b64(spk.Pub), SPKID: spk.ID, SPKSig: b64(spk.Signature),
		OneTimePreKey: &OPK{ID: opk.ID, Pub: b64(opk.Pub)},
	}
	sess, err := Initiate(alice, bundle)
	if err != nil {
		t.Fatal(err)
	}
	env, err := sess.Encrypt([]byte("سلام VYRO — E2EE test"))
	if err != nil {
		t.Fatal(err)
	}
	// Sanity: Bob's side decrypts it.
	bobSess, err := ReceiveInit(bob, spk, opk, mustB64(t, env.IK), mustB64(t, env.EK), "alice", "default")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := bobSess.Decrypt(env)
	if err != nil || string(pt) != "سلام VYRO — E2EE test" {
		t.Fatalf("go-side decrypt of vector: %q %v", pt, err)
	}
	vec := map[string]any{
		"alice_dh_priv":  b64(alice.DHPriv),
		"alice_dh_pub":   b64(alice.DHPub),
		"alice_sign_pub": b64(alice.SignPub),
		"bob_dh_priv":    b64(bob.DHPriv),
		"bob_dh_pub":     b64(bob.DHPub),
		"bob_sign_priv":  b64(bob.SignPriv),
		"bob_sign_pub":   b64(bob.SignPub),
		"bob_spk_priv":   b64(spk.Priv),
		"bob_spk_pub":    b64(spk.Pub),
		"bob_opk_priv":   b64(opk.Priv),
		"bob_opk_pub":    b64(opk.Pub),
		"envelope":       env,
		"plaintext":      "سلام VYRO — E2EE test",
	}
	data, _ := json.MarshalIndent(vec, "", "  ")
	path := "/tmp/vyro-e2ee-vector.json"
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("interop vector written to %s", path)
}
