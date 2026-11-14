# VYRO — Architecture

Slogan: **Connect Beyond Limits**

## Overview

```
                 ┌─────────────┐   ┌─────────────┐
                 │  Flutter    │   │  Next.js    │
                 │  Android /  │   │  Web        │
                 │  iOS / Win /│   │  (web/)     │
                 │  macOS      │   │             │
                 │  (mobile/)  │   └──────┬──────┘
                 └──────┬──────┘          │
                        │  HTTPS + WSS    │
                        ▼                 ▼
                 ┌────────────────────────────────┐
                 │  Go backend (backend/)         │
                 │  REST + WebSocket hub          │
                 │  JWT auth · bcrypt · rate limit│
                 └──────┬───────────────┬─────────┘
                        │               │
                ┌───────▼──────┐  ┌─────▼─────┐
                │  PostgreSQL  │  │  Redis    │
                │  (migrations │  │  (future: │
                │   001_init)  │  │  limits / │
                │              │  │  presence)│
                └──────────────┘  └───────────┘
```

One backend serves all clients. Clients share the same REST + WebSocket
contract (`backend/internal/handlers`). Shared code is maximised by
having a single API; Flutter covers mobile + desktop, Next.js covers web.

## Components

| Component | Tech | Path | Status v0.1 |
|---|---|---|---|
| Backend API + WS | Go 1.23, stdlib `net/http`, `gorilla/websocket`, `golang-jwt/jwt/v5`, `x/crypto/bcrypt` | `backend/` | Working, tested (`go test ./...` passes) |
| Database | PostgreSQL 16 (schema ready) | `backend/migrations/001_init.sql` | Wired in v0.2: `PostgresStore` (`backend/internal/store/postgres.go`) — server uses it when `DATABASE_URL` is set |
| Cache / limits | Redis 7 (docker-compose) | `backend/docker-compose.yml` | Container defined; rate limiting is still in-memory |
| Web client | Next.js 14 (App Router), React, TypeScript | `web/` | Skeleton + working login/chat UI against the Go API (needs `npm install`) |
| Mobile/desktop | Flutter 3 (Dart) | `mobile/` | Skeleton + login/chat screens against the Go API (needs Flutter SDK) |
| Calls | WebRTC (STUN + self-hosted coturn TURN, v0.6) | `internal/ws/hub.go`, `web/lib/use-call.ts` | Signaling done + audio calls in web (v0.5); TURN client wired (v0.6); live media untested |
| E2EE | VYRO E2EE v1 (X3DH + Double Ratchet, audited primitives only) | `backend/internal/e2ee/`, `web/lib/e2ee.ts` | 1:1 DMs done (v0.5), proven interoperable; groups not done |
| Web Push | VAPID (RFC 8292) | `internal/push/`, `web/public/sw.js` | Done (v0.5); subscriptions in-memory; real browser push service untested |

## Backend layout

```
backend/
  cmd/server/main.go          # entry point, route table
  internal/
    config/                   # env config
    models/                   # User, Chat, Message, Reaction, Story
    store/                    # StoreIface + MemoryStore + PostgresStore
    auth/                     # bcrypt, JWT, RateLimiter
    middleware/               # JWT auth, CORS, JSON helpers
    handlers/                 # REST handlers (auth, users, chats, messages, keys, push)
    ws/                       # WebSocket hub (rooms + call signaling)
    e2ee/                     # VYRO E2EE v1: X3DH + Double Ratchet (v0.5)
    push/                     # Web Push / VAPID (v0.5)
  migrations/001_init.sql     # PostgreSQL schema (004_v05.sql adds E2EE sign-key columns)
  Dockerfile / docker-compose.yml / .env.example
```

Handlers depend only on `store.StoreIface`, so swapping
`MemoryStore` → `PostgresStore` needs no handler changes. That is the
deliberate seam for the next milestone.

## API contract (v0.5)

v0.5 adds: `GET /api/keys/{userId}/bundle` (X3DH bundle: identity +
signed pre-key + one claimed one-time pre-key), `POST
/api/chats/{id}/messages` with `kind="e2ee"` (ciphertext envelope;
editing/scheduling rejected), `GET /api/push/vapid-public-key`,
`POST/DELETE /api/push/subscribe`, `POST /api/push/test`, and
WebSocket `call-offer` / `call-answer` / `call-ice` / `call-end`
frames (relayed verbatim per chat room, optional `to_user_id`).
`VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` configure
push; without them the server generates an ephemeral pair per run.

## API contract (v0.4)

v0.4 adds: `POST /api/messages/{id}/read` (read receipts),
`GET /api/users/{id}/presence`, location/sticker kinds + `ttl_seconds`
on `POST /api/chats/{id}/messages`, scheduled messages
(`POST/GET /api/chats/{id}/scheduled`, `DELETE /api/scheduled/{id}`),
pin/unpin/archive/unarchive/folder on chats, member admin endpoints
(`GET/POST /api/chats/{id}/members`, `DELETE .../members/{userId}`,
`POST .../members/{userId}/role`), stories (`POST/GET /api/stories`),
bots (`POST/GET/DELETE /api/bots`, `POST /api/bots/{id}/webhook`,
`GET /api/bot/updates`, `POST /api/bot/send` with `X-Bot-Token`), and
E2EE key endpoints (`POST /api/keys/identity`, `GET
/api/keys/{userId}/{deviceId}`, `POST /api/keys/prekeys`,
`GET /api/keys/{userId}/prekey`). See README for the full table.

## API contract (v0.1)

Base: `http://localhost:8080`. Auth: `Authorization: Bearer <jwt>`.

| Method & path | Purpose |
|---|---|
| `GET /health` | Liveness |
| `POST /api/auth/register` | Register (email, username, password ≥ 8) → user + token |
| `POST /api/auth/login` | Login → token (5 tries / 15 min per IP+email, else 429) |
| `GET /api/users/me` | Current user |
| `GET /api/users?search=` | Find users |
| `POST /api/users/{id}/block` | Block a user |
| `GET /api/chats` · `POST /api/chats` | List / create dm / group / channel |
| `GET /api/chats/{id}/messages` | List messages (limit ≤ 100, offset) |
| `POST /api/chats/{id}/messages` | Send text message (broadcasts over WS) |
| `PUT /api/messages/{id}` | Edit own message |
| `DELETE /api/messages/{id}` | Delete own message (soft delete) |
| `POST/DELETE /api/messages/{id}/reactions` | Add / remove emoji reaction |
| `GET /api/messages/search?q=` | Search own chats |
| `GET /ws?token=` | WebSocket: `{"type":"message","chat_id":…,"message":…}`, `message_edited`, `message_deleted`, `presence` |

## Data model

See `migrations/001_init.sql`: `users`, `identity_keys`,
`one_time_prekeys`, `chats`, `chat_members`, `messages`,
`reactions`, `stories`, `blocks`, `reports`, `devices`.
`002_uploads.sql` adds upload columns on `messages`.
`003_v04.sql` (v0.4) adds `messages.lat/lng/location_label/expires_at`,
`users.is_bot`, and the tables `message_receipts`, `chat_settings`,
`scheduled_messages`, `bots` (token stored only as SHA-256 hash).
`004_v05.sql` (v0.5) adds `identity_keys.sign_key` /
`signed_prekey_sig` (Ed25519 public key + signature over the signed
pre-key) for E2EE v1.

## WebSocket (v0.4: per-chat rooms)

One hub (`internal/ws/hub.go`), clients grouped by chat room. A
connection joins the rooms of its user's chats (passed in by the
handler) plus any room added later via a `{"type":"subscribe"}`
frame. `BroadcastToChat` delivers only to members of that chat and
returns the user ids that received it live — the sender uses that to
mark messages delivered. Inbound frames: `{"type":"typing"}` (forwarded
only to the other members of that room) and `{"type":"subscribe"}`.
Outbound events: `message`, `message_edited`, `message_deleted`,
`presence`, `typing`, `message_read`, `message_delivered`. Presence is
reported through a callback into `Store.SetPresence` (online +
last-seen). Live two-client latency measured 2026-11-12: median
0.35 ms on same-machine loopback with the in-memory store (see README).
Still planned: heartbeat, reconnect with `last_message_id` catch-up.

## WebSocket call signaling (v0.5)

Inbound frames `call-offer` / `call-answer` / `call-ice` / `call-end`
(`{"type","chat_id","payload","to_user_id"}`) are relayed verbatim to
the other members of the chat room — or only to `to_user_id` when set
(1:1 calls). The server never parses or stores the payload; the sender
is in the room, so membership is already enforced. Tested live with
two clients (`TestV05CallSignaling`, `backend/scripts/live-test.cjs`).
Media itself is peer-to-peer WebRTC (STUN, plus the self-hosted
coturn TURN relay wired in v0.6 — see below);
no live media exchange between two real peers was tested in this
environment (no mic/camera), only signaling.

## Delivery / read receipts (v0.4)

`Message.status` is `sent` → `delivered` → `read`, computed from
`message_receipts` (`delivered_to` / `read_by` in the API). Listing a
chat's messages marks the reader's incoming messages delivered;
`POST /api/messages/{id}/read` marks one read and broadcasts
`message_read` to the chat room. Expired (TTL) messages are hidden
from lists/search and soft-deleted by `PurgeExpired`.

## Scheduled messages (v0.4)

`POST /api/chats/{id}/scheduled` stores a message with `send_at`
(RFC3339, ≥ 5 s in the future, ≤ 1 year). A dispatcher goroutine in
`cmd/server/main.go` ticks every 5 s and calls
`Handler.ProcessDueScheduled()`, which sends due messages as real
messages and broadcasts them. Cancel with `DELETE /api/scheduled/{id}`.

## Bots (v0.4)

`POST /api/bots` creates a bot and a real bot user account
(`bot_<username>`, `is_bot=true`); the raw token is returned once and
only its SHA-256 hash is stored. Bots poll with `X-Bot-Token`
(`GET /api/bot/updates`), send with `POST /api/bot/send`, and can
register a webhook (`POST /api/bots/{id}/webhook`, best-effort POST,
2 s timeout) that is notified when someone DMs the bot.

## Rate limiting with Redis (v0.4)

`internal/auth/redislimiter.go` implements the same `Limiter`
interface as the in-memory limiter, using go-redis v9.8 fixed-window
INCR + EXPIRE. The server uses it only when `REDIS_URL` (or
`REDIS_ADDR`) is set *and* Redis answers PING; otherwise it keeps the
in-memory limiter and logs a warning. Any Redis error fails open to
the in-memory fallback for that key (logged once). **Tested against
real Redis 7.0.15 in v0.5** (`TestRedisLimiterLive`, live server log
"VYRO rate limiting: Redis-backed"). go-redis is pinned to v9.8:
older versions send `CLIENT SETINFO` on connect, which Redis < 7.2
rejects (`DisableIdentity` is set in `NewRedisLimiter`).

## Calls (WebRTC) — v0.5: signaling done, media untested live

Signaling is implemented over the existing WebSocket (see "WebSocket
call signaling" above) and wired into the web client
(`web/lib/use-call.ts`, audio-only in v0.5): offer/answer/ICE flow
between two peers, incoming-call banner, hang-up. Media is
peer-to-peer via WebRTC with free public STUN
(`stun:stun.l.google.com:19302`), plus a self-hosted coturn TURN
relay in v0.6 (`turn/turnserver.conf`, `turn` service in
`docker-compose.yml`, wired into the web client in
`web/lib/use-call.ts` via `NEXT_PUBLIC_TURN_URL`) — calls fall back
to the relay behind symmetric NAT instead of failing. **Not tested:**
a live media exchange between two real peers (no mic/camera in this
environment) — only signaling was tested end-to-end, and the coturn
server itself was not run here (no Docker in this environment).
Video calls are not wired yet (audio only).

## Web Push (VAPID) — v0.5

Free, standards-based push for the web client (RFC 8030 / 8291 /
8292), no FCM/APNs account. Server: `internal/push/push.go`
(encryption via `webpush-go`, VAPID signing), in-memory subscription
registry, endpoints `GET /api/push/vapid-public-key`,
`POST/DELETE /api/push/subscribe`, `POST /api/push/test`
(`internal/handlers/v05.go`). VAPID keys come from
`VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY`; when unset the server
generates an ephemeral pair for the run and warns (subscriptions from
a previous pair stop validating after a restart — set the env vars in
production). On `SendMessage`, members not reached over the WebSocket
get a content-free push ("New message from <name>" / "New encrypted
message"). The web client registers `public/sw.js` and subscribes
from the sidebar (`web/lib/push-client.ts`). Subscriptions the push
service reports gone (410/404) are pruned. Limits: subscriptions are
in-memory (a restart drops them; the client re-subscribes on login);
real delivery through a browser push service (e.g. FCM) was NOT
tested — the crypto path + delivery to a local stand-in endpoint
were proven (`internal/push/push_test.go`,
`backend/scripts/live-test.cjs`).

## E2EE — v0.5

See `docs/SECURITY.md` for the full design. In short: X3DH + Double
Ratchet ("VYRO E2EE v1", 1:1 DMs, one device), Go
(`backend/internal/e2ee/`) and TypeScript/WebCrypto (`web/lib/e2ee.ts`,
`web/lib/e2ee-store.ts`) implementations proven interoperable in both
directions. Key bundle endpoint `GET /api/keys/{userId}/bundle`
(`internal/handlers/v05.go`); messages of `kind="e2ee"` carry the
ciphertext envelope as `Content` and are excluded from search.

## Zero-cost rule (until 10,000 registered users)

- Only free / open-source tools: Go, Flutter, Next.js, PostgreSQL,
  Redis, Docker images above — all free.
- No paid SaaS, no paid API, no paid hosting. Local dev and
  self-hosted Docker are the deployment path in v0.1.
- Features that need a paid service (e.g. managed TURN, SMS
  verification, push-notification providers with paid tiers) stay
  disabled, with a free alternative noted in the README.
- Free push: web push (VAPID, self-hosted) is implemented in v0.5;
  FCM's free tier is not wired (the browser push service delivers
  VAPID pushes without any VYRO-side FCM account).

## Roadmap

1. **v0.1 (this repo)** — auth, DM/group/channel, text messages,
   edit/delete/react, search, block, WebSocket live feed, in-memory store.
2. **v0.2** — `PostgresStore` implementing `StoreIface` ✅ (done
   2026-11-12, tested against real PostgreSQL).
3. **v0.3** — File/image upload ✅ (done 2026-11-12, tested live
   against real PostgreSQL, survives restart).
4. **v0.4** — ✅ (done 2026-11-12): per-chat WS rooms, receipts +
   typing + presence, location/sticker/TTL/scheduled messages,
   pin/archive/folders, stories, bots, group admin enforcement,
   Redis-backed rate limiting, E2EE key foundation (keys only),
   familiar web/mobile UI.
5. **v0.5** — ✅ (done 2026-11-12): real E2EE for 1:1 DMs (VYRO E2EE
   v1 — X3DH + Double Ratchet, Go + TypeScript/WebCrypto, proven
   interoperable; Safety Number in the web UI), WebRTC call signaling
   + audio calls in the web client (media untested live), Web Push
   (VAPID) with content-free notifications, Redis limiter tested
   against real Redis 7.0.15, v0.4 Postgres methods tested against
   real PostgreSQL 16, one-time pre-key republish fix.
6. **v0.6** — Group E2EE (Sender Keys), multi-device sessions, video
   calls + TURN (self-hosted coturn), live media test with real peers,
   voice-message recording in clients, AI assistant (local / free-tier
   model only), auto-translation (free / self-hosted model only),
   business tools, independent review of the E2EE construction.
