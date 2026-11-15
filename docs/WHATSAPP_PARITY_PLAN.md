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
- DONE 2026-11-15 (Stage 1, commit 056c03e, DELIVERED ~10:30): mute
  chat (per-user, bell-off icon + grey unread badge), pin message
  (one per chat, banner + unpin, long-press Pin/Unpin), contact
  share (attach → Share contact → contact card, tap opens a DM),
  delete-for-me / delete-for-everyone choice dialog. Backend
  migration 011; verified live.

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
  analyze clean, flutter test 33/33. DELIVERED 2026-11-15 ~11:55
  (commit f4ddb09): APKs arm64 33,491,056 B (85996c71…9cd1),
  armeabi 26,618,446 B, x86_64 36,027,236 B; verified live (landing
  hash, live APK SHA-256 match, /api/calls probe 401, /health ok,
  migration 013 applied cleanly). NOTE: the first auto-deploy of
  f4ddb09 stalled on Render (log silent after cache download) —
  canceled and manually redeployed, Live in 1m 20s.
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
- DONE 2026-11-15 (Stage 4): polls — backend migration 014
  (polls + poll_votes, options JSON; poll_votes PK poll+user+option),
  model Poll {id, question, options, votes, myVotes, total},
  endpoints POST /api/chats/{id}/polls (member; question ≤300 chars,
  2–12 options, each ≤100 chars), GET /api/chats/{id}/polls,
  GET /api/polls/{id}, POST /api/polls/{id}/vote (re-vote replaces;
  empty list clears; single-choice = 0 or 1 option), DELETE
  /api/polls/{id} (creator or admin/owner; also deletes the poll
  message); a poll is a message of kind "poll" (content JSON
  {"poll_id", "question"}) broadcast like any message, so it appears
  in the chat flow with live results. Mobile: attach-sheet "Poll" →
  question + options composer (2–12, add/remove, allow-multiple
  switch), poll bubble with tap-to-vote and live counts + percentage
  bars, change/clear-vote support, "Delete poll" in the message
  actions (creator or admin/owner). Group pin tightening: group
  pin/unpin requires admin/owner (requireRole), DMs stay
  either-member; pinning posts a system notice ("X pinned a
  message", kind "system", rendered as a centered grey chip —
  clients cannot send this kind). TestStage4Polls passes; backend
  go test ./... all pass; mobile analyze clean, flutter test 33/33.
  DELIVERED 2026-11-15 ~12:25 (commit e26df31): APKs arm64
  33,622,324 B (a20fbfb4…118ac), armeabi 26,667,807 B
  (aa24ebf8…8ccac), x86_64 36,092,977 B (c89b961f…7df6408e7d7);
  verified live (landing hash, live APK SHA-256 match, /api/polls
  probe 401, /health ok, migration 014 applied cleanly).

### 6. Communities
- PARTIAL: tab lists real groups; "New community" opens the group flow.
- MISSING: real community model — community with subgroups,
  admin announcement group, member management, invites, admin settings.
- DONE 2026-11-15 (Stage 6, commit f47b9e0, DELIVERED ~13:32 +0430):
  real community model (migration 016: communities +
  community_members + community_subgroups; Community {id, name,
  description, owner_id, announcement_chat_id, member_count,
  subgroup_count, my_role}). POST /api/communities (creates the
  announcement group chat too), GET /api/communities, GET
  /api/communities/{id} (community + subgroups), PATCH
  /api/communities/{id} (name/description; also renames the
  announcement group), GET/POST /api/communities/{id}/members (+
  /{userID}/role, DELETE /{userID}), POST
  /api/communities/{id}/leave, POST/DELETE
  /api/communities/{id}/subgroups[/{chatID}] (link an existing
  group or create a new one; only groups; the group chat survives
  unlinking). Rules: only community admins/owner may post in the
  announcement group (enforced in SendMessage; followers see the
  normal input bar there only if they are admins); joining the
  announcement group via its invite link also joins the community;
  adding/removing a member mirrors the announcement group. Mobile:
  Communities tab lists real communities (member + group counts),
  detail sheet (announcement group row, subgroups with open/remove,
  Add group / Add members / Members / Edit community / Leave),
  create dialog (name + description). TestStage6Communities passes;
  backend go test ./... all pass; mobile analyze clean, flutter
  test 33/33. APKs safari-0.6.0 built 13:25 +0430: arm64 33,622,505 B
  SHA-256 3917a311…e7be2d, armeabi 26,717,141 B 961881a8…6df,
  x86_64 36,158,695 B 64e98010…51b — VERIFIED LIVE (landing hash,
  live APK SHA match, /api/communities probe 401, /health ok,
  migration 016 applied cleanly).

### 7. Channels
- DONE: channel type, create, open/follow (as a chat), invite links.
- DONE 2026-11-15 (Stage 5): view counter
  on posts (migration 015: chats.is_public, message_views table with
  PK message+user so re-views never inflate the count; Message.ViewCount
  populated in ListMessages for channel chats; POST /api/messages/{id}/view
  + GET /api/messages/{id}/views, channel-only), channel search
  (GET /api/channels/search?q= — public channels by title, most-followed
  first, max 20; private channels never appear), follow/unfollow
  (POST /api/chats/{id}/follow — public only, 403 for private; POST
  /api/chats/{id}/unfollow — leaves a channel or group, owner cannot
  leave), admin management (only admins/owner may post in a channel —
  followers get 403 and see a read-only notice instead of the input
  bar; PATCH /api/chats/{id} — admin/owner edit title + public flag;
  create-channel dialog has a Public/Private switch and allows
  zero members; members sheet "Edit channel" tile for admins).
  TestStage5Channels passes; backend go test ./... all pass; mobile
  analyze clean, flutter test 33/33. DELIVERED 2026-11-15 ~13:00
  (commit 391c10d): APKs arm64 33,622,328 B (2eb8f550…0ca16a3b85),
  armeabi 26,700,569 B (d1b9edd0…995c36), x86_64 36,092,977 B
  (18aaa4bf…93a09); verified live (landing hash, live APK SHA-256
  match, /api/channels/search probe 401, /health ok, migration 015
  applied cleanly).

### 8. Extras
- DONE: disappearing messages, chat lock PIN, Data Saver + usage
  meter, media upload/download, light/dark themes, FCM push, block
  (backend + mobile UI), report user, contact search, linked-devices
  page, profile photo/about editing.
- MISSING: app lock (PIN/fingerprint for the whole app), backup &
  restore, storage-management page, website link preview,
  notification settings, chat share (export), account transfer,
  anti-spam tools.

### 7. Accounts (Stage 7, DONE 2026-11-15, commit 20b2093, DELIVERED ~14:00 +0430)
- DONE 2026-11-15: profile editing — PATCH /api/users/me
  (display_name 1-100, bio/about max 500, avatar_url must be an
  /uploads/ path); bio now included in PublicUser and GET /me.
  Mobile: Settings profile card shows bio and opens the edit dialog
  (name + about + photo via gallery pick + upload).
- DONE 2026-11-15: user reports — POST /api/users/{id}/report
  (reason 1-500; stored in the reports table from migration 001
  with status "open"; self-report 400, unknown user 404). Mobile:
  DM chat menu -> Report (reason + "Also block this user").
- DONE 2026-11-15: block enforcement — SendMessage now 403s in a DM
  when either side blocked the other (IsBlocked is symmetric);
  DELETE /api/users/{id}/block unblocks; GET /api/users/me/blocked
  lists my blocks. Mobile: DM chat menu Block/Unblock (label from
  the blocked list, loaded at login).
- DONE 2026-11-15: contact search — the search sheet filters users
  by name/username live (up to 8, tap opens a DM); "New direct
  message" is now a searchable picker instead of taking the first
  user.
- DONE 2026-11-15: linked devices — GET /api/devices + DELETE
  /api/devices/{id} already existed (v0.6); mobile Settings ->
  Linked devices lists them (name/platform, "This device" marked)
  with remove. TestStage7Accounts passes; go test ./... all pass;
  mobile analyze clean, flutter test 33/33.
- APKs safari-0.6.0 built 13:55 +0430: arm64 33,622,787 B
  SHA-256 24f380fc…cc14f, armeabi 26,750,186 B 385c266e…55c9d,
  x86_64 36,158,972 B ea255277…7f731 — VERIFIED LIVE (landing hash,
  live APK SHA match, PATCH /api/users/me probe 401, /health ok).
- STILL MISSING (Stage 8): app lock, backup/restore, storage page,
  link preview, notification settings, chat share/export, account
  transfer, anti-spam. Phone OTP still blocked-with-reason.

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
4. Calls: history + missed, in-call controls, group calls (signaling). — DONE 2026-11-15 (Stage 3 above), DELIVERED ~11:55.
5. Polls + pinned message (groups). — DONE 2026-11-15 (Stage 4 above), DELIVERED ~12:25.
6. Channels: view counter, search, admin management. — DONE 2026-11-15 (Stage 5 above), DELIVERED ~13:00.
7. Communities: real model (subgroups + announcement group). — DONE 2026-11-15 (Stage 6, commit f47b9e0, DELIVERED).
8. Accounts: profile photo/about, block/report, contact search, linked devices. — DONE 2026-11-15 (Stage 7, commit 20b2093, DELIVERED).
9. Extras: app lock, backup/restore, storage page, link preview, notification settings, chat share/export, account transfer, anti-spam. — NEXT (Stage 8).
10. iOS/desktop feasibility check (no Mac — report honestly). — NEXT (Stage 9).
