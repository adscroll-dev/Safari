# VYRO Web (Next.js)

Web client for VYRO — **Connect Beyond Limits**.

## What works in v0.1

- Register / login against the Go backend (`http://localhost:8080`).
- List chats, create a DM (pick a user), send text messages.
- Live updates over WebSocket (`/ws`).
- Message search, edit / delete of your own messages (via the API
  used by the chat view).
- Light / dark theme (toggle in the header), RTL-ready layout
  (`dir="rtl"` when the language is Persian — see `lib/i18n.ts`).

## Run

```bash
# 1. Start the backend first (see ../README.md)
cd ../backend && go run ./cmd/server

# 2. In another terminal:
cd web
npm install
npm run dev
# Open http://localhost:3000
```

Set `NEXT_PUBLIC_API_URL` to point at a different backend
(default `http://localhost:8080`).

## Notes

- No paid service is used. No analytics, no tracking.
- File / image / voice messages are now wired in this client: the 📎
  button uploads via `POST /api/uploads` and sends the result as a
  media message (text box = optional caption); images, videos and
  voice notes render inline, other files download. Files are fetched
  with the login token, because `GET /uploads/*` requires auth.
  Verified by `npm run build` (Next 14.2.35, types pass) on
  2026-11-12; a live send against a running Go backend was not
  tested in this environment (no Go toolchain here).
- Calls, stories and E2EE are **not** in this client yet — the UI does
  not show fake buttons for them.
