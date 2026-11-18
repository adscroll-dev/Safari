# SAFARI — ORBITAL NEXUS — Design Brief (v1.0)

Source: ChatGPT design brief, 2026-11-18 (Canvas document "SAFARI — ORBITAL NEXUS" in the Safari review chat).
Direction: Futuristic / Spatial / Premium / Minimal / Interactive.
Goal: A messenger with its OWN identity — NOT a copy of WhatsApp, Telegram or Signal.

## 1. Philosophy
- The app is a living digital space, not a list of chats with buttons.
- Three principles: Spatial Interface (depth, layers, visual connection), Living Interface (soft, intelligent motion in response to interaction), Quiet Intelligence (striking but never sacrifices readability or speed).
- What makes Safari different: floating surfaces with custom shapes instead of repetitive rectangular cards; a spatial Dock instead of a normal tab bar; subtle light indicators for unread/presence/active call; orbital lines and controlled curves at key points; each screen has its own composition (the four screens must NOT be the same list with a different title); animations explain function, not decoration; enough empty space for a luxurious, professional feel.
- HARD RULE: no spatial element may cause clutter, slowness, or harder access to features.

## 2. Colors (exact)
Main (dark spatial theme):
- Void Black #050812 — main background
- Deep Space #080F20 — page backgrounds
- Nebula Navy #101B32 — main surfaces
- Orbital Surface #14223B — cards and panels
- Elevated Surface #1A2C49 — highlighted elements
- Electric Cyan #35D9F4 — main brand color
- Plasma Blue #477BFF — main actions
- Nebula Violet #9877FF — complementary
- Aurora Mint #55E6BC — success / positive state
- Solar Amber #FFBF69 — warning / needs attention
- Signal Coral #FF637D — error, missed call, delete
Text:
- Stellar White #F3F7FF — headings and main text
- Moonlight #B8C5D9 — secondary text
- Muted Blue #7889A5 — low-importance text
- Disabled #52617A — inactive elements
Gradients:
- Primary Orbit: #35D9F4 → #477BFF (main button, active indicator, main icons)
- Nebula Flow: #477BFF → #9877FF (decorative, selected state, special surfaces)
- Aurora Glow: #35D9F4 → #55E6BC (success, confirmation)
Rules: gradient only on important elements; no long text on gradient; no heavy gradient on whole page backgrounds; max 1–2 glowing elements per screen; glow very subtle and disableable on weak devices.
Glass: floating panels 88–94% surface opacity; border white 6–12% opacity; blur 12–20px on main panels; shadow wide, faint, dark-blue; normal content cards without heavy blur; BackdropFilter only for limited panels — NEVER blur on every list item.

## 3. Typography
- Fonts BUNDLED in the project (same look on all devices): Persian Vazirmatn; Arabic Vazirmatn (check Arabic letter shaping); English Manrope.
- Sizes: page title 28sp; section title 21sp; card title 17sp; body text 15sp; message text 15sp; secondary text 13sp; small captions 11sp; Dock label 10sp.
- Weights: Heading Bold 700; Section SemiBold 600; Body Regular 400; Button SemiBold 600.
- Rules: Persian/Arabic RTL; English LTR; never hard-set direction for the whole app; numbers/times/phone numbers in local readable format; mixed text (English name in Persian message) must render correctly; enough line height for Persian/Arabic; no text pushed off-screen by shrinking font — use wrapping and correct measurement.

## 4. Spacing & shapes (4dp grid)
- XS 4, SM 8, MD 12, LG 16, XL 20, XXL 24, Section 32, Large Section 40 (dp)
- Page rules: horizontal mobile margin 20dp; section gap 24–32dp; list item gap 12dp; min touch target 48dp; main button height 52–56dp.
- Radii: small button 14dp; main button 18dp; card 22dp; floating panel 26dp; Dock 30dp; Bottom Sheet 28dp; Avatar full circle.
- Avoid rounding EVERYTHING excessively — some panels should have fairly geometric corners to form the spatial identity.

## 5. App shell & ORBIT DOCK
- Shell: very dark spatial background; dedicated header per section; independently scrolling content; floating Dock at bottom; safe area for system nav bar and gestures; Dock floats over content and the last content must not hide behind it.
- ORBIT DOCK (the main visual signature):
  - Horizontal floating panel, radius 30dp, Deep Space with controlled opacity, very subtle border, dark-blue shadow.
  - Four destinations: Chats, Updates, Communities, Calls.
  - Custom linear icons; short label under/beside icon depending on device width.
  - Active section marked with an elliptical or light-arc indicator; active icon slightly larger (subtle scale); page change = fade + very short vertical move.
  - Unread badge ONLY when real data exists.
  - Dock may hide/behave per current structure on the chat page; back path always clear and reachable.
  - Do NOT use Flutter's default BottomNavigationBar with a normal look — build a custom component.

## 6. Chats page (STAGE 1 SCOPE)
- Header: small planet logo on the language-correct side; title "Chats" (translated); search button; new-chat button; existing settings/filters access kept. Big simple title; no crowded multi-row header.
- Search Capsule: height 48dp, Nebula Navy background, search icon, translated hint, clear-text ability, real results from the app's current search. Search must not be decorative — keep all current search behavior.
- Filters (horizontal scroll): All, Unread, Personal, Groups — only filters compatible with the real data model; keep any extra current filters. Active filter = capsule with subtle glowing border.
- Conversation tile: an independent floating surface per item. Structure: avatar first; contact/group name; real last-message preview; last-message time; unread indicator; real message status if supported; mute/pin indicator if the feature exists. Design: background near Deep Space; no heavy divider lines; inner padding 14dp; avatar ~52dp; gap between items 10dp. Online/last-seen/encryption status ONLY from real server/client data.
- Empty state: minimal open-orbit + small planet illustration; short translated text; start-chat button; no fake names or counts.
- Loading: skeleton in Nebula Navy, very slow shimmer, no sample data in place of real data.

## 7. Updates page (LATER STAGE — direction only)
- A living stream, not a simple copied story list. Header: Updates title, create button, existing privacy settings. Recent Signals: horizontal section with avatar, dedicated status ring (new = full/near-full colored arc; seen = faint arc; loading = subtle spinner), name, time, unseen indicator; color/ring alone must not carry the meaning — add text/accessibility cue. Update blocks with real content, publisher name, real time, reactions/replies only if currently supported, current management/privacy options. Create Update button with brand colors (text/image/video/other currently supported types); keep all current create/publish/delete/privacy paths.

## 8. Communities page (LATER STAGE — direction only)
- A tidy communication map. Header: Communities title, search, create/join per current features, management access. Each community = Hub Panel: real icon/image, name, short real description, activity/member info only from valid data, unread count if any, access to its groups/subgroups. Panel: radius 24dp, Orbital Surface background, very faint decorative orbital lines, limited brand accent. Inside: community header, real info/image, list of groups/channels, user role/access, messages and management. Keep all current community features (roles, invites, join, leave, management, permissions). No fake members/groups/counters.

## 9. Calls page (LATER STAGE — direction only)
- An advanced communication center. Header: Calls title, start-call button, filters if currently present. Recent calls rows: avatar, name, voice/video type, incoming/outgoing/missed direction, time, duration only if really recorded, call-back button. Colors: successful = Aurora Mint; missed = Signal Coral; normal in/out = Moonlight or Electric Cyan. No fake times/durations. Start-call picker: contact search, real contact list, voice/video buttons, mic/camera permission state if needed. Keep all current call features incl. group calls. Active call screen: contact image/avatar, name, real connection state, real duration, mic/speaker/camera/end-call controls; end-call button clearly distinguishable by color and position.

## 10. Chat page (LATER STAGE — direction only)
- Header: back button, avatar, contact/group name, real online/presence info, voice/video call buttons, existing features menu; group info/members reachable. Background: very dark and minimal; a very subtle orbital pattern in corners is allowed but must not reduce readability; no watermark or big distracting image behind messages. Bubbles: sent = very soft gradient Plasma Blue → dark blue, asymmetric modern radius, Stellar White text; received = Orbital Surface, subtle border, Stellar White text. In RTL, sent/received placement follows semantic direction, not fixed left/right. Keep ALL current message types (text, image, video, file, voice, reply, reaction, edit, delete, forward, send/read status, others) with a dedicated component per type — not one simple bubble for everything. Message status indicators must match the real system state; never show an encryption indicator for a chat whose encryption state is unconfirmed. Composer: a dedicated floating panel (attach button, text input, emoji button if currently present, voice-record button if supported, send button); empty text shows record/current control, text present shows send; input grows with text up to a max height then scrolls inside; moves with the keyboard; last message must not hide behind composer/keyboard. Reply preview: small panel above composer with sender name, original message summary, thin colored bar on the semantic side, close button. Attachment panel: glass Bottom Sheet with icon grid, ONLY actually supported attachment types (image, camera, file, location, others that exist); no decorative option without real function.

## 11. Signature elements
- ORBIT SIGNATURE: orbital indicator inspired by the planet+ring logo — active Dock section, loading state, important selection, section entry animation. Repeat in a few key places to become the brand signature, not everywhere.
- NEXUS AVATAR: avatars with a subtle orbital ring frame — very thin outer ring; real presence indicator at a point on the orbit; second ring only when a real meaningful state exists; neutral color for unknown state. Must NOT fake online or security status.
- SIGNAL TRACE: a very subtle light line/arc showing movement between screens (Dock → Chats, chat → contact profile, Calls → active call). Only for important transitions, not game-like.
- ORBITAL MESSAGE: dedicated style for important or threaded replies — small orbital indicator, subtle border, visual link to the original message. Must not change real messenger structure, only improve display.
- NEXUS EMPTY WORLD: empty pages use a minimal planet+orbit illustration instead of default icons, each page its own: Chats = conversation orbit; Updates = planet with broadcast arc; Communities = several connected orbits; Calls = spatial communication wave. Light, scalable, no text inside the image.

## 12. Animation & motion
- Timing: button tap 100–140ms; hover/pressed 120ms; small state change 160ms; panel open 220ms; page change 240–280ms; bottom sheet 260ms; new message 180ms.
- Curves: easeOutCubic, easeInOutCubic, fastOutSlowIn.
- Dock: icon slightly larger on select; orbital indicator moves to new position; accent color changes slowly; no strong bounce.
- New message: short fade + 6–10dp vertical move; no jump or annoying overshoot.
- Bottom sheet: slide from bottom, soft background fade, blur only if hardware allows.
- Logo: very slow ring motion, limited glow, NO continuous fast rotation; static version allowed to save resources.
- Reduce Motion: if the system enables it, disable non-essential animations, turn motion into short fades; no function may depend on animation.

## 13. RTL (Persian, Arabic, English)
- Persian RTL, Arabic RTL, English LTR. Use Directionality and EdgeInsetsDirectional. Wide use of EdgeInsets.only(left:...) and Alignment.centerLeft for semantic layouts is FORBIDDEN.
- Directional icons must mirror correctly in RTL; non-directional icons (camera, mic, search, settings) must NOT mirror for no reason.
- Dock destination order in RTL must match the language structure; back button, arrows, menus, drawers positioned per current direction.
- Test mixed text: Persian message with English name; phone number in Arabic sentence; URL in Persian text; emoji in RTL text; date/time; English file name; contact name mixing Persian and Latin letters.
- No visible text hardcoded — all strings from the app's current localization system; translations complete for Persian, English, Arabic; do not change the Safari brand name.

## 14. Responsive
- Mobile-first but correct on larger devices. Small mobile: controlled margins/spacing, Dock must not take too much space, text wraps correctly. Normal mobile: standard layout. Large mobile/tablet: limit content width, two-column list+detail where suitable, Dock sized to width; no fixed width for main content.

## 15. Flutter performance
- const on static widgets; ListView.builder for long lists; avoid unnecessary rebuilds; limited BackdropFilter; RepaintBoundary where suitable; no permanent animations on all list items; proper image/cache management; SVG or icon font for simple icons; optimized raster images for complex elements; avoid heavy/multiple shadows; check performance on mid-range devices; decorative animations must not run continuously while scrolling long lists.

## 16. Keep ALL current features
- UI/UX change only — never an excuse to remove or weaken features. Before starting: identify all current screens and routes, all buttons/actions, all loading/error/empty/success states, all APIs and data models, all message/group/community/call/settings features. For each feature, define its new place in the UI. Then build the new design on the same features.
- FORBIDDEN: removing a feature because the design is crowded; changing an API without real need; changing data-model contracts; removing security controls; changing E2EE behavior; changing permissions without need; showing unconfirmed security status; building sample data to fill the UI; adding buttons that don't work. If a feature has no clear place in the new UI, design a proper access path before removing or changing it.

## 17. Real-data rules (HARD)
- No fake data in the Release build: contact names, profile pictures, last message, message time, unread counts, online status, community member counts, call counts, call duration, encryption status, counters and badges. Loading = skeleton; empty = empty state; network error = error state; real data = show real data; unknown = neutral or not shown. Mock data allowed in Debug for testing but must never leak into Release.

## 18. Flutter component structure (suggestion)
- Modular structure; names may adapt to the current project architecture. Suggested: lib/core/theme/{safari_colors, safari_typography, safari_spacing, safari_shapes, safari_theme}.dart; lib/core/widgets/{orbit_dock, orbit_avatar, glass_panel, safari_button, safari_search, safari_empty_state, safari_skeleton, orbit_indicator}.dart; features/{chats, updates, communities, calls, chat}/...
- IMPORTANT: this project currently keeps most UI in lib/main.dart (~9,400 lines) + lib/theme.dart. KEEP that architecture — extend theme.dart and main.dart, add new widget files where sensible. Do NOT rewrite the app from zero and do NOT do a full refactor to a new folder structure in this stage.

## 19. Acceptance criteria
- Visual identity: four main screens share the design language; custom Safari Dock implemented; planet logo identity kept; UI does not look like a copy of other messengers; colors/typography match the Design Tokens.
- Function: all current buttons work; no route or feature removed; messages send/receive correctly; voice/video calls work; community features kept; Updates display and publish correctly; search and filters work.
- Languages: full Persian RTL, full Arabic RTL, full English LTR; mixed text renders correctly; no overflow in long text.
- Quality: no fake data in Release; loading/empty/error states complete; animations smooth; mid-range device performance acceptable; accessibility and Reduce Motion respected; dark mode unified on all screens.
- Security: E2EE logic unchanged; keys/sensitive info not moved into UI; encryption status only from real source; camera/mic permissions not unnecessarily touched.

## 20. Execution order
- Phase 1: review the project. Phase 2: Design System. Phase 3: main Shell (background, header, ORBIT DOCK, transitions). Phase 4: main screens (Chats, Updates, Communities, Calls). Phase 5: Chat. Phase 6: animation and details. Phase 7: test and fix. Phase 8: Release prep.
- ChatGPT's key recommendation: do NOT rewrite all four screens at once without control. Build the Design System + ORBIT DOCK first, then complete the Chats page and get approval; then extend the same design language to the other three screens and Chat. Before coding, build a real prototype of the Chats page + Dock; when the visual identity is approved, start full implementation of the rest.
- STAGE 1 (this run): Design System (colors, typography, spacing, shapes, base components, theme, localization) + main Shell (background, header, ORBIT DOCK, transitions) + Chats page complete (§6). Then report + screenshots. The other screens and Chat come in the next stage after approval.
