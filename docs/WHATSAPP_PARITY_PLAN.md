# Safari — WhatsApp parity plan (master brief, 2026-11-15)

Standing rules from the user: do NOT delete the current project, do NOT
rewrite from zero, do NOT break anything that works. Keep the stack
(Flutter mobile, Go backend, PostgreSQL, Next.js download site). Work
autonomously stage by stage. Zero-cost until 10,000 registered users.
Never present a demo feature as complete.

## Stage 0 — DONE (2026-11-15 morning): WhatsApp look
- WhatsApp theme, 4 tabs (Chats / Updates / Communities / Calls),
  Settings in the ⋮ menu, WhatsApp chat list (preview + unread badge +
  time labels + Archived row), Updates (Status + Channels),
  Communities (real groups), Calls (honest empty state), WhatsApp
  Settings, chat screen (date chips, E2EE banner, WhatsApp input bar),
  WhatsApp bubbles with blue read ticks, login/code screens restyled.
- Backend: ListChats returns last_message + unread_count;
  MarkRead marks all earlier messages read (WhatsApp semantics).
- Tests: flutter analyze clean, flutter test 33/33, go test ./... pass.
- Still to do in this stage: fresh screenshots, fresh split APKs,
  commit + push, verify live site, deliver to the user.

## Stage 1 — DONE (2026-11-15): chat gaps
- Mute chat (per-user setting; chat ⋮ menu Mute/Unmute; bell-off
  icon + grey unread badge in the chat list), pin message (one per
  chat; pinned banner with sender + text, tap for the full message,
  X to unpin; long-press Pin/Unpin), contact share (attach sheet →
  Share contact → pick a Safari user → contact card; tapping the
  card opens a DM with them), delete-for-me / delete-for-everyone
  choice dialog (delete-for-me hides the message only for the
  caller; chat preview + unread counts skip it).
- Backend: migration 011 (chat_settings.muted,
  chats.pinned_message_id, message_deletions table); endpoints
  /mute, /unmute, /pin-message (POST/DELETE),
  /api/messages/{id}/delete-for-me; contact is a new message kind
  (content JSON {"user_id"}, server validates the user exists);
  TestStage1ChatGaps passes; go test ./... all pass.
- Mobile: api.dart mute/unmute/pinMessage/unpinMessage/
  deleteMessageForMe/sendContact; main.dart message-actions sheet
  (Pin/Unpin + delete choice), chat ⋮ menu mute toggle, _chatRow
  muted state, _chatPreview "Contact" label, pinned banner,
  attach-sheet contact picker, contact card in _messageBubble.
  flutter analyze clean, flutter test 33/33.
- Also fixed: pin-chat menu used to replace the open chat with the
  settings response; it now re-reads the fresh chat from the list.
- DELIVERED 2026-11-15 ~10:30 +0430 (commit 056c03e, pushed to
  adscroll-dev/Safari, remote HEAD verified): fresh split APKs
  arm64 33,425,019 B SHA-256 8cfb77d3…c67e, armeabi 26,536,037 B
  d5c08563…35b1, x86_64 35,961,213 B df201cc3…0f2c; download page
  hashes/sizes updated (33.4/26.5/36.0 MB). VERIFIED LIVE: landing
  HTTP 200 with the new hash, live APK download matches SHA-256,
  /api/chats/{id}/mute probe returns 401 (new route live) and
  /health ok with store postgres — migration 011 applied cleanly
  on the first live run.

## Gap list (verify each by code inspection before building)

### 1. Accounts
- DONE: email register/login + 6-digit codes every login, auto-username.
- MISSING: phone-number signup/login with SMS OTP (needs a free SMS
  route — Twilio trial is limited; check free options, else mark as
  blocked-with-reason and keep email as the working path).
- VERIFY: profile photo upload, about/bio field, change email/number,
  session/device management (linked devices page).

### 2. Private chat
- DONE: text, photo/video, file, voice (+record/playback speed),
  location, reply, forward, edit, delete, reactions, timestamps,
  sent/delivered/read ticks, typing, message search, pin chat,
  archive, delete chat, folders, drafts, chat lock PIN, scheduled,
  disappearing.
- VERIFIED 2026-11-15 by code inspection: delete is a single
  sender-only hard delete (DELETE /api/messages/{id}, broadcast
  `message_deleted` to the chat; mobile shows one "Delete" option) —
  effectively delete-for-everyone for own messages, with NO
  delete-for-me option and no choice UI. Mute chat: NOT built (only
  the word "muted" as a colour name in theme.dart). Pin MESSAGE: NOT
  built (zero references; only pin *chat* exists). Share contact: NOT
  built (no contact message kind; only "View contact" in the members
  sheet).

### 3. Calls
- DONE: 1:1 voice/video signaling (WebRTC), Opus 12 kbps cap, video
  starts low, audio-only fallback, TURN wiring on web.
- DONE 2026-11-15 (Stage 3): call history (migration 013
  call_records; POST/GET /api/calls; caller records ended/missed,
  callee + group members see it; TestStage3Calls passes), missed-call
  marker (red "Missed" in the Calls tab), in-call mic mute / camera
  on-off / front-back switch, live duration label, ICE reconnect
  (restartIce once on failure), group calls (mesh: one peer
  connection per member, offers via the existing to_user_id relay,
  join/leave handling, per-member video tiles), incoming-call banner
  on every screen (with caller/group name) + "return to call" bar,
  call-back from history. Backend go test ./... all pass; mobile
  analyze clean, flutter test 33/33.
- MISSING: live two-phone media test still never done — needs her
  phone + a second phone. Speaker toggle not built (no API in the
  WebRTC fork). No "busy" handling when a second call arrives
  mid-call.

### 4. Status / Updates
- DONE: photo/video status post + view, 24h expiry, sponsored
  stories (demo inventory, capped), Promote flow.
- DONE 2026-11-15 (Stage 2, commit 2531bbf): text status (700 chars,
  9 background colours), audio status (Opus 12 kbps, same Data
  Saver encoder as voice messages), delete own status, viewers
  list (owner only, "Viewed by" sheet), status privacy
  everyone/nobody (Settings → Privacy; enforced in list/view/
  react/reply), reply to a status (also a real DM message to the
  owner, so they get notified), react to a status (8 emojis, one
  per user, changeable). Backend migration 012 + TestStage2Status
  passes; mobile analyze clean, flutter test 33/33. DELIVERED
  2026-11-15 ~11:10: APKs arm64 33,490,663 B (893bf636…b7f7),
  armeabi 26,568,904 B, x86_64 35,961,324 B; verified live (landing
  hash, live APK SHA-256 match, /api/stories/{id}/view probe 401,
  /health ok, migration 012 applied cleanly).
- MISSING: status privacy "except specific people" (only
  everyone/nobody so far, same as last-seen).

### 5. Groups
- DONE: create, members, admin/owner roles, invite links, all message
  types, group E2EE (Sender Keys).
- VERIFY: group photo/about editing, member permissions.
- MISSING: polls, pinned message, group calls.
- STAGE 4 PLAN (2026-11-15, reviewed while the Stage 3 APK built):
  Polls need a FULL build — zero poll code exists anywhere today
  (backend or mobile). Plan: (a) backend migration 014 — polls
  table (id, chat_id, creator_id, question, options JSON,
  allow_multiple, created_at) + poll_votes (poll_id, user_id,
  option_index, PK poll+user+option); model Poll {id, question,
  options, votes per option, myVotes, total}; endpoints
  POST /api/chats/{id}/polls (member; question ≤300 chars, 2–12
  options), GET /api/chats/{id}/polls, POST /api/polls/{id}/vote
  (one vote per option; re-vote replaces), DELETE /api/polls/{id}
  (creator/admin); poll is also a message kind so it appears in the
  chat flow with live results; (b) mobile: attach-sheet "Poll" →
  question + options composer, poll bubble with tap-to-vote and live
  percentages, change-vote support. Pinned message in groups is
  ALMOST done — Stage 1 pin/unpin works in every chat already, but
  any member can pin; Stage 4 tightens it: group pin/unpin requires
  admin/owner (requireRole helper exists in v04.go), DMs stay
  either-member, and pinning posts a system notice in the chat
  ("X pinned a message") like WhatsApp. Group calls: DONE in
  Stage 3.

### 6. Communities
- PARTIAL: tab lists real groups; "New community" opens the group flow.
- MISSING: real community model — community with subgroups,
  admin announcement group, member management, invites, admin settings.

### 7. Channels
- DONE: channel type, create, open/follow (as a chat), invite links.
- MISSING: view counter on posts, channel search, admin management
  beyond the members sheet.

### 8. Extras
- DONE: disappearing messages, chat lock PIN, Data Saver + usage
  meter, media upload/download, light/dark themes, FCM push, block.
- MISSING: app lock (PIN/fingerprint for the whole app), backup &
  restore, storage-management page, website link preview,
  notification settings, chat share (export), account transfer,
  linked-devices page, contact search, report user, anti-spam tools.

### 9. Platforms
- DONE: Android APK (split per ABI), Flutter web build (used for
  screenshots + the web artifact demo).
- MISSING: iOS (needs a Mac + Xcode), Windows/macOS desktop builds
  (Flutter desktop — verify the app compiles for them; no test device).

### 10. Blockers (unchanged)
- Mailjet business verification ticket #4340089 still under review —
  real-user registration emails wait on it.
- Safari-svg GitHub deletion blocked on her password (secure card
  f66516e5, unsubmitted).
- No release keystore yet (APKs are debug-signed).
- No live two-device test (calls, FCM delivery, messaging between
  two real accounts on phones).

## Order of work (next stages)
1. Finish Stage 0 delivery (screenshots + APK + push). — DONE 2026-11-15.
2. Chat gaps: mute chat, pin message, contact share, delete-for-me /
   delete-for-everyone check. — DONE 2026-11-15 (Stage 1 above).
3. Status gaps: text/audio status, delete, viewers, privacy, reply/react. — DONE 2026-11-15 (Stage 2 above).
4. Calls: history + missed, in-call controls, group calls (signaling). — DONE 2026-11-15 (Stage 3 above).
5. Polls + pinned message (groups).
6. Channels: view counter + search.
7. Communities: real model (subgroups + announcement group).
8. Accounts: phone OTP (if a free route exists), profile photo/about,
   linked devices, app lock, backup/restore, storage page, link preview.
9. iOS/desktop feasibility check (no Mac — report honestly).
