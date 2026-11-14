# VYRO — Security Design

> **Status (v0.5, honest):** E2EE for 1:1 DMs is implemented and tested
> (see "End-to-End Encryption (v0.5)" below) — but it is **VYRO E2EE v1**,
> a Signal-style X3DH + Double Ratchet construction built from audited
> primitives, **not** libsignal, **not** Signal-interoperable, and **not**
> independently audited as a protocol. Do not market VYRO as "Signal" or
> as "audited end-to-end encryption". Group E2EE does not exist yet.
>
> **Messages in 1:1 DMs with E2EE turned on are ciphertext on the
> server.** Plaintext messages (E2EE off), group messages, and all
> metadata (sender, chat, timestamp) are still visible to the server.

## Implemented in v0.1

| Control | How |
|---|---|
| Password storage | bcrypt, cost 12, via `golang.org/x/crypto/bcrypt`. Plaintext passwords are never stored, logged, or returned. Minimum 8 characters (`internal/auth/auth.go`). |
| Authentication | JWT HS256, `golang-jwt/jwt/v5`, signed with `JWT_SECRET` from the environment. Expiry 24 h by default (`JWT_EXPIRY_HOURS`). Tokens are validated on every protected route (`internal/middleware/auth.go`). |
| Login rate limiting | In-memory fixed window: 5 failed attempts per 15 min per (IP + email). Returns HTTP 429. Successful login resets the counter (`internal/auth/ratelimit.go`). Since v0.4 a Redis-backed limiter (`internal/auth/redislimiter.go`, go-redis v9.8, same `Limiter` interface) is used when `REDIS_URL` is set and Redis answers PING, with automatic fail-open to the in-memory limiter on Redis errors. **Tested against real Redis 7.0.15 in v0.5** (`TestRedisLimiterLive`; go-redis pinned to v9.8 because older versions send `CLIENT SETINFO`, which Redis < 7.2 rejects). |
| Email verification (v0.7) | Registration and login both finish with a random 6-digit code emailed to the user (`POST /api/auth/register` → `verify-register`, `POST /api/auth/login` → `verify-login`; `POST /api/auth/resend-code` re-sends). Codes expire after 10 minutes, are single-use, allow max 5 failed attempts (the 5th returns 429 and invalidates the code), and resends are capped at one per 60 s per (email, purpose). **Only the SHA-256 hash of a code is stored** (`email_codes.code_hash`); the plain code exists only in the email and is never logged or returned by the API. Pending registrations (`pending_users`) hold an already bcrypt-hashed password and are deleted on verification — no user row exists before the code is confirmed. Mail is sent via plain `net/smtp` (STARTTLS, Gmail relay by default, `SMTP_*` env); when no SMTP is configured the endpoints return 503 and never fake a code. Tested in `internal/handlers/verify_test.go` (happy path, wrong/expired code, 5-attempt invalidation, resend cooldown + invalidation, 503 without mailer) and live against a local SMTP sink (`backend/scripts/smtp-sink`). |
| Group/channel admin enforcement (v0.4) | Roles `owner`/`admin`/`member` are enforced in handlers, not just in the schema: only owner/admin can add/remove members, only the owner can change roles, the owner cannot be removed or demote themselves. Tested in `handlers_v04_test.go` (TestV04Admin). |
| Bot tokens (v0.4) | A bot token is returned exactly once at creation; only its SHA-256 hash is stored (`bots.token_hash`). Bot API calls authenticate with `X-Bot-Token`. Listing bots never leaks the token. |
| Disappearing messages (v0.4) | `ttl_seconds` (60..604800) sets `expires_at`; expired messages are hidden from lists/search and soft-deleted by `PurgeExpired`. This is server-side deletion, not cryptographic erasure — copies on other devices are out of scope. |
| Authorisation | Membership checks on every chat/message route: a user can only read, send, edit or react in chats they belong to; only the sender can edit/delete their own message. Non-members get 403 (tested in `handlers_test.go`). |
| Transport | HTTPS in production (reverse proxy / TLS termination — see ARCHITECTURE.md). The dev server listens on plain HTTP on localhost only. |
| Input validation | Email regex, username 3–32 chars, message max 10 000 chars, chat-type allow-list, `kind` allow-list (only `text` accepted in v0.1). |
| Secrets | No secrets in the repo. `.env.example` documents variables; `.env` is git-ignored. The dev JWT default is logged as a warning when unset. |
| Web Push privacy (v0.5) | Push payloads never contain message content — only "New message from <name>" (or "New encrypted message" for E2EE). The push service therefore never sees plaintext (`internal/handlers/v05.go`, `internal/push/push.go`). Subscriptions that the push service reports gone (410/404) are removed. |
| Call signaling (v0.5) | WebRTC offer/answer/ICE are relayed verbatim through the WebSocket room of the chat; the server never inspects or stores them, and only room members receive them (`internal/ws/hub.go`). Media is peer-to-peer (STUN, plus the self-hosted coturn TURN relay wired in v0.6). |

## End-to-End Encryption — VYRO E2EE v1 (implemented in v0.5)

**Scope: 1:1 DMs only, one device per user (device `"default"`).**
Group E2EE does not exist; group messages are plaintext on the server.

### What it is

A Signal-*style* construction — X3DH for session establishment plus
the Double Ratchet for ongoing messages — implemented in Go
(`backend/internal/e2ee/e2ee.go`) and in TypeScript/WebCrypto
(`web/lib/e2ee.ts`) from the same specification. It uses **only
established, audited primitives**, never custom ciphers:

| Piece | Primitive |
|---|---|
| Key agreement | X25519 (`crypto/ecdh` in Go; WebCrypto `X25519` in the browser) |
| Key derivation | HKDF-SHA256 + HMAC-SHA256 (`golang.org/x/crypto/hkdf`; WebCrypto HMAC, HKDF implemented over it so salt/info handling matches Go exactly) |
| Message encryption | AES-256-GCM (`crypto/aes`/`cipher`; WebCrypto `AES-GCM`), 12-byte random nonce, ratchet header `{dh,pn,n}` as AEAD associated data |
| Signed pre-key | Ed25519 signature over the signed pre-key public key (`crypto/ed25519`; WebCrypto `Ed25519`) |

It is **not libsignal**, **not Signal-interoperable**, and the protocol
as a whole has **not been independently audited**. The Go and TS
implementations were proven against each other (see "Interop proof").

### Protocol (both sides identical)

1. **Identity.** Each device generates an X25519 identity keypair and
   a separate Ed25519 signing keypair on first run. Public parts are
   uploaded (`POST /api/keys/identity`); private parts never leave the
   device. The signed pre-key's public part is signed with the Ed25519
   key; peers verify the signature (`Bundle.Verify`) before use.
2. **One-time pre-keys.** Each device uploads a batch (web client: 20).
   The server hands each one out exactly once
   (`GET /api/keys/{userId}/bundle` claims one atomically). **v0.5 fix:**
   re-publishing a key id *replaces* the stored public key — a device
   that reinstalls and re-publishes must never have its old, now
   undecryptable public key handed out (`TestPreKeyRepublish`).
3. **X3DH.** `DH1=DH(IK_A,SPK_B)`, `DH2=DH(EK_A,IK_B)`,
   `DH3=DH(EK_A,SPK_B)`, `DH4=DH(EK_A,OPK_B)` when a one-time pre-key
   was available. `SK = HKDF(salt=32 zero bytes, DH1‖DH2‖DH3‖DH4,
   info="vyro-x3dh-v1", 64)`; `RK=SK[:32]`, `CKr=SK[32:]`.
4. **Double Ratchet.** `KDF_RK = HKDF(salt=RK, DH output,
   info="vyro-ratchet-v1", 64)`; `KDF_CK: CK′=HMAC(CK,0x01),
   MK=HMAC(CK,0x02)`; message key `= HKDF(salt=MK, info="vyro-msg-v1",
   32)`. Skipped (out-of-order) message keys are capped at 1000.
5. **Envelope** (stored in `Message.Content` for `kind="e2ee"`, JSON,
   base64 fields): `{v:1, ik, ek, opk, dh, pn, n, nonce, ct}`;
   `ik`/`ek`/`opk` only on the first (X3DH initiation) message. The
   server validates the envelope shape (`ValidateEnvelope`) and stores
   it; it never decrypts, logs, or searches it, and search excludes
   `kind='e2ee'`. Encrypted messages cannot be edited (delete + resend)
   and cannot be scheduled.
6. **Safety Number.** `Fingerprint(ikA, ikB)` = first 8 bytes hex of
   `SHA256(ikA‖ikB)`, shown in the web client when E2EE is on for a DM.
   A changed fingerprint means a changed key — verify out-of-band.

### Key management rules

- Private keys: device-local only, never sent to the server, no key
  escrow, no backdoor. Losing all devices means losing E2EE history.
- Web client stores private keys in `localStorage` (dev-grade — any
  script on the origin can read it). A production client must use
  non-extractable WebCrypto keys or the OS keystore. The UI does not
  hide this: the README states it.
- Sessions are per-device; multi-device (one session per device) is not
  implemented — a user has one device, `"default"`.
- The server never sees plaintext for E2EE DMs, but it still sees
  metadata (sender, chat, timestamp) and the *fact* that a message is
  encrypted.

### Interop proof (2026-11-12)

- Go test vectors: `TestServerCannotRead`, tamper/wrong-key failures,
  forward secrecy, out-of-order delivery
  (`backend/internal/e2ee/e2ee_test.go`).
- Cross-implementation: the Go-produced vector
  (`web/lib/e2ee-vector.json`) decrypts in the TypeScript client, and a
  TypeScript-produced envelope decrypts in Go
  (`web/lib/e2ee.interop.cjs`; run it per the header comment). Both
  directions were verified live against the real server too
  (`backend/scripts/live-test.cjs` — the server stores ciphertext only,
  Alice decrypts it).
- Live E2EE flow test over HTTP: `TestV05KeyBundleAndE2EEFlow`
  (`backend/internal/handlers/handlers_v05_test.go`).

### Known limits (honest)

- Not libsignal, not Signal-interoperable, no independent audit.
- 1:1 DMs only; no group E2EE (no Sender Keys yet).
- One device per user; no multi-device sessions, no key-change warnings
  in the UI beyond the Safety Number display.
- WebCrypto X25519 needs a current browser (Chrome 137+, Safari 26+,
  Firefox 144+); older browsers get E2EE disabled, never silent
  plaintext. The client detects support (`e2eeSupported()`).
- Forward secrecy holds per message key; the Double Ratchet gives
  future secrecy only when both sides keep exchanging messages.

## Other planned controls (not yet implemented)

- **Spam / abuse:** per-user send rate limits (Redis), report flow
  (`reports` table exists), block flow (implemented in the store and
  API for DMs), CAPTCHA-free proof-of-work is *not* planned; use
  rate limits + reports + email verification (implemented in v0.7:
  6-digit codes, hash-only storage — see the table above).
- **File uploads:** size/type allow-list, random object names,
  no executable serving. Implemented in v0.3 (`POST /api/uploads`,
  `GET /uploads/{name}`) — but files are NOT scanned for malware yet;
  do not expose the endpoint to untrusted users without a scanner.
- **Audit / logging:** never log message contents, passwords, tokens,
  keys, or verification codes. Log auth failures (without passwords)
  and rate-limit hits.
- **Dependencies:** keep `go.mod` pinned; run `go vet`, `govulncheck`
  (when available) in CI before any release.
- **Backups:** PostgreSQL backups must be encrypted at rest; the DB
  holds ciphertext in the E2EE build, but user metadata still needs
  access control.

## Threat model (summary)

| Attacker | Mitigation |
|---|---|
| Passive network observer | TLS + E2EE for 1:1 DMs (v0.5) |
| Compromised / curious server | E2EE for 1:1 DMs (v0.5); server still sees metadata and all non-E2EE content |
| Stolen database | bcrypt passwords; E2EE ciphertext is unreadable without device keys; verification codes are stored as SHA-256 hashes only, so a stolen `email_codes` table cannot be used to log in |
| Brute-force login | Rate limiting (implemented, Redis-backed when available) + strong passwords + email verification codes (v0.7: a stolen password alone no longer yields a token) |
| Stolen device | OS keystore + app lock (planned); remote session revoke (planned). Web client keys in localStorage are dev-grade — see above. |
| Malicious client / spam | Rate limits, blocks, reports (partially implemented) |

## What this document is not

It is not a security audit, and VYRO v0.5 must not be marketed as
"audited end-to-end encryption" — the E2EE construction is Signal-style
but not libsignal, not Signal-interoperable, and not independently
reviewed. Group chats are not end-to-end encrypted at all.
