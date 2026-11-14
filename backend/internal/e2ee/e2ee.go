// Package e2ee implements VYRO E2EE v1 for 1:1 DMs.
//
// Protocol (documented in docs/SECURITY.md — read it before changing this):
// X3DH session establishment + Double Ratchet, exactly the Signal
// protocol's construction, but implemented here over audited primitives
// only: X25519 (crypto/ecdh), HKDF-SHA256 + HMAC-SHA256
// (golang.org/x/crypto/hkdf, crypto/hmac), AES-256-GCM (crypto/aes,
// crypto/cipher) and Ed25519 signatures (crypto/ed25519) for the signed
// pre-key. This is NOT libsignal and NOT Signal-interoperable: no
// libsignal code is used, the wire format is VYRO's own, and the
// protocol has not been independently audited. Do not market it as
// "Signal Protocol" — call it "VYRO E2EE v1".
//
// Scope: 1:1 DMs, one device per user (device "default"). Group E2EE
// (Sender Keys) is NOT implemented. The server stores and relays only
// the ciphertext envelope; it never sees plaintext and cannot decrypt
// it (proven by TestServerCannotRead in e2ee_test.go).
//
// Private keys never leave the device. The server only ever receives
// public keys (identity DH + signing keys, signed pre-key, one-time
// pre-keys) via the existing /api/keys endpoints.
package e2ee

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/hkdf"
)

const (
	// ProtocolVersion is the envelope version. Bump on any wire change.
	ProtocolVersion = 1
	// MaxSkippedKeys bounds memory used by out-of-order message keys.
	MaxSkippedKeys = 1000
)

var x25519 = ecdh.X25519()

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
func unb64(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty base64 field")
	}
	return base64.StdEncoding.DecodeString(s)
}

// ---------------------------------------------------------------- keys

// Identity is a device's long-term key material. DH is the X3DH identity
// key (X25519); Sign is a separate Ed25519 key used only to sign the
// signed pre-key (X25519 keys cannot sign).
type Identity struct {
	DHPriv   []byte // 32 bytes X25519 private
	DHPub    []byte // 32 bytes X25519 public
	SignPriv ed25519.PrivateKey
	SignPub  ed25519.PublicKey
}

// GenerateIdentity creates a fresh identity. Private keys are returned
// to the caller (the device); they must be stored in the OS keystore /
// non-extractable WebCrypto key, never uploaded.
func GenerateIdentity() (*Identity, error) {
	dhPriv, err := x25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	dhPub := dhPriv.PublicKey().Bytes()
	_, signPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Identity{
		DHPriv: dhPriv.Bytes(), DHPub: dhPub,
		SignPriv: signPriv, SignPub: signPriv.Public().(ed25519.PublicKey),
	}, nil
}

// IdentityFromSeeds builds an identity deterministically from 32-byte
// seeds. Used ONLY for test vectors / interop fixtures — never in prod.
func IdentityFromSeeds(dhSeed, signSeed []byte) (*Identity, error) {
	if len(dhSeed) != 32 || len(signSeed) != 32 {
		return nil, errors.New("seeds must be 32 bytes")
	}
	dhPriv, err := x25519.NewPrivateKey(dhSeed)
	if err != nil {
		return nil, err
	}
	dhPub := dhPriv.PublicKey().Bytes()
	signPriv := ed25519.NewKeyFromSeed(signSeed)
	return &Identity{
		DHPriv: dhPriv.Bytes(), DHPub: dhPub,
		SignPriv: signPriv, SignPub: signPriv.Public().(ed25519.PublicKey),
	}, nil
}

// SignedPreKey is a medium-term DH keypair whose public part is signed
// with the identity signing key so peers can verify it came from the
// identity owner.
type SignedPreKey struct {
	ID        int
	Priv      []byte // 32 bytes X25519 private (device only)
	Pub       []byte // 32 bytes X25519 public
	Signature []byte // Ed25519 signature over Pub
}

func GenerateSignedPreKey(id int, ident *Identity) (*SignedPreKey, error) {
	priv, err := x25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := priv.PublicKey().Bytes()
	return &SignedPreKey{
		ID: id, Priv: priv.Bytes(), Pub: pub,
		Signature: ed25519.Sign(ident.SignPriv, pub),
	}, nil
}

// OneTimePreKey is a single-use DH keypair. The server hands each public
// key out exactly once, then it must never be reused.
type OneTimePreKey struct {
	ID   int
	Priv []byte
	Pub  []byte
}

func GenerateOneTimePreKey(id int) (*OneTimePreKey, error) {
	priv, err := x25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := priv.PublicKey().Bytes()
	return &OneTimePreKey{ID: id, Priv: priv.Bytes(), Pub: pub}, nil
}

// Bundle is the public key material a sender fetches to start a session.
// It is exactly what GET /api/keys/{userId}/bundle returns.
type Bundle struct {
	UserID        string `json:"user_id"`
	DeviceID      string `json:"device_id"`
	DHPub         string `json:"dh_pub"`          // identity DH public key, base64
	SignPub       string `json:"sign_pub"`        // Ed25519 public key, base64
	SPKPub        string `json:"spk_pub"`         // signed pre-key public, base64
	SPKID         int    `json:"spk_id"`          // signed pre-key id
	SPKSig        string `json:"spk_sig"`         // Ed25519 signature over SPK pub, base64
	OneTimePreKey *OPK   `json:"one_time_prekey"` // nil when exhausted
}

// OPK is one one-time pre-key as returned in a bundle.
type OPK struct {
	ID  int    `json:"id"`
	Pub string `json:"pub"`
}

// Verify checks the signed pre-key signature against the identity
// signing key. A bundle that fails verification must never be used.
func (b *Bundle) Verify() error {
	signPub, err := unb64(b.SignPub)
	if err != nil || len(signPub) != ed25519.PublicKeySize {
		return errors.New("invalid sign_pub")
	}
	spk, err := unb64(b.SPKPub)
	if err != nil || len(spk) != 32 {
		return errors.New("invalid spk_pub")
	}
	sig, err := unb64(b.SPKSig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("invalid spk_sig")
	}
	if !ed25519.Verify(ed25519.PublicKey(signPub), spk, sig) {
		return errors.New("signed pre-key signature verification failed")
	}
	return nil
}

// ------------------------------------------------------------- primitives

func dh(priv, pub []byte) ([]byte, error) {
	privKey, err := x25519.NewPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	pubKey, err := x25519.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return privKey.ECDH(pubKey)
}

func hkdfDerive(salt, ikm, info []byte, length int) []byte {
	r := hkdf.New(sha256.New, ikm, salt, info)
	out := make([]byte, length)
	if _, err := r.Read(out); err != nil {
		panic(err) // hkdf.Read only fails on bad length
	}
	return out
}

func hmacSHA256(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return h.Sum(nil)
}

func newDHKeypair() (priv, pub []byte, err error) {
	k, err := x25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	pub = k.PublicKey().Bytes()
	return k.Bytes(), pub, nil
}

// kdfRK is the Double Ratchet root-key KDF: (RK, CK) = HKDF(RK, DH-output).
func kdfRK(rk, dhOut []byte) (newRK, ck []byte) {
	out := hkdfDerive(rk, dhOut, []byte("vyro-ratchet-v1"), 64)
	return out[:32], out[32:]
}

// kdfCK is the chain-key KDF: CK' = HMAC(CK, 0x01), MK = HMAC(CK, 0x02).
func kdfCK(ck []byte) (nextCK, mk []byte) {
	return hmacSHA256(ck, []byte{0x01}), hmacSHA256(ck, []byte{0x02})
}

// msgKey derives the AES-256 key for one message key.
func msgKey(mk []byte) []byte {
	return hkdfDerive(mk, nil, []byte("vyro-msg-v1"), 32)
}

func aeadEncrypt(key, nonce, ad, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, ad), nil
}

func aeadDecrypt(key, nonce, ad, ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, ad)
}

// ---------------------------------------------------------------- envelope

// Envelope is the wire format stored in Message.Content for kind "e2ee"
// (JSON, base64 fields). ik/ek/opk are present only on the first
// message of a session (the X3DH initiation message).
type Envelope struct {
	V     int    `json:"v"`
	IK    string `json:"ik,omitempty"`  // sender identity DH pub, b64
	EK    string `json:"ek,omitempty"`  // sender ephemeral pub, b64
	OPKID int    `json:"opk,omitempty"` // one-time pre-key id used (0 = none)
	DH    string `json:"dh"`            // sender ratchet pub, b64
	PN    int    `json:"pn"`            // previous chain message count
	N     int    `json:"n"`             // message number in this chain
	Nonce string `json:"nonce"`         // AES-GCM nonce, b64
	CT    string `json:"ct"`            // AES-GCM ciphertext, b64
}

// headerAD binds the ciphertext to its ratchet header (authenticated
// data for AES-GCM). Any tampering with dh/pn/n breaks decryption.
func (e *Envelope) headerAD() []byte {
	h := struct {
		DH string `json:"dh"`
		PN int    `json:"pn"`
		N  int    `json:"n"`
	}{DH: e.DH, PN: e.PN, N: e.N}
	b, _ := json.Marshal(h)
	return b
}

// ValidateEnvelope parses and sanity-checks an envelope string (what the
// server stores). It does NOT decrypt — the server cannot decrypt.
func ValidateEnvelope(content string) (*Envelope, error) {
	if len(content) == 0 || len(content) > 16384 {
		return nil, errors.New("envelope size out of range")
	}
	var env Envelope
	if err := json.Unmarshal([]byte(content), &env); err != nil {
		return nil, fmt.Errorf("envelope is not valid JSON: %w", err)
	}
	if env.V != ProtocolVersion {
		return nil, fmt.Errorf("unsupported envelope version %d", env.V)
	}
	if env.PN < 0 || env.N < 0 {
		return nil, errors.New("negative pn/n")
	}
	dh, err := unb64(env.DH)
	if err != nil || len(dh) != 32 {
		return nil, errors.New("invalid dh")
	}
	nonce, err := unb64(env.Nonce)
	if err != nil || len(nonce) != 12 {
		return nil, errors.New("invalid nonce")
	}
	ct, err := unb64(env.CT)
	if err != nil || len(ct) == 0 {
		return nil, errors.New("invalid ct")
	}
	for _, f := range []string{env.IK, env.EK} {
		if f != "" {
			b, err := unb64(f)
			if err != nil || len(b) != 32 {
				return nil, errors.New("invalid ik/ek")
			}
		}
	}
	return &env, nil
}

// ------------------------------------------------------------- session

// Session is one Double Ratchet session with one peer. It holds secret
// key material — serialize it only to device-local storage, never send
// it to the server. Marshal/Unmarshal use the JSON form below.
type Session struct {
	PeerUserID string `json:"peer_user_id"`
	PeerDevice string `json:"peer_device"`
	// Ratchet state.
	DHsPriv []byte `json:"dhs_priv"`
	DHsPub  []byte `json:"dhs_pub"`
	DHrPub  []byte `json:"dhr_pub"`
	RK      []byte `json:"rk"`
	CKs     []byte `json:"cks,omitempty"` // nil = needs a ratchet step
	CKr     []byte `json:"ckr,omitempty"`
	Ns      int    `json:"ns"`
	Nr      int    `json:"nr"`
	PN      int    `json:"pn"`
	// Skipped message keys, keyed "base64(dh):n".
	Skipped map[string]string `json:"skipped,omitempty"`
	// First is true until this session's first message is sent (the
	// X3DH initiation message carries ik/ek/opk).
	First bool `json:"first,omitempty"`
	// EKPub / OPKID / OwnIKPub are the X3DH initiation values for the
	// first message (public data, safe to serialize with the session).
	EKPub    []byte `json:"ek_pub,omitempty"`
	OPKID    int    `json:"opk_id,omitempty"`
	OwnIKPub []byte `json:"own_ik_pub,omitempty"`
	// spkPriv is the receiver's signed-pre-key private key, kept only
	// until the first ratchet step replaces DHs. Serialized so a
	// session restored before the first reply still works.
	SPKPriv []byte `json:"spk_priv,omitempty"`
}

func (s *Session) skippedKey(dh []byte, n int) string {
	return b64(dh) + ":" + fmt.Sprint(n)
}

// Initiate runs X3DH as the sender and returns a session ready to
// encrypt its first message. own is the sender's identity; ek is a
// fresh ephemeral keypair generated by the caller (or nil to generate
// one here). bundle must already be verified (Bundle.Verify).
func Initiate(own *Identity, peer *Bundle) (*Session, error) {
	if err := peer.Verify(); err != nil {
		return nil, err
	}
	ekPriv, ekPub, err := newDHKeypair()
	if err != nil {
		return nil, err
	}
	peerIK, err := unb64(peer.DHPub)
	if err != nil {
		return nil, err
	}
	peerSPK, err := unb64(peer.SPKPub)
	if err != nil {
		return nil, err
	}
	dh1, err := dh(own.DHPriv, peerSPK)
	if err != nil {
		return nil, err
	}
	dh2, err := dh(ekPriv, peerIK)
	if err != nil {
		return nil, err
	}
	dh3, err := dh(ekPriv, peerSPK)
	if err != nil {
		return nil, err
	}
	secret := append(append(append([]byte{}, dh1...), dh2...), dh3...)
	opkID := 0
	if peer.OneTimePreKey != nil {
		opkPub, err := unb64(peer.OneTimePreKey.Pub)
		if err != nil {
			return nil, err
		}
		dh4, err := dh(ekPriv, opkPub)
		if err != nil {
			return nil, err
		}
		secret = append(secret, dh4...)
		opkID = peer.OneTimePreKey.ID
	}
	zero := make([]byte, 32)
	sk := hkdfDerive(zero, secret, []byte("vyro-x3dh-v1"), 64)
	dhsPriv, dhsPub, err := newDHKeypair()
	if err != nil {
		return nil, err
	}
	s := &Session{
		PeerUserID: peer.UserID, PeerDevice: peer.DeviceID,
		DHsPriv: dhsPriv, DHsPub: dhsPub, DHrPub: peerSPK,
		RK: sk[:32], CKr: sk[32:], First: true,
		Skipped: map[string]string{},
		EKPub:   ekPub, OPKID: opkID, OwnIKPub: own.DHPub,
	}
	return s, nil
}

// ReceiveInit runs X3DH as the receiver for the first message of a
// session. own is the receiver's identity, spk is its signed pre-key
// (private part needed for the first ratchet step), opk is the
// one-time pre-key the sender used (nil if none was used — the sender
// tells us via envelope.OPKID; the caller looks the private key up).
// senderIK and senderEK come from the envelope.
func ReceiveInit(own *Identity, spk *SignedPreKey, opk *OneTimePreKey, senderIK, senderEK []byte, peerUserID, peerDevice string) (*Session, error) {
	dh1, err := dh(spk.Priv, senderIK)
	if err != nil {
		return nil, err
	}
	dh2, err := dh(own.DHPriv, senderEK)
	if err != nil {
		return nil, err
	}
	dh3, err := dh(spk.Priv, senderEK)
	if err != nil {
		return nil, err
	}
	secret := append(append(append([]byte{}, dh1...), dh2...), dh3...)
	if opk != nil {
		dh4, err := dh(opk.Priv, senderEK)
		if err != nil {
			return nil, err
		}
		secret = append(secret, dh4...)
	}
	zero := make([]byte, 32)
	sk := hkdfDerive(zero, secret, []byte("vyro-x3dh-v1"), 64)
	return &Session{
		PeerUserID: peerUserID, PeerDevice: peerDevice,
		DHsPriv: spk.Priv, DHsPub: spk.Pub,
		RK: sk[:32], CKr: sk[32:], First: false,
		Skipped: map[string]string{}, SPKPriv: spk.Priv,
	}, nil
}

// Encrypt encrypts one message. On the first call of a session the
// envelope carries ik/ek/opk (the X3DH initiation message).
func (s *Session) Encrypt(plaintext []byte) (*Envelope, error) {
	if s.CKs == nil {
		// Ratchet step: new sending chain from the current DH pair.
		dhOut, err := dh(s.DHsPriv, s.DHrPub)
		if err != nil {
			return nil, err
		}
		s.RK, s.CKs = kdfRK(s.RK, dhOut)
		s.PN = s.Nr
		s.Ns = 0
	}
	var ck, mk []byte
	ck, mk = kdfCK(s.CKs)
	s.CKs = ck
	header := Envelope{V: ProtocolVersion, DH: b64(s.DHsPub), PN: s.PN, N: s.Ns}
	s.Ns++
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct, err := aeadEncrypt(msgKey(mk), nonce, header.headerAD(), plaintext)
	if err != nil {
		return nil, err
	}
	header.Nonce = b64(nonce)
	header.CT = b64(ct)
	if s.First {
		header.IK = b64(s.OwnIKPub)
		header.EK = b64(s.EKPub)
		header.OPKID = s.OPKID
		s.First = false
	}
	return &header, nil
}

// Decrypt decrypts one envelope, advancing the ratchet as needed.
// It handles out-of-order messages via skipped message keys.
func (s *Session) Decrypt(env *Envelope) ([]byte, error) {
	if env.V != ProtocolVersion {
		return nil, fmt.Errorf("unsupported envelope version %d", env.V)
	}
	dhPub, err := unb64(env.DH)
	if err != nil || len(dhPub) != 32 {
		return nil, errors.New("invalid dh in envelope")
	}
	// Skipped message key?
	if mkB64, ok := s.Skipped[s.skippedKey(dhPub, env.N)]; ok {
		mk, err := unb64(mkB64)
		if err != nil {
			return nil, err
		}
		delete(s.Skipped, s.skippedKey(dhPub, env.N))
		return s.decryptWithKey(mk, env)
	}
	// New ratchet public key: skip the old receiving chain, then step.
	if !equalBytes(dhPub, s.DHrPub) {
		if err := s.skipTo(env.PN); err != nil {
			return nil, err
		}
		dhOut, err := dh(s.DHsPriv, dhPub)
		if err != nil {
			return nil, err
		}
		s.RK, s.CKr = kdfRK(s.RK, dhOut)
		s.DHrPub = dhPub
		// Fresh sending keypair for the next ratchet step.
		priv, pub, err := newDHKeypair()
		if err != nil {
			return nil, err
		}
		s.DHsPriv, s.DHsPub = priv, pub
		s.CKs = nil
		s.Nr = 0
	}
	// Advance the receiving chain to env.N, storing skipped keys.
	for s.Nr < env.N {
		var mk []byte
		s.CKr, mk = kdfCK(s.CKr)
		if len(s.Skipped) >= MaxSkippedKeys {
			return nil, errors.New("too many skipped message keys")
		}
		s.Skipped[s.skippedKey(s.DHrPub, s.Nr)] = b64(mk)
		s.Nr++
	}
	var mk []byte
	s.CKr, mk = kdfCK(s.CKr)
	s.Nr++
	return s.decryptWithKey(mk, env)
}

func (s *Session) decryptWithKey(mk []byte, env *Envelope) ([]byte, error) {
	nonce, err := unb64(env.Nonce)
	if err != nil || len(nonce) != 12 {
		return nil, errors.New("invalid nonce")
	}
	ct, err := unb64(env.CT)
	if err != nil {
		return nil, err
	}
	pt, err := aeadDecrypt(msgKey(mk), nonce, env.headerAD(), ct)
	if err != nil {
		return nil, errors.New("decryption failed (wrong key or tampered message)")
	}
	return pt, nil
}

// skipTo stores message keys for the current receiving chain up to n.
func (s *Session) skipTo(n int) error {
	for s.Nr < n {
		var mk []byte
		s.CKr, mk = kdfCK(s.CKr)
		if len(s.Skipped) >= MaxSkippedKeys {
			return errors.New("too many skipped message keys")
		}
		s.Skipped[s.skippedKey(s.DHrPub, s.Nr)] = b64(mk)
		s.Nr++
	}
	return nil
}

// Marshal serializes the session for device-local storage.
func (s *Session) Marshal() ([]byte, error) { return json.Marshal(s) }

// UnmarshalSession restores a session from Marshal output.
func UnmarshalSession(data []byte) (*Session, error) {
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Skipped == nil {
		s.Skipped = map[string]string{}
	}
	return &s, nil
}

// Fingerprint is the Safety Number for two identity keys: the first
// 16 hex chars of SHA256(ikA || ikB). Shown in the UI so users can
// verify out-of-band; a changed fingerprint means a changed key.
func Fingerprint(ikA, ikB []byte) string {
	h := sha256.Sum256(append(append([]byte{}, ikA...), ikB...))
	return fmt.Sprintf("%x", h[:8])
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
