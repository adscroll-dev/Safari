import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:image_picker/image_picker.dart';
import 'package:local_auth/local_auth.dart';
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:video_player/video_player.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'api.dart';
import 'call.dart';
import 'chat_extras.dart';
import 'data_saver.dart';
import 'e2ee_client.dart';
import 'fcm_push.dart';
import 'link_preview.dart';
import 'local_ai.dart';
import 'recording_bytes.dart';
import 'share_file.dart';
import 'theme.dart';
import 'voice_recorder.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  // Firebase (FCM push) on mobile only: the web build uses Web Push
  // (VAPID) and has no Firebase options. Failures are swallowed inside
  // initFirebaseForMain — the app works without push.
  await FcmPush.initFirebaseForMain();
  runApp(const VyroApp());
}

/// Safari — Connect Beyond Limits.
/// v0.6 mobile: real auth + chat list + text/location/voice messages,
/// delivery/read ticks, typing indicator, 24h stories (view + post),
/// pin/archive/folders, disappearing and scheduled messages, groups
/// and channels with member admin, reply/edit/delete/reactions,
/// message search, 1:1 + group E2EE (X3DH + Double Ratchet / Sender
/// Keys), voice/video calls (Opus capped at 12 kbps), and Data Saver
/// (default ON) with a visible data-usage meter.
/// No fake features: file picking, calls and push are only shown when
/// they are really wired; push notifications on mobile use FCM
/// (Firebase Cloud Messaging) — see lib/fcm_push.dart. Real delivery
/// to a phone is untested until a phone test.
class VyroApp extends StatefulWidget {
  const VyroApp({super.key});
  @override
  State<VyroApp> createState() => _VyroAppState();
}

class _VyroAppState extends State<VyroApp> {
  ThemeMode _mode = ThemeMode.system;
  // English is the default language for new users (user rule).
  // The choice is saved, so Persian stays available in Settings.
  Locale _locale = const Locale('en');

  @override
  void initState() {
    super.initState();
    _loadLocale();
  }

  Future<void> _loadLocale() async {
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString('vyro_lang');
    if (mounted && (saved == 'fa' || saved == 'en')) {
      final code = saved == 'fa' ? 'fa' : 'en';
      setState(() => _locale = Locale(code));
    }
  }

  Future<void> _setLocale(Locale locale) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('vyro_lang', locale.languageCode);
    setState(() => _locale = locale);
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Safari',
      themeMode: _mode,
      theme: vyroLightTheme(),
      darkTheme: vyroDarkTheme(),
      locale: _locale,
      supportedLocales: const [Locale('en'), Locale('fa')],
      localizationsDelegates: const [
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: HomePage(
        onToggleTheme: () => setState(() =>
            _mode = _mode == ThemeMode.dark ? ThemeMode.light : ThemeMode.dark),
        onToggleLang: () => _setLocale(_locale.languageCode == 'en'
            ? const Locale('fa')
            : const Locale('en')),
      ),
    );
  }
}

/// The ring ON the planet logo, drawn as a bright GLINT that travels
/// around the ring's own ellipse — the ring itself is the thick glossy
/// one in the original logo image, which the hero shows unchanged.
/// Ellipse measured from the logo (512px square): center (258,275),
/// a=212, b=76.7, tilt -18.1deg. The auth hero shows the logo at
/// 112px inside a 176x112 canvas, tile x-offset 32 — so the ring
/// center is (88.4, 60.2), a=46.4, b=16.8 here. The sweep gradient
/// is mostly transparent with one bright cyan-white-blue segment
/// (~25% of the circle); it travels around forever — like the iPhone
/// Clock, it never stops. [front] selects the bottom half (drawn over
/// the planet, brighter) or the top half (drawn behind it, dimmer).
class _OrbitRingPainter extends CustomPainter {
  _OrbitRingPainter({required this.angle, required this.front});

  final double angle;
  final bool front;

  static const _cx = 88.4;
  static const _cy = 60.2;
  static const _a = 46.4;
  static const _b = 16.8;
  static const _tilt = -0.3159; // -18.1deg in radians

  @override
  void paint(Canvas canvas, Size size) {
    final Shader shader = SweepGradient(
      center: FractionalOffset.center,
      startAngle: angle,
      endAngle: angle + 6.28318,
      colors: const [
        Color(0x00000000),
        Color(0x00000000),
        Color(0xFFFFFFFF),
        Color(0xFF7DD3FC),
        Color(0xFF00C2FF),
        Color(0xFF2563EB),
        Color(0x00000000),
      ],
      stops: const [0.0, 0.72, 0.80, 0.86, 0.92, 0.96, 1.0],
    ).createShader(Rect.fromCircle(center: Offset.zero, radius: _a));
    final Rect rect =
        Rect.fromCenter(center: Offset.zero, width: _a * 2, height: _b * 2);
    canvas.save();
    canvas.translate(_cx, _cy);
    canvas.rotate(_tilt);
    // In the rotated frame the ellipse's major axis is horizontal;
    // canvas angles run clockwise from 3 o'clock (y grows downward):
    // 0..pi is the bottom half (front, over the planet), pi..2pi the top.
    // Glow pass: same glint, wider + blurred, so it shines.
    canvas.drawArc(
      rect,
      front ? 0.0 : 3.14159,
      3.14159,
      false,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = front ? 16.0 : 14.0
        ..strokeCap = StrokeCap.round
        ..shader = shader
        ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 4),
    );
    // Core pass: fat bright glint sitting on the ring band.
    canvas.drawArc(
      rect,
      front ? 0.0 : 3.14159,
      3.14159,
      false,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = front ? 11.0 : 9.0
        ..strokeCap = StrokeCap.round
        ..shader = shader,
    );
    canvas.restore();
  }

  @override
  bool shouldRepaint(_OrbitRingPainter old) =>
      old.angle != angle || old.front != front;
}

class HomePage extends StatefulWidget {
  const HomePage(
      {super.key, required this.onToggleTheme, required this.onToggleLang});
  final VoidCallback onToggleTheme;
  final VoidCallback onToggleLang;
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage>
    with WidgetsBindingObserver, TickerProviderStateMixin {
  final _api = ApiClient();
  final _usage = DataUsageTracker();
  final _email = TextEditingController();
  final _displayName = TextEditingController();
  final _password = TextEditingController();
  final _draft = TextEditingController();
  final _searchCtrl = TextEditingController();
  Map<String, dynamic>? _user;
  List<dynamic> _chats = [];
  // Stage 7: the user IDs I blocked (loaded at login).
  final Set<String> _blockedUserIds = {};
  // Stage 8: whole-app lock (PIN + fingerprint) and notification
  // settings (local preferences).
  AppLock? _appLock;
  bool _locked = false;
  final _pinCtrl = TextEditingController();
  bool _notifyEnabled = true;
  bool _notifySound = true;
  bool _notifyVibrate = true;
  final _linkPreviews = LinkPreviewCache();
  // Stage 6: the communities I belong to (real model, not groups).
  List<dynamic> _communities = [];
  // Users by id, loaded with the chat list so a DM can show the other
  // person's name instead of the literal type ("dm").
  Map<String, Map<String, dynamic>> _usersById = {};
  List<dynamic> _messages = [];
  List<dynamic> _stories = [];
  Map<String, dynamic>? _activeChat;
  String? _error;
  bool _registerMode = false;
  // 3D registration page: show/hide password (eye icon).
  bool _obscurePassword = true;
  // Offline mode (WhatsApp-style): true while the server is
  // unreachable; the app then shows the on-device cache.
  bool _offline = false;
  // Messages typed while offline, waiting to send when back online.
  final List<Map<String, dynamic>> _outbox = [];
  // Animated registration page: the orbs, bubbles and logo drift.
  late final AnimationController _authAnim;
  // Planet logo orbit: the ring around the logo tile rotates forever,
  // like the iPhone Clock — never still (8s per revolution).
  late final AnimationController _ringAnim;
  int _tabIndex = 0; // 0 = chats, 1 = updates, 2 = communities, 3 = calls
  // WhatsApp navigation: Settings lives in the ⋮ menu, not a tab.
  bool _showSettings = false;
  // Archived chats hide behind the "Archived" row (WhatsApp-style).
  bool _showArchived = false;
  String _typingUser = '';
  int _ttlSeconds = 0; // disappearing-message TTL (0 = off)
  DateTime _lastTypingAt = DateTime.fromMillisecondsSinceEpoch(0);
  WebSocketChannel? _ws;
  E2eeClient? _e2ee;
  CallController? _call;
  final RTCVideoRenderer _remoteRenderer = RTCVideoRenderer();
  IncomingCall? _incomingCall;
  // Stage 3: call history (loaded at login, refreshed after a call).
  List<dynamic> _calls = [];
  bool _callInitiatedByMe = false;
  // Stage 3: one video renderer per member of a group call.
  final Map<String, RTCVideoRenderer> _groupRenderers = {};
  Timer? _callTicker;
  String _deviceId = 'mobile-default';
  final Set<String> _e2eeChats = {}; // chat ids with E2EE enabled
  Map<String, dynamic>? _replyTo;
  String? _editingId;
  List<dynamic> _searchResults = [];
  bool _searching = false;
  bool _dataSaver = true;
  // Last-seen privacy (Telegram/WhatsApp-style): "everyone" or
  // "nobody". Loaded from the server after login.
  String _lastSeenVisibility = 'everyone';
  // Status privacy (Stage 2): who may see my status — "everyone"
  // (default) or "nobody". Loaded from the server after login.
  String _statusPrivacy = 'everyone';
  // Stage 11 (Settings parity): avatar/about/groups privacy and
  // read receipts, from the combined /privacy-settings endpoint.
  // "everyone" or "nobody"; read receipts default on.
  String _avatarPrivacy = 'everyone';
  String _aboutPrivacy = 'everyone';
  String _groupsPrivacy = 'everyone';
  bool _readReceipts = true;
  // Stage 11 + wallpaper upgrade: beautiful professional wallpapers —
  // solid colours and gradients — picked per chat (WhatsApp-style) or
  // as the default for all chats (Settings). 'default' is the theme
  // chat background. One colour = solid, two = gradient.
  static const Map<String, List<int>> wallpapers = {
    'default': [],
    'beige': [0xFFECE5DD],
    'mint': [0xFFD8F3DC],
    'sky': [0xFFDCEFFA],
    'rose': [0xFFF6DCE0],
    'slate': [0xFF2A2F2B],
    'lavender': [0xFFE6E0F5],
    'peach': [0xFFFBE3D3],
    'sand': [0xFFEFE6CF],
    'ocean': [0xFFD3E9F2],
    'forest': [0xFFD5E8D4],
    'charcoal': [0xFF1F2C34],
    'grad-teal': [0xFF008069, 0xFF25D366],
    'grad-sunset': [0xFFFF9A56, 0xFFFF6B9D],
    'grad-ocean': [0xFF4A90D9, 0xFF7B61FF],
    'grad-forest': [0xFF43A047, 0xFF00897B],
    'grad-lavender': [0xFF9D7BFF, 0xFFFF7EB6],
    'grad-midnight': [0xFF1F2C34, 0xFF4A235A],
  };
  static const Map<String, String> wallpaperNames = {
    'default': 'Default',
    'beige': 'Beige',
    'mint': 'Mint',
    'sky': 'Sky',
    'rose': 'Rose',
    'slate': 'Slate',
    'lavender': 'Lavender',
    'peach': 'Peach',
    'sand': 'Sand',
    'ocean': 'Ocean',
    'forest': 'Forest',
    'charcoal': 'Charcoal',
    'grad-teal': 'Teal Dawn',
    'grad-sunset': 'Sunset',
    'grad-ocean': 'Ocean Blue',
    'grad-forest': 'Forest',
    'grad-lavender': 'Lavender Dream',
    'grad-midnight': 'Midnight',
  };
  // Global default wallpaper id.
  String _wallpaperId = 'default';
  // Per-chat overrides: chat id -> wallpaper id. Absent = global.
  final Map<String, String> _chatWallpapers = {};
  // Downloaded media bytes, keyed by file_url (Data Saver: only after
  // the user taps Download, or when Data Saver is off).
  final Map<String, Uint8List> _downloaded = {};
  final Set<String> _downloading = {};
  // Decrypted E2EE plaintexts, keyed by message id.
  final Map<String, String> _decrypted = {};
  // Real unread markers: chat ids that received a live message while
  // another chat was open (from the WebSocket — never faked).
  final Set<String> _unreadChatIds = {};
  // Per-message reaction state is rendered from m['reactions'].
  final List<String> _reactionEmojis = ['❤️', '👍', '😂', '😮', '😢', '🙏'];
  // v0.7 chat extras (on-device only): user-created folders, chat lock
  // with a PIN, and per-chat message drafts.
  ChatFolders? _folders;
  ChatLock? _chatLock;
  MessageDrafts? _drafts;
  List<String> _folderNames = [];
  String _folderFilter = ''; // '' = all chats
  String _chatTypeFilter = 'All';
  String _callFilter = 'All';
  // Chats unlocked with the PIN during this session (lock re-applies
  // when the app restarts).
  final Set<String> _unlockedThisSession = {};
  Timer? _draftSaveTimer;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _authAnim =
        AnimationController(vsync: this, duration: const Duration(seconds: 7))
          ..repeat();
    _ringAnim =
        AnimationController(vsync: this, duration: const Duration(seconds: 8))
          ..repeat();
    _remoteRenderer.initialize();
    _initPrefs();
    // Offline mode: reopen the last session from the device cache.
    unawaited(_restoreSession());
    // Stage 3: refresh the in-call duration label once a second
    // while a call is active (cheap: setState only during a call).
    _callTicker = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted && _call != null && _call!.state != CallState.idle) {
        setState(() {});
      }
    });
    _draft.addListener(() {
      final empty = _draft.text.isEmpty;
      if (empty != _draftEmpty) {
        setState(() => _draftEmpty = empty);
      }
      // v0.7 drafts: save the unsent text of the open chat (debounced),
      // except while editing an existing message.
      if (_activeChat != null && _editingId == null) {
        _draftSaveTimer?.cancel();
        _draftSaveTimer = Timer(const Duration(milliseconds: 500), () {
          _drafts?.save(_activeChat!['id'] as String, _draft.text);
        });
      }
    });
  }

  bool _draftEmpty = true;

  Future<void> _initPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    _dataSaver = prefs.getBool('vyro_data_saver') ?? true;
    _api.dataSaver = _dataSaver;
    _deviceId = prefs.getString('vyro_device_id') ?? '';
    if (_deviceId.isEmpty) {
      final rnd = Random.secure();
      _deviceId =
          'mobile-${List.generate(8, (_) => rnd.nextInt(256).toRadixString(16).padLeft(2, '0')).join()}';
      await prefs.setString('vyro_device_id', _deviceId);
    }
    await _usage.load();
    _api.usage = _usage;
    _folders = ChatFolders(prefs);
    _chatLock = ChatLock(prefs);
    _appLock = AppLock(prefs);
    _notifyEnabled = prefs.getBool('vyro_notify_enabled_v1') ?? true;
    _notifySound = prefs.getBool('vyro_notify_sound_v1') ?? true;
    _notifyVibrate = prefs.getBool('vyro_notify_vibrate_v1') ?? true;
    // Wallpaper: id-based setting, with migration from the old
    // Stage 11 single-colour int, plus per-chat overrides.
    _wallpaperId = prefs.getString('vyro_wallpaper_id') ?? 'default';
    if (!wallpapers.containsKey(_wallpaperId)) {
      final old = prefs.getInt('vyro_wallpaper_color');
      _wallpaperId = _wallpaperIdForArgb(old) ?? 'default';
    }
    final perChat = prefs.getString('vyro_chat_wallpapers');
    if (perChat != null) {
      try {
        final decoded = jsonDecode(perChat) as Map<String, dynamic>;
        for (final e in decoded.entries) {
          if (e.value is String && wallpapers.containsKey(e.value as String)) {
            _chatWallpapers[e.key] = e.value as String;
          }
        }
      } catch (_) {}
    }
    _drafts = MessageDrafts(prefs);
    _folderNames = _folders!.list();
    if (mounted) setState(() {});
  }

  Future<void> _setDataSaver(bool on) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool('vyro_data_saver', on);
    setState(() {
      _dataSaver = on;
      _api.dataSaver = on;
    });
  }

  /// Last-seen privacy (Telegram/WhatsApp-style): save the choice on
  /// the server; the previous value stays if saving fails.
  Future<void> _setLastSeenVisibility(String v) async {
    final previous = _lastSeenVisibility;
    setState(() => _lastSeenVisibility = v);
    try {
      await _api.setPrivacy(v);
    } catch (e) {
      setState(() {
        _lastSeenVisibility = previous;
        _error = e.toString();
      });
    }
  }

  /// Status privacy (Stage 2): who may see my status. Same pattern
  /// as last-seen privacy.
  Future<void> _setStatusPrivacy(String v) async {
    final previous = _statusPrivacy;
    setState(() => _statusPrivacy = v);
    try {
      await _api.setStatusPrivacy(v);
    } catch (e) {
      setState(() {
        _statusPrivacy = previous;
        _error = e.toString();
      });
    }
  }

  /// Stage 11: avatar / about / groups privacy. One combined
  /// endpoint; the previous value stays if saving fails.
  Future<void> _setPrivacyField(String field, String v) async {
    const keys = {
      'avatar': 'avatar_privacy',
      'about': 'about_privacy',
      'groups': 'groups_privacy',
    };
    final key = keys[field]!;
    String current() => field == 'avatar'
        ? _avatarPrivacy
        : field == 'about'
            ? _aboutPrivacy
            : _groupsPrivacy;
    void apply(String val) {
      setState(() {
        if (field == 'avatar') {
          _avatarPrivacy = val;
        } else if (field == 'about') {
          _aboutPrivacy = val;
        } else {
          _groupsPrivacy = val;
        }
      });
    }

    final previous = current();
    apply(v);
    try {
      await _api.setPrivacySettings({key: v});
    } catch (e) {
      apply(previous);
      setState(() => _error = e.toString());
    }
  }

  /// Stage 11: read receipts. Off means I send no blue ticks and
  /// see none on my own messages either (enforced server-side).
  Future<void> _setReadReceipts(bool v) async {
    final previous = _readReceipts;
    setState(() => _readReceipts = v);
    try {
      await _api.setPrivacySettings({'read_receipts': v});
    } catch (e) {
      setState(() {
        _readReceipts = previous;
        _error = e.toString();
      });
    }
  }

  /// Wallpaper id for a legacy Stage 11 ARGB colour (migration).
  String? _wallpaperIdForArgb(int? argb) {
    if (argb == null) return null;
    for (final e in wallpapers.entries) {
      if (e.value.length == 1 && e.value.first == argb) return e.key;
    }
    return null;
  }

  /// The wallpaper id in effect for a chat: its own choice, else the
  /// global default. 'default' means the theme chat background.
  String _wallpaperForChat(String? chatId) {
    if (chatId != null && _chatWallpapers.containsKey(chatId)) {
      return _chatWallpapers[chatId]!;
    }
    return _wallpaperId;
  }

  /// Box decoration for a wallpaper id (solid colour or gradient).
  BoxDecoration _wallpaperDecoration(String id, bool dark) {
    final colors = wallpapers[id];
    if (colors == null || colors.isEmpty) {
      return BoxDecoration(
          color: dark
              ? WaColors.darkChatBackground
              : WaColors.lightChatBackground);
    }
    if (colors.length == 1) {
      return BoxDecoration(color: Color(colors.first));
    }
    return BoxDecoration(
      gradient: LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: colors.map((c) => Color(c)).toList(),
      ),
    );
  }

  /// Small preview tile for a wallpaper (solid or gradient).
  Widget _wallpaperTile(String id, bool dark, {double size = 56}) {
    final colors = wallpapers[id];
    final solid =
        colors != null && colors.length == 1 ? Color(colors.first) : null;
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: solid ??
            (dark ? WaColors.darkChatBackground : WaColors.lightChatBackground),
        gradient: colors != null && colors.length > 1
            ? LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: colors.map((c) => Color(c)).toList())
            : null,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.grey.shade300),
        boxShadow: const [
          BoxShadow(
              color: Color(0x22000000), blurRadius: 6, offset: Offset(0, 3)),
        ],
      ),
    );
  }

  /// Set the global default wallpaper (Settings) or one chat's own.
  Future<void> _setWallpaper(String id, {String? chatId}) async {
    final prefs = await SharedPreferences.getInstance();
    if (chatId == null) {
      await prefs.setString('vyro_wallpaper_id', id);
      setState(() => _wallpaperId = id);
    } else {
      if (id == 'default') {
        _chatWallpapers.remove(chatId);
      } else {
        _chatWallpapers[chatId] = id;
      }
      await prefs.setString(
          'vyro_chat_wallpapers', jsonEncode(_chatWallpapers));
      setState(() {});
    }
  }

  /// Beautiful wallpaper picker: a grid of colours and gradients.
  /// chatId == null edits the global default; otherwise this chat
  /// only (Default there means "use my global wallpaper").
  Future<void> _showWallpaperPicker({String? chatId}) async {
    if (!mounted) return;
    final current = chatId == null ? _wallpaperId : _wallpaperForChat(chatId);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final picked = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (context) {
        Widget section(String title, List<String> ids) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(4, 16, 4, 8),
                  child: Text(title,
                      style: const TextStyle(
                          fontSize: 15, fontWeight: FontWeight.w700)),
                ),
                GridView.builder(
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: 3,
                    mainAxisSpacing: 12,
                    crossAxisSpacing: 12,
                    childAspectRatio: 1.35,
                  ),
                  itemCount: ids.length,
                  itemBuilder: (context, i) {
                    final id = ids[i];
                    final selected = id == current;
                    return GestureDetector(
                      onTap: () => Navigator.pop(context, id),
                      child: Column(
                        children: [
                          Stack(
                            alignment: Alignment.center,
                            children: [
                              _wallpaperTile(id, dark),
                              if (selected)
                                Container(
                                  padding: const EdgeInsets.all(2),
                                  decoration: const BoxDecoration(
                                      color: Color(0x80000000),
                                      shape: BoxShape.circle),
                                  child: const Icon(Icons.check,
                                      color: Colors.white, size: 18),
                                ),
                            ],
                          ),
                          const SizedBox(height: 4),
                          Text(wallpaperNames[id] ?? id,
                              style: const TextStyle(fontSize: 12)),
                        ],
                      ),
                    );
                  },
                ),
              ],
            );
        return SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              Center(
                child: Container(
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                      color: Colors.grey.shade300,
                      borderRadius: BorderRadius.circular(2)),
                ),
              ),
              const SizedBox(height: 12),
              Text(
                chatId == null ? 'Chat wallpaper' : 'Wallpaper for this chat',
                style:
                    const TextStyle(fontSize: 18, fontWeight: FontWeight.w800),
              ),
              if (chatId != null)
                const Padding(
                  padding: EdgeInsets.only(top: 4),
                  child: Text(
                      'Only this chat uses it. Choose Default to use your global wallpaper.',
                      style: TextStyle(fontSize: 13, color: Colors.grey)),
                ),
              section('Colours', const [
                'default',
                'beige',
                'mint',
                'sky',
                'rose',
                'slate',
                'lavender',
                'peach',
                'sand',
                'ocean',
                'forest',
                'charcoal',
              ]),
              section('Gradients', const [
                'grad-teal',
                'grad-sunset',
                'grad-ocean',
                'grad-forest',
                'grad-lavender',
                'grad-midnight',
              ]),
            ],
          ),
        );
      },
    );
    // Dismissed, or the same as now: leave the wallpaper alone.
    if (picked == null || picked == current) return;
    await _setWallpaper(picked, chatId: chatId);
  }

  /// Stage 11: blocked contacts — the list from the server, with
  /// an Unblock button per person. Names come from the loaded users.
  Future<void> _showBlockedContacts() async {
    List<dynamic> blocked;
    try {
      blocked = await _api.listBlocked();
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    if (!mounted) return;
    final ids = blocked.cast<String>();
    await showModalBottomSheet(
      context: context,
      builder: (context) => Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Blocked contacts (${ids.length})',
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            if (ids.isEmpty)
              const ListTile(
                title: Text('Nobody blocked'),
                subtitle: Text('Blocked people cannot message or call you. '
                    'Block someone from any chat menu.'),
              ),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: ids.length,
                itemBuilder: (context, i) {
                  final id = ids[i];
                  final name = _usersById[id]?['display_name'] as String? ??
                      _usersById[id]?['username'] as String? ??
                      id;
                  return ListTile(
                    contentPadding: EdgeInsets.zero,
                    leading: WaAvatar(label: name, radius: 20),
                    title: Text(name),
                    trailing: TextButton(
                      child: const Text('Unblock'),
                      onPressed: () async {
                        Navigator.pop(context);
                        try {
                          await _api.unblockUser(id);
                          setState(() => _blockedUserIds.remove(id));
                        } catch (e) {
                          setState(() => _error = e.toString());
                        }
                      },
                    ),
                  );
                },
              ),
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }

  /// Stage 11: delete my account — two confirmations, then the
  /// server deletes the account and everything it owns. The app
  /// signs out to the login screen; the deleted token yields
  /// nothing afterwards.
  Future<void> _deleteAccount() async {
    final first = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete my account?'),
        content: const Text('This permanently deletes your account, '
            'messages, groups, stories and call history. '
            'This cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Continue'),
          ),
        ],
      ),
    );
    if (first != true || !mounted) return;
    final second = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete forever?'),
        content: const Text('Your Safari account will be gone for good. '
            'Tap Delete to confirm.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (second != true || !mounted) return;
    try {
      await _api.deleteAccount();
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    await _ws?.sink.close();
    _api.clearToken();
    // Offline mode: forget the saved session and cache too, so the
    // next person on this phone starts clean.
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('vyro_session_token');
    await prefs.remove('vyro_session_user');
    await prefs.remove('vyro_cached_chats');
    await prefs.remove('vyro_outbox');
    for (final k in prefs
        .getKeys()
        .where((k) => k.startsWith('vyro_cached_msgs_'))
        .toList()) {
      await prefs.remove(k);
    }
    if (!mounted) return;
    setState(() {
      _user = null;
      _codeStep = false;
      _showSettings = false;
      _activeChat = null;
      _messages = [];
      _chats = [];
      _stories = [];
      _calls = [];
      _communities = [];
      _blockedUserIds.clear();
      _error = null;
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Stage 8: arm the app lock when the app leaves the foreground
    // (WhatsApp locks on background). Unlocking needs the PIN or a
    // fingerprint — the chats stay hidden until then.
    if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.hidden ||
        state == AppLifecycleState.detached) {
      if (_appLock?.enabled == true && _user != null && !_locked) {
        setState(() => _locked = true);
      }
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _authAnim.dispose();
    _ringAnim.dispose();
    _ws?.sink.close();
    _call?.dispose();
    _remoteRenderer.dispose();
    _callTicker?.cancel();
    for (final r in _groupRenderers.values) {
      r.dispose();
    }
    _groupRenderers.clear();
    _resendTimer?.cancel();
    _draftSaveTimer?.cancel();
    _codeCtrl.dispose();
    _email.dispose();
    _displayName.dispose();
    _password.dispose();
    _draft.dispose();
    _searchCtrl.dispose();
    super.dispose();
  }

  // Email verification state (v0.6): register and login both send a
  // 6-digit code; no token is issued until the code is verified.
  bool _codeStep = false;
  String _codeEmail = '';
  String _assignedUsername = '';
  String _codePurpose = 'login'; // 'register' | 'login'
  int _resendIn = 0;
  Timer? _resendTimer;
  final _codeCtrl = TextEditingController();

  Future<void> _auth() async {
    setState(() => _error = null);
    try {
      final out = _registerMode
          ? await _api.register(
              _email.text.trim(), _displayName.text.trim(), _password.text)
          : await _api.login(_email.text.trim(), _password.text);
      if (out['status'] == 'code_sent') {
        // No token yet: the user must enter the emailed 6-digit code.
        // On registration the server also tells us the username it
        // chose from the user's name; show it on the code screen.
        setState(() {
          _codeEmail = _email.text.trim();
          _codePurpose = _registerMode ? 'register' : 'login';
          _assignedUsername =
              _registerMode ? (out['username'] as String? ?? '') : '';
          _codeStep = true;
        });
        _startResendTimer();
        return;
      }
      // Older server that still returns a token directly.
      await _finishLogin(out['user'] as Map<String, dynamic>);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  Future<void> _verifyCode() async {
    final code = _codeCtrl.text.trim();
    if (code.length != 6 || int.tryParse(code) == null) {
      setState(() => _error = 'Enter the 6-digit code from your email');
      return;
    }
    setState(() => _error = null);
    try {
      final out = _codePurpose == 'register'
          ? await _api.verifyRegister(_codeEmail, code)
          : await _api.verifyLogin(_codeEmail, code);
      _resendTimer?.cancel();
      setState(() => _codeStep = false);
      await _finishLogin(out['user'] as Map<String, dynamic>);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  Future<void> _resendCode() async {
    if (_resendIn > 0) return;
    try {
      await _api.resendCode(_codeEmail, _codePurpose);
      _startResendTimer();
      setState(() => _error = null);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  void _startResendTimer() {
    _resendTimer?.cancel();
    setState(() => _resendIn = 60);
    _resendTimer = Timer.periodic(const Duration(seconds: 1), (t) {
      if (!mounted) {
        t.cancel();
        return;
      }
      if (_resendIn <= 1) {
        t.cancel();
        setState(() => _resendIn = 0);
      } else {
        setState(() => _resendIn--);
      }
    });
  }

  Future<void> _finishLogin(Map<String, dynamic> u,
      {bool persist = true}) async {
    // Offline mode: remember the session on the device, so the app
    // opens without internet like WhatsApp.
    if (persist) {
      final prefs = await SharedPreferences.getInstance();
      final token = _api.token;
      if (token != null) {
        await prefs.setString('vyro_session_token', token);
      }
      await prefs.setString('vyro_session_user', jsonEncode(u));
    }
    setState(() => _user = u);
    // E2EE setup: generate/load the device identity and publish
    // public keys. Private keys never leave the device.
    _e2ee = E2eeClient(_api, _deviceId);
    _e2ee!.setOwnUserId(u['id'] as String);
    try {
      await _e2ee!.ensureSetup(u['id'] as String);
    } catch (e) {
      // Offline: no error — the chats still open from the cache.
      if (!_isNetworkError(e)) {
        setState(() => _error = 'E2EE setup failed (chats still work '
            'unencrypted): $e');
      }
    }
    _call = CallController(
      sendSignal: (chatId, type, toUserId, payload) {
        _ws?.sink.add(jsonEncode({
          'type': type,
          'chat_id': chatId,
          'to_user_id': toUserId,
          'payload': payload,
        }));
      },
      usage: _usage,
    );
    // Show the remote video through a renderer (RTCVideoView takes
    // a renderer, not a MediaStream): attach each new remote stream.
    _call!.onRemoteStream.listen((s) {
      _remoteRenderer.srcObject = s;
      if (mounted) setState(() {});
    });
    // Stage 3: group-call remote streams get one renderer each.
    _call!.onGroupStreams.listen((_) {
      _syncGroupRenderers();
      if (mounted) setState(() {});
    });
    await _refreshChats();
    // Stage 7: blocked users (for Block/Unblock labels) + the
    // communities for the Communities tab.
    try {
      final blocked = await _api.listBlocked();
      if (mounted) {
        setState(() => _blockedUserIds.addAll(blocked.cast<String>()));
      }
    } catch (_) {}
    await _refreshCommunities();
    // Stage 3: call history for the Calls tab.
    await _refreshCalls();
    // Stage 11: combined privacy settings — last seen, status,
    // avatar/about/groups privacy, read receipts — in one request
    // instead of two. Falls back to the older single-setting
    // endpoints if the server does not have it yet.
    try {
      final s = await _api.getPrivacySettings();
      if (mounted) {
        setState(() {
          _lastSeenVisibility =
              s['last_seen_visibility'] as String? ?? 'everyone';
          _statusPrivacy = s['status_privacy'] as String? ?? 'everyone';
          _avatarPrivacy = s['avatar_privacy'] as String? ?? 'everyone';
          _aboutPrivacy = s['about_privacy'] as String? ?? 'everyone';
          _groupsPrivacy = s['groups_privacy'] as String? ?? 'everyone';
          _readReceipts = s['read_receipts'] as bool? ?? true;
        });
      }
    } catch (_) {
      try {
        final privacy = await _api.getPrivacy();
        if (mounted) {
          setState(() => _lastSeenVisibility =
              privacy['last_seen_visibility'] as String? ?? 'everyone');
        }
      } catch (_) {}
      try {
        final statusPrivacy = await _api.getStatusPrivacy();
        if (mounted) {
          setState(() => _statusPrivacy =
              statusPrivacy['status_privacy'] as String? ?? 'everyone');
        }
      } catch (_) {}
    }
    try {
      final stories = await _api.listStories();
      setState(() => _stories = stories);
    } catch (_) {}
    _connectWs();
    // FCM push: register this device's token so the server can wake
    // the app when a message arrives while it is closed. Best effort:
    // a failure here never breaks login or chat.
    await FcmPush.init(
      api: _api,
      deviceId: _deviceId,
      onOpenChat: (chatId) => _openChatById(chatId),
      onForegroundMessage: (title, body, chatId) {
        if (!mounted || !_notifyEnabled) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('$title: $body'),
            action: chatId != null
                ? SnackBarAction(
                    label: 'Open',
                    onPressed: () => _openChatById(chatId),
                  )
                : null,
          ),
        );
      },
    );
  }

  void _connectWs() {
    final token = _api.token;
    if (token == null) return;
    final wsUrl = _api.baseUrl.replaceFirst('http', 'ws');
    _ws = WebSocketChannel.connect(Uri.parse('$wsUrl/ws?token=$token'));
    _ws!.stream.listen((raw) {
      final ev = jsonDecode(raw) as Map<String, dynamic>;
      final type = ev['type'] as String? ?? '';
      if (type == 'message' && ev['message'] != null) {
        final msg = ev['message'] as Map<String, dynamic>;
        if (_activeChat != null && msg['chat_id'] == _activeChat!['id']) {
          setState(() => _messages = [..._messages, msg]);
          _maybeDecrypt(msg);
        } else {
          // Real unread marker for a chat that is not open.
          setState(() => _unreadChatIds.add(msg['chat_id'] as String));
        }
        _refreshChats();
      } else if (type == 'message_sync' && ev['message'] != null) {
        // Multi-device: our own other device sent a message.
        final msg = ev['message'] as Map<String, dynamic>;
        if (_activeChat != null && msg['chat_id'] == _activeChat!['id']) {
          setState(() => _messages = [..._messages, msg]);
        }
        _refreshChats();
      } else if (type == 'typing' &&
          _activeChat != null &&
          ev['chat_id'] == _activeChat!['id']) {
        setState(() => _typingUser = (ev['user_id'] as String?) ?? 'Someone');
        Future.delayed(const Duration(seconds: 3), () {
          if (mounted) setState(() => _typingUser = '');
        });
      } else if ((type == 'message_read' || type == 'message_delivered') &&
          ev['message_id'] != null) {
        final status = type == 'message_read' ? 'read' : 'delivered';
        setState(() {
          _messages = _messages.map((m) {
            final mm = Map<String, dynamic>.from(m as Map);
            if (mm['id'] == ev['message_id']) mm['status'] = status;
            return mm;
          }).toList();
        });
      } else if (type == 'call-offer' && ev['payload'] != null) {
        final payload = ev['payload'] as Map<String, dynamic>;
        final peerId = ev['user_id'] as String? ?? '';
        final call = payload['group'] == true
            ? _call?.handleGroupOffer(ev['chat_id'] as String, peerId, payload)
            : _call?.handleOffer(ev['chat_id'] as String, peerId, payload);
        if (call != null) setState(() => _incomingCall = call);
      } else if (type == 'call-answer' && ev['payload'] != null) {
        final payload = ev['payload'] as Map<String, dynamic>;
        final peerId = ev['user_id'] as String? ?? '';
        if (_call?.isGroup == true) {
          _call?.handleGroupAnswer(peerId, payload);
        } else {
          _call?.handleAnswer(payload);
        }
      } else if (type == 'call-ice' && ev['payload'] != null) {
        final payload = ev['payload'] as Map<String, dynamic>;
        final peerId = ev['user_id'] as String? ?? '';
        if (_call?.isGroup == true) {
          _call?.handleGroupIce(peerId, payload);
        } else {
          _call?.handleIce(payload);
        }
      } else if (type == 'call-end') {
        // Stage 3: if I started this call and the other side ended
        // it (declined, or hung up while I was ringing), record it
        // as missed/ended before the controller resets.
        final wasConnected = _call?.connected ?? false;
        final duration = _call?.durationSeconds ?? 0;
        final chatId = _call?.chatId ?? '';
        final peerId = _call?.peerId ?? '';
        final video = _call?.video ?? false;
        final group = _call?.isGroup ?? false;
        final initiatedByMe = _callInitiatedByMe;
        final endedBy = ev['user_id'] as String? ?? '';
        if (group) {
          _call?.handleRemoteEndForPeer(endedBy);
        } else {
          _call?.handleRemoteEnd();
        }
        setState(() => _incomingCall = null);
        _callInitiatedByMe = false;
        if (initiatedByMe && chatId.isNotEmpty) {
          _recordEndedCall(
            chatId: chatId,
            peerId: group ? null : peerId,
            video: video,
            wasConnected: wasConnected,
            durationSeconds: duration,
          );
        }
      }
    }, onError: (_) {
      // Offline: the socket died. The cached chats stay on screen;
      // sending queues in the outbox until the internet is back.
      if (mounted) setState(() => _offline = true);
    }, onDone: () {
      if (mounted) setState(() => _offline = true);
    });
  }

  // ── Offline mode (WhatsApp-style): the app opens without
  // internet and shows the on-device cache. ─────────────────────

  /// True when an error means "no internet" rather than a real
  /// server problem (wrong password, expired code, ...).
  bool _isNetworkError(Object e) {
    final s = e.toString();
    return s.contains('SocketException') ||
        s.contains('Failed host lookup') ||
        s.contains('Connection refused') ||
        s.contains('Network is unreachable') ||
        s.contains('Connection closed') ||
        s.contains('timed out') ||
        s.contains('HttpException') ||
        s.contains('ClientException');
  }

  /// Slim amber banner shown while offline.
  Widget _offlineBanner() {
    if (!_offline) return const SizedBox.shrink();
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Container(
      width: double.infinity,
      color: dark ? const Color(0xFF3A2E0A) : Colors.amber.shade100,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
        children: [
          Icon(Icons.wifi_off,
              size: 16, color: dark ? const Color(0xFFFBBF24) : null),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              "You're offline — showing saved chats. Messages will send when you're back online.",
              style: TextStyle(
                  fontSize: 12, color: dark ? const Color(0xFFFDE68A) : null),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _saveCachedChats() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString('vyro_cached_chats', jsonEncode(_chats));
    } catch (_) {}
  }

  Future<void> _loadCachedChats() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString('vyro_cached_chats');
      if (raw == null) return;
      final decoded = jsonDecode(raw) as List;
      if (mounted) {
        setState(() => _chats = List<dynamic>.from(decoded));
      }
    } catch (_) {}
  }

  Future<void> _saveCachedMessages(String chatId) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final recent = _messages.length > 100
          ? _messages.sublist(_messages.length - 100)
          : _messages;
      await prefs.setString('vyro_cached_msgs_$chatId', jsonEncode(recent));
    } catch (_) {}
  }

  Future<List<dynamic>> _loadCachedMessages(String chatId) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString('vyro_cached_msgs_$chatId');
      if (raw == null) return [];
      return List<dynamic>.from(jsonDecode(raw) as List);
    } catch (_) {
      return [];
    }
  }

  Future<void> _loadOutbox() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString('vyro_outbox');
      if (raw != null) {
        final decoded = jsonDecode(raw) as List;
        _outbox
          ..clear()
          ..addAll(decoded.map((e) => Map<String, dynamic>.from(e as Map)));
      }
    } catch (_) {}
  }

  Future<void> _saveOutbox() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString('vyro_outbox', jsonEncode(_outbox));
    } catch (_) {}
  }

  /// Messages typed while offline for one chat, as chat bubbles
  /// with a clock (pending) instead of ticks.
  List<Map<String, dynamic>> _queuedMessagesFor(String chatId) {
    return _outbox
        .where((o) => o['chat_id'] == chatId)
        .map((o) => <String, dynamic>{
              'id': o['local_id'],
              'chat_id': chatId,
              'sender_id': _user?['id'],
              'content': o['content'],
              'kind': 'text',
              'pending': true,
              'created_at': o['created_at'],
            })
        .toList();
  }

  /// Send the messages that were queued while offline, oldest first.
  /// Stops at the first failure (still offline) and keeps the rest.
  Future<void> _flushOutbox() async {
    if (_outbox.isEmpty || _offline) return;
    final pending = List<Map<String, dynamic>>.from(_outbox);
    for (final item in pending) {
      try {
        final msg = await _api.sendMessage(
          item['chat_id'] as String,
          item['content'] as String,
          replyToId: item['reply_to'] as String?,
        );
        _outbox.remove(item);
        await _saveOutbox();
        if (mounted &&
            _activeChat != null &&
            _activeChat!['id'] == item['chat_id']) {
          setState(() {
            _messages = _messages.map((m) {
              final mm = Map<String, dynamic>.from(m as Map);
              return mm['id'] == item['local_id'] ? msg : mm;
            }).toList();
          });
        }
      } catch (_) {
        break;
      }
    }
  }

  /// Reopen the last session without internet, like WhatsApp: the
  /// saved token signs the app in and the cached chats show at once.
  Future<void> _restoreSession() async {
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString('vyro_session_token');
    final rawUser = prefs.getString('vyro_session_user');
    if (token == null || rawUser == null) return;
    try {
      final u = jsonDecode(rawUser) as Map<String, dynamic>;
      _api.token = token;
      await _loadCachedChats();
      await _loadOutbox();
      await _finishLogin(u, persist: false);
    } catch (_) {
      // Corrupt or expired session: fall through to the login page.
      _api.clearToken();
    }
  }

  Future<void> _refreshChats() async {
    try {
      final chats = await _api.listChats();
      setState(() {
        _chats = chats;
        _offline = false;
      });
      await _saveCachedChats();
      await _flushOutbox();
    } catch (e) {
      if (_isNetworkError(e)) {
        // Offline: keep the cached chats on screen, no scary error.
        if (mounted) setState(() => _offline = true);
        if (_chats.isEmpty) await _loadCachedChats();
      } else {
        setState(() => _error = e.toString());
      }
    }
    // Best-effort user directory for DM titles (never blocks the list).
    try {
      final users = await _api.listUsers();
      final map = <String, Map<String, dynamic>>{};
      for (final u in users) {
        final m = u as Map<String, dynamic>;
        final id = m['id'] as String?;
        if (id != null) map[id] = m;
      }
      if (mounted) setState(() => _usersById = map);
    } catch (_) {}
  }

  /// Stage 6: load my communities for the Communities tab.
  Future<void> _refreshCommunities() async {
    try {
      final communities = await _api.listCommunities();
      if (mounted) setState(() => _communities = communities);
    } catch (_) {}
  }

  /// Display title for a chat: its own title, else — for a DM — the
  /// other person's display name (Telegram/WhatsApp show the person,
  /// never the literal word "dm").
  String _chatTitle(Map<String, dynamic> chat) {
    final t = (chat['title'] as String?)?.trim();
    if (t != null && t.isNotEmpty) return t;
    if (chat['type'] == 'dm') {
      final me = _user?['id'] as String?;
      final members =
          (chat['member_ids'] as List?)?.map((m) => '$m').toList() ??
              const <String>[];
      final peerId = members.firstWhere((m) => m != me, orElse: () => '');
      final peer = _usersById[peerId];
      if (peer != null) {
        final name = (peer['display_name'] as String?)?.trim();
        if (name != null && name.isNotEmpty) return name;
        final uname = (peer['username'] as String?)?.trim();
        if (uname != null && uname.isNotEmpty) return '@$uname';
      }
    }
    return (chat['type'] as String?) ?? 'dm';
  }

  /// Decrypt an E2EE message in place (stores plaintext in
  /// _decrypted so the UI never shows ciphertext).
  Future<void> _maybeDecrypt(Map<String, dynamic> msg) async {
    if (msg['kind'] != 'e2ee' || _e2ee == null) return;
    final id = msg['id'] as String;
    if (_decrypted.containsKey(id)) return;
    try {
      final chatType = _activeChat?['type'] as String? ?? 'dm';
      final text = chatType == 'dm'
          ? await _e2ee!
              .decryptDm(msg['sender_id'] as String, msg['content'] as String)
          : await _e2ee!
              .decryptGroup(msg['chat_id'] as String, msg['content'] as String);
      if (mounted) setState(() => _decrypted[id] = text);
    } catch (_) {
      if (mounted) {
        setState(() => _decrypted[id] = '🔒 Encrypted message (key missing)');
      }
    }
  }

  /// Opens a chat by id (used when the user taps an FCM notification).
  /// Does nothing when the chat is not in the loaded list — the user
  /// can still open it from the chat list.
  void _openChatById(String chatId) {
    for (final c in _chats) {
      if (c is Map<String, dynamic> && c['id'] == chatId) {
        unawaited(_openChat(c));
        return;
      }
    }
  }

  Future<void> _openChat(Map<String, dynamic> chat) async {
    final chatId = chat['id'] as String;
    // v0.7 chat lock: a locked chat asks for the PIN first (unless it
    // was already unlocked during this session).
    if (_chatLock?.isLocked(chatId) == true &&
        !_unlockedThisSession.contains(chatId)) {
      final ok = await _askPin('Enter PIN to open this chat');
      if (ok != true) return;
      _unlockedThisSession.add(chatId);
    }
    setState(() {
      _activeChat = chat;
      _replyTo = null;
      _editingId = null;
      _peerIdCache = '';
      _unreadChatIds.remove(chat['id'] as String);
      // v0.7 drafts: restore the unsent text of this chat, if any.
      _draft.text = _drafts?.draftFor(chatId) ?? '';
    });
    // Offline mode: show the saved messages at once, then refresh.
    final cached = await _loadCachedMessages(chatId);
    if (cached.isNotEmpty && mounted) {
      setState(() => _messages = cached);
      for (final m in cached) {
        _maybeDecrypt(m as Map<String, dynamic>);
      }
    }
    // Messages typed while offline, still waiting to send.
    final queued = _queuedMessagesFor(chatId);
    if (queued.isNotEmpty && mounted) {
      setState(() => _messages = [..._messages, ...queued]);
    }
    try {
      final msgs = await _api.listMessages(chat['id'] as String);
      if (!mounted) return;
      setState(() {
        _messages = [...msgs, ...queued];
        _offline = false;
      });
      await _saveCachedMessages(chatId);
      for (final m in msgs) {
        _maybeDecrypt(m as Map<String, dynamic>);
      }
      final fromOthers =
          msgs.where((m) => (m as Map)['sender_id'] != _user!['id']).toList();
      if (fromOthers.isNotEmpty) {
        try {
          await _api.markRead((fromOthers.last as Map)['id'] as String);
        } catch (_) {}
      }
    } catch (e) {
      // Offline: the saved messages above stay on screen.
      if (_isNetworkError(e) && mounted) {
        setState(() => _offline = true);
      }
    }
  }

  Future<void> _refreshMessages() async {
    if (_activeChat == null) return;
    try {
      final msgs = await _api.listMessages(_activeChat!['id'] as String);
      if (mounted) {
        final queued = _queuedMessagesFor(_activeChat!['id'] as String);
        setState(() {
          _messages = [...msgs, ...queued];
          _offline = false;
        });
        await _saveCachedMessages(_activeChat!['id'] as String);
        for (final m in msgs) {
          _maybeDecrypt(m as Map<String, dynamic>);
        }
        // Stage 5: opening a channel counts my view of each post
        // from others (one row per viewer; fire-and-forget).
        if (_activeChat!['type'] == 'channel') {
          for (final m in msgs) {
            final mm = m as Map<String, dynamic>;
            if (mm['sender_id'] != _user?['id']) {
              unawaited(_api
                  .recordMessageView(mm['id'] as String)
                  .catchError((_) => <String, dynamic>{}));
            }
          }
        }
      }
    } catch (_) {}
  }

  void _sendTyping() {
    if (_activeChat == null || _ws == null) return;
    final now = DateTime.now();
    if (now.difference(_lastTypingAt).inSeconds < 2) return;
    _lastTypingAt = now;
    _ws!.sink
        .add(jsonEncode({'type': 'typing', 'chat_id': _activeChat!['id']}));
  }

  Future<void> _send() async {
    if (_activeChat == null || _draft.text.trim().isEmpty) return;
    final content = _draft.text;
    final replyTo = _replyTo;
    final editingId = _editingId;
    _draft.clear();
    // v0.7 drafts: the text was sent, so the saved draft goes away.
    await _drafts?.clear(_activeChat!['id'] as String);
    setState(() {
      _replyTo = null;
      _editingId = null;
    });
    try {
      if (editingId != null) {
        final msg = await _api.editMessage(editingId, content);
        setState(() {
          _messages = _messages.map((m) {
            final mm = Map<String, dynamic>.from(m as Map);
            return mm['id'] == editingId ? msg : mm;
          }).toList();
        });
        return;
      }
      final chatId = _activeChat!['id'] as String;
      final chatType = _activeChat!['type'] as String? ?? 'dm';
      Map<String, dynamic> msg;
      if (_e2eeChats.contains(chatId) && _e2ee != null) {
        if (chatType == 'dm') {
          final otherId = await _peerUserId();
          if (otherId.isEmpty) {
            throw Exception('Cannot encrypt: peer user not found');
          }
          final env = await _e2ee!.encryptDm(otherId, content);
          msg = await _api.sendE2eeMessage(chatId, env);
          _decrypted[msg['id'] as String] = content;
        } else {
          final members = await _api.listMembers(chatId);
          final memberIds =
              members.map((m) => (m as Map)['user_id'] as String).toList();
          final env = await _e2ee!.encryptGroup(chatId, content, memberIds);
          msg = await _api.sendE2eeMessage(chatId, env);
          _decrypted[msg['id'] as String] = content;
        }
      } else {
        msg = await _api.sendMessage(chatId, content,
            ttlSeconds: _ttlSeconds == 0 ? null : _ttlSeconds,
            replyToId: replyTo == null ? null : replyTo['id'] as String);
      }
      setState(() => _messages = [..._messages, msg]);
    } catch (e) {
      // Offline: queue the message like WhatsApp — it goes out when
      // the internet is back. (Plain chats only: E2EE sends need the
      // server for keys, so those still report the error.)
      final chatId = _activeChat?['id'] as String?;
      if (_isNetworkError(e) &&
          editingId == null &&
          chatId != null &&
          !_e2eeChats.contains(chatId)) {
        final localId = 'pending-${DateTime.now().millisecondsSinceEpoch}';
        final pending = <String, dynamic>{
          'id': localId,
          'chat_id': chatId,
          'sender_id': _user?['id'],
          'content': content,
          'kind': 'text',
          'pending': true,
          'created_at': DateTime.now().toIso8601String(),
        };
        _outbox.add({
          'local_id': localId,
          'chat_id': chatId,
          'content': content,
          'reply_to': replyTo == null ? null : replyTo['id'] as String,
          'created_at': pending['created_at'],
        });
        await _saveOutbox();
        setState(() {
          _offline = true;
          _messages = [..._messages, pending];
        });
        return;
      }
      setState(() => _error = e.toString());
    }
  }

  Future<void> _shareLocation() async {
    if (_activeChat == null) return;
    final latCtrl = TextEditingController();
    final lngCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Share location'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
                controller: latCtrl,
                decoration:
                    const InputDecoration(labelText: 'Latitude (-90..90)'),
                keyboardType: const TextInputType.numberWithOptions(
                    signed: true, decimal: true)),
            TextField(
                controller: lngCtrl,
                decoration:
                    const InputDecoration(labelText: 'Longitude (-180..180)'),
                keyboardType: const TextInputType.numberWithOptions(
                    signed: true, decimal: true)),
          ],
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Send')),
        ],
      ),
    );
    if (ok != true) return;
    final lat = double.tryParse(latCtrl.text.trim());
    final lng = double.tryParse(lngCtrl.text.trim());
    if (lat == null || lng == null) {
      setState(() => _error = 'Enter valid latitude and longitude');
      return;
    }
    try {
      final msg =
          await _api.sendLocation(_activeChat!['id'] as String, lat, lng);
      setState(() => _messages = [..._messages, msg]);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  Future<void> _scheduleDraft() async {
    if (_activeChat == null || _draft.text.trim().isEmpty) return;
    final minutesCtrl = TextEditingController(text: '60');
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Schedule message'),
        content: TextField(
            controller: minutesCtrl,
            decoration:
                const InputDecoration(labelText: 'Send in how many minutes?'),
            keyboardType: TextInputType.number),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Schedule')),
        ],
      ),
    );
    if (ok != true) return;
    final minutes = int.tryParse(minutesCtrl.text.trim());
    if (minutes == null || minutes <= 0) return;
    try {
      await _api.scheduleMessage(_activeChat!['id'] as String, _draft.text,
          DateTime.now().add(Duration(minutes: minutes)).toIso8601String());
      _draft.clear();
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Create a group or channel: pick members, give it a title.
  Future<void> _createGroupOrChannel({bool channel = false}) async {
    final users = await _api.listUsers();
    if (!mounted) return;
    String type = channel ? 'channel' : 'group';
    // Stage 5: channels are public (discoverable) unless the
    // creator marks them private.
    bool channelPublic = true;
    final titleCtrl = TextEditingController();
    final selected = <String>{};
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: Text(channel ? 'New channel' : 'New group / channel'),
          content: SizedBox(
            width: double.maxFinite,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Row(children: [
                  ChoiceChip(
                      label: const Text('Group'),
                      selected: type == 'group',
                      onSelected: (_) => setDialogState(() => type = 'group')),
                  const SizedBox(width: 8),
                  ChoiceChip(
                      label: const Text('Channel'),
                      selected: type == 'channel',
                      onSelected: (_) =>
                          setDialogState(() => type = 'channel')),
                ]),
                if (type == 'channel')
                  SwitchListTile(
                    title: const Text('Public channel'),
                    subtitle: const Text('Anyone can find and follow it'),
                    value: channelPublic,
                    onChanged: (v) => setDialogState(() => channelPublic = v),
                    contentPadding: EdgeInsets.zero,
                  ),
                TextField(
                    controller: titleCtrl,
                    decoration: const InputDecoration(labelText: 'Title')),
                const SizedBox(height: 8),
                Flexible(
                  child: ListView(
                    shrinkWrap: true,
                    children: users.map((u) {
                      final m = u as Map<String, dynamic>;
                      final id = m['id'] as String;
                      final name = (m['username'] as String?) ?? id;
                      return CheckboxListTile(
                        title: Text(name),
                        value: selected.contains(id),
                        onChanged: (v) => setDialogState(() {
                          if (v == true) {
                            selected.add(id);
                          } else {
                            selected.remove(id);
                          }
                        }),
                      );
                    }).toList(),
                  ),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Create')),
          ],
        ),
      ),
    );
    if (ok != true || (selected.isEmpty && type != 'channel')) return;
    try {
      final chat = await _api.createChat(type, selected.toList(),
          title: titleCtrl.text.trim(),
          public: type == 'channel' ? channelPublic : null);
      await _refreshChats();
      await _openChat(chat);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Join a group/channel with a pasted invite link or code: preview
  /// the chat first (title, type, member count), then join.
  Future<void> _joinViaInvite() async {
    final ctrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Join via invite link'),
        content: TextField(
          controller: ctrl,
          decoration: const InputDecoration(
            labelText: 'Paste the invite link or code',
            hintText: 'https://…/join/Ab3x…',
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Continue')),
        ],
      ),
    );
    if (ok != true) return;
    final token = ApiClient.extractInviteToken(ctrl.text);
    if (token.isEmpty) return;
    try {
      final preview = await _api.previewInvite(token);
      if (!mounted) return;
      final join = await showDialog<bool>(
        // ignore: use_build_context_synchronously
        context: context,
        builder: (context) => AlertDialog(
          title: Text(preview['title'] as String? ?? 'Join chat'),
          content: Text(
              '${preview['type'] ?? 'group'} · ${preview['member_count'] ?? '?'} members'),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Join')),
          ],
        ),
      );
      if (join != true) return;
      final res = await _api.joinViaInvite(token);
      await _refreshChats();
      if (!mounted) return;
      final chatId = res['chat_id'] as String?;
      final chat = _chats.firstWhere((c) => (c as Map)['id'] == chatId,
          orElse: () => null);
      if (chat != null) {
        await _openChat(chat as Map<String, dynamic>);
      }
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 5: rename a channel and flip it between public and
  /// private (admins and owner only).
  Future<void> _editChannel() async {
    if (_activeChat == null) return;
    final chatId = _activeChat!['id'] as String;
    final titleCtrl =
        TextEditingController(text: (_activeChat!['title'] as String?) ?? '');
    bool isPublic = _activeChat!['is_public'] == true;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Edit channel'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: titleCtrl,
                decoration: const InputDecoration(labelText: 'Channel name'),
                maxLength: 100,
              ),
              SwitchListTile(
                title: const Text('Public channel'),
                subtitle: const Text('Anyone can find and follow it'),
                value: isPublic,
                onChanged: (v) => setDialogState(() => isPublic = v),
                contentPadding: EdgeInsets.zero,
              ),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Save')),
          ],
        ),
      ),
    );
    if (ok != true) return;
    try {
      final updated = await _api.updateChat(chatId,
          title: titleCtrl.text.trim(), public: isPublic);
      await _refreshChats();
      setState(() => _activeChat = updated);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Members + admin sheet for a group/channel.
  Future<void> _showMembers() async {
    if (_activeChat == null) return;
    final chatId = _activeChat!['id'] as String;
    try {
      final members = await _api.listMembers(chatId);
      // The active invite link, if any (owners/admins manage it).
      List<dynamic> invites = [];
      try {
        invites = await _api.listInvites(chatId);
      } catch (_) {
        // Non-admins get a 403 here; the members list still shows.
      }
      final activeInvites =
          invites.where((i) => (i as Map)['revoked'] != true).toList();
      final activeInvite = activeInvites.isEmpty
          ? null
          : activeInvites.first as Map<String, dynamic>;
      final myRole = members.map((m) => m as Map<String, dynamic>).firstWhere(
              (m) => m['user_id'] == _user!['id'],
              orElse: () => <String, dynamic>{})['role'] as String? ??
          '';
      final canManageInvite = myRole == 'owner' || myRole == 'admin';
      if (!mounted) return;
      await showModalBottomSheet(
        context: context,
        builder: (context) => Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Members (${members.length})',
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 8),
              // Stage 5: channel admins can rename the channel and
              // flip it between public and private.
              if (canManageInvite && _activeChat!['type'] == 'channel')
                ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: const Icon(Icons.edit_outlined),
                  title: const Text('Edit channel'),
                  subtitle: Text(_activeChat!['is_public'] == true
                      ? 'Public channel'
                      : 'Private channel'),
                  onTap: () async {
                    Navigator.pop(context);
                    await _editChannel();
                  },
                ),
              // Invite link (Telegram-style): owners/admins create one
              // link anyone can join with; revoke it to close the door.
              if (canManageInvite) ...[
                ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: const Icon(Icons.link),
                  title: const Text('Invite link'),
                  subtitle: Text(activeInvite == null
                      ? 'No active link — create one to let people join'
                      : activeInvite['link'] as String? ?? ''),
                  trailing: activeInvite == null
                      ? FilledButton(
                          onPressed: () async {
                            Navigator.pop(context);
                            try {
                              await _api.createInvite(chatId);
                              await _showMembers();
                            } catch (e) {
                              setState(() => _error = e.toString());
                            }
                          },
                          child: const Text('Create'),
                        )
                      : Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: const Icon(Icons.copy),
                              tooltip: 'Copy link',
                              onPressed: () async {
                                await Clipboard.setData(ClipboardData(
                                    text: activeInvite['link'] as String));
                                if (!context.mounted) return;
                                ScaffoldMessenger.of(context).showSnackBar(
                                  const SnackBar(
                                      content: Text('Invite link copied')),
                                );
                              },
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline),
                              tooltip: 'Revoke link',
                              onPressed: () async {
                                Navigator.pop(context);
                                try {
                                  await _api.revokeInvite(
                                      chatId, activeInvite['token'] as String);
                                  await _showMembers();
                                } catch (e) {
                                  setState(() => _error = e.toString());
                                }
                              },
                            ),
                          ],
                        ),
                ),
                const Divider(),
              ],
              Flexible(
                child: ListView(
                  shrinkWrap: true,
                  children: members.map((m) {
                    final mm = m as Map<String, dynamic>;
                    final uid = mm['user_id'] as String;
                    final role = mm['role'] as String? ?? 'member';
                    final isMe = uid == _user!['id'];
                    return ListTile(
                      title: Text(isMe ? '$uid (you)' : uid),
                      subtitle: Text('Role: $role'),
                      trailing: isMe
                          ? null
                          : PopupMenuButton<String>(
                              onSelected: (v) async {
                                Navigator.pop(context);
                                try {
                                  if (v == 'remove') {
                                    await _api.removeMember(chatId, uid);
                                  } else {
                                    await _api.setMemberRole(chatId, uid, v);
                                  }
                                } catch (e) {
                                  setState(() => _error = e.toString());
                                }
                              },
                              itemBuilder: (_) => const [
                                PopupMenuItem(
                                    value: 'admin', child: Text('Make admin')),
                                PopupMenuItem(
                                    value: 'member',
                                    child: Text('Make member')),
                                PopupMenuItem(
                                    value: 'remove', child: Text('Remove')),
                              ],
                            ),
                    );
                  }).toList(),
                ),
              ),
              const SizedBox(height: 8),
              FilledButton.icon(
                icon: const Icon(Icons.person_add),
                label: const Text('Add member'),
                onPressed: () async {
                  Navigator.pop(context);
                  final users = await _api.listUsers();
                  if (!mounted) return;
                  final picked = await showDialog<Map<String, dynamic>>(
                    // ignore: use_build_context_synchronously
                    context: context,
                    builder: (context) => SimpleDialog(
                      title: const Text('Add member'),
                      children: users
                          .map((u) => SimpleDialogOption(
                                onPressed: () => Navigator.pop(context, u),
                                child: Text(u['username'] as String? ??
                                    u['id'] as String),
                              ))
                          .toList(),
                    ),
                  );
                  if (picked != null) {
                    try {
                      await _api.addMember(chatId, picked['id'] as String);
                    } catch (e) {
                      setState(() => _error = e.toString());
                    }
                  }
                },
              ),
            ],
          ),
        ),
      );
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  Future<void> _showScheduled() async {
    if (_activeChat == null) return;
    try {
      final items = await _api.listScheduled(_activeChat!['id'] as String);
      if (!mounted) return;
      await showModalBottomSheet(
        context: context,
        builder: (context) => Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Scheduled messages (${items.length})',
                  style: Theme.of(context).textTheme.titleMedium),
              Flexible(
                child: ListView(
                  shrinkWrap: true,
                  children: items.map((s) {
                    final sm = s as Map<String, dynamic>;
                    return ListTile(
                      title: Text(sm['content'] as String? ?? ''),
                      subtitle: Text('Sends at ${sm['send_at']}'),
                      trailing: IconButton(
                        icon: const Icon(Icons.cancel_outlined),
                        tooltip: 'Cancel',
                        onPressed: () async {
                          await _api.cancelScheduled(sm['id'] as String);
                          if (context.mounted) Navigator.pop(context);
                        },
                      ),
                    );
                  }).toList(),
                ),
              ),
            ],
          ),
        ),
      );
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  Future<void> _search() async {
    final q = _searchCtrl.text.trim();
    if (q.isEmpty) return;
    setState(() => _searching = true);
    try {
      final results = await _api.searchMessages(q);
      setState(() => _searchResults = results);
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _searching = false);
    }
  }

  /// Post a status: choose the kind — photo/video, text, or voice
  /// (Stage 2). No fake statuses: failures are shown as errors.
  Future<void> _postStory() async {
    final choice = await showModalBottomSheet<String>(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.photo_camera_outlined),
              title: const Text('Photo or video'),
              onTap: () => Navigator.pop(context, 'photo'),
            ),
            ListTile(
              leading: const Icon(Icons.text_fields_outlined),
              title: const Text('Text status'),
              onTap: () => Navigator.pop(context, 'text'),
            ),
            ListTile(
              leading: const Icon(Icons.mic_outlined),
              title: const Text('Voice status'),
              onTap: () => Navigator.pop(context, 'voice'),
            ),
          ],
        ),
      ),
    );
    if (choice == 'photo') {
      await _postPhotoStory();
    } else if (choice == 'text') {
      await _postTextStory();
    } else if (choice == 'voice') {
      await _postVoiceStory();
    }
  }

  /// Post a photo story: pick a photo, compress it (Data Saver),
  /// upload, create the story. No fake stories: failures are shown
  /// as errors.
  Future<void> _postPhotoStory() async {
    final picker = ImagePicker();
    final picked =
        await picker.pickImage(source: ImageSource.gallery, maxWidth: 1920);
    if (picked == null) return;
    try {
      final bytes = await picked.readAsBytes();
      final compressed = DataSaver.compressImage(bytes);
      final up = await _api.uploadFile(compressed,
          'story-${DateTime.now().millisecondsSinceEpoch}.jpg', 'image/jpeg');
      final captionCtrl = TextEditingController();
      if (!mounted) return;
      final ok = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Post story'),
          content: TextField(
              controller: captionCtrl,
              decoration:
                  const InputDecoration(labelText: 'Caption (optional)')),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Post')),
          ],
        ),
      );
      if (ok != true) return;
      await _api.createStory(up['url'] as String,
          caption: captionCtrl.text.trim());
      final stories = await _api.listStories();
      setState(() => _stories = stories);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Text statuses (Stage 2): the same background colours the server
  /// allows, so a colour the server would reject can never be picked.
  static const List<String> _textStatusColors = [
    '#008069',
    '#25D366',
    '#7C3AED',
    '#DC2626',
    '#EA580C',
    '#CA8A04',
    '#0EA5E9',
    '#EC4899',
    '#111B21',
  ];

  static Color _colorFromHex(String hex) {
    final h = hex.replaceFirst('#', '');
    return Color(int.parse('FF$h', radix: 16));
  }

  /// Post a text status (Stage 2): text + a background colour.
  Future<void> _postTextStory() async {
    final textCtrl = TextEditingController();
    String color = _textStatusColors[0];
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Text status'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: textCtrl,
                maxLength: 700,
                maxLines: 4,
                decoration:
                    const InputDecoration(labelText: 'Write your status…'),
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                children: [
                  for (final c in _textStatusColors)
                    GestureDetector(
                      onTap: () => setDialogState(() => color = c),
                      child: Container(
                        width: 32,
                        height: 32,
                        decoration: BoxDecoration(
                          color: _colorFromHex(c),
                          shape: BoxShape.circle,
                          border: Border.all(
                            color:
                                color == c ? Colors.white : Colors.transparent,
                            width: 3,
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Post')),
          ],
        ),
      ),
    );
    if (ok != true) return;
    final text = textCtrl.text.trim();
    if (text.isEmpty) return;
    try {
      await _api.createStory('',
          kind: 'text', textContent: text, backgroundColor: color);
      final stories = await _api.listStories();
      setState(() => _stories = stories);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Post a voice status (Stage 2): record with the same Data Saver
  /// Opus 12 kbps encoder as voice messages, upload, create the story.
  Future<void> _postVoiceStory() async {
    final recorder = AudioRecorder();
    bool recording = false;
    int seconds = 0;
    String? mime;
    Timer? timer;
    try {
      final posted = await showDialog<bool>(
        context: context,
        builder: (context) => StatefulBuilder(
          builder: (context, setDialogState) {
            Future<void> toggle() async {
              if (recording) {
                timer?.cancel();
                final path = await recorder.stop();
                setDialogState(() => recording = false);
                if (path == null || path.isEmpty) {
                  setState(() => _error = 'Recording produced no file');
                  if (context.mounted) Navigator.pop(context, false);
                  return;
                }
                try {
                  final bytes = await readRecordedBytes(path);
                  if (bytes.isEmpty) {
                    throw Exception('Recording is empty');
                  }
                  final fileName = path.split('/').last.split('?').first;
                  final up = await _api.uploadFile(
                      bytes, fileName, mime ?? 'audio/ogg',
                      category: UsageCategory.voice);
                  await _api.createStory(up['url'] as String, kind: 'audio');
                  final stories = await _api.listStories();
                  setState(() => _stories = stories);
                  if (context.mounted) Navigator.pop(context, true);
                } catch (e) {
                  setState(() => _error = e.toString());
                  if (context.mounted) Navigator.pop(context, false);
                }
                return;
              }
              if (!await recorder.hasPermission()) {
                setState(() => _error = 'Microphone permission denied');
                if (context.mounted) Navigator.pop(context, false);
                return;
              }
              final startedMime = await startVoiceRecording(recorder);
              if (startedMime == null) {
                setState(() => _error = 'Could not start recording');
                if (context.mounted) Navigator.pop(context, false);
                return;
              }
              mime = startedMime;
              seconds = 0;
              timer = Timer.periodic(
                const Duration(seconds: 1),
                (_) => setDialogState(() => seconds++),
              );
              setDialogState(() => recording = true);
            }

            final m = (seconds ~/ 60).toString().padLeft(2, '0');
            final s = (seconds % 60).toString().padLeft(2, '0');
            return AlertDialog(
              title: const Text('Voice status'),
              content: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(recording ? '● Recording $m:$s' : 'Tap to record',
                      style: const TextStyle(fontSize: 16)),
                  const SizedBox(height: 12),
                  IconButton(
                    icon: Icon(
                      recording
                          ? Icons.stop_circle_outlined
                          : Icons.mic_outlined,
                      size: 48,
                    ),
                    color: WaColors.accent,
                    onPressed: toggle,
                  ),
                  if (recording)
                    const Text('Tap stop to post your voice status.'),
                ],
              ),
              actions: [
                TextButton(
                  onPressed: () async {
                    timer?.cancel();
                    if (recording) await recorder.stop();
                    if (context.mounted) Navigator.pop(context, false);
                  },
                  child: const Text('Cancel'),
                ),
              ],
            );
          },
        ),
      );
      if (posted != true) {
        timer?.cancel();
        try {
          await recorder.stop();
        } catch (_) {}
      }
    } finally {
      timer?.cancel();
      recorder.dispose();
    }
  }

  /// Send a photo from the gallery (compressed first — Data Saver).
  Future<void> _sendPhoto() async {
    if (_activeChat == null) return;
    final picker = ImagePicker();
    final picked =
        await picker.pickImage(source: ImageSource.gallery, maxWidth: 1920);
    if (picked == null) return;
    try {
      final bytes = await picked.readAsBytes();
      final compressed = DataSaver.compressImage(bytes);
      final up = await _api.uploadFile(compressed,
          'photo-${DateTime.now().millisecondsSinceEpoch}.jpg', 'image/jpeg');
      final msg = await _api.sendMediaMessage(_activeChat!['id'] as String, up);
      setState(() => _messages = [..._messages, msg]);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Data Saver: download a media file only when the user taps it
  /// (or immediately when Data Saver is off).
  Future<void> _downloadMedia(String fileUrl) async {
    if (_downloaded.containsKey(fileUrl) || _downloading.contains(fileUrl)) {
      return;
    }
    setState(() => _downloading.add(fileUrl));
    try {
      final bytes = await _api.downloadFile(fileUrl);
      setState(() {
        _downloaded[fileUrl] = bytes;
        _downloading.remove(fileUrl);
      });
    } catch (e) {
      setState(() {
        _downloading.remove(fileUrl);
        _error = e.toString();
      });
    }
  }

  void _showMessageActions(Map<String, dynamic> m) {
    final mine = m['sender_id'] == _user!['id'];
    final content =
        _decrypted[m['id'] as String] ?? (m['content'] as String? ?? '');
    final isPinned = (_activeChat?['pinned_message'] as Map?)?['id'] == m['id'];
    showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.reply),
              title: const Text('Reply'),
              onTap: () {
                Navigator.pop(context);
                setState(() => _replyTo = m);
              },
            ),
            ListTile(
              leading: const Icon(Icons.emoji_emotions_outlined),
              title: const Text('React'),
              onTap: () {
                Navigator.pop(context);
                _showReactions(m);
              },
            ),
            // Stage 1: pin / unpin this message in the chat.
            ListTile(
              leading:
                  Icon(isPinned ? Icons.push_pin : Icons.push_pin_outlined),
              title: Text(isPinned ? 'Unpin' : 'Pin'),
              onTap: () async {
                Navigator.pop(context);
                final chatId = _activeChat?['id'] as String;
                try {
                  if (isPinned) {
                    await _api.unpinMessage(chatId);
                  } else {
                    await _api.pinMessage(chatId, m['id'] as String);
                  }
                  await _refreshChats();
                  // Keep the open chat's pinned banner in sync.
                  final fresh = _chats.firstWhere(
                      (c) => (c as Map)['id'] == chatId,
                      orElse: () => _activeChat!);
                  setState(() => _activeChat = fresh as Map<String, dynamic>);
                } catch (e) {
                  setState(() => _error = e.toString());
                }
              },
            ),
            // v0.7 forwarding: E2EE messages cannot be forwarded
            // (their ciphertext is only decryptable in this chat).
            if ((m['kind'] as String? ?? 'text') != 'e2ee')
              ListTile(
                leading: const Icon(Icons.forward_outlined),
                title: const Text('Forward'),
                onTap: () {
                  Navigator.pop(context);
                  _showForwardPicker(m);
                },
              ),
            if (mine) ...[
              ListTile(
                leading: const Icon(Icons.edit_outlined),
                title: const Text('Edit'),
                onTap: () {
                  Navigator.pop(context);
                  setState(() {
                    _editingId = m['id'] as String;
                    _draft.text = content;
                  });
                },
              ),
            ],
            // Stage 4: delete the whole poll (creator or admin/owner).
            // The server deletes the poll and its chat message.
            if ((m['kind'] as String? ?? 'text') == 'poll' && _canDeletePoll(m))
              ListTile(
                leading: const Icon(Icons.bar_chart_outlined),
                title: const Text('Delete poll'),
                onTap: () async {
                  Navigator.pop(context);
                  final pollId = _parsePollId(m['content'] as String?);
                  if (pollId == null) return;
                  try {
                    await _api.deletePoll(pollId);
                    await _refreshChats();
                    setState(() => _messages
                        .removeWhere((x) => (x as Map)['id'] == m['id']));
                  } catch (e) {
                    setState(() => _error = e.toString());
                  }
                },
              ),
            // Stage 1: WhatsApp-style delete choice — for me, or for
            // everyone (own messages only).
            ListTile(
              leading: const Icon(Icons.delete_outline),
              title: const Text('Delete'),
              onTap: () {
                Navigator.pop(context);
                _showDeleteChoice(m, mine);
              },
            ),
          ],
        ),
      ),
    );
  }

  /// Stage 1: delete choice dialog. "Delete for me" hides the message
  /// only for the caller; "Delete for everyone" (own messages only)
  /// removes it for all members.
  void _showDeleteChoice(Map<String, dynamic> m, bool mine) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete message?'),
        content: Text(mine
            ? 'Delete this message for yourself or for everyone?'
            : 'Delete this message for yourself?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () async {
              Navigator.pop(context);
              try {
                await _api.deleteMessageForMe(m['id'] as String);
                setState(() =>
                    _messages.removeWhere((x) => (x as Map)['id'] == m['id']));
              } catch (e) {
                setState(() => _error = e.toString());
              }
            },
            child: const Text('Delete for me'),
          ),
          if (mine)
            TextButton(
              onPressed: () async {
                Navigator.pop(context);
                try {
                  await _api.deleteMessage(m['id'] as String);
                  setState(() => _messages
                      .removeWhere((x) => (x as Map)['id'] == m['id']));
                } catch (e) {
                  setState(() => _error = e.toString());
                }
              },
              child: const Text('Delete for everyone'),
            ),
        ],
      ),
    );
  }

  /// v0.7 forwarding: pick a chat and send a copy there, keeping the
  /// original sender as attribution ("Forwarded from <name>").
  void _showForwardPicker(Map<String, dynamic> m) {
    final currentId = _activeChat?['id'] as String?;
    showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text('Forward to…',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
            ),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: _chats.length,
                itemBuilder: (context, i) {
                  final c = _chats[i] as Map<String, dynamic>;
                  final title = _chatTitle(c);
                  return ListTile(
                    leading: WaAvatar(label: title, radius: 22),
                    title: Text(title),
                    subtitle: Text(c['type'] as String? ?? ''),
                    onTap: () async {
                      Navigator.pop(context);
                      final messenger = ScaffoldMessenger.of(context);
                      try {
                        await _api.forwardMessage(c['id'] as String, m);
                        if (c['id'] == currentId) {
                          await _refreshMessages();
                        }
                        messenger.showSnackBar(
                          SnackBar(content: Text('Forwarded to $title')),
                        );
                      } catch (e) {
                        if (mounted) {
                          setState(() => _error = e.toString());
                        }
                      }
                    },
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// v0.7 chat lock: asks for the 4-8 digit PIN and returns true when
  /// it matches. Returns false on cancel or a wrong PIN.
  Future<bool?> _askPin(String title) async {
    final ctrl = TextEditingController();
    String? error;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: Text(title),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: ctrl,
                keyboardType: TextInputType.number,
                obscureText: true,
                maxLength: 8,
                autofocus: true,
                decoration: InputDecoration(
                  labelText: 'PIN (4-8 digits)',
                  errorText: error,
                ),
                onSubmitted: (_) async {
                  if (await (_chatLock?.verifyPin(ctrl.text.trim()) ??
                      Future.value(false))) {
                    if (context.mounted) Navigator.pop(context, true);
                  } else {
                    setDialogState(() => error = 'Wrong PIN');
                  }
                },
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () async {
                if (await (_chatLock?.verifyPin(ctrl.text.trim()) ??
                    Future.value(false))) {
                  if (context.mounted) Navigator.pop(context, true);
                } else {
                  setDialogState(() => error = 'Wrong PIN');
                }
              },
              child: const Text('Unlock'),
            ),
          ],
        ),
      ),
    );
    ctrl.dispose();
    return ok;
  }

  /// v0.7 chat lock: set or change the PIN (asks for the new PIN twice).
  Future<void> _setChatPin() async {
    final first = TextEditingController();
    final second = TextEditingController();
    String? error;
    final saved = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: Text(_chatLock?.hasPin == true
              ? 'Change chat lock PIN'
              : 'Set chat lock PIN'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: first,
                keyboardType: TextInputType.number,
                obscureText: true,
                maxLength: 8,
                autofocus: true,
                decoration:
                    const InputDecoration(labelText: 'New PIN (4-8 digits)'),
              ),
              TextField(
                controller: second,
                keyboardType: TextInputType.number,
                obscureText: true,
                maxLength: 8,
                decoration:
                    InputDecoration(labelText: 'Repeat PIN', errorText: error),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () async {
                final a = first.text.trim();
                final b = second.text.trim();
                if (a != b) {
                  setDialogState(() => error = 'PINs do not match');
                  return;
                }
                final err = await _chatLock?.setPin(a);
                if (err != null) {
                  setDialogState(() => error = err);
                  return;
                }
                if (context.mounted) Navigator.pop(context, true);
              },
              child: const Text('Save PIN'),
            ),
          ],
        ),
      ),
    );
    first.dispose();
    second.dispose();
    if (saved == true && mounted) {
      setState(() {});
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Chat lock PIN saved')),
      );
    }
  }

  void _showReactions(Map<String, dynamic> m) {
    showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceEvenly,
            children: _reactionEmojis
                .map((e) => IconButton(
                      iconSize: 32,
                      icon: Text(e, style: const TextStyle(fontSize: 28)),
                      onPressed: () async {
                        Navigator.pop(context);
                        try {
                          await _api.addReaction(m['id'] as String, e);
                          await _refreshMessages();
                        } catch (err) {
                          setState(() => _error = err.toString());
                        }
                      },
                    ))
                .toList(),
          ),
        ),
      ),
    );
  }

  /// Local Safari assistant (on-device, rule-based — not a cloud AI).
  /// Clearly labeled as such in the UI.
  void _showAssistant() {
    final input = TextEditingController();
    String answer = '';
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => StatefulBuilder(
        builder: (context, setSheetState) => Padding(
          padding: EdgeInsets.only(
              bottom: MediaQuery.of(context).viewInsets.bottom,
              left: 16,
              right: 16,
              top: 16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('🤖 Safari Assistant',
                  style: Theme.of(context).textTheme.titleMedium),
              const Text(
                'Local assistant — runs on your device, rule-based. '
                'Not a cloud AI; it only knows how Safari works.',
                style: TextStyle(fontSize: 12, color: Colors.grey),
              ),
              const SizedBox(height: 12),
              Row(children: [
                Expanded(
                  child: TextField(
                    controller: input,
                    decoration: const InputDecoration(
                        hintText: 'Ask how to use Safari…'),
                    onSubmitted: (_) {
                      final r = assistantReply(input.text);
                      setSheetState(() => answer = r.text);
                    },
                  ),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: () {
                    final r = assistantReply(input.text);
                    setSheetState(() => answer = r.text);
                  },
                  child: const Text('Ask'),
                ),
              ]),
              if (answer.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  child: Text(answer),
                ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }

  /// Translate a text message with the local dictionary. The dialog
  /// shows the coverage, so a partial translation is never presented
  /// as complete.
  void _showTranslation(Map<String, dynamic> m, String content) {
    final to = detectLang(content) == AiLang.fa ? AiLang.en : AiLang.fa;
    final tr = translateLocal(content, to);
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('Translation (${tr.from.name} → ${tr.to.name})'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(tr.text),
            const SizedBox(height: 12),
            Text(
              'Local dictionary: ${tr.knownWords}/${tr.totalWords} words known '
              '(${(tr.coverage * 100).round()}% coverage). Unknown words are '
              'left unchanged — this is not a full translator.',
              style: const TextStyle(fontSize: 12, color: Colors.grey),
            ),
          ],
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Close')),
        ],
      ),
    );
  }

  /// Bots manager: list my bots, create one (token shown once),
  /// set a webhook, delete. Uses the real /api/bots endpoints.
  Future<void> _showBots() async {
    try {
      final bots = await _api.listBots();
      if (!mounted) return;
      await showModalBottomSheet(
        context: context,
        isScrollControlled: true,
        builder: (context) => Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('My bots (${bots.length})',
                  style: Theme.of(context).textTheme.titleMedium),
              const Text(
                'Bots receive messages via webhook or polling '
                '(/api/bot/updates) with their token. The token is shown '
                'once at creation — the server stores only its hash.',
                style: TextStyle(fontSize: 12, color: Colors.grey),
              ),
              Flexible(
                child: ListView(
                  shrinkWrap: true,
                  children: bots.map((b) {
                    final bm = b as Map<String, dynamic>;
                    return ListTile(
                      leading: const CircleAvatar(child: Text('🤖')),
                      title: Text(bm['name'] as String? ?? ''),
                      subtitle: Text(
                          '@${bm['username'] as String? ?? ''} · webhook: ${(bm['webhook_url'] as String?)?.isNotEmpty == true ? bm['webhook_url'] : 'not set'}'),
                      trailing: PopupMenuButton<String>(
                        onSelected: (v) async {
                          Navigator.pop(context);
                          if (v == 'delete') {
                            await _api.deleteBot(bm['id'] as String);
                          } else if (v == 'webhook') {
                            final urlCtrl = TextEditingController();
                            final ok = await showDialog<bool>(
                              context: context,
                              builder: (context) => AlertDialog(
                                title: const Text('Bot webhook URL'),
                                content: TextField(
                                    controller: urlCtrl,
                                    decoration: const InputDecoration(
                                        hintText: 'https://your-server/hook')),
                                actions: [
                                  TextButton(
                                      onPressed: () =>
                                          Navigator.pop(context, false),
                                      child: const Text('Cancel')),
                                  FilledButton(
                                      onPressed: () =>
                                          Navigator.pop(context, true),
                                      child: const Text('Save')),
                                ],
                              ),
                            );
                            if (ok == true) {
                              await _api.setBotWebhook(
                                  bm['id'] as String, urlCtrl.text.trim());
                            }
                          }
                        },
                        itemBuilder: (_) => const [
                          PopupMenuItem(
                              value: 'webhook', child: Text('Set webhook')),
                          PopupMenuItem(
                              value: 'delete', child: Text('Delete bot')),
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ),
              const SizedBox(height: 8),
              FilledButton.icon(
                icon: const Icon(Icons.add),
                label: const Text('Create a bot'),
                onPressed: () async {
                  Navigator.pop(context);
                  final nameCtrl = TextEditingController();
                  final userCtrl = TextEditingController();
                  final ok = await showDialog<bool>(
                    context: context,
                    builder: (context) => AlertDialog(
                      title: const Text('Create a bot'),
                      content: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          TextField(
                              controller: nameCtrl,
                              decoration:
                                  const InputDecoration(labelText: 'Bot name')),
                          TextField(
                              controller: userCtrl,
                              decoration: const InputDecoration(
                                  labelText: 'Bot username')),
                        ],
                      ),
                      actions: [
                        TextButton(
                            onPressed: () => Navigator.pop(context, false),
                            child: const Text('Cancel')),
                        FilledButton(
                            onPressed: () => Navigator.pop(context, true),
                            child: const Text('Create')),
                      ],
                    ),
                  );
                  if (ok != true) return;
                  try {
                    final created = await _api.createBot(
                        nameCtrl.text.trim(), userCtrl.text.trim());
                    if (!mounted) return;
                    await showDialog(
                      // ignore: use_build_context_synchronously
                      context: context,
                      builder: (context) => AlertDialog(
                        title: const Text('Bot created — save the token'),
                        content: SelectableText(
                          'Token (shown once, copy it now):\n${created['token'] ?? '(not returned)'}',
                        ),
                        actions: [
                          FilledButton(
                              onPressed: () => Navigator.pop(context),
                              child: const Text('Done')),
                        ],
                      ),
                    );
                  } catch (e) {
                    setState(() => _error = e.toString());
                  }
                },
              ),
            ],
          ),
        ),
      );
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  // ---------------------------------------------------------- UI

  // ── 3D registration screens (approved mockup): deep teal→emerald
  // gradient, floating glass spheres and chat bubbles, a glossy logo
  // tile, and a frosted glass card. English is the default language;
  // the pill on the page switches to فارسی.

  /// The 3D backdrop shared by the login, register and code screens.
  /// Animated: the glass spheres and chat bubbles drift slowly, so
  /// the page is always moving.
  Widget _authBackdrop({required Widget child}) {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            Color(0xFF013B34),
            Color(0xFF008069),
            Color(0xFF00A884),
            Color(0xFF25D366),
          ],
          stops: [0.0, 0.45, 0.75, 1.0],
        ),
      ),
      child: AnimatedBuilder(
        animation: _authAnim,
        builder: (context, _) {
          final t = _authAnim.value * 6.28318;
          return Stack(
            children: [
              // Floating glass spheres — depth, near and far.
              Positioned(
                  top: 64 + sin(t) * 14,
                  left: 24 + cos(t * 0.7) * 8,
                  child: _authOrb(64, 0.20)),
              Positioned(
                  top: 120 + sin(t + 1.3) * 10,
                  right: 28 + cos(t * 0.8 + 1.0) * 8,
                  child: _authOrb(40, 0.25)),
              Positioned(
                  top: 250 + sin(t + 2.1) * 8,
                  left: 8 + cos(t * 0.6 + 2.0) * 6,
                  child: _authOrb(24, 0.18)),
              Positioned(
                  bottom: 64 + sin(t + 0.8) * 12,
                  right: 36 + cos(t * 0.7 + 0.5) * 8,
                  child: _authOrb(52, 0.20)),
              Positioned(
                  bottom: 190 + sin(t + 2.8) * 9,
                  left: 30 + cos(t * 0.9 + 1.7) * 6,
                  child: _authOrb(30, 0.15)),
              // Big soft chat bubbles drifting behind the card.
              Positioned(
                  bottom: 110 + sin(t + 1.9) * 10,
                  left: -18 + cos(t * 0.5) * 6,
                  child: Icon(Icons.chat_bubble,
                      size: 120, color: Colors.white.withOpacity(0.10))),
              Positioned(
                  top: 330 + sin(t + 0.4) * 12,
                  right: -24 + cos(t * 0.6 + 0.9) * 6,
                  child: Icon(Icons.chat_bubble,
                      size: 96, color: Colors.white.withOpacity(0.08))),
              Positioned.fill(child: child),
            ],
          );
        },
      ),
    );
  }

  /// One floating glass sphere for the 3D backdrop.
  Widget _authOrb(double size, double opacity) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: Colors.white.withOpacity(opacity),
        border: Border.all(color: Colors.white.withOpacity(0.35), width: 1.5),
        boxShadow: const [
          BoxShadow(
              color: Color(0x30000000), blurRadius: 20, offset: Offset(0, 8)),
        ],
      ),
    );
  }

  /// Frosted glass card holding the registration fields.
  Widget _authGlassCard({required List<Widget> children}) {
    return Container(
      padding: const EdgeInsets.all(22),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.94),
        borderRadius: BorderRadius.circular(28),
        border: Border.all(color: Colors.white.withOpacity(0.8)),
        boxShadow: const [
          BoxShadow(
              color: Color(0x40000000), blurRadius: 28, offset: Offset(0, 14)),
          BoxShadow(
              color: Color(0x20000000), blurRadius: 6, offset: Offset(0, 3)),
        ],
      ),
      child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch, children: children),
    );
  }

  /// One big soft 3D input field with a label above it.
  Widget _authField({
    required TextEditingController controller,
    required String label,
    required IconData icon,
    TextInputType keyboardType = TextInputType.text,
    bool obscureText = false,
    Widget? suffix,
    void Function(String)? onSubmitted,
    int? maxLength,
    TextAlign textAlign = TextAlign.start,
    TextStyle? style,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(label,
            style: const TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w700,
                color: Color(0xFF1F2C34))),
        const SizedBox(height: 6),
        Container(
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(18),
            boxShadow: const [
              BoxShadow(
                  color: Color(0x14000000),
                  blurRadius: 10,
                  offset: Offset(0, 4)),
              BoxShadow(
                  color: Color(0x0A000000),
                  blurRadius: 2,
                  offset: Offset(0, 1)),
            ],
          ),
          child: TextField(
            controller: controller,
            keyboardType: keyboardType,
            obscureText: obscureText,
            maxLength: maxLength,
            textAlign: textAlign,
            style: style ?? const TextStyle(fontSize: 16),
            onSubmitted: onSubmitted,
            decoration: InputDecoration(
              prefixIcon: Icon(icon, color: const Color(0xFF008069)),
              suffixIcon: suffix,
              hintText: label,
              border: InputBorder.none,
              contentPadding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
              counterText: '',
            ),
          ),
        ),
      ],
    );
  }

  /// Big gradient-green 3D button (Create account / Log in / Verify).
  Widget _authButton({required String label, required VoidCallback onPressed}) {
    return Container(
      decoration: BoxDecoration(
        gradient: const LinearGradient(
            colors: [Color(0xFF25D366), Color(0xFF00A884)]),
        borderRadius: BorderRadius.circular(999),
        boxShadow: const [
          BoxShadow(
              color: Color(0x5500A884), blurRadius: 18, offset: Offset(0, 8)),
          BoxShadow(
              color: Color(0x33000000), blurRadius: 4, offset: Offset(0, 2)),
        ],
      ),
      child: FilledButton(
        onPressed: onPressed,
        style: FilledButton.styleFrom(
          backgroundColor: Colors.transparent,
          shadowColor: Colors.transparent,
          padding: const EdgeInsets.symmetric(vertical: 16),
          shape:
              RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
        ),
        child: Text(label,
            style: const TextStyle(
                fontSize: 17,
                fontWeight: FontWeight.w800,
                color: Colors.white)),
      ),
    );
  }

  /// Language picker pill (English default, فارسی second).
  Widget _authLangPill(bool isFa) {
    return PopupMenuButton<String>(
      onSelected: (code) {
        if ((code == 'fa') != isFa) widget.onToggleLang();
      },
      itemBuilder: (_) => const [
        PopupMenuItem(value: 'en', child: Text('English')),
        PopupMenuItem(value: 'fa', child: Text('فارسی')),
      ],
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        decoration: BoxDecoration(
          color: Colors.white.withOpacity(0.20),
          borderRadius: BorderRadius.circular(999),
          border: Border.all(color: Colors.white.withOpacity(0.45)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.language, size: 18, color: Colors.white),
            const SizedBox(width: 6),
            Text(isFa ? 'فارسی' : 'English',
                style: const TextStyle(
                    color: Colors.white, fontWeight: FontWeight.w700)),
            const SizedBox(width: 2),
            const Icon(Icons.keyboard_arrow_down,
                size: 18, color: Colors.white),
          ],
        ),
      ),
    );
  }

  /// Glossy glass logo tile + Safari name + slogan.
  /// The tile floats gently up and down.
  Widget _authLogoHeader({required bool isFa}) {
    return Column(
      children: [
        AnimatedBuilder(
          animation: _authAnim,
          builder: (context, child) => Transform.translate(
            offset: Offset(0, sin(_authAnim.value * 6.28318 + 1.0) * 7),
            child: child,
          ),
          child: SizedBox(
            width: 176,
            height: 112,
            child: Stack(
              alignment: Alignment.center,
              children: [
                // Orbit ring, back half: behind the planet, dimmed.
                AnimatedBuilder(
                  animation: _ringAnim,
                  builder: (context, _) => Opacity(
                    opacity: 0.45,
                    child: CustomPaint(
                      size: const Size(176, 112),
                      painter: _OrbitRingPainter(
                        angle: _ringAnim.value * 6.28318,
                        front: false,
                      ),
                    ),
                  ),
                ),
                Container(
                  width: 112,
                  height: 112,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(32),
                    color: Colors.white.withOpacity(0.20),
                    border: Border.all(
                        color: Colors.white.withOpacity(0.7), width: 2),
                    boxShadow: const [
                      BoxShadow(
                          color: Color(0x40000000),
                          blurRadius: 24,
                          offset: Offset(0, 12)),
                      BoxShadow(
                          color: Color(0x33FFFFFF),
                          blurRadius: 2,
                          offset: Offset(0, -1)),
                    ],
                  ),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(30),
                    child: Image.asset(
                      'assets/safari-logo.png',
                      width: 112,
                      height: 112,
                      fit: BoxFit.cover,
                    ),
                  ),
                ),
                // Orbit ring, front half: over the planet, bright.
                AnimatedBuilder(
                  animation: _ringAnim,
                  builder: (context, _) => CustomPaint(
                    size: const Size(176, 112),
                    painter: _OrbitRingPainter(
                      angle: _ringAnim.value * 6.28318,
                      front: true,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 18),
        const Text(
          'Safari',
          style: TextStyle(
            fontSize: 42,
            fontWeight: FontWeight.w900,
            color: Colors.white,
            letterSpacing: 5,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          isFa ? 'فراتر از مرزها وصل شو' : 'Connect Beyond Limits',
          style: TextStyle(
            fontSize: 15,
            color: Colors.white.withOpacity(0.92),
            fontWeight: FontWeight.w500,
          ),
        ),
      ],
    );
  }

  /// Soft red error box for auth failures.
  Widget _authError(String message) {
    return Container(
      margin: const EdgeInsets.only(top: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0xFFFDECEA),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: const Color(0xFFF5C6C2)),
      ),
      child: Row(
        children: [
          const Icon(Icons.error_outline, color: Color(0xFFB3261E), size: 20),
          const SizedBox(width: 8),
          Expanded(
            child: Text(message,
                style: const TextStyle(color: Color(0xFFB3261E), fontSize: 13)),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final isFa = Localizations.localeOf(context).languageCode == 'fa';
    // Stage 8: app lock — PIN or fingerprint before any chat is shown.
    if (_locked) return _buildLockScreen();
    // Email verification step (v0.6): 6-digit code, 10-minute expiry,
    // resend after a 60-second cooldown. No token until verified.
    if (_user == null && _codeStep) {
      return Scaffold(
        body: _authBackdrop(
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(24, 56, 24, 32),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Align(
                    alignment: AlignmentDirectional.centerEnd,
                    child: _authLangPill(isFa)),
                const SizedBox(height: 28),
                const Icon(Icons.mark_email_read_outlined,
                    size: 64, color: Colors.white),
                const SizedBox(height: 12),
                const Text(
                  'Check your email',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.w800,
                    color: Colors.white,
                  ),
                ),
                const SizedBox(height: 24),
                _authGlassCard(children: [
                  Text(
                    'We sent a 6-digit verification code to $_codeEmail. '
                    'It expires in 10 minutes. Never share it with anyone.',
                    style:
                        const TextStyle(fontSize: 14, color: Color(0xFF1F2C34)),
                  ),
                  if (_assignedUsername.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Text(
                      'Your username: @$_assignedUsername',
                      style: const TextStyle(
                          fontSize: 14, fontWeight: FontWeight.w700),
                    ),
                  ],
                  const SizedBox(height: 20),
                  _authField(
                    controller: _codeCtrl,
                    label: '6-digit code',
                    icon: Icons.pin_outlined,
                    keyboardType: TextInputType.number,
                    maxLength: 6,
                    textAlign: TextAlign.center,
                    style: const TextStyle(
                        fontSize: 26,
                        letterSpacing: 10,
                        fontWeight: FontWeight.w700),
                    onSubmitted: (_) => _verifyCode(),
                  ),
                  const SizedBox(height: 20),
                  _authButton(
                      label: 'Verify and continue', onPressed: _verifyCode),
                  const SizedBox(height: 6),
                  TextButton(
                    onPressed: _resendIn > 0 ? null : _resendCode,
                    child: Text(_resendIn > 0
                        ? 'Resend code in ${_resendIn}s'
                        : 'Resend code'),
                  ),
                  TextButton(
                    onPressed: () => setState(() {
                      _codeStep = false;
                      _resendTimer?.cancel();
                    }),
                    child: const Text('Back to login'),
                  ),
                  if (_error != null) _authError(_error!),
                ]),
              ],
            ),
          ),
        ),
      );
    }
    if (_user == null) {
      return Scaffold(
        body: _authBackdrop(
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(24, 56, 24, 32),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Align(
                    alignment: AlignmentDirectional.centerEnd,
                    child: _authLangPill(isFa)),
                const SizedBox(height: 20),
                _authLogoHeader(isFa: isFa),
                const SizedBox(height: 28),
                _authGlassCard(children: [
                  if (_registerMode) ...[
                    _authField(
                      controller: _displayName,
                      label: 'Your name',
                      icon: Icons.person_outline,
                    ),
                    const SizedBox(height: 6),
                    const Text(
                      'Your username is chosen automatically from your name.',
                      style: TextStyle(fontSize: 12, color: Colors.grey),
                    ),
                    const SizedBox(height: 14),
                  ],
                  _authField(
                    controller: _email,
                    label: 'Email',
                    icon: Icons.mail_outline,
                    keyboardType: TextInputType.emailAddress,
                  ),
                  const SizedBox(height: 14),
                  _authField(
                    controller: _password,
                    label: 'Password',
                    icon: Icons.lock_outline,
                    obscureText: _obscurePassword,
                    suffix: IconButton(
                      icon: Icon(_obscurePassword
                          ? Icons.visibility_outlined
                          : Icons.visibility_off_outlined),
                      onPressed: () =>
                          setState(() => _obscurePassword = !_obscurePassword),
                    ),
                  ),
                  const SizedBox(height: 20),
                  _authButton(
                    label: _registerMode ? 'Create account' : 'Log in',
                    onPressed: _auth,
                  ),
                  const SizedBox(height: 6),
                  TextButton(
                      onPressed: () =>
                          setState(() => _registerMode = !_registerMode),
                      child: Text(_registerMode
                          ? 'Already have an account? Log in'
                          : 'New to Safari? Create an account')),
                  if (_error != null) _authError(_error!),
                ]),
                const SizedBox(height: 16),
                // Official-download note, restyled small on the 3D page.
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.16),
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: Colors.white.withOpacity(0.35)),
                  ),
                  child: const Row(
                    children: [
                      Icon(Icons.verified_user_outlined, color: Colors.white),
                      SizedBox(width: 10),
                      Expanded(
                        child: Text(
                          'Download Safari only from the official Safari '
                          'website. APKs from other sites are not safe.',
                          style: TextStyle(fontSize: 12, color: Colors.white),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }
    return Scaffold(
      appBar: _buildAppBar(),
      body: _activeChat == null
          ? (_showSettings
              ? _buildSettingsBody()
              : Column(
                  children: [
                    if (_incomingCall != null) _incomingCallBanner(),
                    if (_incomingCall == null &&
                        (_call?.state == CallState.inCall ||
                            _call?.state == CallState.outgoing))
                      _inCallReturnBar(),
                    Expanded(child: _buildHomeBody()),
                  ],
                ))
          : _buildChatBody(),
      bottomNavigationBar:
          _activeChat == null && !_showSettings ? _buildSafariNavBar() : null,
      floatingActionButton: _buildFab(),
    );
  }

  /// Safari app bar, three modes: chat header (avatar + name +
  /// status, call icons, ⋮ menu), Settings (back + title), or the
  /// main bar ("Safari" + search + camera + ⋮ menu).
  PreferredSizeWidget _buildAppBar() {
    if (_activeChat != null) return _buildChatAppBar();
    if (_showSettings) {
      return vyroAppBar(
        context,
        title: 'Settings',
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => setState(() => _showSettings = false),
        ),
        actions: const [],
      );
    }
    return vyroAppBar(
      context,
      titleWidget: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(9),
            child: Image.asset(
              'assets/safari-logo-transparent.png',
              width: 32,
              height: 32,
              fit: BoxFit.cover,
            ),
          ),
          const SizedBox(width: 10),
          const Text('Safari'),
        ],
      ),
      actions: [
        IconButton(
          icon: const Icon(Icons.search),
          tooltip: 'Search',
          onPressed: _showSearchSheet,
        ),
        IconButton(
          icon: const Icon(Icons.camera_alt_outlined),
          tooltip: 'Camera',
          onPressed: _postStory,
        ),
        PopupMenuButton<String>(
          icon: const Icon(Icons.more_vert),
          onSelected: (v) {
            switch (v) {
              case 'new_group':
                _showNewChatSheet();
                break;
              case 'settings':
                setState(() => _showSettings = true);
                break;
              case 'theme':
                widget.onToggleTheme();
                break;
              case 'lang':
                widget.onToggleLang();
                break;
              case 'assistant':
                _showAssistant();
                break;
            }
          },
          itemBuilder: (context) => const [
            PopupMenuItem(value: 'new_group', child: Text('New group')),
            PopupMenuItem(value: 'settings', child: Text('Settings')),
            PopupMenuItem(value: 'assistant', child: Text('Safari assistant')),
            PopupMenuItem(value: 'theme', child: Text('Dark mode')),
            PopupMenuItem(value: 'lang', child: Text('فارسی / English')),
          ],
        ),
      ],
    );
  }

  /// Chat header: avatar + name + subtitle (typing / members /
  /// @username), video + voice call buttons, ⋮ menu with the rest
  /// (members, pin, archive, scheduled, E2EE, search).
  PreferredSizeWidget _buildChatAppBar() {
    final chat = _activeChat!;
    final type = chat['type'] as String? ?? 'dm';
    final title = _chatTitle(chat);
    final isE2ee = _e2eeChats.contains(chat['id']);
    final subtitle = _typingUser.isNotEmpty
        ? '$_typingUser is typing…'
        : type == 'group' || type == 'channel'
            ? '${(chat['member_ids'] as List?)?.length ?? 0} members'
            : _peerUsername(chat);
    return vyroAppBar(
      context,
      titleWidget: InkWell(
        onTap: type == 'dm' ? null : _showMembers,
        child: Row(
          children: [
            WaAvatar(label: title, radius: 20),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                      color: Colors.white,
                    ),
                  ),
                  Text(
                    subtitle,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 12,
                      color: Colors.white.withOpacity(0.8),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
      leading: IconButton(
        icon: const Icon(Icons.arrow_back),
        onPressed: () => setState(() => _activeChat = null),
      ),
      actions: [
        IconButton(
          icon: const Icon(Icons.videocam_outlined),
          tooltip: 'Video call',
          onPressed: () => _startCall(video: true),
        ),
        IconButton(
          icon: const Icon(Icons.call),
          tooltip: 'Voice call',
          onPressed: () => _startCall(video: false),
        ),
        PopupMenuButton<String>(
          icon: const Icon(Icons.more_vert),
          onSelected: (v) async {
            switch (v) {
              case 'members':
                _showMembers();
                break;
              case 'pin':
                // SetChatSetting returns the settings, not the chat —
                // re-read the fresh chat from the list (same for mute).
                if (chat['pinned'] == true) {
                  await _api.unpinChat(chat['id'] as String);
                } else {
                  await _api.pinChat(chat['id'] as String);
                }
                await _refreshChats();
                setState(() => _activeChat = _chats.firstWhere(
                    (c) => (c as Map)['id'] == chat['id'],
                    orElse: () => chat) as Map<String, dynamic>);
                break;
              case 'mute':
                // Stage 1: mute / unmute notifications for this chat.
                if (chat['muted'] == true) {
                  await _api.unmuteChat(chat['id'] as String);
                } else {
                  await _api.muteChat(chat['id'] as String);
                }
                await _refreshChats();
                setState(() => _activeChat = _chats.firstWhere(
                    (c) => (c as Map)['id'] == chat['id'],
                    orElse: () => chat) as Map<String, dynamic>);
                break;
              case 'archive':
                await _api.archiveChat(chat['id'] as String);
                setState(() => _activeChat = null);
                await _refreshChats();
                break;
              case 'scheduled':
                _showScheduled();
                break;
              case 'e2ee':
                setState(() {
                  if (isE2ee) {
                    _e2eeChats.remove(chat['id']);
                  } else {
                    _e2eeChats.add(chat['id']);
                  }
                });
                break;
              case 'search':
                _showSearchSheet();
                break;
              case 'export':
                await _exportChat(chat);
                break;
              case 'wallpaper':
                await _showWallpaperPicker(chatId: chat['id'] as String);
                break;
              case 'block':
                final peerId = _peerId(chat);
                if (peerId.isEmpty) break;
                try {
                  if (_blockedUserIds.contains(peerId)) {
                    await _api.unblockUser(peerId);
                    setState(() => _blockedUserIds.remove(peerId));
                  } else {
                    await _api.blockUser(peerId);
                    setState(() => _blockedUserIds.add(peerId));
                  }
                } catch (e) {
                  setState(() => _error = e.toString());
                }
                break;
              case 'report':
                final peerId = _peerId(chat);
                if (peerId.isNotEmpty) _reportPeer(peerId);
                break;
            }
          },
          itemBuilder: (context) => [
            PopupMenuItem(
              value: 'members',
              child: Text(type == 'dm' ? 'View contact' : 'Members & admins'),
            ),
            PopupMenuItem(
              value: 'pin',
              child: Text(chat['pinned'] == true ? 'Unpin chat' : 'Pin chat'),
            ),
            PopupMenuItem(
              value: 'mute',
              child: Text(chat['muted'] == true
                  ? 'Unmute notifications'
                  : 'Mute notifications'),
            ),
            const PopupMenuItem(value: 'archive', child: Text('Archive chat')),
            const PopupMenuItem(
                value: 'scheduled', child: Text('Scheduled messages')),
            PopupMenuItem(
              value: 'e2ee',
              child: Row(
                children: [
                  Text(isE2ee ? 'E2EE on' : 'E2EE off'),
                  const SizedBox(width: 8),
                  Icon(isE2ee ? Icons.lock : Icons.lock_open, size: 16),
                ],
              ),
            ),
            const PopupMenuItem(value: 'search', child: Text('Search')),
            const PopupMenuItem(value: 'export', child: Text('Export chat')),
            const PopupMenuItem(value: 'wallpaper', child: Text('Wallpaper')),
            if (type == 'dm')
              PopupMenuItem(
                value: 'block',
                child: Text(_blockedUserIds.contains(_peerId(chat))
                    ? 'Unblock'
                    : 'Block'),
              ),
            if (type == 'dm')
              const PopupMenuItem(value: 'report', child: Text('Report')),
          ],
        ),
      ],
    );
  }

  /// The other member's id of a DM (for block/report actions).
  String _peerId(Map<String, dynamic> chat) {
    final me = _user?['id'] as String?;
    final members = chat['member_ids'] as List?;
    if (members == null) return '';
    for (final id in members) {
      if (id != me) return id as String? ?? '';
    }
    return '';
  }

  /// @username of the other member of a DM (chat header subtitle).
  String _peerUsername(Map<String, dynamic> chat) {
    final me = _user?['id'] as String?;
    final members = chat['member_ids'] as List?;
    if (members == null) return '';
    for (final id in members) {
      if (id != me) {
        final u = _usersById[id as String? ?? ''];
        final name = u?['username'] as String?;
        if (name != null && name.isNotEmpty) return '@$name';
      }
    }
    return '';
  }

  /// Safari's floating glass bottom navigation: a rounded
  /// translucent bar with a cyan-gradient pill on the active tab.
  /// Tab order matches the old NavigationBar (Chats, Updates,
  /// Communities, Calls); Flutter mirrors the Row automatically
  /// in RTL, so the visual order follows the device direction.
  Widget _buildSafariNavBar() {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final isPersian = Localizations.localeOf(context).languageCode == 'fa';
    const items = [
      (Icons.chat_bubble_outline, Icons.chat_bubble, 'Chats', 'گفت‌وگوها', 0),
      (Icons.circle_outlined, Icons.circle, 'Updates', 'به‌روزرسانی‌ها', 1),
      (Icons.group_outlined, Icons.group, 'Communities', 'انجمن‌ها', 2),
      (Icons.call_outlined, Icons.call, 'Calls', 'تماس‌ها', 3),
    ];
    final inactive =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    return SafeArea(
      top: false,
      child: Container(
        margin: const EdgeInsets.fromLTRB(16, 0, 16, 12),
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 6),
        decoration: BoxDecoration(
          color: dark
              ? WaColors.darkSurface.withOpacity(0.88)
              : Colors.white.withOpacity(0.92),
          borderRadius: BorderRadius.circular(28),
          border: Border.all(
            color: dark ? WaColors.darkBorder : WaColors.lightBorder,
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(dark ? 0.45 : 0.12),
              blurRadius: 20,
              offset: const Offset(0, 8),
            ),
            if (dark)
              const BoxShadow(
                color: Color(0x1A22D3EE),
                blurRadius: 24,
              ),
          ],
        ),
        child: Row(
          children: [
            for (final (icon, activeIcon, label, faLabel, i) in items)
              Expanded(
                child: GestureDetector(
                  onTap: () => setState(() => _tabIndex = i),
                  behavior: HitTestBehavior.opaque,
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 220),
                    curve: Curves.easeOut,
                    padding: const EdgeInsets.symmetric(vertical: 7),
                    decoration: BoxDecoration(
                      gradient: _tabIndex == i
                          ? const LinearGradient(
                              colors: [Color(0xFF22D3EE), Color(0xFF2563EB)],
                              begin: Alignment.topLeft,
                              end: Alignment.bottomRight,
                            )
                          : null,
                      borderRadius: BorderRadius.circular(22),
                      boxShadow: _tabIndex == i
                          ? [
                              const BoxShadow(
                                color: Color(0x5522D3EE),
                                blurRadius: 12,
                                offset: Offset(0, 4),
                              ),
                            ]
                          : null,
                    ),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          _tabIndex == i ? activeIcon : icon,
                          size: 22,
                          color: _tabIndex == i ? Colors.white : inactive,
                        ),
                        const SizedBox(height: 2),
                        Text(
                          isPersian ? faLabel : label,
                          style: TextStyle(
                            fontSize: 11,
                            fontWeight: _tabIndex == i
                                ? FontWeight.w700
                                : FontWeight.w500,
                            color: _tabIndex == i ? Colors.white : inactive,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  /// Gradient FAB with a cyan glow, matching the premium theme.
  Widget _safariFab({
    required IconData icon,
    required VoidCallback onPressed,
    bool small = false,
  }) {
    final size = small ? 44.0 : 56.0;
    return FloatingActionButton(
      onPressed: onPressed,
      backgroundColor: Colors.transparent,
      elevation: 0,
      mini: small,
      shape: const CircleBorder(),
      child: Container(
        width: size,
        height: size,
        decoration: const BoxDecoration(
          gradient: VyroColors.brandGradient,
          shape: BoxShape.circle,
          boxShadow: [
            BoxShadow(
              color: Color(0x5522D3EE),
              blurRadius: 14,
              offset: Offset(0, 5),
            ),
          ],
        ),
        child: Icon(icon, color: Colors.white, size: small ? 22 : 26),
      ),
    );
  }

  /// One FAB per tab. Chats: new message. Updates: camera (new
  /// status). Calls: new call (goes to Chats to pick a chat).
  /// Communities: none. Inside a chat: back.
  Widget? _buildFab() {
    if (_activeChat != null) {
      return _safariFab(
        icon: Icons.arrow_back,
        onPressed: () => setState(() => _activeChat = null),
        small: true,
      );
    }
    if (_showSettings) return null;
    switch (_tabIndex) {
      case 1:
        return _safariFab(icon: Icons.camera_alt, onPressed: _postStory);
      case 2:
        return null;
      case 3:
        return _safariFab(
          icon: Icons.add_call,
          onPressed: () => _startNewCall(video: false),
        );
      case 0:
      default:
        return _safariFab(icon: Icons.message, onPressed: _showNewChatSheet);
    }
  }

  Widget _buildHomeBody() {
    switch (_tabIndex) {
      case 1:
        return _buildUpdatesBody();
      case 2:
        return _buildCommunitiesBody();
      case 3:
        return _buildCallsBody();
      case 0:
      default:
        return _buildChatsBody();
    }
  }

  // --- Chats tab: Safari list -------------------------------
  // Avatar, name, last-message preview, time, and cyan unread badge.
  // Pinned chats show a pin; archived chats hide behind the
  // "Archived" row at the bottom, exactly like WhatsApp.

  Widget _buildChatsBody() {
    final archived =
        _chats.where((c) => (c as Map)['archived'] == true).toList();
    final folderChats = _showArchived
        ? archived
        : _visibleChats.where((c) => (c as Map)['archived'] != true).toList();
    final shown = folderChats.where((chat) {
      final c = chat as Map<String, dynamic>;
      switch (_chatTypeFilter) {
        case 'Personal':
          return c['type'] == 'dm';
        case 'Groups':
          return c['type'] == 'group';
        case 'Unread':
          return _unreadCount(c) > 0;
        default:
          return true;
      }
    }).toList();
    return Column(
      children: [
        _offlineBanner(),
        if (_showArchived)
          ListTile(
            leading: const Icon(Icons.arrow_back),
            title: const Text('Archived'),
            subtitle: Text('${archived.length} chat(s)'),
            onTap: () => setState(() => _showArchived = false),
          )
        else ...[
          _chatFilterChips(),
        ],
        Expanded(
          child: shown.isEmpty && !_showArchived
              ? SafariEmptyState(
                  icon: Icons.chat_bubble_outline,
                  title: _chats.isEmpty ? 'No chats yet' : 'No matching chats',
                  subtitle: _chats.isEmpty
                      ? 'Start a conversation and your chats will appear here.'
                      : 'Try another filter to see your chats.',
                  buttonLabel: _chats.isEmpty ? 'New chat' : null,
                  onPressed: _chats.isEmpty ? _showNewChatSheet : null,
                  emblem: Container(
                    width: 120,
                    height: 120,
                    decoration: const BoxDecoration(
                      shape: BoxShape.circle,
                      gradient: RadialGradient(
                        colors: [Color(0x3322D3EE), Color(0x00000000)],
                      ),
                    ),
                    child: Image.asset(
                      'assets/safari-logo-transparent.png',
                      fit: BoxFit.contain,
                    ),
                  ),
                )
              : ListView.builder(
                  itemCount: shown.length +
                      (!_showArchived &&
                              archived.isNotEmpty &&
                              _chatTypeFilter == 'All' &&
                              _folderFilter.isEmpty
                          ? 1
                          : 0),
                  itemBuilder: (context, i) {
                    if (i == shown.length &&
                        !_showArchived &&
                        archived.isNotEmpty &&
                        _chatTypeFilter == 'All' &&
                        _folderFilter.isEmpty) {
                      return ListTile(
                        leading: const CircleAvatar(
                          child: Icon(Icons.archive_outlined, size: 20),
                        ),
                        title: const Text('Archived'),
                        trailing: Text('${archived.length}'),
                        onTap: () => setState(() => _showArchived = true),
                      );
                    }
                    return _chatRow(shown[i] as Map<String, dynamic>);
                  },
                ),
        ),
      ],
    );
  }

  Widget _chatRow(Map<String, dynamic> c) {
    final title = _chatTitle(c);
    final unread = _unreadCount(c);
    final timeLabel = _chatTimeLabel(c);
    final pinned = c['pinned'] == true;
    // Stage 1: muted chats show a bell-off icon and a grey (not
    // cyan) unread badge, like WhatsApp.
    final muted = c['muted'] == true;
    final locked = _chatLock?.isLocked(c['id'] as String) == true;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    final green = dark ? WaColors.accentDark : WaColors.accent;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: SafariGlassCard(
        padding: EdgeInsets.zero,
        radius: 18,
        child: ListTile(
          leading: WaAvatar(label: title, radius: 26),
          title: Row(
            children: [
              Expanded(
                child: Text(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: unread > 0 ? FontWeight.w600 : FontWeight.w500,
                  ),
                ),
              ),
              if (pinned)
                Padding(
                  padding: const EdgeInsets.only(right: 4),
                  child: Icon(Icons.push_pin, size: 15, color: secondary),
                ),
              if (muted)
                Padding(
                  padding: const EdgeInsets.only(right: 4),
                  child:
                      Icon(Icons.notifications_off, size: 15, color: secondary),
                ),
              if (timeLabel.isNotEmpty)
                Text(
                  timeLabel,
                  style: TextStyle(
                    fontSize: 12,
                    color: unread > 0 ? green : secondary,
                    fontWeight:
                        unread > 0 ? FontWeight.w600 : FontWeight.normal,
                  ),
                ),
            ],
          ),
          subtitle: Row(
            children: [
              Expanded(
                child: Text(
                  _chatPreview(c),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 14,
                    color: unread > 0
                        ? Theme.of(context).colorScheme.onSurface
                        : secondary,
                  ),
                ),
              ),
              if (locked)
                Padding(
                  padding: const EdgeInsets.only(left: 4),
                  child: Icon(Icons.lock, size: 14, color: secondary),
                ),
              if (unread > 0) ...[
                const SizedBox(width: 6),
                Container(
                  constraints:
                      const BoxConstraints(minWidth: 20, minHeight: 20),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    gradient: muted
                        ? null
                        : const LinearGradient(
                            colors: [Color(0xFF22D3EE), Color(0xFF2563EB)],
                            begin: Alignment.topLeft,
                            end: Alignment.bottomRight,
                          ),
                    color: muted ? secondary : null,
                    borderRadius: BorderRadius.circular(10),
                    boxShadow: muted
                        ? null
                        : const [
                            BoxShadow(
                              color: Color(0x5522D3EE),
                              blurRadius: 6,
                              offset: Offset(0, 2),
                            ),
                          ],
                  ),
                  alignment: Alignment.center,
                  child: Text(
                    unread > 99 ? '99+' : '$unread',
                    style: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w700,
                      color: Colors.white,
                    ),
                  ),
                ),
              ],
            ],
          ),
          onTap: () => _openChat(c),
          onLongPress: () => _showChatActions(c),
        ),
      ),
    );
  }

  /// Unread count from the server (ListChats), falling back to the
  /// live WebSocket marker for chats that got a message while open.
  int _unreadCount(Map<String, dynamic> c) {
    final n = c['unread_count'];
    if (n is int) return n;
    return _unreadChatIds.contains(c['id'] as String) ? 1 : 0;
  }

  /// Safari time label: HH:MM today, "Yesterday", weekday, else date.
  String _chatTimeLabel(Map<String, dynamic> c) {
    final lm = c['last_message'] as Map<String, dynamic>?;
    final raw = (lm?['created_at'] as String?) ?? (c['updated_at'] as String?);
    if (raw == null || raw.isEmpty) return '';
    DateTime t;
    try {
      t = DateTime.parse(raw).toLocal();
    } catch (_) {
      return raw.length >= 16 ? raw.substring(11, 16) : '';
    }
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final day = DateTime(t.year, t.month, t.day);
    final hhmm =
        '${t.hour.toString().padLeft(2, '0')}:${t.minute.toString().padLeft(2, '0')}';
    if (day == today) return hhmm;
    if (day == today.subtract(const Duration(days: 1))) {
      return 'Yesterday';
    }
    if (today.difference(day).inDays < 7) {
      const weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
      return weekdays[t.weekday - 1];
    }
    return '${t.day.toString().padLeft(2, '0')}/${t.month.toString().padLeft(2, '0')}/${t.year}';
  }

  /// Last-message preview for the chat list. E2EE messages never show
  /// ciphertext; media kinds show their WhatsApp label.
  String _chatPreview(Map<String, dynamic> c) {
    final lm = c['last_message'] as Map<String, dynamic>?;
    if (lm == null) return '';
    final kind = lm['kind'] as String? ?? 'text';
    String body;
    switch (kind) {
      case 'e2ee':
        body = 'Encrypted message';
        break;
      case 'image':
        body = 'Photo';
        break;
      case 'video':
        body = 'Video';
        break;
      case 'voice':
        body = 'Voice message';
        break;
      case 'file':
        body = 'File';
        break;
      case 'location':
        body = 'Location';
        break;
      case 'sticker':
        body = 'Sticker';
        break;
      case 'contact':
        body = 'Contact';
        break;
      case 'poll':
        body = _pollPreviewText(lm['content'] as String?);
        break;
      case 'system':
        body = (lm['content'] as String?) ?? '';
        break;
      default:
        body = (lm['content'] as String?) ?? '';
    }
    final mine = lm['sender_id'] == _user?['id'];
    if (mine) {
      final status = lm['status'] as String?;
      final ticks = status == 'read' || status == 'delivered'
          ? '✓✓ '
          : status == 'sent'
              ? '✓ '
              : '';
      return '${ticks}You: $body';
    }
    final type = c['type'] as String? ?? 'dm';
    if (type == 'group' || type == 'channel') {
      final sender = _usersById[lm['sender_id'] as String? ?? ''];
      final name = (sender?['display_name'] as String?) ??
          (sender?['username'] as String?) ??
          '';
      if (name.isNotEmpty) return '$name: $body';
    }
    return body;
  }

  Widget _sectionHeader(String text) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 18, 16, 6),
      child: Align(
        alignment: AlignmentDirectional.centerStart,
        child: Text(
          text.toUpperCase(),
          style: TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w700,
            letterSpacing: 0.8,
            color:
                dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary,
          ),
        ),
      ),
    );
  }

  int? _channelFollowerCount(Map<String, dynamic> channel) {
    final count = channel['follower_count'];
    if (count is int) return count;
    final members = channel['member_ids'];
    return members is List ? members.length : null;
  }

  // --- Updates tab: Stories + Channels (Safari) -------------------
  // Real user stories have cyan-purple rings; channel actions use
  // existing APIs and channel rows come from the loaded chat list.

  Widget _buildUpdatesBody() {
    final stories = _stories.cast<Map<String, dynamic>>();
    final channels =
        _chats.where((c) => (c as Map)['type'] == 'channel').toList();
    final myName = (_user?['display_name'] as String?) ??
        (_user?['username'] as String?) ??
        'Me';
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    return ListView(
      padding: const EdgeInsets.only(bottom: 24),
      children: [
        _sectionHeader('Stories'),
        SizedBox(
          height: 106,
          child: ListView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            children: [
              _storyRailItem(
                label: 'My status',
                onTap: _postStory,
                avatar: Stack(
                  children: [
                    WaAvatar(label: myName, radius: 30),
                    Positioned(
                      right: 0,
                      bottom: 0,
                      child: Container(
                        width: 22,
                        height: 22,
                        decoration: BoxDecoration(
                          gradient: VyroColors.brandGradient,
                          shape: BoxShape.circle,
                          border: Border.all(
                            color: Theme.of(context).scaffoldBackgroundColor,
                            width: 2,
                          ),
                        ),
                        child: const Icon(Icons.add,
                            size: 14, color: Colors.white),
                      ),
                    ),
                  ],
                ),
              ),
              for (final st in stories)
                _storyRailItem(
                  label: _storyTitle(st),
                  onTap: () => _viewStory(st),
                  onLongPress: st['user_id'] == _user?['id']
                      ? () => _deleteStory(st)
                      : null,
                  avatar: WaAvatar(
                    label: _storyTitle(st),
                    radius: 30,
                    ring: true,
                  ),
                ),
            ],
          ),
        ),
        _sectionHeader('Channels'),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 10),
          child: SafariHeroCard(
            icon: Icons.campaign_outlined,
            title: 'Create your channel',
            subtitle: 'Share updates with your followers',
            actionLabel: 'Create channel',
            onTap: () => _createGroupOrChannel(channel: true),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
          child: SafariGlassCard(
            onTap: _showChannelSearch,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            child: Row(
              children: [
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: WaColors.accent.withOpacity(0.14),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: const Icon(Icons.search, color: WaColors.accent),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('Find channels',
                          style: TextStyle(
                              fontSize: 15, fontWeight: FontWeight.w600)),
                      Text('Search public channels to follow',
                          style: TextStyle(fontSize: 13, color: secondary)),
                    ],
                  ),
                ),
                Icon(Icons.chevron_right, color: secondary),
              ],
            ),
          ),
        ),
        if (channels.isEmpty)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
            child: SafariGlassCard(
              padding: const EdgeInsets.all(20),
              child: Column(
                children: [
                  Container(
                    width: 64,
                    height: 64,
                    decoration: const BoxDecoration(
                      gradient: VyroColors.brandGradient,
                      shape: BoxShape.circle,
                      boxShadow: [
                        BoxShadow(
                          color: Color(0x4422D3EE),
                          blurRadius: 18,
                          offset: Offset(0, 6),
                        ),
                      ],
                    ),
                    child: const Icon(Icons.campaign_outlined,
                        size: 30, color: Colors.white),
                  ),
                  const SizedBox(height: 14),
                  const Text('No channels yet',
                      style:
                          TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
                  const SizedBox(height: 4),
                  Text('Channels you follow will appear here.',
                      textAlign: TextAlign.center,
                      style: TextStyle(fontSize: 13, color: secondary)),
                  const SizedBox(height: 14),
                  FilledButton.icon(
                    onPressed: _showChannelSearch,
                    icon: const Icon(Icons.search, size: 18),
                    label: const Text('Find channels'),
                  ),
                ],
              ),
            ),
          )
        else
          for (final ch in channels)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
              child: SafariGlassCard(
                onTap: () => _openChat(ch),
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                child: Row(
                  children: [
                    WaAvatar(
                        label: _chatTitle(ch as Map<String, dynamic>),
                        radius: 24),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(_chatTitle(ch),
                              style: const TextStyle(
                                  fontSize: 15, fontWeight: FontWeight.w600)),
                          if (_channelFollowerCount(ch) != null)
                            Text(
                              '${_channelFollowerCount(ch)} ${_channelFollowerCount(ch) == 1 ? 'follower' : 'followers'}',
                              style: TextStyle(fontSize: 13, color: secondary),
                            ),
                        ],
                      ),
                    ),
                    if (_unreadCount(ch) > 0) ...[
                      Container(
                        constraints:
                            const BoxConstraints(minWidth: 22, minHeight: 22),
                        padding: const EdgeInsets.symmetric(horizontal: 6),
                        alignment: Alignment.center,
                        decoration: const BoxDecoration(
                          gradient: VyroColors.brandGradient,
                          shape: BoxShape.circle,
                          boxShadow: [
                            BoxShadow(
                              color: Color(0x5522D3EE),
                              blurRadius: 8,
                            ),
                          ],
                        ),
                        child: Text(
                          _unreadCount(ch) > 99 ? '99+' : '${_unreadCount(ch)}',
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 11,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      const SizedBox(width: 6),
                    ],
                    Icon(Icons.chevron_right, color: secondary),
                  ],
                ),
              ),
            ),
      ],
    );
  }

  /// One item of the horizontal status rail: a circular avatar
  /// (with the gradient story ring for unviewed updates) and a
  /// name below it.
  Widget _storyRailItem({
    required String label,
    required Widget avatar,
    required VoidCallback onTap,
    VoidCallback? onLongPress,
  }) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 6),
      child: GestureDetector(
        onTap: onTap,
        onLongPress: onLongPress,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            avatar,
            const SizedBox(height: 6),
            SizedBox(
              width: 66,
              child: Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style:
                    const TextStyle(fontSize: 12, fontWeight: FontWeight.w500),
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// Stage 5: search public channels and follow them.
  void _showChannelSearch() {
    final ctrl = TextEditingController();
    List<dynamic> results = [];
    bool searching = false;
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => StatefulBuilder(
        builder: (context, setSheetState) => Padding(
          padding: EdgeInsets.only(
              bottom: MediaQuery.of(context).viewInsets.bottom,
              left: 16,
              right: 16,
              top: 16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(children: [
                Expanded(
                  child: TextField(
                    controller: ctrl,
                    decoration: const InputDecoration(
                        hintText: 'Search channels…',
                        prefixIcon: Icon(Icons.search)),
                    onSubmitted: (_) async {
                      setSheetState(() => searching = true);
                      try {
                        final r = await _api.searchChannels(ctrl.text.trim());
                        setSheetState(() {
                          results = r;
                          searching = false;
                        });
                      } catch (e) {
                        setSheetState(() => searching = false);
                        setState(() => _error = e.toString());
                      }
                    },
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.search),
                  onPressed: () async {
                    setSheetState(() => searching = true);
                    try {
                      final r = await _api.searchChannels(ctrl.text.trim());
                      setSheetState(() {
                        results = r;
                        searching = false;
                      });
                    } catch (e) {
                      setSheetState(() => searching = false);
                      setState(() => _error = e.toString());
                    }
                  },
                ),
              ]),
              const SizedBox(height: 8),
              if (searching) const CircularProgressIndicator(),
              Flexible(
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: results.length,
                  itemBuilder: (context, i) {
                    final ch = results[i] as Map<String, dynamic>;
                    final id = ch['id'] as String;
                    final joined = _chats.any((c) => (c as Map)['id'] == id);
                    final followers = ch['follower_count'] as int? ?? 0;
                    return ListTile(
                      leading: WaAvatar(
                          label: (ch['title'] as String?) ?? 'Channel',
                          radius: 26),
                      title: Text((ch['title'] as String?) ?? 'Channel'),
                      subtitle: Text(
                          '$followers ${followers == 1 ? 'follower' : 'followers'}'),
                      trailing: joined
                          ? const Text('Following',
                              style: TextStyle(color: WaColors.accent))
                          : FilledButton(
                              onPressed: () async {
                                try {
                                  await _api.followChannel(id);
                                  await _refreshChats();
                                  setSheetState(() {});
                                } catch (e) {
                                  setState(() => _error = e.toString());
                                }
                              },
                              child: const Text('Follow'),
                            ),
                      onTap: joined
                          ? () async {
                              Navigator.pop(context);
                              final chat = _chats
                                  .firstWhere((c) => (c as Map)['id'] == id);
                              await _openChat(chat as Map<String, dynamic>);
                            }
                          : null,
                    );
                  },
                ),
              ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }

  // --- Communities tab (Safari) -----------------------------------
  // A community brings related groups together. Here a community is
  // a group: the list shows real groups, and "New community"
  // opens the new-group flow.

  /// Stage 6: real communities (name, description, member and
  /// subgroup counts). Tapping one opens its detail sheet.
  /// Stage 6: real communities (name, description, member and
  /// subgroup counts). Tapping one opens its detail sheet.
  Widget _buildCommunitiesBody() {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
      children: [
        SafariHeroCard(
          icon: Icons.group_add_outlined,
          title: 'Create community',
          subtitle: 'Bring related groups together',
          actionLabel: 'Create community',
          onTap: _createCommunity,
        ),
        const SizedBox(height: 20),
        if (_communities.isEmpty)
          SafariEmptyState(
            icon: Icons.groups_outlined,
            title: 'No communities yet',
            subtitle: 'Create a community to bring related groups together.',
            buttonLabel: 'New community',
            onPressed: _createCommunity,
          )
        else
          for (final c in _communities)
            Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: SafariGlassCard(
                onTap: () => _showCommunityDetail(c),
                child: Row(
                  children: [
                    Builder(
                      builder: (context) {
                        final name = c['name'] as String? ?? 'Community';
                        final g = WaAvatar.gradientFor(name);
                        final initial = name.isEmpty
                            ? '?'
                            : name.substring(0, 1).toUpperCase();
                        return Container(
                          width: 52,
                          height: 52,
                          decoration: BoxDecoration(
                            gradient: LinearGradient(
                              colors: g,
                              begin: Alignment.topLeft,
                              end: Alignment.bottomRight,
                            ),
                            borderRadius: BorderRadius.circular(14),
                            boxShadow: const [
                              BoxShadow(
                                color: Color(0x3322D3EE),
                                blurRadius: 8,
                                offset: Offset(0, 3),
                              ),
                            ],
                          ),
                          alignment: Alignment.center,
                          child: Text(
                            initial,
                            style: const TextStyle(
                              color: Colors.white,
                              fontWeight: FontWeight.bold,
                              fontSize: 22,
                            ),
                          ),
                        );
                      },
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            c['name'] as String? ?? 'Community',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                                fontSize: 16, fontWeight: FontWeight.w600),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            _communityCountsLabel(c),
                            style: TextStyle(
                              fontSize: 13,
                              color: Theme.of(context).brightness ==
                                      Brightness.dark
                                  ? WaColors.darkTextSecondary
                                  : WaColors.lightTextSecondary,
                            ),
                          ),
                        ],
                      ),
                    ),
                    Icon(Icons.chevron_right, color: secondary),
                  ],
                ),
              ),
            ),
      ],
    );
  }

  String _communityCountsLabel(Map<String, dynamic> community) {
    final counts = <String>[];
    final members = community['member_count'];
    final groups = community['subgroup_count'];
    if (members is num) counts.add('$members members');
    if (groups is num) counts.add('$groups groups');
    return counts.join(' · ');
  }

  /// Stage 6: create a community (name + description). The server
  /// also creates the admin-only announcement group.
  Future<void> _createCommunity() async {
    final nameCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('New community'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: nameCtrl,
              decoration: const InputDecoration(labelText: 'Community name'),
              maxLength: 100,
            ),
            TextField(
              controller: descCtrl,
              decoration:
                  const InputDecoration(labelText: 'Description (optional)'),
              maxLength: 500,
            ),
          ],
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Create')),
        ],
      ),
    );
    if (ok != true || nameCtrl.text.trim().isEmpty) return;
    try {
      await _api.createCommunity(nameCtrl.text.trim(),
          description: descCtrl.text.trim());
      await _refreshCommunities();
      await _refreshChats();
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 6: community detail — announcement group, subgroups,
  /// members and admin settings.
  Future<void> _showCommunityDetail(Map<String, dynamic> community) async {
    final id = community['id'] as String;
    Map<String, dynamic> detail;
    try {
      detail = await _api.getCommunity(id);
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    if (!mounted) return;
    final c = detail['community'] as Map<String, dynamic>;
    final subgroups = (detail['subgroups'] as List?) ?? [];
    final myRole = c['my_role'] as String? ?? '';
    final isAdmin = myRole == 'admin' || myRole == 'owner';
    final announcementId = c['announcement_chat_id'] as String?;
    final announcement = announcementId == null
        ? null
        : _chats.cast<Map?>().firstWhere((ch) => ch?['id'] == announcementId,
            orElse: () => null);
    await showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text((c['name'] as String?) ?? 'Community',
                style: Theme.of(context).textTheme.titleLarge),
            if ((c['description'] as String?)?.isNotEmpty == true)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(c['description'] as String),
              ),
            Text(
                '${c['member_count'] ?? 0} members · ${subgroups.length} groups',
                style: Theme.of(context).textTheme.bodySmall),
            const SizedBox(height: 12),
            if (announcement != null)
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.campaign_outlined),
                title:
                    Text((announcement['title'] as String?) ?? 'Announcements'),
                subtitle: const Text('Announcement group · admins only'),
                onTap: () async {
                  Navigator.pop(context);
                  await _openChat(announcement as Map<String, dynamic>);
                },
              ),
            const Divider(),
            Text('Groups', style: Theme.of(context).textTheme.titleMedium),
            if (subgroups.isEmpty)
              const ListTile(
                title: Text('No groups yet'),
                subtitle: Text('Admins can add groups to this community.'),
              ),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: subgroups.length,
                itemBuilder: (context, i) {
                  final g = subgroups[i] as Map<String, dynamic>;
                  return ListTile(
                    contentPadding: EdgeInsets.zero,
                    leading: WaAvatar(
                        label: (g['title'] as String?) ?? 'Group', radius: 26),
                    title: Text((g['title'] as String?) ?? 'Group'),
                    subtitle: Text(
                        '${((g['member_ids'] as List?)?.length ?? 0)} members'),
                    trailing: isAdmin
                        ? IconButton(
                            icon: const Icon(Icons.remove_circle_outline),
                            tooltip: 'Remove from community',
                            onPressed: () async {
                              Navigator.pop(context);
                              try {
                                await _api.removeSubgroup(
                                    id, g['id'] as String);
                                await _refreshCommunities();
                              } catch (e) {
                                setState(() => _error = e.toString());
                              }
                            },
                          )
                        : null,
                    onTap: () async {
                      Navigator.pop(context);
                      await _openChat(g);
                    },
                  );
                },
              ),
            ),
            const Divider(),
            if (isAdmin) ...[
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.group_add_outlined),
                title: const Text('Add group'),
                onTap: () async {
                  Navigator.pop(context);
                  await _addSubgroup(c);
                },
              ),
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.person_add_outlined),
                title: const Text('Add members'),
                onTap: () async {
                  Navigator.pop(context);
                  await _addCommunityMembers(c);
                },
              ),
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.people_outline),
                title: const Text('Members'),
                onTap: () async {
                  Navigator.pop(context);
                  await _showCommunityMembers(c);
                },
              ),
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.edit_outlined),
                title: const Text('Edit community'),
                onTap: () async {
                  Navigator.pop(context);
                  await _editCommunity(c);
                },
              ),
            ] else
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.people_outline),
                title: const Text('Members'),
                onTap: () async {
                  Navigator.pop(context);
                  await _showCommunityMembers(c);
                },
              ),
            if (myRole != 'owner')
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.exit_to_app, color: Colors.red),
                title: const Text('Leave community',
                    style: TextStyle(color: Colors.red)),
                onTap: () async {
                  Navigator.pop(context);
                  try {
                    await _api.leaveCommunity(id);
                    await _refreshCommunities();
                    await _refreshChats();
                  } catch (e) {
                    setState(() => _error = e.toString());
                  }
                },
              ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }

  /// Stage 6: add a subgroup — pick an existing group or create
  /// a new one (admins only).
  Future<void> _addSubgroup(Map<String, dynamic> community) async {
    final id = community['id'] as String;
    final announcementId = community['announcement_chat_id'] as String?;
    final existing = _chats
        .where((ch) => ch['type'] == 'group' && ch['id'] != announcementId)
        .toList();
    final titleCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Add group'),
          content: SizedBox(
            width: double.maxFinite,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Flexible(
                  child: ListView(
                    shrinkWrap: true,
                    children: existing.map((ch) {
                      final m = ch as Map<String, dynamic>;
                      return ListTile(
                        title: Text((m['title'] as String?) ?? 'Group'),
                        onTap: () async {
                          Navigator.pop(context, true);
                          try {
                            await _api.addSubgroup(id,
                                chatId: m['id'] as String);
                            await _refreshCommunities();
                          } catch (e) {
                            setState(() => _error = e.toString());
                          }
                        },
                      );
                    }).toList(),
                  ),
                ),
                const Divider(),
                TextField(
                  controller: titleCtrl,
                  decoration:
                      const InputDecoration(labelText: 'Or create a new group'),
                  maxLength: 100,
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Create new')),
          ],
        ),
      ),
    );
    if (ok != true || titleCtrl.text.trim().isEmpty) return;
    try {
      await _api.addSubgroup(id, title: titleCtrl.text.trim());
      await _refreshCommunities();
      await _refreshChats();
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 6: add members to a community (admins only). They are
  /// also added to the announcement group.
  Future<void> _addCommunityMembers(Map<String, dynamic> community) async {
    final id = community['id'] as String;
    final users = await _api.listUsers();
    if (!mounted) return;
    final selected = <String>{};
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Add members'),
          content: SizedBox(
            width: double.maxFinite,
            child: ListView(
              shrinkWrap: true,
              children: users.map((u) {
                final m = u as Map<String, dynamic>;
                final uid = m['id'] as String;
                final name = (m['display_name'] as String?) ??
                    (m['username'] as String?) ??
                    uid;
                return CheckboxListTile(
                  title: Text(name),
                  value: selected.contains(uid),
                  onChanged: (v) => setDialogState(() {
                    if (v == true) {
                      selected.add(uid);
                    } else {
                      selected.remove(uid);
                    }
                  }),
                );
              }).toList(),
            ),
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Add')),
          ],
        ),
      ),
    );
    if (ok != true || selected.isEmpty) return;
    try {
      for (final uid in selected) {
        await _api.addCommunityMember(id, uid);
      }
      await _refreshCommunities();
      await _refreshChats();
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 6: members list with roles; admins can promote,
  /// demote and remove (never the owner).
  Future<void> _showCommunityMembers(Map<String, dynamic> community) async {
    final id = community['id'] as String;
    List<dynamic> members;
    try {
      members = await _api.listCommunityMembers(id);
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    if (!mounted) return;
    final myRole = community['my_role'] as String? ?? '';
    final isAdmin = myRole == 'admin' || myRole == 'owner';
    await showModalBottomSheet(
      context: context,
      builder: (context) => Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Members (${members.length})',
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: members.length,
                itemBuilder: (context, i) {
                  final m = members[i] as Map<String, dynamic>;
                  final uid = m['user_id'] as String;
                  final role = m['role'] as String? ?? 'member';
                  final name =
                      _usersById[uid]?['display_name'] as String? ?? uid;
                  final isMe = uid == _user?['id'];
                  return ListTile(
                    contentPadding: EdgeInsets.zero,
                    leading: WaAvatar(label: name, radius: 22),
                    title: Text(isMe ? '$name (you)' : name),
                    subtitle: Text('Role: $role'),
                    trailing: !isAdmin || isMe || role == 'owner'
                        ? null
                        : PopupMenuButton<String>(
                            onSelected: (v) async {
                              Navigator.pop(context);
                              try {
                                if (v == 'remove') {
                                  await _api.removeCommunityMember(id, uid);
                                } else {
                                  await _api.setCommunityMemberRole(id, uid, v);
                                }
                                await _refreshCommunities();
                              } catch (e) {
                                setState(() => _error = e.toString());
                              }
                            },
                            itemBuilder: (_) => const [
                              PopupMenuItem(
                                  value: 'admin', child: Text('Make admin')),
                              PopupMenuItem(
                                  value: 'member', child: Text('Make member')),
                              PopupMenuItem(
                                  value: 'remove', child: Text('Remove')),
                            ],
                          ),
                  );
                },
              ),
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }

  /// Stage 6: rename a community / edit its description
  /// (admins and the owner).
  Future<void> _editCommunity(Map<String, dynamic> community) async {
    final id = community['id'] as String;
    final nameCtrl =
        TextEditingController(text: (community['name'] as String?) ?? '');
    final descCtrl = TextEditingController(
        text: (community['description'] as String?) ?? '');
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Edit community'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: nameCtrl,
              decoration: const InputDecoration(labelText: 'Community name'),
              maxLength: 100,
            ),
            TextField(
              controller: descCtrl,
              decoration: const InputDecoration(labelText: 'Description'),
              maxLength: 500,
            ),
          ],
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Save')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.updateCommunity(id,
          name: nameCtrl.text.trim(), description: descCtrl.text.trim());
      await _refreshCommunities();
      await _refreshChats();
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 7: report a user (abuse/spam), optionally blocking them
  /// too. The report is stored for review; blocking stops messages
  /// in both directions.
  Future<void> _reportPeer(String userId) async {
    final reasonCtrl = TextEditingController();
    var alsoBlock = false;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Report user'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: reasonCtrl,
                decoration: const InputDecoration(labelText: 'Reason'),
                maxLength: 500,
              ),
              CheckboxListTile(
                title: const Text('Also block this user'),
                value: alsoBlock,
                onChanged: (v) => setDialogState(() => alsoBlock = v == true),
              ),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Report')),
          ],
        ),
      ),
    );
    if (ok != true || reasonCtrl.text.trim().isEmpty) return;
    try {
      await _api.reportUser(userId, reasonCtrl.text.trim());
      if (alsoBlock && !_blockedUserIds.contains(userId)) {
        await _api.blockUser(userId);
        setState(() => _blockedUserIds.add(userId));
      }
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 7: edit my profile — name, about (bio) and photo.
  /// The photo is uploaded like any other image; the server only
  /// accepts /uploads/ avatar URLs.
  Future<void> _editProfile() async {
    final nameCtrl =
        TextEditingController(text: (_user?['display_name'] as String?) ?? '');
    final bioCtrl =
        TextEditingController(text: (_user?['bio'] as String?) ?? '');
    var avatarUrl = _user?['avatar_url'] as String?;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Edit profile'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextButton.icon(
                icon: const Icon(Icons.photo_camera_outlined),
                label: const Text('Change photo'),
                onPressed: () async {
                  final picker = ImagePicker();
                  final picked = await picker.pickImage(
                      source: ImageSource.gallery, maxWidth: 512);
                  if (picked == null) return;
                  try {
                    final bytes = await picked.readAsBytes();
                    final compressed = DataSaver.compressImage(bytes);
                    final up = await _api.uploadFile(
                        compressed,
                        'avatar-${DateTime.now().millisecondsSinceEpoch}.jpg',
                        'image/jpeg');
                    setDialogState(() => avatarUrl = up['url'] as String?);
                  } catch (e) {
                    setState(() => _error = e.toString());
                  }
                },
              ),
              TextField(
                controller: nameCtrl,
                decoration: const InputDecoration(labelText: 'Name'),
                maxLength: 100,
              ),
              TextField(
                controller: bioCtrl,
                decoration: const InputDecoration(labelText: 'About'),
                maxLength: 500,
              ),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Save')),
          ],
        ),
      ),
    );
    if (ok != true) return;
    try {
      final updated = await _api.updateProfile(
          displayName: nameCtrl.text.trim(),
          bio: bioCtrl.text.trim(),
          avatarUrl: avatarUrl);
      setState(() => _user = updated);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 7: linked devices — every device registered for my
  /// account, with remove (never the device I am on now).
  Future<void> _showLinkedDevices() async {
    List<dynamic> devices;
    try {
      devices = await _api.listDevices();
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    if (!mounted) return;
    await showModalBottomSheet(
      context: context,
      builder: (context) => Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Linked devices (${devices.length})',
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            if (devices.isEmpty)
              const ListTile(
                title: Text('No other devices'),
                subtitle: Text(
                    'When you log in on another phone or computer, it shows up here.'),
              ),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: devices.length,
                itemBuilder: (context, i) {
                  final d = devices[i] as Map<String, dynamic>;
                  final isThis = d['device_id'] == _deviceId;
                  final name = (d['name'] as String?)?.isNotEmpty == true
                      ? d['name'] as String
                      : (d['device_id'] as String? ?? 'Device');
                  return ListTile(
                    contentPadding: EdgeInsets.zero,
                    leading: Icon((d['platform'] as String?) == 'android'
                        ? Icons.phone_android
                        : (d['platform'] as String?) == 'ios'
                            ? Icons.phone_iphone
                            : Icons.computer_outlined),
                    title: Text(name),
                    subtitle: Text(
                        '${d['platform'] ?? ''}${isThis ? ' · This device' : ''}'),
                    trailing: isThis
                        ? null
                        : IconButton(
                            icon: const Icon(Icons.delete_outline),
                            tooltip: 'Remove device',
                            onPressed: () async {
                              Navigator.pop(context);
                              try {
                                await _api
                                    .removeDevice(d['device_id'] as String);
                              } catch (e) {
                                setState(() => _error = e.toString());
                              }
                            },
                          ),
                  );
                },
              ),
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }

  // --- Stage 8 helpers: app lock, notifications, export, email ---

  /// Stage 8: app lock screen — PIN entry, plus a fingerprint
  /// button when the user enabled biometrics.
  Widget _buildLockScreen() {
    return Scaffold(
      backgroundColor: WaColors.lightHeader,
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Image.asset('assets/safari-logo.png', width: 72, height: 72),
              const SizedBox(height: 16),
              const Text('Safari is locked',
                  style: TextStyle(
                      fontSize: 22,
                      fontWeight: FontWeight.w700,
                      color: Colors.white)),
              const SizedBox(height: 8),
              const Text('Enter your PIN to continue',
                  style: TextStyle(color: Colors.white70)),
              const SizedBox(height: 24),
              TextField(
                controller: _pinCtrl,
                keyboardType: TextInputType.number,
                obscureText: true,
                maxLength: 8,
                textAlign: TextAlign.center,
                style: const TextStyle(
                    color: Colors.white, fontSize: 24, letterSpacing: 8),
                decoration: const InputDecoration(
                  counterText: '',
                  enabledBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.white54)),
                  focusedBorder: UnderlineInputBorder(
                      borderSide: BorderSide(color: Colors.white)),
                ),
                onSubmitted: (_) => _unlockWithPin(),
              ),
              const SizedBox(height: 16),
              FilledButton(
                  onPressed: _unlockWithPin, child: const Text('Unlock')),
              if (_appLock?.biometric == true) ...[
                const SizedBox(height: 12),
                IconButton(
                  icon: const Icon(Icons.fingerprint,
                      color: Colors.white, size: 36),
                  tooltip: 'Unlock with fingerprint',
                  onPressed: _tryBiometric,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _unlockWithPin() async {
    final ok = await _appLock?.verifyPin(_pinCtrl.text.trim());
    _pinCtrl.clear();
    if (ok == true) {
      setState(() => _locked = false);
    } else {
      setState(() => _error = 'Wrong PIN');
    }
  }

  /// Stage 8: fingerprint unlock. A failure (no sensor, not
  /// enrolled, cancelled) leaves the PIN field as the way in.
  Future<void> _tryBiometric() async {
    try {
      final auth = LocalAuthentication();
      final ok = await auth.authenticate(localizedReason: 'Unlock Safari');
      if (ok && mounted) setState(() => _locked = false);
    } catch (_) {}
  }

  /// Stage 8: app lock settings — set/change/remove the PIN,
  /// enable/disable the lock, fingerprint on/off (only when the
  /// device has biometrics).
  Future<void> _showAppLockSettings() async {
    final lock = _appLock;
    if (lock == null) return;
    var canBiometric = false;
    try {
      canBiometric = await LocalAuthentication().canCheckBiometrics;
    } catch (_) {}
    final pinCtrl = TextEditingController();
    final confirmCtrl = TextEditingController();
    var enable = lock.enabled;
    var biometric = lock.biometric;
    if (!mounted) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('App lock'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SwitchListTile(
                title: const Text('Lock Safari'),
                subtitle: const Text(
                    'Ask for the PIN when the app returns from the background.'),
                value: enable,
                onChanged: (v) => setDialogState(() => enable = v),
              ),
              if (canBiometric)
                SwitchListTile(
                  title: const Text('Fingerprint'),
                  subtitle: const Text('Also unlock with your fingerprint.'),
                  value: biometric,
                  onChanged: (v) => setDialogState(() => biometric = v),
                ),
              TextField(
                controller: pinCtrl,
                keyboardType: TextInputType.number,
                obscureText: true,
                maxLength: 8,
                decoration:
                    const InputDecoration(labelText: 'New PIN (4-8 digits)'),
              ),
              TextField(
                controller: confirmCtrl,
                keyboardType: TextInputType.number,
                obscureText: true,
                maxLength: 8,
                decoration: const InputDecoration(labelText: 'Confirm PIN'),
              ),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel')),
            if (lock.hasPin)
              TextButton(
                  onPressed: () async {
                    await lock.removePin();
                    if (context.mounted) Navigator.pop(context, true);
                  },
                  child: const Text('Remove PIN',
                      style: TextStyle(color: Colors.red))),
            FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Save')),
          ],
        ),
      ),
    );
    if (ok != true) return;
    try {
      if (pinCtrl.text.isNotEmpty) {
        if (pinCtrl.text != confirmCtrl.text) {
          setState(() => _error = 'PINs do not match');
          return;
        }
        final err = await lock.setPin(pinCtrl.text.trim());
        if (err != null) {
          setState(() => _error = err);
          return;
        }
      }
      await lock.setEnabled(enable);
      await lock.setBiometric(biometric && canBiometric);
      setState(() {});
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 8: save one notification preference (local only).
  Future<void> _setNotifyPref(String which, bool value) async {
    final prefs = await SharedPreferences.getInstance();
    switch (which) {
      case 'enabled':
        await prefs.setBool('vyro_notify_enabled_v1', value);
        setState(() => _notifyEnabled = value);
        break;
      case 'sound':
        await prefs.setBool('vyro_notify_sound_v1', value);
        setState(() => _notifySound = value);
        break;
      case 'vibrate':
        await prefs.setBool('vyro_notify_vibrate_v1', value);
        setState(() => _notifyVibrate = value);
        break;
    }
  }

  /// Stage 8: export the loaded messages of this chat as a text
  /// file and open the system share sheet. Only messages already
  /// loaded on this device are included.
  Future<void> _exportChat(Map<String, dynamic> chat) async {
    final lines = <String>['Safari chat export — ${_chatTitle(chat)}'];
    for (final m in _messages) {
      final map = m as Map<String, dynamic>;
      final when = (map['created_at'] as String?) ?? '';
      final sender = (map['sender_id'] == _user?['id'])
          ? 'You'
          : (_usersById[map['sender_id'] as String? ?? '']?['display_name']
                  as String?) ??
              'Someone';
      final kind = map['kind'] as String? ?? 'text';
      final body =
          kind == 'text' ? (map['content'] as String? ?? '') : '[$kind]';
      lines.add('[$when] $sender: $body');
    }
    try {
      final dir = await getTemporaryDirectory();
      final file = File(
          '${dir.path}/safari-chat-${DateTime.now().millisecondsSinceEpoch}.txt');
      await file.writeAsString(lines.join('\n'));
      final shared =
          await ShareFile.shareFile(file.path, text: 'Safari chat export');
      if (!shared) {
        setState(() => _error =
            'Sharing is not available here — the export is saved at ${file.path}');
      }
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Stage 8: change the account email — a 6-digit code goes to the
  /// NEW address first, then the account moves to it.
  Future<void> _changeEmail() async {
    final emailCtrl = TextEditingController();
    final codeCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Change email'),
        content: TextField(
          controller: emailCtrl,
          keyboardType: TextInputType.emailAddress,
          decoration: const InputDecoration(labelText: 'New email address'),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Send code')),
        ],
      ),
    );
    if (ok != true || emailCtrl.text.trim().isEmpty) return;
    final newEmail = emailCtrl.text.trim();
    try {
      await _api.requestEmailChange(newEmail);
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    if (!mounted) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Enter the code'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('A 6-digit code was sent to $newEmail.'),
            TextField(
              controller: codeCtrl,
              keyboardType: TextInputType.number,
              maxLength: 6,
              decoration: const InputDecoration(labelText: 'Code'),
            ),
          ],
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Change email')),
        ],
      ),
    );
    if (confirmed != true || codeCtrl.text.trim().isEmpty) return;
    try {
      final updated =
          await _api.verifyEmailChange(newEmail, codeCtrl.text.trim());
      setState(() => _user = updated);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  // --- Stage 9 helpers: backup, storage page ---

  /// Stage 9: storage page — per-chat message counts + media bytes
  /// from the server, plus a clear-cache button for downloads kept
  /// on this device.
  Future<void> _showStoragePage() async {
    late Map<String, dynamic> stats;
    try {
      stats = await _api.storageStats();
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    if (!mounted) return;
    final rows = (stats['chats'] as List<dynamic>?) ?? [];
    final totalMessages = stats['total_messages'] ?? 0;
    final totalBytes = (stats['total_bytes'] as num?) ?? 0;
    String mb(num bytes) => '${(bytes / 1048576).toStringAsFixed(2)} MB';
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (context) => DraggableScrollableSheet(
        expand: false,
        initialChildSize: 0.7,
        minChildSize: 0.4,
        maxChildSize: 0.95,
        builder: (context, scroll) => ListView(
          controller: scroll,
          children: [
            const ListTile(
              title: Text('Storage',
                  style: TextStyle(fontWeight: FontWeight.bold)),
            ),
            ListTile(
              title: const Text('Total messages'),
              trailing: Text('$totalMessages'),
            ),
            ListTile(
              title: const Text('Total media'),
              trailing: Text(mb(totalBytes)),
            ),
            const Divider(),
            for (final row in rows)
              ListTile(
                title: Text(_chatTitleFromMap(row)),
                subtitle: Text('${row['message_count']} messages'),
                trailing: Text(mb((row['bytes'] as num?) ?? 0)),
              ),
            if (rows.isEmpty) const ListTile(title: Text('No chats yet.')),
            const Divider(),
            ListTile(
              leading: const Icon(Icons.delete_outline),
              title: const Text('Clear downloaded media cache'),
              subtitle: const Text(
                  'Removes photos, videos and files downloaded on this '
                  'phone. Your chats stay on the server — media can be '
                  'downloaded again.'),
              onTap: () async {
                _downloaded.clear();
                await _usage.reset();
                if (context.mounted) Navigator.pop(context);
                setState(() {});
              },
            ),
          ],
        ),
      ),
    );
  }

  String _chatTitleFromMap(dynamic row) {
    if (row is Map<String, dynamic>) {
      final title = (row['title'] as String?)?.trim() ?? '';
      if (title.isNotEmpty) return title;
      return (row['type'] as String?) ?? 'Chat';
    }
    return 'Chat';
  }

  /// Stage 9: backup — your chats already live on Safari's server,
  /// so signing in anywhere restores them. This sheet offers a
  /// personal JSON copy (download) and a re-sync from the server
  /// (restore).
  Future<void> _showBackup() async {
    await showModalBottomSheet<void>(
      context: context,
      builder: (context) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const ListTile(
            title:
                Text('Backup', style: TextStyle(fontWeight: FontWeight.bold)),
            subtitle: Text('Your chats are saved on Safari\'s server — '
                'signing in on any device restores them. Keep an extra '
                'copy of your own, or re-download everything now.'),
          ),
          ListTile(
            leading: const Icon(Icons.download_outlined),
            title: const Text('Download backup file'),
            subtitle: const Text(
                'Your profile, chats and latest messages as a JSON file.'),
            onTap: () async {
              Navigator.pop(context);
              await _downloadBackup();
            },
          ),
          ListTile(
            leading: const Icon(Icons.restore_outlined),
            title: const Text('Restore from server'),
            subtitle: const Text('Re-download all chats and messages now.'),
            onTap: () async {
              Navigator.pop(context);
              await _refreshChats();
              if (_activeChat != null) await _refreshMessages();
              setState(() => _error = 'Restored from the server.');
            },
          ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }

  /// Stage 9: download the backup JSON and open the share sheet.
  Future<void> _downloadBackup() async {
    try {
      final json = await _api.exportMyData();
      final dir = await getTemporaryDirectory();
      final file = File(
          '${dir.path}/safari-backup-${DateTime.now().millisecondsSinceEpoch}.json');
      await file.writeAsString(json);
      final shared =
          await ShareFile.shareFile(file.path, text: 'Safari backup');
      if (!shared) {
        setState(() => _error =
            'Sharing is not available here — the backup is saved at ${file.path}');
      }
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  // --- Calls tab (Safari) -----------------------------------------
  /// Stage 3: Calls tab — real call history with missed markers.
  /// Tap a row to call that person/group back.
  Widget _buildCallsBody() {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    final isPersian = Localizations.localeOf(context).languageCode == 'fa';
    final filteredCalls = _calls.cast<Map<String, dynamic>>().where((call) {
      switch (_callFilter) {
        case 'Missed':
          return call['status'] == 'missed';
        case 'Incoming':
          return call['direction'] == 'incoming';
        case 'Outgoing':
          return call['direction'] == 'outgoing';
        default:
          return true;
      }
    }).toList();
    const filters = [
      ('All', 'همه'),
      ('Missed', 'ازدست‌رفته'),
      ('Incoming', 'ورودی'),
      ('Outgoing', 'خروجی'),
    ];
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 2),
          child: Row(
            children: [
              Expanded(
                child: SafariGradientButton(
                  icon: Icons.call,
                  label: isPersian ? 'تماس صوتی' : 'Voice call',
                  onPressed: () => _startNewCall(video: false),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: SafariGradientButton(
                  icon: Icons.videocam,
                  label: isPersian ? 'تماس تصویری' : 'Video call',
                  onPressed: () => _startNewCall(video: true),
                ),
              ),
            ],
          ),
        ),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 10),
          child: Row(
            children: [
              for (final (filter, faLabel) in filters) ...[
                SafariFilterChip(
                  label: isPersian ? faLabel : filter,
                  selected: _callFilter == filter,
                  onTap: () => setState(() => _callFilter = filter),
                ),
                const SizedBox(width: 8),
              ],
            ],
          ),
        ),
        Expanded(
          child: filteredCalls.isEmpty
              ? SafariEmptyState(
                  icon: Icons.call_outlined,
                  title: _calls.isEmpty
                      ? 'No calls yet'
                      : 'No ${_callFilter.toLowerCase()} calls',
                  subtitle: _calls.isEmpty
                      ? 'To make a call, choose a chat and start a call.'
                      : 'Your call history will appear here.',
                  buttonLabel: _calls.isEmpty ? 'Start a call' : null,
                  onPressed:
                      _calls.isEmpty ? () => _startNewCall(video: false) : null,
                )
              : ListView.builder(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  itemCount: filteredCalls.length,
                  itemBuilder: (context, index) {
                    final c = filteredCalls[index];
                    final missed = c['status'] == 'missed';
                    final name = (c['peer_name'] as String?) ?? '';
                    return Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 12, vertical: 4),
                      child: SafariGlassCard(
                        padding: EdgeInsets.zero,
                        radius: 18,
                        child: ListTile(
                          leading: WaAvatar(label: name, radius: 24),
                          title: Text(name),
                          subtitle: Row(
                            children: [
                              Icon(
                                missed
                                    ? Icons.call_missed
                                    : c['direction'] == 'outgoing'
                                        ? Icons.call_made
                                        : Icons.call_received,
                                size: 15,
                                color: missed
                                    ? const Color(0xFFF87171)
                                    : WaColors.accent,
                              ),
                              const SizedBox(width: 5),
                              Expanded(
                                child: Text(
                                  _callSubtitle(c),
                                  style: TextStyle(
                                    color: missed
                                        ? const Color(0xFFF87171)
                                        : secondary,
                                  ),
                                ),
                              ),
                            ],
                          ),
                          trailing: Container(
                            width: 42,
                            height: 42,
                            decoration: const BoxDecoration(
                              gradient: VyroColors.brandGradient,
                              shape: BoxShape.circle,
                              boxShadow: [
                                BoxShadow(
                                  color: Color(0x4422D3EE),
                                  blurRadius: 10,
                                ),
                              ],
                            ),
                            child: IconButton(
                              icon: Icon(
                                c['call_type'] == 'video'
                                    ? Icons.videocam
                                    : Icons.call,
                                size: 20,
                              ),
                              color: Colors.white,
                              onPressed: () => _callBack(c),
                            ),
                          ),
                          onTap: () => _callBack(c),
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  String _callSubtitle(Map<String, dynamic> c) {
    final missed = c['status'] == 'missed';
    final startedAt = _parseTime(c['started_at'] as String?);
    final parts = <String>[];
    if (missed) {
      parts.add('Missed');
    } else {
      final duration = c['duration_seconds'];
      if (duration is int) {
        parts.add(
          '${(duration ~/ 60).toString().padLeft(2, '0')}:'
          '${(duration % 60).toString().padLeft(2, '0')}',
        );
      }
    }
    if (startedAt != null) parts.add(_callTimeLabel(startedAt));
    return parts.join(' · ');
  }

  /// Safari time label for a DateTime (Stage 3): HH:MM today,
  /// "Yesterday", weekday, or dd/MM/yyyy — same rules as the chat list.
  String _callTimeLabel(DateTime t) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final day = DateTime(t.year, t.month, t.day);
    final hhmm =
        '${t.hour.toString().padLeft(2, '0')}:${t.minute.toString().padLeft(2, '0')}';
    if (day == today) return hhmm;
    if (day == today.subtract(const Duration(days: 1))) {
      return 'Yesterday';
    }
    if (today.difference(day).inDays < 7) {
      const weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
      return weekdays[t.weekday - 1];
    }
    return '${t.day.toString().padLeft(2, '0')}/${t.month.toString().padLeft(2, '0')}/${t.year}';
  }

  DateTime? _parseTime(String? iso) {
    if (iso == null || iso.isEmpty) return null;
    return DateTime.tryParse(iso)?.toLocal();
  }

  /// Call back from a history row (Stage 3): open the chat and start
  /// the same kind of call (video or voice).
  Future<void> _callBack(Map<String, dynamic> c) async {
    final chatId = c['chat_id'] as String?;
    if (chatId == null) return;
    Map<String, dynamic>? chat;
    for (final ch in _chats) {
      if ((ch as Map)['id'] == chatId) {
        chat = ch as Map<String, dynamic>;
        break;
      }
    }
    if (chat == null) {
      await _refreshChats();
      for (final ch in _chats) {
        if ((ch as Map)['id'] == chatId) {
          chat = ch as Map<String, dynamic>;
          break;
        }
      }
    }
    if (chat == null) {
      setState(() => _error = 'Chat not found');
      return;
    }
    _openChat(chat);
    // Let the chat screen settle before starting the call.
    await Future.delayed(const Duration(milliseconds: 300));
    await _startCall(video: c['call_type'] == 'video');
  }

  Future<void> _startNewCall({required bool video}) async {
    final availableChats = _chats
        .where((chat) {
          final type = (chat as Map)['type'];
          return type == 'dm' || type == 'group';
        })
        .cast<Map<String, dynamic>>()
        .toList();
    final selected = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      builder: (context) => SafeArea(
        child: SizedBox(
          height: MediaQuery.of(context).size.height * 0.65,
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  video ? 'Start a video call' : 'Start a voice call',
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              Expanded(
                child: availableChats.isEmpty
                    ? SafariEmptyState(
                        icon: Icons.chat_bubble_outline,
                        title: 'No chats available',
                        subtitle: 'Start a conversation before calling.',
                        buttonLabel: 'New chat',
                        onPressed: () {
                          Navigator.pop(context);
                          _showNewChatSheet();
                        },
                      )
                    : ListView.builder(
                        itemCount: availableChats.length,
                        itemBuilder: (context, index) {
                          final chat = availableChats[index];
                          final title = _chatTitle(chat);
                          return ListTile(
                            leading: WaAvatar(label: title, radius: 22),
                            title: Text(title),
                            subtitle: Text(
                              chat['type'] == 'group' ? 'Group' : 'Personal',
                            ),
                            onTap: () => Navigator.pop(context, chat),
                          );
                        },
                      ),
              ),
            ],
          ),
        ),
      ),
    );
    if (selected == null || !mounted) return;
    await _openChat(selected);
    await Future.delayed(const Duration(milliseconds: 300));
    await _startCall(video: video);
  }

  /// v0.7: chats visible under the active folder filter.
  List<dynamic> get _visibleChats {
    if (_folderFilter.isEmpty) return _chats;
    return _chats.where((c) => (c as Map)['folder'] == _folderFilter).toList();
  }

  /// Type and folder filters share one row. Long-press a folder chip
  /// to rename or delete it.
  Widget _chatFilterChips() {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final isPersian = Localizations.localeOf(context).languageCode == 'fa';
    final selectedBg = dark ? const Color(0xFF06B6D4) : const Color(0xFF0891B2);
    const selectedFg = Colors.white;
    final unselectedBg = dark ? WaColors.darkSurface2 : const Color(0xFFEDF2F7);
    final unselectedFg = dark ? WaColors.darkText : WaColors.lightText;
    const filters = [
      ('All', 'همه'),
      ('Personal', 'شخصی'),
      ('Groups', 'گروه‌ها'),
      ('Unread', 'خوانده‌نشده'),
    ];
    Widget chip(String label, bool selected, VoidCallback onTap,
        {VoidCallback? onLongPress}) {
      final w = GestureDetector(
        onLongPress: onLongPress,
        child: ChoiceChip(
          label: Text(label),
          selected: selected,
          showCheckmark: false,
          selectedColor: selectedBg,
          backgroundColor: unselectedBg,
          labelStyle: TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w600,
            color: selected ? selectedFg : unselectedFg,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(18),
            side: BorderSide.none,
          ),
          onSelected: (_) => onTap(),
        ),
      );
      return w;
    }

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
      child: Row(
        children: [
          for (final (filter, faLabel) in filters) ...[
            chip(
              isPersian ? faLabel : filter,
              _chatTypeFilter == filter && _folderFilter.isEmpty,
              () => setState(() {
                if (filter == 'All' ||
                    (_chatTypeFilter == filter && _folderFilter.isEmpty)) {
                  _chatTypeFilter = 'All';
                  _folderFilter = '';
                } else {
                  _chatTypeFilter = filter;
                  _folderFilter = '';
                }
              }),
            ),
            const SizedBox(width: 6),
          ],
          for (final f in _folderNames) ...[
            chip(
                f,
                _folderFilter == f && _chatTypeFilter == 'All',
                () => setState(() {
                      if (_folderFilter == f && _chatTypeFilter == 'All') {
                        _folderFilter = '';
                      } else {
                        _folderFilter = f;
                        _chatTypeFilter = 'All';
                      }
                    }),
                onLongPress: () => _manageFolder(f)),
            const SizedBox(width: 6),
          ],
          ActionChip(
            label: const Text('+ Folder'),
            backgroundColor: unselectedBg,
            labelStyle: TextStyle(
                fontSize: 13, fontWeight: FontWeight.w600, color: unselectedFg),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(18),
              side: BorderSide.none,
            ),
            onPressed: _createFolder,
          ),
        ],
      ),
    );
  }

  /// v0.7: create a user folder (name 1..32 chars, unique).
  Future<void> _createFolder() async {
    final ctrl = TextEditingController();
    String? error;
    final name = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('New folder'),
          content: TextField(
            controller: ctrl,
            autofocus: true,
            maxLength: 32,
            decoration: InputDecoration(
              labelText: 'Folder name',
              errorText: error,
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () async {
                final err = await _folders?.add(ctrl.text);
                if (err != null) {
                  setDialogState(() => error = err);
                  return;
                }
                if (context.mounted) Navigator.pop(context, ctrl.text.trim());
              },
              child: const Text('Create'),
            ),
          ],
        ),
      ),
    );
    ctrl.dispose();
    if (name != null && mounted) {
      setState(() => _folderNames = _folders?.list() ?? []);
    }
  }

  /// v0.7: rename or delete a user folder (long-press on its chip).
  Future<void> _manageFolder(String folder) async {
    final ctrl = TextEditingController(text: folder);
    String? error;
    final action = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: Text('Folder "$folder"'),
          content: TextField(
            controller: ctrl,
            maxLength: 32,
            decoration: InputDecoration(
              labelText: 'Folder name',
              errorText: error,
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, 'delete'),
              child: const Text('Delete', style: TextStyle(color: Colors.red)),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () async {
                final err = await _folders?.rename(folder, ctrl.text);
                if (err != null) {
                  setDialogState(() => error = err);
                  return;
                }
                if (context.mounted) {
                  Navigator.pop(context, ctrl.text.trim());
                }
              },
              child: const Text('Rename'),
            ),
          ],
        ),
      ),
    );
    ctrl.dispose();
    if (action == null || !mounted) return;
    if (action == 'delete') {
      await _folders?.remove(folder);
      // Chats in the deleted folder go back to "All".
      for (final c in _chats) {
        final m = c as Map<String, dynamic>;
        if (m['folder'] == folder) {
          try {
            await _api.setFolder(m['id'] as String, '');
          } catch (_) {}
        }
      }
      setState(() {
        _folderNames = _folders?.list() ?? [];
        if (_folderFilter == folder) _folderFilter = '';
      });
      await _refreshChats();
    } else {
      // Renamed: `action` is the new folder name. Move the chats
      // from the old folder name to the new one.
      final newName = action;
      for (final c in _chats) {
        final m = c as Map<String, dynamic>;
        if (m['folder'] == folder) {
          try {
            await _api.setFolder(m['id'] as String, newName);
          } catch (_) {}
        }
      }
      setState(() {
        _folderNames = _folders?.list() ?? [];
        if (_folderFilter == folder) _folderFilter = newName;
      });
      await _refreshChats();
    }
  }

  /// v0.7: long-press a chat — move to a folder, lock/unlock, pin,
  /// archive.
  void _showChatActions(Map<String, dynamic> chat) {
    final chatId = chat['id'] as String;
    final locked = _chatLock?.isLocked(chatId) == true;
    final hasPin = _chatLock?.hasPin == true;
    showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                _chatTitle(chat),
                style:
                    const TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
              ),
            ),
            ListTile(
              leading: const Icon(Icons.folder_outlined),
              title: const Text('Move to folder'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () async {
                Navigator.pop(context);
                await _showMoveToFolder(chat);
              },
            ),
            ListTile(
              leading:
                  Icon(locked ? Icons.lock_open_outlined : Icons.lock_outlined),
              title: Text(locked ? 'Unlock chat' : 'Lock chat'),
              subtitle: hasPin
                  ? null
                  : const Text('Set a PIN first in Settings → Chat lock'),
              onTap: () async {
                Navigator.pop(context);
                if (!hasPin) {
                  await _setChatPin();
                  return;
                }
                if (locked) {
                  await _chatLock?.unlockChat(chatId);
                  _unlockedThisSession.remove(chatId);
                } else {
                  await _chatLock?.lockChat(chatId);
                }
                if (mounted) setState(() {});
              },
            ),
            ListTile(
              leading: Icon(chat['pinned'] == true
                  ? Icons.push_pin_outlined
                  : Icons.push_pin),
              title: Text(chat['pinned'] == true ? 'Unpin' : 'Pin'),
              onTap: () async {
                Navigator.pop(context);
                if (chat['pinned'] == true) {
                  await _api.unpinChat(chatId);
                } else {
                  await _api.pinChat(chatId);
                }
                await _refreshChats();
              },
            ),
            ListTile(
              leading: const Icon(Icons.archive_outlined),
              title: Text(chat['archived'] == true ? 'Unarchive' : 'Archive'),
              onTap: () async {
                Navigator.pop(context);
                if (chat['archived'] == true) {
                  await _api.unarchiveChat(chatId);
                } else {
                  await _api.archiveChat(chatId);
                }
                await _refreshChats();
              },
            ),
          ],
        ),
      ),
    );
  }

  /// v0.7: pick a folder for a chat (or None to remove it from folders).
  Future<void> _showMoveToFolder(Map<String, dynamic> chat) async {
    final chatId = chat['id'] as String;
    final current = chat['folder'] as String? ?? '';
    await showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text('Move to folder',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
            ),
            ListTile(
              leading: const Icon(Icons.folder_off_outlined),
              title: const Text('None (show in All)'),
              trailing: current.isEmpty ? const Icon(Icons.check) : null,
              onTap: () async {
                Navigator.pop(context);
                await _api.setFolder(chatId, '');
                await _refreshChats();
              },
            ),
            for (final f in _folderNames)
              ListTile(
                leading: const Icon(Icons.folder_outlined),
                title: Text(f),
                trailing: current == f ? const Icon(Icons.check) : null,
                onTap: () async {
                  Navigator.pop(context);
                  await _api.setFolder(chatId, f);
                  await _refreshChats();
                },
              ),
            ListTile(
              leading: const Icon(Icons.create_new_folder_outlined),
              title: const Text('New folder…'),
              onTap: () async {
                Navigator.pop(context);
                await _createFolder();
              },
            ),
          ],
        ),
      ),
    );
  }

  /// WhatsApp-style Settings: profile card on top, then grouped
  /// rows — Account (last-seen privacy), Chats (Data Saver + usage
  /// + chat lock), Security (E2EE, bots, official-download warning),
  /// About. Every row does something real.
  Widget _buildSettingsBody() {
    String mb(double v) => '${v.toStringAsFixed(2)} MB';
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    final myName = (_user?['display_name'] as String?) ??
        (_user?['username'] as String?) ??
        'Me';
    final myUsername = _user?['username'] as String? ?? '';
    final myEmail = _user?['email'] as String? ?? '';
    Widget header(String text) => Padding(
          padding: const EdgeInsets.fromLTRB(16, 20, 16, 4),
          child: Align(
            alignment: Alignment.centerLeft,
            child: Text(
              text,
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: secondary,
              ),
            ),
          ),
        );
    // Stage 11: a privacy row — label on the left, Everyone/Nobody
    // picker on the right (WhatsApp-style).
    Widget privacyRow(String title, String subtitle, String value,
            void Function(String) onPick) =>
        ListTile(
          title: Text(title),
          subtitle: Text(subtitle),
          trailing: DropdownButton<String>(
            value: value,
            underline: const SizedBox(),
            items: const [
              DropdownMenuItem(value: 'everyone', child: Text('Everyone')),
              DropdownMenuItem(value: 'nobody', child: Text('Nobody')),
            ],
            onChanged: (v) {
              if (v != null) onPick(v);
            },
          ),
        );
    return ListView(
      children: [
        ListTile(
          leading: WaAvatar(label: myName, radius: 30),
          title: Text(
            myName,
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          subtitle: Text(
            '@$myUsername${myEmail.isEmpty ? '' : ' · $myEmail'}${(_user?['bio'] as String?)?.isNotEmpty == true ? '\n${_user?['bio']}' : ''}',
            style: TextStyle(fontSize: 13, color: secondary),
          ),
          // Stage 7: tap my profile to edit name, about and photo.
          onTap: _editProfile,
        ),
        const Divider(),
        header('Account'),
        ListTile(
          leading: const Icon(Icons.devices_outlined),
          title: const Text('Linked devices'),
          subtitle: const Text('See and remove your other devices'),
          onTap: _showLinkedDevices,
        ),
        ListTile(
          leading: const Icon(Icons.email_outlined),
          title: const Text('Change email'),
          subtitle: Text(_user?['email'] as String? ?? ''),
          onTap: _changeEmail,
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            'Who can see when you were last online. "Nobody" hides your '
            'online status and last seen from everyone.',
            style: TextStyle(fontSize: 13, color: secondary),
          ),
        ),
        RadioListTile<String>(
          title: const Text('Everyone'),
          subtitle:
              const Text('Anyone can see your online status and last seen.'),
          value: 'everyone',
          groupValue: _lastSeenVisibility,
          onChanged: (v) {
            if (v != null) _setLastSeenVisibility(v);
          },
        ),
        RadioListTile<String>(
          title: const Text('Nobody'),
          subtitle:
              const Text('Nobody can see your online status or last seen.'),
          value: 'nobody',
          groupValue: _lastSeenVisibility,
          onChanged: (v) {
            if (v != null) _setLastSeenVisibility(v);
          },
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            'Who can see your status updates. "Nobody" hides your '
            'status from everyone.',
            style: TextStyle(fontSize: 13, color: secondary),
          ),
        ),
        RadioListTile<String>(
          title: const Text('Everyone can see my status'),
          subtitle: const Text('Anyone can see your status updates.'),
          value: 'everyone',
          groupValue: _statusPrivacy,
          onChanged: (v) {
            if (v != null) _setStatusPrivacy(v);
          },
        ),
        RadioListTile<String>(
          title: const Text('Nobody can see my status'),
          subtitle: const Text('Nobody can see your status updates.'),
          value: 'nobody',
          groupValue: _statusPrivacy,
          onChanged: (v) {
            if (v != null) _setStatusPrivacy(v);
          },
        ),
        header('Privacy'),
        SwitchListTile(
          title: const Text('Read receipts'),
          subtitle: const Text('When off, you send no blue ticks and see '
              'none on your own messages either.'),
          value: _readReceipts,
          onChanged: _setReadReceipts,
        ),
        privacyRow(
          'Profile photo',
          'Who can see your profile photo.',
          _avatarPrivacy,
          (v) => _setPrivacyField('avatar', v),
        ),
        privacyRow(
          'About',
          'Who can see your About text.',
          _aboutPrivacy,
          (v) => _setPrivacyField('about', v),
        ),
        privacyRow(
          'Groups',
          'Who can add you to groups. "Nobody" blocks adds — '
              'invite links still work, since joining is your choice.',
          _groupsPrivacy,
          (v) => _setPrivacyField('groups', v),
        ),
        ListTile(
          leading: const Icon(Icons.block_outlined),
          title: const Text('Blocked contacts'),
          subtitle: Text(_blockedUserIds.isEmpty
              ? 'Nobody blocked'
              : '${_blockedUserIds.length} blocked'),
          trailing: const Icon(Icons.chevron_right),
          onTap: _showBlockedContacts,
        ),
        ListTile(
          leading: const Icon(Icons.delete_forever_outlined, color: Colors.red),
          title: const Text('Delete my account',
              style: TextStyle(color: Colors.red)),
          subtitle: const Text('Permanently delete your account, messages, '
              'groups, stories and call history.'),
          onTap: _deleteAccount,
        ),
        header('Notifications'),
        SwitchListTile(
          title: const Text('Notifications'),
          subtitle:
              const Text('Show message notifications while the app is open.'),
          value: _notifyEnabled,
          onChanged: (v) => _setNotifyPref('enabled', v),
        ),
        SwitchListTile(
          title: const Text('Sound'),
          subtitle: const Text('Play a sound with notifications.'),
          value: _notifySound,
          onChanged: (v) => _setNotifyPref('sound', v),
        ),
        SwitchListTile(
          title: const Text('Vibrate'),
          subtitle: const Text('Vibrate with notifications.'),
          value: _notifyVibrate,
          onChanged: (v) => _setNotifyPref('vibrate', v),
        ),
        header('Chats'),
        SwitchListTile(
          title: const Text('Data Saver (default ON)'),
          subtitle: const Text('Tap-to-download for photos, videos, files '
              'and voice messages; low-quality previews.'),
          value: _dataSaver,
          onChanged: _setDataSaver,
        ),
        ListTile(
          leading: const Icon(Icons.wallpaper_outlined),
          title: const Text('Chat wallpaper'),
          subtitle: Text(wallpaperNames[_wallpaperId] ?? 'Default'),
          trailing: _wallpaperTile(
              _wallpaperId, Theme.of(context).brightness == Brightness.dark,
              size: 28),
          onTap: _showWallpaperPicker,
        ),
        ListTile(
          leading: const Icon(Icons.lock_outlined),
          title: Text(_chatLock?.hasPin == true
              ? 'Change chat lock PIN'
              : 'Set chat lock PIN'),
          subtitle: Text(_chatLock?.hasPin == true
              ? '${_chatLock?.lockedChatIds().length ?? 0} chat(s) locked'
              : 'Lock individual chats with a 4-8 digit PIN.'),
          trailing: const Icon(Icons.chevron_right),
          onTap: _setChatPin,
        ),
        if (_chatLock?.hasPin == true)
          ListTile(
            leading: const Icon(Icons.lock_open_outlined),
            title: const Text('Remove chat lock PIN'),
            subtitle: const Text('Unlocks every chat and removes the PIN.'),
            onTap: () async {
              final ok = await _askPin('Enter PIN to remove the lock');
              if (ok == true) {
                await _chatLock?.removePin();
                _unlockedThisSession.clear();
                if (mounted) setState(() {});
              }
            },
          ),
        header('Storage and data'),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            'Safari uses far less data than Telegram and WhatsApp: '
            'voice calls use Opus at 12 kbps (~0.09 MB/min vs ~0.3-0.5 '
            'MB/min on WhatsApp), photos are compressed before upload, '
            'and nothing downloads automatically.',
            style: TextStyle(fontSize: 13, color: secondary),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
          child: Text('Data used this month (${_usage.month})',
              style: Theme.of(context).textTheme.titleMedium),
        ),
        for (final c in UsageCategory.values)
          ListTile(
            title: Text(c.label),
            trailing: Text(mb(_usage.mbFor(c))),
          ),
        ListTile(
          title: const Text('Total'),
          trailing: Text(mb(_usage.totalMb),
              style: const TextStyle(fontWeight: FontWeight.bold)),
        ),
        ListTile(
          leading: const Icon(Icons.storage_outlined),
          title: const Text('Storage'),
          subtitle: const Text('Messages and media per chat, on the server'),
          onTap: _showStoragePage,
        ),
        ListTile(
          leading: const Icon(Icons.backup_outlined),
          title: const Text('Backup'),
          subtitle: const Text(
              'Your chats are saved on the server — download a copy or restore'),
          onTap: _showBackup,
        ),
        TextButton(
          onPressed: () async {
            await _usage.reset();
            setState(() {});
          },
          child: const Text('Reset counters'),
        ),
        header('Security'),
        const ListTile(
          leading: Icon(Icons.verified_user_outlined),
          title: Text('Download Safari only from the official Safari website'),
          subtitle: Text('Never install APKs from other sites or strangers — '
              'unofficial APKs can be modified and are not safe. Check the '
              'version and SHA-256 checksum published on the official site.'),
        ),
        ListTile(
          leading: const Icon(Icons.smart_toy_outlined),
          title: const Text('My bots'),
          subtitle: const Text(
              'Create bots with a token and webhook — they reply via '
              'the bot API.'),
          trailing: const Icon(Icons.chevron_right),
          onTap: _showBots,
        ),
        ListTile(
          leading: const Icon(Icons.lock_outline),
          title: const Text('End-to-end encryption'),
          subtitle: Text(_e2ee?.isReady == true
              ? 'Ready on this device (device $_deviceId). Private keys '
                  'never leave the device.'
              : 'Not set up yet — log in to generate keys.'),
        ),
        ListTile(
          leading: const Icon(Icons.screen_lock_portrait_outlined),
          title: const Text('App lock'),
          subtitle: Text(_appLock?.enabled == true
              ? 'On — PIN${_appLock?.biometric == true ? ' + fingerprint' : ''} required when the app returns from the background.'
              : 'Off — set a PIN to lock Safari when it goes to the background.'),
          onTap: _showAppLockSettings,
        ),
        const ListTile(
          leading: Icon(Icons.shield_outlined),
          title: Text('Anti-spam'),
          subtitle: Text('One account can send at most 60 messages a '
              'minute — a flood script is stopped automatically. '
              'Report spam from any chat menu (Report), and block the '
              'sender from the same menu.'),
        ),
        header('About'),
        const ListTile(
          leading: Icon(Icons.info_outline),
          title: Text('Safari 0.6 (mobile)'),
          subtitle: Text('Go backend · Next.js web · Flutter mobile. '
              'Push notifications on mobile use FCM (Firebase).'),
        ),
        const SizedBox(height: 24),
      ],
    );
  }

  /// Title shown for a story in the Updates list, by kind.
  String _storyTitle(Map<String, dynamic> st) {
    final kind = st['kind'] as String? ?? 'photo';
    if (kind == 'text') {
      final text = (st['text_content'] as String?)?.trim() ?? '';
      return text.isNotEmpty ? text : 'Text status';
    }
    if (kind == 'audio') return 'Voice message';
    final caption = (st['caption'] as String?)?.trim() ?? '';
    return caption.isNotEmpty ? caption : 'Story';
  }

  /// Record a story view, ignoring failures (Stage 2).
  Future<void> _recordStoryView(String id) async {
    try {
      await _api.recordStoryView(id);
    } catch (_) {}
  }

  /// View a story (Stage 2): records a view for other people's
  /// stories, renders text/audio/photo kinds, and offers reply +
  /// reactions (others) or viewers + delete (my own story).
  void _viewStory(Map<String, dynamic> st) {
    final kind = st['kind'] as String? ?? 'photo';
    final mine = st['user_id'] == _user?['id'];
    if (!mine) {
      // Fire-and-forget: a failed view record never blocks viewing.
      _recordStoryView(st['id'] as String);
    }
    final mediaUrl = st['media_url'] as String?;
    if (kind != 'text') {
      if (mediaUrl == null) return;
      if (!_downloaded.containsKey(mediaUrl)) {
        if (_dataSaver) {
          // In Data Saver mode, ask before downloading a story.
          showDialog(
            context: context,
            builder: (context) => AlertDialog(
              title: const Text('Download story?'),
              content: const Text('Data Saver is on. Download this story '
                  'to view it?'),
              actions: [
                TextButton(
                    onPressed: () => Navigator.pop(context),
                    child: const Text('Cancel')),
                FilledButton(
                    onPressed: () {
                      Navigator.pop(context);
                      _downloadMedia(mediaUrl);
                    },
                    child: const Text('Download')),
              ],
            ),
          );
          return;
        }
        _downloadMedia(mediaUrl);
      }
    }
    final replyCtrl = TextEditingController();
    String myReaction = st['my_reaction'] as String? ?? '';
    showDialog(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) {
          Widget content;
          if (kind == 'text') {
            content = Container(
              width: double.infinity,
              height: 300,
              decoration: BoxDecoration(
                color: _colorFromHex(
                    (st['background_color'] as String?)?.isNotEmpty == true
                        ? st['background_color'] as String
                        : '#008069'),
                borderRadius: BorderRadius.circular(12),
              ),
              alignment: Alignment.center,
              padding: const EdgeInsets.all(20),
              child: Text(
                (st['text_content'] as String?) ?? '',
                style: const TextStyle(
                    color: Colors.white,
                    fontSize: 22,
                    fontWeight: FontWeight.w600),
                textAlign: TextAlign.center,
              ),
            );
          } else if (kind == 'audio') {
            content = VoiceMessagePlayer(
              api: _api,
              fileUrl: mediaUrl!,
              fileName: 'Voice status',
            );
          } else {
            final bytes = _downloaded[mediaUrl!];
            content = bytes == null
                ? const Text('Downloading…')
                : Image.memory(bytes, height: 300);
          }
          return AlertDialog(
            title: Text(_storyTitle(st)),
            content: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  content,
                  if (!mine) ...[
                    const SizedBox(height: 12),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        for (final emoji in const [
                          '❤️',
                          '😂',
                          '😮',
                          '😢',
                          '🙏',
                          '👍',
                          '🔥',
                          '😍'
                        ])
                          IconButton(
                            padding: EdgeInsets.zero,
                            constraints: const BoxConstraints(),
                            icon: Text(emoji,
                                style: TextStyle(
                                    fontSize: 22,
                                    color: myReaction == emoji
                                        ? null
                                        : Colors.grey)),
                            onPressed: () async {
                              try {
                                if (myReaction == emoji) {
                                  await _api
                                      .deleteStoryReaction(st['id'] as String);
                                  setDialogState(() => myReaction = '');
                                } else {
                                  await _api.reactToStory(
                                      st['id'] as String, emoji);
                                  setDialogState(() => myReaction = emoji);
                                }
                                st['my_reaction'] = myReaction;
                              } catch (e) {
                                setState(() => _error = e.toString());
                              }
                            },
                          ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        Expanded(
                          child: TextField(
                            controller: replyCtrl,
                            decoration: const InputDecoration(
                                hintText: 'Reply to status…'),
                          ),
                        ),
                        IconButton(
                          icon: const Icon(Icons.send),
                          color: WaColors.accent,
                          onPressed: () async {
                            final text = replyCtrl.text.trim();
                            if (text.isEmpty) return;
                            try {
                              await _api.replyToStory(st['id'] as String, text);
                              replyCtrl.clear();
                              await _refreshChats();
                              if (context.mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(
                                  const SnackBar(content: Text('Reply sent')),
                                );
                              }
                            } catch (e) {
                              setState(() => _error = e.toString());
                            }
                          },
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            actions: [
              if (mine) ...[
                TextButton(
                  onPressed: () {
                    Navigator.pop(context);
                    _showStoryViewers(st);
                  },
                  child: const Text('Viewed by'),
                ),
                TextButton(
                  onPressed: () {
                    Navigator.pop(context);
                    _deleteStory(st);
                  },
                  child:
                      const Text('Delete', style: TextStyle(color: Colors.red)),
                ),
              ],
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Close'),
              ),
            ],
          );
        },
      ),
    ).then((_) async {
      // Refresh: view counts and my reaction may have changed.
      try {
        final stories = await _api.listStories();
        if (mounted) setState(() => _stories = stories);
      } catch (_) {}
    });
  }

  /// Viewers, replies and reactions on my own story (Stage 2).
  Future<void> _showStoryViewers(Map<String, dynamic> st) async {
    try {
      final results = await Future.wait([
        _api.listStoryViews(st['id'] as String),
        _api.listStoryReplies(st['id'] as String),
        _api.listStoryReactions(st['id'] as String),
      ]);
      if (!mounted) return;
      final views = results[0];
      final replies = results[1];
      final reactions = results[2];
      await showModalBottomSheet(
        context: context,
        builder: (context) => SafeArea(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('Viewed by ${views.length}',
                    style: const TextStyle(
                        fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                if (views.isEmpty)
                  const Text('No views yet.')
                else
                  for (final v in views)
                    ListTile(
                      leading: WaAvatar(
                          label: (v['display_name'] as String?) ?? 'S',
                          radius: 20),
                      title: Text((v['display_name'] as String?) ?? ''),
                      subtitle: Text('@${v['username'] ?? ''}'),
                    ),
                if (replies.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  const Text('Replies',
                      style:
                          TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                  for (final r in replies)
                    ListTile(
                      leading: WaAvatar(
                          label: (r['display_name'] as String?) ?? 'S',
                          radius: 20),
                      title: Text((r['display_name'] as String?) ?? ''),
                      subtitle: Text((r['content'] as String?) ?? ''),
                    ),
                ],
                if (reactions.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  const Text('Reactions',
                      style:
                          TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                  for (final r in reactions)
                    ListTile(
                      leading: Text((r['emoji'] as String?) ?? '',
                          style: const TextStyle(fontSize: 24)),
                      title: Text((r['display_name'] as String?) ?? ''),
                      subtitle: Text('@${r['username'] ?? ''}'),
                    ),
                ],
              ],
            ),
          ),
        ),
      );
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  /// Delete my own story after a confirmation (Stage 2).
  Future<void> _deleteStory(Map<String, dynamic> st) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete status?'),
        content: const Text('This status will be deleted for everyone.'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Delete')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.deleteStory(st['id'] as String);
      final stories = await _api.listStories();
      setState(() => _stories = stories);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  void _showNewChatSheet() {
    showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.person_add_outlined),
              title: const Text('New direct message'),
              onTap: () async {
                Navigator.pop(context);
                final users = await _api.listUsers();
                if (!context.mounted || users.isEmpty) return;
                // Stage 7: contact search — pick by name or username.
                final queryCtrl = TextEditingController();
                final picked = await showDialog<Map<String, dynamic>>(
                  context: context,
                  builder: (context) => StatefulBuilder(
                    builder: (context, setDialogState) => AlertDialog(
                      title: const Text('New direct message'),
                      content: SizedBox(
                        width: double.maxFinite,
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            TextField(
                              controller: queryCtrl,
                              decoration: const InputDecoration(
                                  hintText: 'Search contacts…',
                                  prefixIcon: Icon(Icons.search)),
                              onChanged: (_) => setDialogState(() {}),
                            ),
                            Flexible(
                              child: ListView(
                                shrinkWrap: true,
                                children: users.where((u) {
                                  final m = u as Map<String, dynamic>;
                                  if (m['id'] == _user?['id']) return false;
                                  final q = queryCtrl.text.trim().toLowerCase();
                                  if (q.isEmpty) return true;
                                  final name =
                                      ((m['display_name'] as String?) ?? '')
                                          .toLowerCase();
                                  final username =
                                      ((m['username'] as String?) ?? '')
                                          .toLowerCase();
                                  return name.contains(q) ||
                                      username.contains(q);
                                }).map((u) {
                                  final m = u as Map<String, dynamic>;
                                  return ListTile(
                                    leading: WaAvatar(
                                        label: (m['display_name'] as String?) ??
                                            (m['username'] as String?) ??
                                            '?',
                                        radius: 20),
                                    title: Text(
                                        (m['display_name'] as String?) ??
                                            (m['username'] as String?) ??
                                            ''),
                                    subtitle: Text(
                                        '@${m['username'] as String? ?? ''}'),
                                    onTap: () => Navigator.pop(context, m),
                                  );
                                }).toList(),
                              ),
                            ),
                          ],
                        ),
                      ),
                      actions: [
                        TextButton(
                            onPressed: () => Navigator.pop(context),
                            child: const Text('Cancel')),
                      ],
                    ),
                  ),
                );
                if (picked == null) return;
                final chat = await _api.createDm(picked['id'] as String);
                await _refreshChats();
                await _openChat(chat);
              },
            ),
            ListTile(
              leading: const Icon(Icons.group_add_outlined),
              title: const Text('New group / channel'),
              onTap: () {
                Navigator.pop(context);
                _createGroupOrChannel();
              },
            ),
            ListTile(
              leading: const Icon(Icons.link),
              title: const Text('Join via invite link'),
              onTap: () {
                Navigator.pop(context);
                _joinViaInvite();
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showSearchSheet() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      // Stage 7: StatefulBuilder so contact matches filter live
      // while typing (message results still refresh on submit).
      builder: (context) => StatefulBuilder(
        builder: (context, setSheetState) => Padding(
          padding: EdgeInsets.only(
              bottom: MediaQuery.of(context).viewInsets.bottom,
              left: 16,
              right: 16,
              top: 16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(children: [
                Expanded(
                  child: TextField(
                    controller: _searchCtrl,
                    decoration: const InputDecoration(
                        hintText: 'Search messages and contacts…',
                        prefixIcon: Icon(Icons.search)),
                    onChanged: (_) => setSheetState(() {}),
                    onSubmitted: (_) async {
                      await _search();
                      setSheetState(() {});
                    },
                  ),
                ),
                IconButton(
                    icon: const Icon(Icons.search),
                    onPressed: () async {
                      await _search();
                      setSheetState(() {});
                    }),
              ]),
              const SizedBox(height: 8),
              if (_searching) const CircularProgressIndicator(),
              // Stage 7: contact search — matching users by name
              // or username; tap opens a DM with them.
              ..._contactSearchResults().map((u) => ListTile(
                    leading: WaAvatar(
                        label: (u['display_name'] as String?) ??
                            (u['username'] as String?) ??
                            '?',
                        radius: 20),
                    title: Text((u['display_name'] as String?) ??
                        (u['username'] as String?) ??
                        ''),
                    subtitle: Text('@${u['username'] as String? ?? ''}'),
                    onTap: () async {
                      Navigator.pop(context);
                      final chat = await _api.createDm(u['id'] as String);
                      await _refreshChats();
                      await _openChat(chat);
                    },
                  )),
              Flexible(
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: _searchResults.length,
                  itemBuilder: (context, i) {
                    final m = _searchResults[i] as Map<String, dynamic>;
                    return ListTile(
                      title: Text(m['content'] as String? ?? ''),
                      subtitle: Text('in chat ${m['chat_id']}'),
                      onTap: () async {
                        Navigator.pop(context);
                        final chat = _chats.firstWhere(
                            (c) => (c as Map)['id'] == m['chat_id'],
                            orElse: () => null);
                        if (chat != null) {
                          await _openChat(chat as Map<String, dynamic>);
                        }
                      },
                    );
                  },
                ),
              ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }

  /// Stage 7: users whose name or username matches the current
  /// search text (up to 8), excluding myself.
  List<Map<String, dynamic>> _contactSearchResults() {
    final q = _searchCtrl.text.trim().toLowerCase();
    if (q.isEmpty) return [];
    return _usersById.values
        .where((u) {
          if (u['id'] == _user?['id']) return false;
          final name = ((u['display_name'] as String?) ?? '').toLowerCase();
          final username = ((u['username'] as String?) ?? '').toLowerCase();
          return name.contains(q) || username.contains(q);
        })
        .take(8)
        .toList();
  }

  /// Write one call-history record (Stage 3) and refresh the Calls
  /// tab. Fire-and-forget: history must never break the call UI.
  Future<void> _recordEndedCall({
    required String chatId,
    String? peerId,
    required bool video,
    required bool wasConnected,
    required int durationSeconds,
  }) async {
    try {
      await _api.recordCall(
        chatId: chatId,
        calleeId: peerId,
        callType: video ? 'video' : 'voice',
        status: wasConnected ? 'ended' : 'missed',
        durationSeconds: wasConnected ? durationSeconds : 0,
      );
      await _refreshCalls();
    } catch (_) {}
  }

  /// Refresh the Calls tab history (Stage 3). Best-effort.
  Future<void> _refreshCalls() async {
    try {
      final calls = await _api.listCalls();
      if (mounted) setState(() => _calls = calls);
    } catch (_) {}
  }

  /// End the current call (1:1 or group) and record it in history
  /// when I started it (Stage 3).
  Future<void> _endCallAndRecord() async {
    final call = _call;
    if (call == null) return;
    final wasConnected = call.connected;
    final duration = call.durationSeconds;
    final chatId = call.chatId;
    final peerId = call.peerId;
    final video = call.video;
    final group = call.isGroup;
    final initiatedByMe = _callInitiatedByMe;
    if (group) {
      await call.endGroupCall();
    } else {
      await call.endCall();
    }
    _callInitiatedByMe = false;
    if (mounted) setState(() {});
    if (initiatedByMe && chatId.isNotEmpty) {
      await _recordEndedCall(
        chatId: chatId,
        peerId: group ? null : peerId,
        video: video,
        wasConnected: wasConnected,
        durationSeconds: duration,
      );
    }
  }

  /// One video renderer per member of a group call (Stage 3):
  /// create renderers for new remote streams, drop the ones gone.
  Future<void> _syncGroupRenderers() async {
    final streams = _call?.groupRemoteStreams ?? {};
    for (final entry in streams.entries) {
      var r = _groupRenderers[entry.key];
      if (r == null) {
        r = RTCVideoRenderer();
        await r.initialize();
        _groupRenderers[entry.key] = r;
      }
      r.srcObject = entry.value;
    }
    final gone =
        _groupRenderers.keys.where((k) => !streams.containsKey(k)).toList();
    for (final k in gone) {
      await _groupRenderers[k]?.dispose();
      _groupRenderers.remove(k);
    }
  }

  /// Incoming-call banner (Stage 3): shown above every screen — the
  /// chat list, the tabs, and inside a chat — so a call is never
  /// missed just because the user is not inside that chat.
  Widget _incomingCallBanner() {
    final call = _incomingCall!;
    final video = _call?.video == true;
    String who;
    if (call.group) {
      String title = 'Group';
      for (final c in _chats) {
        if ((c as Map)['id'] == call.chatId) {
          title = _chatTitle(c as Map<String, dynamic>);
          break;
        }
      }
      who = 'Incoming group ${video ? 'video' : 'voice'} call · $title';
    } else {
      final peer = _usersById[call.peerId];
      final name = (peer?['display_name'] as String?) ?? 'Unknown';
      who = 'Incoming ${video ? 'video' : 'voice'} call · $name';
    }
    return Container(
      color: Colors.green.shade100,
      padding: const EdgeInsets.all(12),
      child: Row(children: [
        const Icon(Icons.call),
        const SizedBox(width: 8),
        Expanded(child: Text(who)),
        FilledButton(
          onPressed: _acceptIncomingCall,
          child: const Text('Accept'),
        ),
        const SizedBox(width: 8),
        OutlinedButton(
          onPressed: _declineIncomingCall,
          child: const Text('Decline'),
        ),
      ]),
    );
  }

  /// Accept the ringing call (Stage 3): a group invite joins the
  /// mesh (offering to every other member except the inviter).
  /// Opens the chat first, so the in-call controls are visible.
  Future<void> _acceptIncomingCall() async {
    final call = _incomingCall;
    if (call == null) return;
    setState(() => _incomingCall = null);
    _callInitiatedByMe = false;
    _openChatById(call.chatId);
    if (call.group) {
      final members = await _api.listMembers(call.chatId);
      final others = <String>[];
      for (final m in members) {
        final uid = (m as Map)['user_id'] as String;
        if (uid != _user!['id'] && uid != call.peerId) {
          others.add(uid);
        }
      }
      await _call?.acceptGroupCall(call, others);
    } else {
      await _call?.acceptCall(call);
    }
    if (mounted) setState(() {});
  }

  /// "Return to call" bar (Stage 3): when a call is active and the
  /// user is on the chat list, one tap brings the call screen back.
  Widget _inCallReturnBar() {
    final chatId = _call?.chatId ?? '';
    return InkWell(
      onTap: () {
        if (chatId.isNotEmpty) _openChatById(chatId);
      },
      child: Container(
        color: Colors.green.shade100,
        padding: const EdgeInsets.all(12),
        child: Row(children: [
          const Icon(Icons.call, color: Colors.green),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
                'In call — ${_callDurationLabel()} · Tap to return to call'),
          ),
          const Icon(Icons.keyboard_arrow_up),
        ]),
      ),
    );
  }

  /// Decline the ringing call (Stage 3). The caller records it as a
  /// missed call in their history.
  void _declineIncomingCall() {
    final call = _incomingCall;
    if (call == null) return;
    if (call.group) {
      _call?.declineGroupCall(call);
    } else {
      _call?.declineCall(call);
    }
    setState(() => _incomingCall = null);
  }

  String _callDurationLabel() {
    final s = _call?.durationSeconds ?? 0;
    final m = s ~/ 60;
    final sec = s % 60;
    return '${m.toString().padLeft(2, '0')}:${sec.toString().padLeft(2, '0')}';
  }

  /// Start a call from the open chat (Stage 3): a group chat starts a
  /// mesh group call with every other member; a DM starts a 1:1 call.
  Future<void> _startCall({required bool video}) async {
    if (_activeChat == null || _call == null) return;
    try {
      if (_activeChat!['type'] == 'group') {
        final members = await _api.listMembers(_activeChat!['id'] as String);
        final peerIds = <String>[];
        for (final m in members) {
          final uid = (m as Map)['user_id'] as String;
          if (uid != _user!['id']) peerIds.add(uid);
        }
        if (peerIds.isEmpty) {
          setState(() => _error = 'No one else in this group to call');
          return;
        }
        _callInitiatedByMe = true;
        await _call!.startGroupCall(
            chatId: _activeChat!['id'] as String,
            peerIds: peerIds,
            video: video);
      } else {
        final peerId = await _peerUserId();
        if (peerId.isEmpty) {
          setState(() => _error = 'Cannot call: peer user not found');
          return;
        }
        _callInitiatedByMe = true;
        await _call!.startCall(
            chatId: _activeChat!['id'] as String, peerId: peerId, video: video);
      }
      if (mounted) setState(() {});
    } catch (e) {
      setState(() => _error = 'Call failed: $e');
    }
  }

  /// The other participant of the active DM, from the member list.
  String _peerIdCache = '';
  Future<String> _peerUserId() async {
    if (_peerIdCache.isNotEmpty) return _peerIdCache;
    if (_activeChat == null) return '';
    try {
      final members = await _api.listMembers(_activeChat!['id'] as String);
      for (final m in members) {
        final uid = (m as Map)['user_id'] as String;
        if (uid != _user!['id']) {
          _peerIdCache = uid;
          return uid;
        }
      }
    } catch (_) {}
    return '';
  }

  /// Stage 1: WhatsApp-style pinned-message banner at the top of a
  /// chat. Tap shows the full message; the X unpins it.
  Widget _pinnedBanner(Map<String, dynamic> pm) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final kind = pm['kind'] as String? ?? 'text';
    final text = kind == 'text'
        ? (pm['content'] as String? ?? '')
        : kind == 'image'
            ? 'Photo'
            : kind == 'video'
                ? 'Video'
                : kind == 'voice'
                    ? 'Voice message'
                    : kind == 'file'
                        ? 'File'
                        : kind == 'location'
                            ? 'Location'
                            : kind == 'contact'
                                ? 'Contact'
                                : kind == 'e2ee'
                                    ? 'Encrypted message'
                                    : 'Message';
    final sender = _usersById[pm['sender_id'] as String? ?? ''];
    final name = pm['sender_id'] == _user?['id']
        ? 'You'
        : (sender?['display_name'] as String?) ??
            (sender?['username'] as String?) ??
            '';
    return Container(
      margin: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: dark ? WaColors.darkBubbleIn : Colors.white,
        borderRadius: BorderRadius.circular(8),
      ),
      child: InkWell(
        onTap: () => showDialog(
          context: context,
          builder: (context) => AlertDialog(
            title: Text(name.isEmpty ? 'Pinned message' : 'Pinned by $name'),
            content: Text(text),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Close'),
              ),
            ],
          ),
        ),
        child: Row(
          children: [
            Icon(Icons.push_pin,
                size: 16, color: dark ? WaColors.accentDark : WaColors.accent),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (name.isNotEmpty)
                    Text(name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w700,
                            color:
                                dark ? WaColors.accentDark : WaColors.accent)),
                  Text(text,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontSize: 13)),
                ],
              ),
            ),
            IconButton(
              icon: const Icon(Icons.close, size: 18),
              tooltip: 'Unpin',
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
              onPressed: () async {
                try {
                  await _api.unpinMessage(_activeChat!['id'] as String);
                  await _refreshChats();
                  final fresh = _chats.firstWhere(
                      (c) => (c as Map)['id'] == _activeChat!['id'],
                      orElse: () => _activeChat!);
                  setState(() => _activeChat = fresh as Map<String, dynamic>);
                } catch (e) {
                  setState(() => _error = e.toString());
                }
              },
            ),
          ],
        ),
      ),
    );
  }

  /// WhatsApp conversation screen: plain chat background, date
  /// chips, a yellow E2EE notice when encryption is on for this
  /// chat, and the WhatsApp input bar (rounded field with attach +
  /// camera inside, green mic/send circle outside).
  Widget _buildChatBody() {
    final callState = _call?.state ?? CallState.idle;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final chatBg = _wallpaperDecoration(
        _wallpaperForChat(_activeChat!['id'] as String?), dark);
    final isE2ee = _e2eeChats.contains(_activeChat!['id']);
    return Container(
      decoration: chatBg,
      child: Column(
        children: [
          _offlineBanner(),
          if (_incomingCall != null) _incomingCallBanner(),
          if (callState == CallState.inCall || callState == CallState.outgoing)
            Container(
              color: dark ? const Color(0xFF0C3D55) : const Color(0xFFD6F3FB),
              padding: const EdgeInsets.all(12),
              child: Row(children: [
                const Icon(Icons.call),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    callState == CallState.outgoing
                        ? 'Calling… (Opus 12 kbps)'
                        : 'In call — ${_callDurationLabel()}${_call?.isGroup == true ? ' · Group' : ''} · ~0.09 MB/min',
                  ),
                ),
                // Stage 3: in-call controls — mic, camera, switch.
                IconButton(
                  icon:
                      Icon(_call?.micMuted == true ? Icons.mic_off : Icons.mic),
                  tooltip: _call?.micMuted == true ? 'Unmute' : 'Mute',
                  onPressed: () {
                    _call?.toggleMute();
                    setState(() {});
                  },
                ),
                // Speaker toggle: loudspeaker on/off (voice + video).
                IconButton(
                  icon: Icon(_call?.speakerOn == true
                      ? Icons.volume_up
                      : Icons.volume_down),
                  tooltip:
                      _call?.speakerOn == true ? 'Speaker off' : 'Speaker on',
                  onPressed: () async {
                    await _call?.toggleSpeaker();
                    setState(() {});
                  },
                ),
                if (_call?.video == true) ...[
                  IconButton(
                    icon: Icon(_call?.cameraOff == true
                        ? Icons.videocam_off
                        : Icons.videocam),
                    tooltip: _call?.cameraOff == true
                        ? 'Turn camera on'
                        : 'Turn camera off',
                    onPressed: () {
                      _call?.toggleCamera();
                      setState(() {});
                    },
                  ),
                  IconButton(
                    icon: const Icon(Icons.cameraswitch),
                    tooltip: 'Switch camera',
                    onPressed: () async {
                      await _call?.switchCamera();
                      setState(() {});
                    },
                  ),
                ],
                if (_call?.video == true && _call?.audioOnly == false)
                  TextButton(
                    onPressed: () async {
                      await _call?.dropToAudioOnly();
                      setState(() {});
                    },
                    child: const Text('Audio only'),
                  ),
                IconButton(
                  icon: const Icon(Icons.call_end, color: Colors.red),
                  tooltip: 'End call',
                  onPressed: _endCallAndRecord,
                ),
              ]),
            ),
          if (_call?.video == true && _call?.remoteStream != null)
            SizedBox(
              height: 200,
              child: RTCVideoView(_remoteRenderer),
            ),
          // Stage 3: group-call video tiles, one per connected member.
          if (_call?.isGroup == true &&
              _call?.video == true &&
              _groupRenderers.isNotEmpty)
            SizedBox(
              height: 160,
              child: GridView.count(
                crossAxisCount: 2,
                childAspectRatio: 4 / 3,
                children: [
                  for (final entry in _groupRenderers.entries)
                    RTCVideoView(entry.value),
                ],
              ),
            ),
          if (isE2ee)
            Container(
              margin: const EdgeInsets.fromLTRB(16, 12, 16, 0),
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: dark ? const Color(0xFF0C2E3E) : const Color(0xFFFFF8C6),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(Icons.lock,
                      size: 15, color: dark ? Colors.white70 : Colors.black54),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      'Messages are end-to-end encrypted. No one outside '
                      'of this chat, not even Safari, can read or listen '
                      'to them.',
                      style: TextStyle(
                        fontSize: 12,
                        color: dark ? Colors.white70 : Colors.black87,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          // Stage 1: pinned-message banner (WhatsApp-style).
          if (_activeChat?['pinned_message'] is Map<String, dynamic>)
            _pinnedBanner(
                _activeChat!['pinned_message'] as Map<String, dynamic>),
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.fromLTRB(8, 12, 8, 8),
              itemCount: _messages.length,
              itemBuilder: (context, i) {
                final m = _messages[i] as Map<String, dynamic>;
                final showDate = i == 0 ||
                    _messageDay(_messages[i - 1] as Map<String, dynamic>) !=
                        _messageDay(m);
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    if (showDate) _dateChip(_messageDay(m)),
                    _messageBubble(m),
                    const SizedBox(height: 4),
                  ],
                );
              },
            ),
          ),
          if (_typingUser.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text('$_typingUser is typing…',
                    style: const TextStyle(
                        fontSize: 12, fontStyle: FontStyle.italic)),
              ),
            ),
          if (_replyTo != null)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              color: dark ? WaColors.darkInputBackground : Colors.grey.shade200,
              child: Row(children: [
                Expanded(
                  child: Text(
                    'Replying to: ${_decrypted[_replyTo!['id'] as String] ?? (_replyTo!['content'] as String? ?? '')}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.close, size: 18),
                  onPressed: () => setState(() => _replyTo = null),
                ),
              ]),
            ),
          if (_editingId != null)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              color: Colors.amber.shade100,
              child: Row(children: [
                const Expanded(
                    child: Text('Editing message…',
                        style: TextStyle(fontSize: 12))),
                IconButton(
                  icon: const Icon(Icons.close, size: 18),
                  onPressed: () => setState(() {
                    _editingId = null;
                    _draft.clear();
                  }),
                ),
              ]),
            ),
          if (_draftEmpty) _smartReplyChips(),
          // Stage 5: in a channel only admins and the owner may
          // post — followers see a read-only notice instead of
          // the input bar.
          if (_activeChat?['type'] == 'channel' &&
              ((_activeChat?['roles'] as Map?)?[_user?['id']] ?? 'member') ==
                  'member')
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
              color: Colors.transparent,
              child: const Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.campaign_outlined, size: 18),
                  SizedBox(width: 8),
                  Text('Only admins can post in this channel',
                      style: TextStyle(fontSize: 13)),
                ],
              ),
            )
          else
            Container(
              padding: const EdgeInsets.fromLTRB(8, 6, 8, 10),
              color: Colors.transparent,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Expanded(
                    child: Container(
                      decoration: BoxDecoration(
                        color: dark
                            ? WaColors.darkInputBackground
                            : WaColors.lightInputBackground,
                        borderRadius: BorderRadius.circular(24),
                        boxShadow: [
                          BoxShadow(
                            color: Colors.black.withOpacity(0.06),
                            blurRadius: 4,
                            offset: const Offset(0, 1),
                          ),
                        ],
                      ),
                      child: Row(
                        children: [
                          Expanded(
                            child: TextField(
                              controller: _draft,
                              decoration: const InputDecoration(
                                hintText: 'Message',
                                border: InputBorder.none,
                                enabledBorder: InputBorder.none,
                                focusedBorder: InputBorder.none,
                                filled: false,
                                contentPadding: EdgeInsets.symmetric(
                                    horizontal: 16, vertical: 12),
                              ),
                              onChanged: (_) => _sendTyping(),
                              onSubmitted: (_) => _send(),
                            ),
                          ),
                          if (_ttlSeconds > 0)
                            Padding(
                              padding: const EdgeInsets.only(right: 4),
                              child: Chip(
                                label: Text(
                                    '⏱ ${_ttlSeconds >= 86400 ? '1d' : _ttlSeconds >= 3600 ? '1h' : '1m'}'),
                                visualDensity: VisualDensity.compact,
                                onDeleted: () =>
                                    setState(() => _ttlSeconds = 0),
                              ),
                            ),
                          IconButton(
                            icon: Icon(
                              Icons.attach_file,
                              color:
                                  dark ? WaColors.darkIcon : WaColors.lightIcon,
                            ),
                            tooltip:
                                'Attach: photo, location, schedule, disappearing',
                            onPressed: _showAttachSheet,
                          ),
                          IconButton(
                            icon: Icon(
                              Icons.camera_alt_outlined,
                              color:
                                  dark ? WaColors.darkIcon : WaColors.lightIcon,
                            ),
                            tooltip: 'Send photo',
                            onPressed: _sendPhoto,
                          ),
                          const SizedBox(width: 4),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(width: 6),
                  if (_draftEmpty)
                    VoiceRecordButton(
                      api: _api,
                      chatId: _activeChat!['id'] as String,
                      onSent: _refreshMessages,
                    )
                  else
                    _WaSendButton(onPressed: _send),
                ],
              ),
            ),
          if (_error != null)
            Text(_error!, style: const TextStyle(color: Colors.red)),
        ],
      ),
    );
  }

  /// Day key (yyyy-mm-dd) of a message, for the date chips.
  String _messageDay(Map<String, dynamic> m) {
    final raw = m['created_at'] as String?;
    if (raw == null || raw.isEmpty) return '';
    try {
      final t = DateTime.parse(raw).toLocal();
      return '${t.year}-${t.month.toString().padLeft(2, '0')}-${t.day.toString().padLeft(2, '0')}';
    } catch (_) {
      return '';
    }
  }

  /// WhatsApp date chip: "Today", "Yesterday", weekday, or date.
  Widget _dateChip(String dayKey) {
    if (dayKey.isEmpty) return const SizedBox.shrink();
    final now = DateTime.now();
    final today =
        '${now.year}-${now.month.toString().padLeft(2, '0')}-${now.day.toString().padLeft(2, '0')}';
    final yesterday = now.subtract(const Duration(days: 1));
    final yesterdayKey =
        '${yesterday.year}-${yesterday.month.toString().padLeft(2, '0')}-${yesterday.day.toString().padLeft(2, '0')}';
    String label = dayKey;
    if (dayKey == today) {
      label = 'Today';
    } else if (dayKey == yesterdayKey) {
      label = 'Yesterday';
    } else {
      try {
        final t = DateTime.parse(dayKey);
        if (now.difference(t).inDays < 7) {
          const weekdays = [
            'Monday',
            'Tuesday',
            'Wednesday',
            'Thursday',
            'Friday',
            'Saturday',
            'Sunday'
          ];
          label = weekdays[t.weekday - 1];
        } else {
          label = '${t.day}/${t.month}/${t.year}';
        }
      } catch (_) {}
    }
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
          decoration: BoxDecoration(
            color: dark ? WaColors.darkBubbleIn : const Color(0xFFFCF8F0),
            borderRadius: BorderRadius.circular(8),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withOpacity(0.08),
                blurRadius: 3,
                offset: const Offset(0, 1),
              ),
            ],
          ),
          child: Text(
            label,
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: dark
                  ? WaColors.darkTextSecondary
                  : WaColors.lightTextSecondary,
            ),
          ),
        ),
      ),
    );
  }

  /// Smart-reply chips for the last incoming text message (local,
  /// rule-based). Empty when nothing sensible matches — no fake chips.
  Widget _smartReplyChips() {
    String? lastIncoming;
    for (var i = _messages.length - 1; i >= 0; i--) {
      final m = _messages[i] as Map<String, dynamic>;
      if (m['sender_id'] != _user!['id'] &&
          (m['kind'] as String? ?? 'text') == 'text') {
        lastIncoming =
            _decrypted[m['id'] as String] ?? (m['content'] as String? ?? '');
        break;
      }
    }
    if (lastIncoming == null || lastIncoming.isEmpty) {
      return const SizedBox.shrink();
    }
    final lang = Localizations.localeOf(context).languageCode == 'fa'
        ? AiLang.fa
        : AiLang.en;
    final replies = smartReplies(lastIncoming, lang);
    if (replies.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.fromLTRB(10, 4, 10, 0),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          children: replies
              .map((r) => Padding(
                    padding: const EdgeInsets.only(right: 6),
                    child: ActionChip(
                      label: Text(r),
                      onPressed: () => _draft.text = r,
                    ),
                  ))
              .toList(),
        ),
      ),
    );
  }

  /// Attach sheet: photo, location, schedule, disappearing timer.
  /// Keeps the input bar clean (premium) while all features stay
  /// one tap away.
  /// Stage 1: pick a Safari user to share as a contact card in this
  /// chat. The card carries their name + @username; tapping it in
  /// the chat opens a DM with them.
  void _showContactPicker() async {
    List<dynamic> users;
    try {
      users = await _api.listUsers();
    } catch (e) {
      setState(() => _error = e.toString());
      return;
    }
    final me = _user?['id'] as String?;
    final others = users
        .where((u) => (u as Map)['id'] != me)
        .cast<Map<String, dynamic>>()
        .toList();
    if (!mounted) return;
    showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text('Share contact',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
            ),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: others.length,
                itemBuilder: (context, i) {
                  final u = others[i];
                  final name = (u['display_name'] as String?)?.trim() ?? '';
                  final username = (u['username'] as String?) ?? '';
                  final label = name.isNotEmpty ? name : '@$username';
                  return ListTile(
                    leading: WaAvatar(label: label, radius: 22),
                    title: Text(label),
                    subtitle: username.isNotEmpty ? Text('@$username') : null,
                    onTap: () async {
                      Navigator.pop(context);
                      try {
                        await _api.sendContact(
                            _activeChat!['id'] as String, u['id'] as String);
                        await _refreshMessages();
                      } catch (e) {
                        setState(() => _error = e.toString());
                      }
                    },
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _showAttachSheet() {
    showModalBottomSheet(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.photo_outlined),
              title: const Text('Send photo (compressed)'),
              onTap: () {
                Navigator.pop(context);
                _sendPhoto();
              },
            ),
            ListTile(
              leading: const Icon(Icons.location_on_outlined),
              title: const Text('Share location'),
              onTap: () {
                Navigator.pop(context);
                _shareLocation();
              },
            ),
            // Stage 1: share a Safari user as a contact card.
            ListTile(
              leading: const Icon(Icons.person_outline),
              title: const Text('Share contact'),
              onTap: () {
                Navigator.pop(context);
                _showContactPicker();
              },
            ),
            // Stage 4: create a poll in this chat.
            ListTile(
              leading: const Icon(Icons.bar_chart_outlined),
              title: const Text('Poll'),
              onTap: () {
                Navigator.pop(context);
                _showPollComposer();
              },
            ),
            ListTile(
              leading: const Icon(Icons.schedule_outlined),
              title: const Text('Schedule message'),
              onTap: () {
                Navigator.pop(context);
                _scheduleDraft();
              },
            ),
            ListTile(
              leading: const Icon(Icons.timer_outlined),
              title: Text(_ttlSeconds == 0
                  ? 'Disappearing messages: off'
                  : 'Disappearing messages: ${_ttlSeconds >= 86400 ? '1 day' : _ttlSeconds >= 3600 ? '1 hour' : '1 minute'}'),
              onTap: () {
                Navigator.pop(context);
                showDialog(
                  context: context,
                  builder: (context) => SimpleDialog(
                    title: const Text('Disappearing messages'),
                    children: [0, 60, 3600, 86400]
                        .map((v) => SimpleDialogOption(
                              onPressed: () {
                                setState(() => _ttlSeconds = v);
                                Navigator.pop(context);
                              },
                              child: Text(v == 0
                                  ? 'Off'
                                  : v == 60
                                      ? '1 minute'
                                      : v == 3600
                                          ? '1 hour'
                                          : '1 day'),
                            ))
                        .toList(),
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }

  /// Safari message bubble: cyan outgoing (#D9FDD3 light /
  /// #005C4B dark), white/dark incoming, small radius with the tail
  /// corner, time + ticks inline at the bottom-right (blue ticks
  /// when read).
  Widget _messageBubble(Map<String, dynamic> m) {
    final kind = m['kind'] as String? ?? 'text';
    // Stage 4: system notices ("X pinned a message") render as a
    // centered grey chip, like WhatsApp — no bubble, no actions.
    if (kind == 'system') {
      final dark = Theme.of(context).brightness == Brightness.dark;
      return Align(
        alignment: Alignment.center,
        child: Container(
          margin: const EdgeInsets.symmetric(vertical: 6),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
          decoration: BoxDecoration(
            color: dark ? WaColors.darkChatBackground : const Color(0xFFF0F2F5),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Text(
            (m['content'] as String?) ?? '',
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 12.5,
              color: dark
                  ? WaColors.darkTextSecondary
                  : WaColors.lightTextSecondary,
            ),
          ),
        ),
      );
    }
    final mine = m['sender_id'] == _user!['id'];
    final status = m['status'] as String?;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final time = (m['created_at'] as String?) ?? '';
    final timeLabel = time.length >= 19 ? time.substring(11, 16) : time;
    final isE2ee = kind == 'e2ee';
    final content = isE2ee
        ? (_decrypted[m['id'] as String] ?? 'Encrypted message')
        : (m['content'] as String? ?? '');
    final replyToId = m['reply_to_id'] as String?;
    String? replyText;
    if (replyToId != null) {
      for (final other in _messages) {
        final om = other as Map<String, dynamic>;
        if (om['id'] == replyToId) {
          replyText = _decrypted[replyToId] ?? (om['content'] as String?);
          break;
        }
      }
    }
    final reactions = m['reactions'] as Map<String, dynamic>?;
    // v0.7 forwarding: "Forwarded from <name>" above the body.
    final fwdLabel = forwardedLabel(m);
    final bubbleColor = mine
        ? (dark ? WaColors.darkBubbleOut : WaColors.lightBubbleOut)
        : (dark ? WaColors.darkBubbleIn : WaColors.lightBubbleIn);
    final textColor = dark ? WaColors.darkText : WaColors.lightText;
    final metaColor =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    final tickColor = status == 'read' ? WaColors.readBlue : metaColor;
    final ticks = !mine
        ? ''
        : m['pending'] == true
            ? ' 🕐'
            : status == 'read' || status == 'delivered'
                ? ' ✓✓'
                : ' ✓';
    // Subtle entrance: fade + rise, so new messages feel alive.
    return TweenAnimationBuilder<double>(
      tween: Tween(begin: 0.0, end: 1.0),
      duration: const Duration(milliseconds: 220),
      builder: (context, t, child) => Opacity(
        opacity: t,
        child: Transform.translate(
          offset: Offset(0, (1 - t) * 10),
          child: child,
        ),
      ),
      child: Align(
        alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
        child: GestureDetector(
          onLongPress: () => _showMessageActions(m),
          child: Container(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.of(context).size.width * 0.78,
            ),
            decoration: BoxDecoration(
              color: bubbleColor,
              borderRadius: BorderRadius.only(
                topLeft: const Radius.circular(18),
                topRight: const Radius.circular(18),
                bottomLeft: Radius.circular(mine ? 18 : 4),
                bottomRight: Radius.circular(mine ? 4 : 18),
              ),
              border: mine
                  ? null
                  : Border.all(
                      color: dark
                          ? const Color(0xFF1C2A44)
                          : const Color(0xFFE2E8F0),
                    ),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.08),
                  blurRadius: 3,
                  offset: const Offset(0, 1),
                ),
              ],
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (isE2ee)
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.lock, size: 12, color: metaColor),
                        const SizedBox(width: 2),
                        Text('E2EE',
                            style: TextStyle(fontSize: 11, color: metaColor)),
                      ],
                    ),
                  if (fwdLabel != null)
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.forward, size: 12, color: metaColor),
                        const SizedBox(width: 2),
                        Text(fwdLabel,
                            style: TextStyle(
                                fontSize: 11,
                                fontStyle: FontStyle.italic,
                                color: metaColor)),
                      ],
                    ),
                  if (replyText != null)
                    Container(
                      padding: const EdgeInsets.all(6),
                      margin: const EdgeInsets.only(bottom: 4),
                      decoration: BoxDecoration(
                        color: mine
                            ? Colors.black.withOpacity(0.08)
                            : Colors.black.withOpacity(0.06),
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(replyText,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                              fontSize: 12,
                              fontStyle: FontStyle.italic,
                              color: textColor)),
                    ),
                  DefaultTextStyle(
                    style: TextStyle(
                      fontSize: 15,
                      height: 1.35,
                      color: textColor,
                    ),
                    child: _messageContent(m, kind, content),
                  ),
                  if (kind == 'text')
                    Align(
                      alignment: Alignment.centerLeft,
                      child: TextButton.icon(
                        icon: const Icon(Icons.translate, size: 14),
                        label: const Text('Translate',
                            style: TextStyle(fontSize: 11)),
                        style: TextButton.styleFrom(
                          padding: EdgeInsets.zero,
                          minimumSize: const Size(0, 26),
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                          foregroundColor: metaColor,
                        ),
                        onPressed: () => _showTranslation(m, content),
                      ),
                    ),
                  if (reactions != null && reactions.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 4),
                      child: Text(
                        reactions.entries
                            .map((e) => '${e.key} ${(e.value as List).length}')
                            .join('  '),
                        style: TextStyle(
                          fontSize: 13,
                          color: textColor,
                        ),
                      ),
                    ),
                  Align(
                    alignment: Alignment.centerRight,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        // Stage 5: view count on channel posts.
                        if (_activeChat?['type'] == 'channel' &&
                            (m['view_count'] as int? ?? 0) > 0)
                          Padding(
                            padding: const EdgeInsets.only(right: 6),
                            child: Text(
                              '👁 ${m['view_count']}',
                              style: TextStyle(fontSize: 11, color: metaColor),
                            ),
                          ),
                        Text(
                          timeLabel,
                          style: TextStyle(fontSize: 11, color: metaColor),
                        ),
                        if (ticks.isNotEmpty)
                          Text(
                            ticks,
                            style: TextStyle(
                              fontSize: 13,
                              color: tickColor,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  /// Message body: text, location, voice, or media. Data Saver:
  /// media is only downloaded after a tap (or immediately when
  /// Data Saver is off).
  /// Stage 4 — Polls (WhatsApp parity). A poll message's content is
  /// JSON {"poll_id": "...", "question": "..."}.
  String? _parsePollId(String? content) {
    if (content == null) return null;
    try {
      final parsed = jsonDecode(content) as Map<String, dynamic>;
      return parsed['poll_id'] as String?;
    } catch (_) {
      return null;
    }
  }

  String _pollPreviewText(String? content) {
    if (content == null) return '📊 Poll';
    try {
      final parsed = jsonDecode(content) as Map<String, dynamic>;
      final q = (parsed['question'] as String?)?.trim() ?? '';
      return q.isEmpty ? '📊 Poll' : '📊 Poll: $q';
    } catch (_) {
      return '📊 Poll';
    }
  }

  bool _canDeletePoll(Map<String, dynamic> m) {
    // The poll creator is the message sender; group admins/owners
    // may also delete it (the server enforces both).
    if (m['sender_id'] == _user!['id']) return true;
    final roles = _activeChat?['roles'] as Map?;
    final role = roles?[_user!['id']] as String?;
    return role == 'admin' || role == 'owner';
  }

  /// Stage 4: poll composer — question + 2..12 options + the
  /// "allow multiple answers" switch.
  void _showPollComposer() {
    final chat = _activeChat;
    if (chat == null) return;
    final question = TextEditingController();
    final optionControllers = List.generate(2, (_) => TextEditingController());
    bool allowMultiple = false;
    showDialog(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Create poll'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: question,
                  decoration: const InputDecoration(
                    labelText: 'Question',
                    hintText: 'Ask something…',
                  ),
                  maxLength: 300,
                ),
                const SizedBox(height: 8),
                ...optionControllers.asMap().entries.map((e) {
                  final i = e.key;
                  final c = e.value;
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 8),
                    child: Row(
                      children: [
                        Expanded(
                          child: TextField(
                            controller: c,
                            decoration: InputDecoration(
                              labelText: 'Option ${i + 1}',
                            ),
                            maxLength: 100,
                          ),
                        ),
                        if (optionControllers.length > 2)
                          IconButton(
                            icon: const Icon(Icons.remove_circle_outline),
                            onPressed: () => setDialogState(() {
                              optionControllers.removeAt(i);
                            }),
                          ),
                      ],
                    ),
                  );
                }),
                if (optionControllers.length < 12)
                  TextButton.icon(
                    icon: const Icon(Icons.add),
                    label: const Text('Add option'),
                    onPressed: () => setDialogState(() {
                      optionControllers.add(TextEditingController());
                    }),
                  ),
                SwitchListTile(
                  title: const Text('Allow multiple answers'),
                  value: allowMultiple,
                  onChanged: (v) => setDialogState(() => allowMultiple = v),
                  contentPadding: EdgeInsets.zero,
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () async {
                final q = question.text.trim();
                final options = optionControllers
                    .map((c) => c.text.trim())
                    .where((o) => o.isNotEmpty)
                    .toList();
                if (q.isEmpty || options.length < 2) {
                  return;
                }
                Navigator.pop(context);
                try {
                  await _api.createPoll(
                      chat['id'] as String, q, options, allowMultiple);
                  await _refreshMessages();
                  await _refreshChats();
                } catch (e) {
                  setState(() => _error = e.toString());
                }
              },
              child: const Text('Create'),
            ),
          ],
        ),
      ),
    );
  }

  /// Stage 4: poll bubble — question, options with live vote counts
  /// and percentage bars; tap to vote (or change / clear the vote).
  Widget _pollBubble(Map<String, dynamic> m, String content) {
    final pollId = _parsePollId(content);
    final fallbackQuestion =
        _pollPreviewText(content).replaceFirst('📊 Poll: ', '');
    if (pollId == null) {
      return const Text('📊 Poll');
    }
    return FutureBuilder<Map<String, dynamic>>(
      future: _api.getPoll(pollId),
      builder: (context, snap) {
        if (!snap.hasData) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(fallbackQuestion,
                  style: const TextStyle(fontWeight: FontWeight.w600)),
              const SizedBox(height: 6),
              const Text('Loading poll…'),
            ],
          );
        }
        final poll = snap.data!;
        final question = (poll['question'] as String?) ?? fallbackQuestion;
        final options = (poll['options'] as List?) ?? [];
        final votes = (poll['votes'] as List?) ?? [];
        final myVotes = ((poll['my_votes'] as List?) ?? [])
            .map((v) => (v as num).toInt())
            .toSet();
        final allowMultiple = poll['allow_multiple'] == true;
        final total = (poll['total_votes'] as num?)?.toInt() ?? 0;
        final dark = Theme.of(context).brightness == Brightness.dark;
        final barBg = dark ? Colors.white12 : Colors.black12;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(question, style: const TextStyle(fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            ...options.asMap().entries.map((e) {
              final i = e.key;
              final label = e.value.toString();
              final count = i < votes.length ? (votes[i] as num).toInt() : 0;
              final pct = total == 0 ? 0.0 : count / total;
              final mine = myVotes.contains(i);
              return InkWell(
                onTap: () async {
                  List<int> next;
                  if (allowMultiple) {
                    next = myVotes.toList();
                    if (mine) {
                      next.remove(i);
                    } else {
                      next.add(i);
                    }
                  } else {
                    next = mine ? [] : [i];
                  }
                  try {
                    await _api.votePoll(pollId, next);
                    // The FutureBuilder re-fetches the poll on the
                    // next build, so the counts update right away.
                    await _refreshMessages();
                  } catch (e) {
                    setState(() => _error = e.toString());
                  }
                },
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            mine
                                ? (allowMultiple
                                    ? Icons.check_box
                                    : Icons.radio_button_checked)
                                : (allowMultiple
                                    ? Icons.check_box_outline_blank
                                    : Icons.radio_button_unchecked),
                            size: 18,
                            color: mine
                                ? WaColors.accent
                                : (dark
                                    ? WaColors.darkTextSecondary
                                    : WaColors.lightTextSecondary),
                          ),
                          const SizedBox(width: 8),
                          Flexible(child: Text(label)),
                          const SizedBox(width: 8),
                          Text('$count',
                              style: TextStyle(
                                color: dark
                                    ? WaColors.darkTextSecondary
                                    : WaColors.lightTextSecondary,
                                fontSize: 12,
                              )),
                        ],
                      ),
                      const SizedBox(height: 3),
                      ClipRRect(
                        borderRadius: BorderRadius.circular(3),
                        child: LinearProgressIndicator(
                          value: pct,
                          minHeight: 5,
                          backgroundColor: barBg,
                          color: mine
                              ? WaColors.accent
                              : (dark
                                  ? WaColors.darkTextSecondary
                                  : WaColors.lightTextSecondary),
                        ),
                      ),
                    ],
                  ),
                ),
              );
            }),
            const SizedBox(height: 4),
            Text(
              total == 0
                  ? 'No votes yet'
                  : '$total ${total == 1 ? 'vote' : 'votes'}',
              style: TextStyle(
                fontSize: 12,
                color: dark
                    ? WaColors.darkTextSecondary
                    : WaColors.lightTextSecondary,
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _messageContent(Map<String, dynamic> m, String kind, String content) {
    if (kind == 'poll') {
      // Stage 4: poll bubble — question + live vote counts.
      return _pollBubble(m, content);
    }
    if (kind == 'voice') {
      final fileUrl = m['file_url'] as String?;
      if (fileUrl == null) return Text(content);
      if (_dataSaver && !_downloaded.containsKey(fileUrl)) {
        return _downloadPlaceholder(
            '🎤 Voice message', m['file_size'] as int?, fileUrl);
      }
      return VoiceMessagePlayer(
        api: _api,
        fileUrl: fileUrl,
        fileName: (m['file_name'] as String?) ?? 'Voice message',
      );
    }
    if (kind == 'location') {
      return Text(
          '📍 ${m['location_label'] ?? 'Location'} (${m['lat']}, ${m['lng']})');
    }
    if (kind == 'contact') {
      // Stage 1: shared contact card — name + @username, tap
      // "Message" to open a DM with them.
      String userId = '';
      try {
        final parsed = jsonDecode(content) as Map<String, dynamic>;
        userId = parsed['user_id'] as String? ?? '';
      } catch (_) {}
      final u = _usersById[userId];
      final name = (u?['display_name'] as String?)?.trim() ?? '';
      final username = (u?['username'] as String?) ?? '';
      final label = name.isNotEmpty
          ? name
          : username.isNotEmpty
              ? '@$username'
              : 'Safari user';
      return InkWell(
        onTap: userId.isEmpty
            ? null
            : () async {
                try {
                  final chat = await _api.createDm(userId);
                  await _refreshChats();
                  await _openChat(chat);
                } catch (e) {
                  setState(() => _error = e.toString());
                }
              },
        child: Container(
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: Colors.black.withOpacity(0.05),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              WaAvatar(label: label, radius: 20),
              const SizedBox(width: 10),
              Flexible(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(label,
                        style: const TextStyle(fontWeight: FontWeight.w600)),
                    if (username.isNotEmpty)
                      Text('@$username', style: const TextStyle(fontSize: 12)),
                    const Text('Tap to message',
                        style: TextStyle(fontSize: 11)),
                  ],
                ),
              ),
            ],
          ),
        ),
      );
    }
    if (kind == 'image' ||
        kind == 'video' ||
        kind == 'file' ||
        kind == 'sticker') {
      final fileUrl = m['file_url'] as String?;
      if (fileUrl == null) return Text(content);
      final bytes = _downloaded[fileUrl];
      if (bytes == null) {
        if (!_dataSaver) {
          // Data Saver off: auto-download in the background.
          Future.microtask(() => _downloadMedia(fileUrl));
        }
        return _downloadPlaceholder(
            kind == 'image'
                ? '🖼️ Photo'
                : kind == 'video'
                    ? '🎬 Video'
                    : '📎 File',
            m['file_size'] as int?,
            fileUrl);
      }
      if (kind == 'image' || kind == 'sticker') {
        return Image.memory(bytes, width: 220);
      }
      if (kind == 'video') {
        return _VideoPlayer(bytes: bytes);
      }
      return Text('📎 ${(m['file_name'] as String?) ?? 'File'} '
          '(${(bytes.length / 1024).toStringAsFixed(0)} KB downloaded)');
    }
    // Stage 8: link preview — a text message with a link shows the
    // page title as a small card above the text.
    final url = LinkPreviewCache.extractUrl(content);
    if (url == null) return Text(content);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        FutureBuilder<String?>(
          future: _linkPreviews.titleFor(url),
          builder: (context, snap) {
            final title = snap.data;
            if (title == null) return const SizedBox.shrink();
            return Container(
              margin: const EdgeInsets.only(bottom: 4),
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: Colors.black.withOpacity(0.05),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.link, size: 16),
                  const SizedBox(width: 6),
                  Flexible(
                    child: Text(title,
                        style: const TextStyle(fontSize: 13),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis),
                  ),
                ],
              ),
            );
          },
        ),
        Text(content),
      ],
    );
  }

  Widget _downloadPlaceholder(String label, int? fileSize, String fileUrl) {
    final size =
        fileSize == null ? '' : ' (${(fileSize / 1024).toStringAsFixed(0)} KB)';
    final downloading = _downloading.contains(fileUrl);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text('$label$size', style: const TextStyle(fontSize: 13)),
        const SizedBox(width: 8),
        downloading
            ? const SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(strokeWidth: 2))
            : FilledButton(
                onPressed: () => _downloadMedia(fileUrl),
                child: const Text('Download'),
              ),
      ],
    );
  }
}

/// Plays a downloaded video from memory (via a temp file, since
/// video_player needs a file or URL).
class _VideoPlayer extends StatefulWidget {
  const _VideoPlayer({required this.bytes});
  final Uint8List bytes;

  @override
  State<_VideoPlayer> createState() => _VideoPlayerState();
}

class _VideoPlayerState extends State<_VideoPlayer> {
  VideoPlayerController? _controller;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    final dir = await getTemporaryDirectory();
    final file = File(
        '${dir.path}/vyro-video-${DateTime.now().millisecondsSinceEpoch}.mp4');
    await file.writeAsBytes(widget.bytes);
    final c = VideoPlayerController.file(file);
    await c.initialize();
    if (mounted) setState(() => _controller = c);
  }

  @override
  void dispose() {
    _controller?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = _controller;
    if (c == null || !c.value.isInitialized) {
      return const SizedBox(
          width: 220,
          height: 120,
          child: Center(child: CircularProgressIndicator()));
    }
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(
          width: 220,
          child: AspectRatio(
            aspectRatio: c.value.aspectRatio,
            child: VideoPlayer(c),
          ),
        ),
        IconButton(
          icon: Icon(c.value.isPlaying ? Icons.pause : Icons.play_arrow),
          onPressed: () =>
              setState(() => c.value.isPlaying ? c.pause() : c.play()),
        ),
      ],
    );
  }
}

/// WhatsApp green circular send button.
class _WaSendButton extends StatelessWidget {
  const _WaSendButton({required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Container(
      width: 48,
      height: 48,
      decoration: BoxDecoration(
        color: dark ? WaColors.accentDark : WaColors.accent,
        shape: BoxShape.circle,
      ),
      child: IconButton(
        icon: const Icon(Icons.send, color: Colors.white),
        tooltip: 'Send',
        onPressed: onPressed,
      ),
    );
  }
}
