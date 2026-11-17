# Safari Mobile / Desktop (Flutter)

Flutter client for Android, iOS, Windows and macOS — one codebase.
Mobile-first: this is where Safari is built first.

## What works in v0.6 (mobile)

- **Auth with email verification**: register and login both send a
  6-digit code to the user's email (10-minute expiry, single-use, max
  5 attempts, 60 s resend cooldown). No token is issued until the code
  is verified. Code entry screen with resend countdown is built in.
- **Chats**: list chats, send text messages, live updates over
  WebSocket, delivery/read ticks, typing indicator, reply / edit /
  delete / reactions, message search, pin / archive / folders,
  disappearing messages (1 min – 1 day) and scheduled messages.
- **Groups & channels**: create group/channel, member list, add /
  remove members, admin / member roles.
- **Stories**: view 24 h stories, post a story (photo compressed
  before upload).
- **Sponsored stories (Stories ads, from day one)**: clearly-labelled
  "Sponsored" story cards appear between user stories (one per 5 user
  stories, max 3 per feed, never first), with a frequency cap (max 3
  sponsored views per session, max 5 per day — persisted). Tapping one
  opens the ad (advertiser name, headline, link copied to clipboard).
  Businesses can submit a sponsored story from the Stories tab
  ("Promote your business") — the request is saved on the device only
  until the ad service is live; nothing is sent anywhere. The built-in
  inventory is DEMO content only (fictional businesses, labelled
  "Sponsored · Demo") — no real advertisers, no real revenue, no fake
  metrics. Sponsored cards carry no remote media, so they cost zero
  bytes (Data Saver). See `lib/sponsored_stories.dart` (11 tests).
- **1:1 + group E2EE** (X3DH + Double Ratchet / Sender Keys) — proven by
  `flutter test` (test/e2ee_test.dart) against the Go-produced vector in
  `../web/lib/e2ee-vector.json`, plus Dart 1:1 and group round-trip
  tests. Toggle per chat with the 🔒 button. Private keys never leave
  the device. Voice messages are NOT E2EE (see below).
- **Voice/video calls** (WebRTC): audio calls and low-start video
  calls, audio-only fallback, incoming-call banner. Signalling goes
  through the backend WebSocket, same as web.
- **Voice messages**: record with Opus at 12 kbps (falls back to
  AAC-LC 16 kbps, then WAV), upload via `/api/uploads`, play back.
- **Local AI (on-device, no cloud)**: 🤖 assistant (rule-based, knows
  how Safari works — honestly labelled, not a cloud AI), dictionary
  translation EN↔FA with a reported coverage % (tap 🌐 on any text
  message), and smart-reply chips for the last incoming message.
  Nothing here makes a network request.
- **Bots**: create/list/delete bots, set webhook URL (Settings → My
  bots). The token is shown once at creation.
- **Data Saver (default ON)** — Safari uses far less data than
  Telegram/WhatsApp: tap-to-download media, photos compressed before
  upload, voice calls Opus 12 kbps (~0.09 MB/min vs ~0.3–0.5 on
  WhatsApp), video calls start at 150 kbps (~1.1 MB/min), usage meter
  in Settings (MB this month, per activity). Numbers are pinned by
  tests (test/local_ai_test.dart).
- **Design**: premium blue-to-purple brand gradient, animated
  "Your World" home view (drifting gradient sky, glassy cards, one
  atmosphere gradient per space), gradient avatars, story rings,
  rounded gradient message bubbles, refined input bar, beautiful
  light and dark themes, subtle motion. Layout stays familiar
  (Telegram/WhatsApp-like) but visibly more polished.
- **Language**: English is the default for new users; Persian (RTL)
  and other languages stay available in Settings (choice is saved).

## New in v0.7 (2026-11-14, tested)

- **Message forwarding**: long-press a message → Forward → pick a
  chat. The copy keeps the original sender ("Forwarded from Alice"),
  its kind and media/location, without re-uploading. E2EE messages
  cannot be forwarded (their ciphertext is only decryptable in the
  original chat) — blocked in the app and on the server. Backend:
  `forwarded_from` / `forwarded_from_id` on messages (migration
  `007_forward.sql`), validated server-side (original sender must
  exist, name ≤ 100 chars). See `lib/api.dart` (`forwardMessage`),
  `lib/chat_extras.dart` (`forwardedLabel`).
- **User-created chat folders**: create/rename/delete folders from the
  home screen ("+ Folder" chip; long-press a chip to manage it),
  long-press a chat → "Move to folder". Folder names live on this
  device; assignment reuses the existing `POST /api/chats/{id}/folder`
  endpoint (max 32 chars). See `lib/chat_extras.dart` (`ChatFolders`).
- **Chat lock with a PIN**: Settings → Chat lock → set a 4-8 digit
  PIN, then long-press a chat → "Lock chat". Locked chats ask for the
  PIN before opening (once per session). Only the salted SHA-256 hash
  of the PIN is stored, never the PIN. Locking hides content on this
  device — it is not encryption. See `lib/chat_extras.dart`
  (`ChatLock`).
- **Message drafts**: unsent text is saved per chat as you type and
  restored when you reopen the chat; sending clears it. On-device
  only. See `lib/chat_extras.dart` (`MessageDrafts`).
- **Voice message playback speed**: the voice player has a 1x / 1.5x /
  2x speed button (just_audio `setSpeed`). See
  `lib/voice_recorder.dart` (`VoiceMessagePlayer`).
- **Bug fixed (found by live-testing the server)**: registering or
  logging in with no SMTP configured **crashed the server** (nil
  pointer panic in `SMTPSender.Send`) instead of answering 503.
  Cause: `mail.NewSMTP` returned a nil `*SMTPSender` that was assigned
  into the `mail.Sender` interface, so `Mailer == nil` was false.
  Fix: `NewSMTP` now returns the `Sender` interface (unconfigured =
  true nil) and `Send` is nil-receiver-safe. Live-tested: register
  without SMTP now returns HTTP 503 `{"error":"email sending is not
  configured"}` and the server stays up. Regression test:
  `TestRegisterWithUnconfiguredSMTPIs503`.

## Run

```bash
# Start the backend first (see ../README.md)
cd ../backend && go run ./cmd/server

# In another terminal (needs the Flutter SDK — free, https://flutter.dev):
cd mobile
flutter pub get
flutter run
# For web/desktop targets: flutter run -d chrome / -d windows / -d macos
```

The API base URL is `http://10.0.2.2:8080` on the Android emulator
and `http://localhost:8080` elsewhere — see `lib/api.dart`
(`ApiClient.baseUrl`). Change it for a real device / deployment.

## Tests

```bash
flutter analyze   # must be clean
flutter test      # 33 tests: E2EE interop + round-trips, Data Saver
                  # numbers, local AI / translation / smart replies,
                  # SDP bitrate caps, sponsored-story caps + submissions,
                  # forwarding labels, folders, chat lock (PIN hash),
                  # drafts, voice playback speed cycle
```

## APK build (Android)

```bash
# Needs: Android SDK (~/Android/sdk, platforms 29/31/33/34/35 +
# build-tools 33.0.1/34.0.0), JDK 17 (~/java, Temurin — downloaded
# with curl because the system JDK does not survive VM restarts here),
# GRADLE_USER_HOME set, and the local Maven repo
# (scripts/fetch-maven.sh + scripts/prime-build.sh).
# This sandbox's egress proxy breaks Java TLS, so Gradle cannot
# download the Android Gradle Plugin itself — artifacts are fetched
# with curl into a file:// repo at ~/.m2-vyro and Gradle reads them
# from there (injected by ~/.gradle/init.d/local-repo.gradle).
# Also required here: -Djava.net.preferIPv4Stack=true on every JVM
# including worker daemons (JAVA_TOOL_OPTIONS — Gradle daemon IPC
# fails on IPv6 loopback) and scripts/normalize-poms.py (some Kotlin
# poms repeat a dependency as <optional>true</optional>, which makes
# Gradle silently drop the jar). prime-build.sh sets all of this up
# and iterates the build, fetching each missing artifact, until it
# succeeds or no new artifact can be found.
./scripts/prime-build.sh 40
# -> build/app/outputs/flutter-apk/app-release.apk (version 0.6.0, id app.vyro)
```

**APK built 2026-11-12** (first real Android build):

- `build/app/outputs/flutter-apk/app-release.apk` — 101,011,626 bytes
  (96.3 MB), package `app.vyro`, version 0.6.0 (versionCode 1),
  minSdk 23, targetSdk 34, label "Safari", Safari gradient launcher icon.
- SHA-256:
  `8a717aa83b26cf916b4f109aa16a2a3b27a84bd93a0fdf41e2b8ca6bc97fc592`
- Also copied to `../web/public/downloads/vyro-0.6.0.apk` and listed
  on the official landing page (`/landing`) with the checksum.
- **Signed with the debug key** (the app's build.gradle uses
  `signingConfigs.debug` for release) — a proper release keystore
  must be created before wide distribution. Do not publish this APK
  as a final release build.

**Per-ABI split APKs built 2026-11-14** (`flutter build apk --release
--split-per-abi`, because the 96 MB fat APK was too big to install):

| ABI | File | Size | SHA-256 |
| --- | --- | --- | --- |
| arm64-v8a (most modern phones) | `app-arm64-v8a-release.apk` | 32,583,679 bytes (32.6 MB) | `f04a52e5f236de2e5b244564200d25749a7da1dcaf47c87f8aae9906a4b7de7f` |
| armeabi-v7a (older 32-bit phones) | `app-armeabi-v7a-release.apk` | 25,694,715 bytes (25.7 MB) | `7d92bd53fcc4c7c521d34ec75e663b4b0a99f1b454c2fbcc58a40dfb66600098` |
| x86_64 (Intel / emulators) | `app-x86_64-release.apk` | 35,119,876 bytes (35.1 MB) | `380d8628f997ff6c14ab04cb9fde6ed5b64c5b424341a5522cf0046e8db3d096` |

Rebuilt 2026-11-14 05:25 UTC: same sizes, now includes the v0.7 code
(forwarding with sender name, chat folders, PIN chat lock, message
drafts, voice playback speed) — verified in the arm64 APK's
`libapp.so` strings. Rebuild needed a Gradle heap raise
(`~/.gradle/gradle.properties`: `-Xmx512m` → `-Xmx3072m`); with 512m
the build failed twice (R8 `OutOfMemoryError: Java heap space`, then
the split packager `OutOfMemoryError`).

All verified with `aapt dump badging`: package `app.vyro`,
versionName 0.6.0 (versionCodes 2001 / 1001 / 4001), label "Safari".
Copied to `../web/public/downloads/vyro-0.6.0-<abi>.apk` and offered
on the download-only landing page (`/landing`, arm64 first, fat APK
kept as fallback). Still debug-signed and never installed/run on a
real device.

The official APK is published ONLY on Safari's own official website
(`/landing` download section, with version + SHA-256 checksum).
Never install Safari APKs from other sites — the app itself warns
about this on the login screen and in Settings.

### Build notes (why the sandbox build needs all this)

- `flutter_webrtc` is vendored as `plugins/flutter_webrtc/` (Android
  only) via a `dependency_overrides` path entry: no released version
  implements Flutter 3.24.5's `SurfaceProducer.Callback`
  (`{onSurfaceCreated, onSurfaceDestroyed}`), so the stock plugin
  does not compile. The fork changes 2 lines in
  `SurfaceTextureRenderer.java` (`onSurfaceAvailable` →
  `onSurfaceCreated`). See `plugins/flutter_webrtc/FORK_README.md`.
  When Safari upgrades to a Flutter whose engine uses the newer
  Callback shape, switch back to the stock 1.x package.
- The Gradle init script (`~/.gradle/init.d/local-repo.gradle`, not
  in the repo) also adds a stand-in `flutter` extension (Flutter's
  defaults: compileSdk 34) to plugin projects, because some plugin
  build scripts read `flutter.compileSdkVersion` on their own project.
- minSdk is 23 (not Flutter's default 21) because `record_android`
  declares minSdk 23.
- The local Maven repo also needs Gradle Module Metadata absent:
  Kotlin multiplatform libraries resolve from poms alone, so the
  app's `build.gradle` forces `activity-ktx:1.9.3` and
  `lifecycle-runtime-ktx:2.7.0`, excludes the duplicate
  `datastore-core-jvm`, and excludes KMP metadata files from
  packaging (`packagingOptions`).

## Notes / not done yet

- iOS needs a Mac with Xcode and cannot be built on this Linux machine
  (no IPA yet).
- APKs built before the local release keystore was configured remain
  debug-signed. New release builds use the release keystore configured
  from the untracked `secrets/key.properties` file. Since that signing
  key differs from the debug key, installing a new release APK over an
  older debug-signed install requires uninstalling the old app first.
- Push notifications are **not** available on mobile yet (no Firebase
  project — FCM is not set up).
- Live two-device testing (real-time speed, calls between real peers,
  E2EE between mobile and web) has not been done.
- Voice messages are NOT end-to-end encrypted (only 1:1 text DMs and
  group text are). Calls are WebRTC peer-to-peer; TURN is configured
  in the backend docker-compose but live media between real peers is
  untested.
