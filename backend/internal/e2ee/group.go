// Group E2EE v0.6: Sender Keys for groups and channels, built on the
// v0.5 E2EE core (X3DH + Double Ratchet, X25519/HKDF/AES-256-GCM/
// Ed25519 — same primitives, same "VYRO E2EE v1" naming).
//
// Design (Signal-style Sender Keys, VYRO's own construction — not
// libsignal, not audited, not Signal-interoperable):
//
//   - Each device that sends to a group creates ONE sender key: a random
//     32-byte chain key plus an Ed25519 signing keypair. It encrypts
//     every group message by ratcheting that chain key forward
//     (next chain = HMAC(chain, 0x01), message key = HMAC(chain, 0x02),
//     same step as the 1:1 chain KDF)
//     and AES-256-GCM-encrypting under a key derived from the message
//     key, signing header+ciphertext with the signing key so members
//     can verify the sender and the server cannot forge messages.
//   - The sender key is distributed to every other member device by
//     encrypting it with the existing 1:1 Double Ratchet session to
//     that device (Session.Encrypt over Distribution.Marshal) and
//     uploading the resulting envelope to the server, which stores it
//     as opaque ciphertext (see models.GroupSenderKey). A new member's
//     device fetches the distributions addressed to it, decrypts them
//     with its 1:1 sessions, and can then decrypt group messages.
//   - When membership changes, each sender rotates its sender key
//     (NewSenderKey) and distributes the new one; messages under the
//     old key stay unreadable to anyone who never received it, and a
//     removed member never receives the new key, so they cannot read
//     messages sent after rotation. The server enforces the second
//     half by deleting distributions addressed to a removed member.
//
// The server never holds a sender key, a chain key, or a message key:
// it stores GroupEnvelope JSON (ciphertext + signature) and
// distribution envelopes only. TestServerCannotReadGroup in
// group_test.go proves the server-held bytes contain no plaintext and
// cannot be decrypted with server-side knowledge alone.
package e2ee

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
)

// GroupProtocolVersion is the group envelope version. Bump on any wire change.
const GroupProtocolVersion = 1

// MaxSkippedGroupKeys bounds memory used by out-of-order group message keys.
const MaxSkippedGroupKeys = 1000

// ------------------------------------------------------------ sender key

// SenderKey is a device's secret key material for one group. It never
// leaves the device except encrypted to a single recipient device via a
// 1:1 session (see Distribution). ChainKey ratchets forward with every
// message; SignPriv signs every envelope.
type SenderKey struct {
	KeyID    string // sender-chosen id for this generation (opaque, <= 64 chars)
	ChainKey []byte // 32 bytes, ratchets forward per message
	SignPriv ed25519.PrivateKey
	SignPub  ed25519.PublicKey
}

// NewSenderKey generates a fresh sender key for a group.
func NewSenderKey(keyID string) (*SenderKey, error) {
	if keyID == "" || len(keyID) > 64 {
		return nil, errors.New("key_id is required (max 64 chars)")
	}
	ck := make([]byte, 32)
	if _, err := rand.Read(ck); err != nil {
		return nil, err
	}
	_, signPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &SenderKey{
		KeyID: keyID, ChainKey: ck,
		SignPriv: signPriv, SignPub: signPriv.Public().(ed25519.PublicKey),
	}, nil
}

// Distribution is the sender-key material a recipient device needs:
// the chain key at the sender's current ratchet position and the
// sender's signing public key, plus the ratchet position (N) so the
// receiver can fast-forward. It is serialized by the sender, encrypted
// with the 1:1 session to the recipient device, and stored by the
// server as opaque ciphertext. The signing PRIVATE key is never
// distributed — only the sender holds it.
type Distribution struct {
	KeyID   string `json:"key_id"`
	ChainKey string `json:"chain_key"` // base64, 32 bytes
	SignPub  string `json:"sign_pub"`  // base64, Ed25519 public key
	N        int    `json:"n"`         // ratchet position of ChainKey
}

// Marshal serializes a sender key for distribution to one recipient
// device. The caller encrypts the bytes with its 1:1 session.
func (k *SenderKey) Marshal(n int) ([]byte, error) {
	return json.Marshal(Distribution{
		KeyID: k.KeyID, ChainKey: b64(k.ChainKey), SignPub: b64(k.SignPub), N: n,
	})
}

// UnmarshalDistribution parses distribution bytes a recipient decrypted
// from its 1:1 session with the sender.
func UnmarshalDistribution(data []byte) (*Distribution, error) {
	var d Distribution
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("invalid sender-key distribution: %w", err)
	}
	if d.KeyID == "" {
		return nil, errors.New("distribution has no key_id")
	}
	ck, err := unb64(d.ChainKey)
	if err != nil || len(ck) != 32 {
		return nil, errors.New("distribution has invalid chain_key")
	}
	sp, err := unb64(d.SignPub)
	if err != nil || len(sp) != ed25519.PublicKeySize {
		return nil, errors.New("distribution has invalid sign_pub")
	}
	if d.N < 0 {
		return nil, errors.New("distribution has negative n")
	}
	return &d, nil
}

// ---------------------------------------------------------- group envelope

// GroupEnvelope is the wire format stored in Message.Content for kind
// "e2ee" in a group/channel chat (JSON, base64 fields). One envelope
// decrypts on every member device that holds the sender key — the
// server stores and relays exactly one copy.
type GroupEnvelope struct {
	V            int    `json:"v"`
	KeyID        string `json:"key_id"`         // sender-key generation
	SenderUser   string `json:"sender_user"`    // sender user id
	SenderDevice string `json:"sender_device"`  // sender device id
	N            int    `json:"n"`              // message number in this sender chain
	Nonce        string `json:"nonce"`          // AES-GCM nonce, b64
	CT           string `json:"ct"`             // AES-GCM ciphertext, b64
	Sig          string `json:"sig"`            // Ed25519 signature over headerAD||nonce||ct, b64
}

// headerAD binds the ciphertext to its header (authenticated data for
// AES-GCM and part of the signed data). Any tampering with key_id /
// sender / n breaks decryption and signature verification.
func (e *GroupEnvelope) headerAD() []byte {
	h := struct {
		KeyID        string `json:"key_id"`
		SenderUser   string `json:"sender_user"`
		SenderDevice string `json:"sender_device"`
		N            int    `json:"n"`
	}{KeyID: e.KeyID, SenderUser: e.SenderUser, SenderDevice: e.SenderDevice, N: e.N}
	b, _ := json.Marshal(h)
	return b
}

func (e *GroupEnvelope) signedData() ([]byte, error) {
	nonce, err := unb64(e.Nonce)
	if err != nil {
		return nil, err
	}
	ct, err := unb64(e.CT)
	if err != nil {
		return nil, err
	}
	out := append(append([]byte{}, e.headerAD()...), nonce...)
	return append(out, ct...), nil
}

// ValidateGroupEnvelope parses and sanity-checks a group envelope
// string (what the server stores). It does NOT decrypt — the server
// cannot decrypt.
func ValidateGroupEnvelope(content string) (*GroupEnvelope, error) {
	if len(content) == 0 || len(content) > 16384 {
		return nil, errors.New("group envelope size out of range")
	}
	var env GroupEnvelope
	if err := json.Unmarshal([]byte(content), &env); err != nil {
		return nil, fmt.Errorf("group envelope is not valid JSON: %w", err)
	}
	if env.V != GroupProtocolVersion {
		return nil, fmt.Errorf("unsupported group envelope version %d", env.V)
	}
	if env.KeyID == "" || len(env.KeyID) > 64 {
		return nil, errors.New("invalid key_id")
	}
	if env.SenderUser == "" || env.SenderDevice == "" {
		return nil, errors.New("invalid sender")
	}
	if env.N < 0 {
		return nil, errors.New("negative n")
	}
	nonce, err := unb64(env.Nonce)
	if err != nil || len(nonce) != 12 {
		return nil, errors.New("invalid nonce")
	}
	ct, err := unb64(env.CT)
	if err != nil || len(ct) == 0 {
		return nil, errors.New("invalid ct")
	}
	sig, err := unb64(env.Sig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return nil, errors.New("invalid sig")
	}
	return &env, nil
}

// ------------------------------------------------------------- sender state

// GroupSender encrypts group messages with one sender key. It holds
// secret key material — serialize it only to device-local storage.
type GroupSender struct {
	SenderUserID   string
	SenderDeviceID string
	Key            *SenderKey
	N              int // next message number
}

// NewGroupSender creates a sender state from a fresh sender key.
func NewGroupSender(userID, deviceID string, key *SenderKey) *GroupSender {
	return &GroupSender{SenderUserID: userID, SenderDeviceID: deviceID, Key: key, N: 0}
}

// Encrypt encrypts one group message, ratcheting the chain key forward.
func (s *GroupSender) Encrypt(plaintext []byte) (*GroupEnvelope, error) {
	if s.Key == nil || len(s.Key.ChainKey) != 32 {
		return nil, errors.New("no sender key")
	}
	nextCK, mk := kdfCK(s.Key.ChainKey)
	s.Key.ChainKey = nextCK
	header := GroupEnvelope{
		V: GroupProtocolVersion, KeyID: s.Key.KeyID,
		SenderUser: s.SenderUserID, SenderDevice: s.SenderDeviceID, N: s.N,
	}
	s.N++
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
	signed, err := header.signedData()
	if err != nil {
		return nil, err
	}
	header.Sig = b64(ed25519.Sign(s.Key.SignPriv, signed))
	return &header, nil
}

// ----------------------------------------------------------- receiver state

// GroupReceiver decrypts group messages from one sender device under
// one sender-key generation. It holds the chain key (secret) and the
// sender's signing public key; build it from a Distribution the sender
// sent over a 1:1 session.
type GroupReceiver struct {
	SenderUserID   string
	SenderDeviceID string
	KeyID          string
	ChainKey       []byte // 32 bytes, ratchets forward per message
	SignPub        ed25519.PublicKey
	N              int // next expected message number
	Skipped        map[int]string // n -> base64 message key, for out-of-order delivery
}

// NewGroupReceiver builds receiver state from a decrypted distribution.
func NewGroupReceiver(senderUserID, senderDeviceID string, d *Distribution) (*GroupReceiver, error) {
	ck, err := unb64(d.ChainKey)
	if err != nil || len(ck) != 32 {
		return nil, errors.New("invalid chain_key")
	}
	sp, err := unb64(d.SignPub)
	if err != nil || len(sp) != ed25519.PublicKeySize {
		return nil, errors.New("invalid sign_pub")
	}
	return &GroupReceiver{
		SenderUserID: senderUserID, SenderDeviceID: senderDeviceID,
		KeyID: d.KeyID, ChainKey: ck, SignPub: ed25519.PublicKey(sp),
		N: d.N, Skipped: map[int]string{},
	}, nil
}

// Decrypt verifies the signature, then decrypts one envelope, advancing
// the chain as needed. Out-of-order messages are handled via skipped
// message keys. A message under a different key_id, sender, or with a
// bad signature is rejected before any decryption is attempted.
func (r *GroupReceiver) Decrypt(env *GroupEnvelope) ([]byte, error) {
	if env.V != GroupProtocolVersion {
		return nil, fmt.Errorf("unsupported group envelope version %d", env.V)
	}
	if env.KeyID != r.KeyID || env.SenderUser != r.SenderUserID || env.SenderDevice != r.SenderDeviceID {
		return nil, errors.New("envelope is not for this sender key")
	}
	sig, err := unb64(env.Sig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return nil, errors.New("invalid sig")
	}
	signed, err := env.signedData()
	if err != nil {
		return nil, errors.New("invalid envelope")
	}
	if !ed25519.Verify(r.SignPub, signed, sig) {
		return nil, errors.New("signature verification failed (wrong sender or tampered message)")
	}
	if mkB64, ok := r.Skipped[env.N]; ok {
		mk, err := unb64(mkB64)
		if err != nil {
			return nil, err
		}
		delete(r.Skipped, env.N)
		return r.decryptWithKey(mk, env)
	}
	if env.N < r.N {
		return nil, errors.New("message already decrypted (replay)")
	}
	for r.N < env.N {
		var mk []byte
		r.ChainKey, mk = kdfCK(r.ChainKey)
		if len(r.Skipped) >= MaxSkippedGroupKeys {
			return nil, errors.New("too many skipped group message keys")
		}
		r.Skipped[r.N] = b64(mk)
		r.N++
	}
	var mk []byte
	r.ChainKey, mk = kdfCK(r.ChainKey)
	r.N++
	return r.decryptWithKey(mk, env)
}

func (r *GroupReceiver) decryptWithKey(mk []byte, env *GroupEnvelope) ([]byte, error) {
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
