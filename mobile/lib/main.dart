import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:video_player/video_player.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'api.dart';
import 'call.dart';
import 'chat_extras.dart';
import 'data_saver.dart';
import 'e2ee_client.dart';
import 'fcm_push.dart';
import 'local_ai.dart';
import 'sponsored_stories.dart';
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

class HomePage extends StatefulWidget {
  const HomePage(
      {super.key, required this.onToggleTheme, required this.onToggleLang});
  final VoidCallback onToggleTheme;
  final VoidCallback onToggleLang;
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final _api = ApiClient();
  final _usage = DataUsageTracker();
  final _email = TextEditingController();
  final _displayName = TextEditingController();
  final _password = TextEditingController();
  final _draft = TextEditingController();
  final _searchCtrl = TextEditingController();
  Map<String, dynamic>? _user;
  List<dynamic> _chats = [];
  // Users by id, loaded with the chat list so a DM can show the other
  // person's name instead of the literal type ("dm").
  Map<String, Map<String, dynamic>> _usersById = {};
  List<dynamic> _messages = [];
  List<dynamic> _stories = [];
  Map<String, dynamic>? _activeChat;
  String? _error;
  bool _registerMode = false;
  int _tabIndex = 0; // 0 = chats, 1 = stories, 2 = settings
  String _typingUser = '';
  int _ttlSeconds = 0; // disappearing-message TTL (0 = off)
  DateTime _lastTypingAt = DateTime.fromMillisecondsSinceEpoch(0);
  WebSocketChannel? _ws;
  E2eeClient? _e2ee;
  CallController? _call;
  final RTCVideoRenderer _remoteRenderer = RTCVideoRenderer();
  IncomingCall? _incomingCall;
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
  // Sponsored stories (Stories ads): frequency-capped, demo inventory
  // only until real advertisers exist. Never faked, never auto-downloaded.
  final _sponsoredTracker = SponsoredCapTracker();
  bool _sponsoredAvailable = true;
  // v0.7 chat extras (on-device only): user-created folders, chat lock
  // with a PIN, and per-chat message drafts.
  ChatFolders? _folders;
  ChatLock? _chatLock;
  MessageDrafts? _drafts;
  List<String> _folderNames = [];
  String _folderFilter = ''; // '' = all chats
  // Chats unlocked with the PIN during this session (lock re-applies
  // when the app restarts).
  final Set<String> _unlockedThisSession = {};
  Timer? _draftSaveTimer;

  @override
  void initState() {
    super.initState();
    _remoteRenderer.initialize();
    _initPrefs();
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

  @override
  void dispose() {
    _ws?.sink.close();
    _call?.dispose();
    _remoteRenderer.dispose();
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
          _assignedUsername = _registerMode
              ? (out['username'] as String? ?? '')
              : '';
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

  Future<void> _finishLogin(Map<String, dynamic> u) async {
    setState(() => _user = u);
    // E2EE setup: generate/load the device identity and publish
    // public keys. Private keys never leave the device.
    _e2ee = E2eeClient(_api, _deviceId);
    _e2ee!.setOwnUserId(u['id'] as String);
    try {
      await _e2ee!.ensureSetup(u['id'] as String);
    } catch (e) {
      setState(() => _error = 'E2EE setup failed (chats still work '
          'unencrypted): $e');
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
    await _refreshChats();
    // Last-seen privacy: load the server setting for the Settings page.
    try {
      final privacy = await _api.getPrivacy();
      if (mounted) {
        setState(() => _lastSeenVisibility =
            privacy['last_seen_visibility'] as String? ?? 'everyone');
      }
    } catch (_) {}
    try {
      final stories = await _api.listStories();
      setState(() => _stories = stories);
    } catch (_) {}
    await _refreshSponsoredCap();
    _connectWs();
    // FCM push: register this device's token so the server can wake
    // the app when a message arrives while it is closed. Best effort:
    // a failure here never breaks login or chat.
    await FcmPush.init(
      api: _api,
      deviceId: _deviceId,
      onOpenChat: (chatId) => _openChatById(chatId),
      onForegroundMessage: (title, body, chatId) {
        if (!mounted) return;
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

  /// Re-check the sponsored-story frequency cap (daily + session).
  Future<void> _refreshSponsoredCap() async {
    final ok = await _sponsoredTracker.canShow();
    if (mounted) setState(() => _sponsoredAvailable = ok);
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
        final call =
            _call?.handleOffer(ev['chat_id'] as String, peerId, payload);
        if (call != null) setState(() => _incomingCall = call);
      } else if (type == 'call-answer' && ev['payload'] != null) {
        _call?.handleAnswer(ev['payload'] as Map<String, dynamic>);
      } else if (type == 'call-ice' && ev['payload'] != null) {
        _call?.handleIce(ev['payload'] as Map<String, dynamic>);
      } else if (type == 'call-end') {
        _call?.handleRemoteEnd();
        setState(() => _incomingCall = null);
      }
    });
  }

  Future<void> _refreshChats() async {
    try {
      final chats = await _api.listChats();
      setState(() => _chats = chats);
    } catch (e) {
      setState(() => _error = e.toString());
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
      final peerId =
          members.firstWhere((m) => m != me, orElse: () => '');
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
    final msgs = await _api.listMessages(chat['id'] as String);
    setState(() => _messages = msgs);
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
  }

  Future<void> _refreshMessages() async {
    if (_activeChat == null) return;
    try {
      final msgs = await _api.listMessages(_activeChat!['id'] as String);
      if (mounted) {
        setState(() => _messages = msgs);
        for (final m in msgs) {
          _maybeDecrypt(m as Map<String, dynamic>);
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
  Future<void> _createGroupOrChannel() async {
    final users = await _api.listUsers();
    if (!mounted) return;
    String type = 'group';
    final titleCtrl = TextEditingController();
    final selected = <String>{};
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('New group / channel'),
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
    if (ok != true || selected.isEmpty) return;
    try {
      final chat = await _api.createChat(type, selected.toList(),
          title: titleCtrl.text.trim());
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
      final chat = _chats.firstWhere(
          (c) => (c as Map)['id'] == chatId,
          orElse: () => null);
      if (chat != null) {
        await _openChat(chat as Map<String, dynamic>);
      }
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
      final activeInvite =
          activeInvites.isEmpty ? null : activeInvites.first as Map<String, dynamic>;
      final myRole = members
          .map((m) => m as Map<String, dynamic>)
          .firstWhere((m) => m['user_id'] == _user!['id'],
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

  /// Post a story: pick a photo, compress it (Data Saver), upload,
  /// create the story. No fake stories: failures are shown as errors.
  Future<void> _postStory() async {
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
              ListTile(
                leading: const Icon(Icons.delete_outline),
                title: const Text('Delete'),
                onTap: () async {
                  Navigator.pop(context);
                  try {
                    await _api.deleteMessage(m['id'] as String);
                    setState(() => _messages
                        .removeWhere((x) => (x as Map)['id'] == m['id']));
                  } catch (e) {
                    setState(() => _error = e.toString());
                  }
                },
              ),
            ],
          ],
        ),
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
                    leading: GradientAvatar(
                        label: title,
                        gradient: atmosphereGradient(c['id'] as String)),
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
                decoration: const InputDecoration(labelText: 'New PIN (4-8 digits)'),
              ),
              TextField(
                controller: second,
                keyboardType: TextInputType.number,
                obscureText: true,
                maxLength: 8,
                decoration: InputDecoration(
                    labelText: 'Repeat PIN', errorText: error),
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
                    decoration:
                        const InputDecoration(hintText: 'Ask how to use Safari…'),
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

  @override
  Widget build(BuildContext context) {
    final isFa = Localizations.localeOf(context).languageCode == 'fa';
    // Email verification step (v0.6): 6-digit code, 10-minute expiry,
    // resend after a 60-second cooldown. No token until verified.
    if (_user == null && _codeStep) {
      return Scaffold(
        body: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Container(
                padding: const EdgeInsets.fromLTRB(24, 64, 24, 36),
                decoration: const BoxDecoration(
                  gradient: VyroColors.brandGradient,
                  borderRadius: BorderRadius.only(
                    bottomLeft: Radius.circular(36),
                    bottomRight: Radius.circular(36),
                  ),
                ),
                child: const Column(
                  children: [
                    Icon(Icons.mark_email_read_outlined,
                        size: 56, color: Colors.white),
                    SizedBox(height: 12),
                    Text(
                      'Check your email',
                      style: TextStyle(
                        fontSize: 26,
                        fontWeight: FontWeight.w800,
                        color: Colors.white,
                      ),
                    ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'We sent a 6-digit verification code to $_codeEmail. '
                      'It expires in 10 minutes. Never share it with anyone.',
                      style: const TextStyle(fontSize: 14),
                    ),
                    if (_assignedUsername.isNotEmpty) ...[
                      const SizedBox(height: 12),
                      Text(
                        'Your username: @$_assignedUsername',
                        style: const TextStyle(
                            fontSize: 14, fontWeight: FontWeight.w600),
                      ),
                    ],
                    const SizedBox(height: 20),
                    TextField(
                      controller: _codeCtrl,
                      decoration: const InputDecoration(
                        labelText: '6-digit code',
                        prefixIcon: Icon(Icons.pin_outlined),
                        hintText: '••••••',
                      ),
                      keyboardType: TextInputType.number,
                      maxLength: 6,
                      textAlign: TextAlign.center,
                      style: const TextStyle(fontSize: 24, letterSpacing: 8),
                      onSubmitted: (_) => _verifyCode(),
                    ),
                    const SizedBox(height: 8),
                    FilledButton(
                      onPressed: _verifyCode,
                      child: const Text('Verify and continue'),
                    ),
                    const SizedBox(height: 8),
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
                    if (_error != null)
                      Text(_error!, style: const TextStyle(color: Colors.red)),
                  ],
                ),
              ),
            ],
          ),
        ),
      );
    }
    if (_user == null) {
      final dark = Theme.of(context).brightness == Brightness.dark;
      return Scaffold(
        body: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Brand hero: blue-to-purple gradient, logo, slogan.
              Container(
                padding: const EdgeInsets.fromLTRB(24, 64, 24, 36),
                decoration: BoxDecoration(
                  gradient: dark
                      ? VyroColors.brandGradientDark
                      : VyroColors.brandGradient,
                  borderRadius: const BorderRadius.only(
                    bottomLeft: Radius.circular(36),
                    bottomRight: Radius.circular(36),
                  ),
                ),
                child: Column(
                  children: [
                    Container(
                      width: 88,
                      height: 88,
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(24),
                        border: Border.all(
                            color: Colors.white.withOpacity(0.6), width: 2),
                        boxShadow: const [
                          BoxShadow(
                            color: Color(0x33000000),
                            blurRadius: 16,
                            offset: Offset(0, 6),
                          ),
                        ],
                      ),
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(22),
                        child: Image.asset(
                          'assets/safari-logo.png',
                          width: 88,
                          height: 88,
                          fit: BoxFit.cover,
                        ),
                      ),
                    ),
                    const SizedBox(height: 16),
                    const Text(
                      'Safari',
                      style: TextStyle(
                        fontSize: 34,
                        fontWeight: FontWeight.w900,
                        color: Colors.white,
                        letterSpacing: 4,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      isFa ? 'فراتر از مرزها وصل شو' : 'Connect Beyond Limits',
                      style: TextStyle(
                        fontSize: 15,
                        color: Colors.white.withOpacity(0.9),
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    // Official-download warning, styled as an info card.
                    Container(
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(
                        color: Colors.amber.shade50,
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: Colors.amber.shade200),
                      ),
                      child: Row(
                        children: [
                          Icon(Icons.verified_user_outlined,
                              color: Colors.amber.shade800),
                          const SizedBox(width: 10),
                          const Expanded(
                            child: Text(
                              'Download Safari only from the official Safari '
                              'website. APKs from other sites are not safe '
                              '— do not install them.',
                              style: TextStyle(fontSize: 13),
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 24),
                    TextField(
                        controller: _email,
                        decoration: const InputDecoration(
                            labelText: 'Email',
                            prefixIcon: Icon(Icons.mail_outline)),
                        keyboardType: TextInputType.emailAddress),
                    const SizedBox(height: 12),
                    if (_registerMode) ...[
                      TextField(
                          controller: _displayName,
                          decoration: const InputDecoration(
                              labelText: 'Your name',
                              prefixIcon: Icon(Icons.person_outline))),
                      const SizedBox(height: 6),
                      const Text(
                        'Your username is chosen automatically from your name.',
                        style: TextStyle(fontSize: 12, color: Colors.grey),
                      ),
                      const SizedBox(height: 12),
                    ],
                    TextField(
                        controller: _password,
                        decoration: const InputDecoration(
                            labelText: 'Password',
                            prefixIcon: Icon(Icons.lock_outline)),
                        obscureText: true),
                    const SizedBox(height: 20),
                    Container(
                      decoration: const BoxDecoration(
                        gradient: VyroColors.brandGradient,
                        borderRadius: BorderRadius.all(Radius.circular(16)),
                        boxShadow: [
                          BoxShadow(
                            color: Color(0x447C3AED),
                            blurRadius: 12,
                            offset: Offset(0, 5),
                          ),
                        ],
                      ),
                      child: FilledButton(
                        onPressed: _auth,
                        style: FilledButton.styleFrom(
                          backgroundColor: Colors.transparent,
                          shadowColor: Colors.transparent,
                          padding: const EdgeInsets.symmetric(vertical: 16),
                        ),
                        child:
                            Text(_registerMode ? 'Create account' : 'Log in'),
                      ),
                    ),
                    const SizedBox(height: 8),
                    TextButton(
                        onPressed: () =>
                            setState(() => _registerMode = !_registerMode),
                        child: Text(_registerMode
                            ? 'Already have an account? Log in'
                            : 'New to Safari? Create an account')),
                    TextButton(
                        onPressed: widget.onToggleLang,
                        child: const Text('فارسی / English')),
                    if (_error != null)
                      Text(_error!, style: const TextStyle(color: Colors.red)),
                  ],
                ),
              ),
            ],
          ),
        ),
      );
    }
    return Scaffold(
      appBar: vyroAppBar(
        context,
        title: _activeChat == null
            ? 'Safari'
            : _chatTitle(_activeChat!),
        actions: [
          IconButton(
            icon: const Text('🤖', style: TextStyle(fontSize: 22)),
            tooltip: 'Safari assistant (local, on-device)',
            onPressed: _showAssistant,
          ),
          if (_activeChat == null) ...[
            IconButton(
              icon: const Icon(Icons.search),
              tooltip: 'Search messages',
              onPressed: () => _showSearchSheet(),
            ),
          ],
          if (_activeChat != null) ...[
            if ((_activeChat!['type'] as String?) == 'dm') ...[
              IconButton(
                icon: const Icon(Icons.call),
                tooltip: 'Voice call (Opus 12 kbps — Data Saver)',
                onPressed: () => _startCall(video: false),
              ),
              IconButton(
                icon: const Icon(Icons.videocam_outlined),
                tooltip: 'Video call (starts low quality)',
                onPressed: () => _startCall(video: true),
              ),
              IconButton(
                icon: Icon(_e2eeChats.contains(_activeChat!['id'])
                    ? Icons.lock
                    : Icons.lock_open),
                tooltip: 'End-to-end encryption',
                onPressed: () {
                  final id = _activeChat!['id'] as String;
                  setState(() {
                    if (_e2eeChats.contains(id)) {
                      _e2eeChats.remove(id);
                    } else {
                      _e2eeChats.add(id);
                    }
                  });
                },
              ),
            ] else ...[
              IconButton(
                icon: const Icon(Icons.group_outlined),
                tooltip: 'Members & admins',
                onPressed: _showMembers,
              ),
              IconButton(
                icon: Icon(_e2eeChats.contains(_activeChat!['id'])
                    ? Icons.lock
                    : Icons.lock_open),
                tooltip: 'End-to-end encryption (group)',
                onPressed: () {
                  final id = _activeChat!['id'] as String;
                  setState(() {
                    if (_e2eeChats.contains(id)) {
                      _e2eeChats.remove(id);
                    } else {
                      _e2eeChats.add(id);
                    }
                  });
                },
              ),
            ],
            IconButton(
              icon: Icon(_activeChat!['pinned'] == true
                  ? Icons.push_pin
                  : Icons.push_pin_outlined),
              tooltip: 'Pin chat',
              onPressed: () async {
                final c = _activeChat!['pinned'] == true
                    ? await _api.unpinChat(_activeChat!['id'] as String)
                    : await _api.pinChat(_activeChat!['id'] as String);
                setState(() => _activeChat = c);
                await _refreshChats();
              },
            ),
            IconButton(
              icon: const Icon(Icons.archive_outlined),
              tooltip: 'Archive chat',
              onPressed: () async {
                await _api.archiveChat(_activeChat!['id'] as String);
                setState(() => _activeChat = null);
                await _refreshChats();
              },
            ),
            IconButton(
              icon: const Icon(Icons.schedule_outlined),
              tooltip: 'Scheduled messages',
              onPressed: _showScheduled,
            ),
          ],
          IconButton(
              icon: const Icon(Icons.brightness_6),
              onPressed: widget.onToggleTheme),
          IconButton(
              icon: const Icon(Icons.language), onPressed: widget.onToggleLang),
        ],
      ),
      body: _activeChat == null ? _buildHomeBody() : _buildChatBody(),
      bottomNavigationBar: _activeChat == null
          ? NavigationBar(
              selectedIndex: _tabIndex,
              onDestinationSelected: (i) => setState(() => _tabIndex = i),
              destinations: const [
                NavigationDestination(
                    icon: Icon(Icons.chat_outlined), label: 'Chats'),
                NavigationDestination(
                    icon: Icon(Icons.circle_outlined), label: 'Stories'),
                NavigationDestination(
                    icon: Icon(Icons.settings_outlined), label: 'Settings'),
              ],
            )
          : null,
      floatingActionButton: _activeChat == null
          ? FloatingActionButton(
              onPressed: _tabIndex == 1
                  ? _postStory
                  : _tabIndex == 2
                      ? null
                      : _showNewChatSheet,
              child: Icon(_tabIndex == 1 ? Icons.add_a_photo : Icons.add),
            )
          : FloatingActionButton.small(
              onPressed: () => setState(() => _activeChat = null),
              child: const Icon(Icons.arrow_back),
            ),
    );
  }

  Widget _buildHomeBody() {
    if (_tabIndex == 2) return _buildSettingsBody();
    if (_tabIndex == 1) {
      // Stories feed: user stories with clearly-labelled sponsored
      // stories interleaved (frequency-capped, demo inventory only).
      final feed = buildStoryFeed(
        _stories.length,
        demoSponsoredStories,
        sponsoredAvailable: _sponsoredAvailable,
      );
      return ListView.builder(
        itemCount: feed.length + 1, // +1: "Promote your business" entry
        itemBuilder: (context, i) {
          if (i == feed.length) {
            return ListTile(
              leading: const CircleAvatar(child: Icon(Icons.campaign_outlined)),
              title: const Text('Promote your business'),
              subtitle: const Text(
                  'Put your business in Stories as a sponsored story.'),
              trailing: const Icon(Icons.chevron_right),
              onTap: _showPromoteSheet,
            );
          }
          final item = feed[i];
          if (item is SponsoredStoryItem) {
            final s = item.story;
            return ListTile(
              leading: StoryRing(
                child: GradientAvatar(label: s.advertiserName, radius: 22),
              ),
              title: Text(s.title),
              subtitle: Text(s.advertiserName),
              trailing: Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: Colors.amber.shade100,
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(color: Colors.amber.shade700),
                ),
                child: Text(
                  s.isDemo ? 'Sponsored · Demo' : 'Sponsored',
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w700,
                    color: Colors.amber.shade900,
                  ),
                ),
              ),
              onTap: () => _viewSponsoredStory(s),
            );
          }
          final st =
              _stories[(item as UserStoryItem).index] as Map<String, dynamic>;
          final mediaUrl = st['media_url'] as String?;
          return ListTile(
            leading: StoryRing(
              child: GradientAvatar(
                label: (st['caption'] as String?)?.isNotEmpty == true
                    ? st['caption'] as String
                    : 'S',
                radius: 22,
              ),
            ),
            title: Text((st['caption'] as String?)?.isNotEmpty == true
                ? st['caption'] as String
                : 'Story'),
            subtitle: Text('Expires: ${st['expires_at']}'),
            trailing: mediaUrl == null
                ? null
                : IconButton(
                    icon: const Icon(Icons.visibility_outlined),
                    tooltip: 'View story',
                    onPressed: () => _viewStory(st),
                  ),
          );
        },
      );
    }
    // The world view: animated sky, greeting, and every chat as a
    // glassy "space" with its own atmosphere gradient.
    return Stack(
      children: [
        const WorldBackground(),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 6),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '🌍 Your World',
                    style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                          fontWeight: FontWeight.w800,
                          letterSpacing: 0.3,
                        ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${_user?['username'] ?? 'Welcome'} · ${_chats.length} chats · ${_stories.length} stories live',
                    style: TextStyle(
                      fontSize: 13,
                      color: Theme.of(context)
                          .colorScheme
                          .onSurface
                          .withOpacity(0.65),
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    'Every space has its own atmosphere.',
                    style: TextStyle(
                      fontSize: 12,
                      color: Theme.of(context)
                          .colorScheme
                          .onSurface
                          .withOpacity(0.45),
                      fontStyle: FontStyle.italic,
                    ),
                  ),
                ],
              ),
            ),
            // v0.7 user-created folders: filter chips above the list.
            _folderChips(),
            Expanded(
              child: ListView.builder(
                padding: const EdgeInsets.fromLTRB(14, 6, 14, 90),
                itemCount: _visibleChats.length,
                itemBuilder: (context, i) {
                  final c = _visibleChats[i] as Map<String, dynamic>;
                  final title = _chatTitle(c);
                  final badges = [
                    if (c['pinned'] == true) '📌',
                    if ((c['folder'] as String?)?.isNotEmpty == true)
                      '📁 ${c['folder']}',
                    if (c['archived'] == true) '· archived',
                    if (_chatLock?.isLocked(c['id'] as String) == true)
                      '🔒 locked',
                  ].join(' ');
                  final hasUnread = _unreadChatIds.contains(c['id'] as String);
                  final updated = (c['updated_at'] as String?) ?? '';
                  final timeLabel =
                      updated.length >= 16 ? updated.substring(11, 16) : '';
                  final type = c['type'] as String? ?? 'dm';
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 10),
                    child: GlassCard(
                      padding: const EdgeInsets.all(12),
                      child: InkWell(
                        onTap: () => _openChat(c),
                        onLongPress: () => _showChatActions(c),
                        child: Row(
                          children: [
                            GradientAvatar(
                              label: title,
                              gradient: atmosphereGradient(c['id'] as String),
                            ),
                            const SizedBox(width: 14),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Row(
                                    children: [
                                      Expanded(
                                        child: Text(
                                          title,
                                          style: const TextStyle(
                                            fontSize: 16,
                                            fontWeight: FontWeight.w600,
                                          ),
                                          maxLines: 1,
                                          overflow: TextOverflow.ellipsis,
                                        ),
                                      ),
                                      if (type != 'dm')
                                        Container(
                                          padding: const EdgeInsets.symmetric(
                                              horizontal: 7, vertical: 2),
                                          margin:
                                              const EdgeInsets.only(right: 6),
                                          decoration: BoxDecoration(
                                            gradient: atmosphereGradient(
                                                c['id'] as String),
                                            borderRadius:
                                                BorderRadius.circular(8),
                                          ),
                                          child: Text(
                                            type.toUpperCase(),
                                            style: const TextStyle(
                                              fontSize: 9,
                                              fontWeight: FontWeight.w800,
                                              color: Colors.white,
                                            ),
                                          ),
                                        ),
                                      if (timeLabel.isNotEmpty)
                                        Text(
                                          timeLabel,
                                          style: TextStyle(
                                            fontSize: 12,
                                            color: hasUnread
                                                ? VyroColors.purple
                                                : Colors.grey,
                                            fontWeight: hasUnread
                                                ? FontWeight.w700
                                                : FontWeight.normal,
                                          ),
                                        ),
                                    ],
                                  ),
                                  const SizedBox(height: 3),
                                  Row(
                                    children: [
                                      Expanded(
                                        child: Text(
                                          '$type $badges'.trim(),
                                          style: TextStyle(
                                            fontSize: 13,
                                            color: Colors.grey.shade600,
                                          ),
                                          maxLines: 1,
                                          overflow: TextOverflow.ellipsis,
                                        ),
                                      ),
                                      if (hasUnread)
                                        Container(
                                          width: 10,
                                          height: 10,
                                          decoration: const BoxDecoration(
                                            gradient: VyroColors.brandGradient,
                                            shape: BoxShape.circle,
                                          ),
                                        ),
                                    ],
                                  ),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ],
    );
  }

  /// v0.7: chats visible under the active folder filter.
  List<dynamic> get _visibleChats {
    if (_folderFilter.isEmpty) return _chats;
    return _chats
        .where((c) => (c as Map)['folder'] == _folderFilter)
        .toList();
  }

  /// v0.7: user-created folder filter chips. Long-press a folder chip
  /// to rename or delete it.
  Widget _folderChips() {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.fromLTRB(20, 2, 20, 8),
      child: Row(
        children: [
          ChoiceChip(
            label: const Text('All'),
            selected: _folderFilter.isEmpty,
            onSelected: (_) => setState(() => _folderFilter = ''),
          ),
          const SizedBox(width: 6),
          for (final f in _folderNames) ...[
            GestureDetector(
              onLongPress: () => _manageFolder(f),
              child: ChoiceChip(
                label: Text(f),
                selected: _folderFilter == f,
                onSelected: (_) => setState(
                    () => _folderFilter = _folderFilter == f ? '' : f),
              ),
            ),
            const SizedBox(width: 6),
          ],
          ActionChip(
            label: const Text('+ Folder'),
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
              leading: Icon(
                  locked ? Icons.lock_open_outlined : Icons.lock_outlined),
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
                trailing:
                    current == f ? const Icon(Icons.check) : null,
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

  Widget _buildSettingsBody() {
    String mb(double v) => '${v.toStringAsFixed(2)} MB';
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('Data Saver', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 4),
        const Text(
          'Safari uses far less data than Telegram and WhatsApp: '
          'voice calls use Opus at 12 kbps (~0.09 MB/min vs ~0.3-0.5 '
          'MB/min on WhatsApp), photos are compressed before upload, '
          'and nothing downloads automatically.',
          style: TextStyle(fontSize: 13),
        ),
        SwitchListTile(
          title: const Text('Data Saver (default ON)'),
          subtitle: const Text('Tap-to-download for photos, videos, files '
              'and voice messages; low-quality previews.'),
          value: _dataSaver,
          onChanged: _setDataSaver,
        ),
        const Divider(),
        Text('Data used this month (${_usage.month})',
            style: Theme.of(context).textTheme.titleMedium),
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
        TextButton(
          onPressed: () async {
            await _usage.reset();
            setState(() {});
          },
          child: const Text('Reset counters'),
        ),
        const Divider(),
        Text('Security', style: Theme.of(context).textTheme.titleMedium),
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
        const Divider(),
        Text('Privacy', style: Theme.of(context).textTheme.titleMedium),
        const Text(
          'Who can see when you were last online. "Nobody" hides your '
          'online status and last seen from everyone — just like '
          'Telegram and WhatsApp.',
          style: TextStyle(fontSize: 13),
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
          subtitle: const Text(
              'Nobody can see your online status or last seen.'),
          value: 'nobody',
          groupValue: _lastSeenVisibility,
          onChanged: (v) {
            if (v != null) _setLastSeenVisibility(v);
          },
        ),
        const Divider(),
        Text('Chat lock', style: Theme.of(context).textTheme.titleMedium),
        const Text(
          'Lock individual chats with a 4-8 digit PIN. A locked chat '
          'asks for the PIN before opening. The PIN is stored only as '
          'a hash on this device — locking hides content on this '
          'device, it is not encryption.',
          style: TextStyle(fontSize: 13),
        ),
        ListTile(
          leading: const Icon(Icons.lock_outlined),
          title: Text(_chatLock?.hasPin == true
              ? 'Change chat lock PIN'
              : 'Set chat lock PIN'),
          subtitle: Text(_chatLock?.hasPin == true
              ? '${_chatLock?.lockedChatIds().length ?? 0} chat(s) locked'
              : 'No PIN set — locked chats are off'),
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
        const Divider(),
        const ListTile(
          leading: Icon(Icons.info_outline),
          title: Text('Safari 0.6 (mobile)'),
          subtitle: Text('Go backend · Next.js web · Flutter mobile. '
              'Push notifications on mobile use FCM (Firebase).'),
        ),
      ],
    );
  }

  void _viewStory(Map<String, dynamic> st) {
    final mediaUrl = st['media_url'] as String?;
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
    showDialog(
      context: context,
      builder: (context) {
        final bytes = _downloaded[mediaUrl];
        return AlertDialog(
          title: Text((st['caption'] as String?) ?? 'Story'),
          content: bytes == null
              ? const Text('Downloading…')
              : Image.memory(bytes, height: 300),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Close')),
          ],
        );
      },
    );
  }

  /// Open a sponsored story: clearly labelled, counts against the
  /// frequency cap, and costs zero bytes (demo entries carry no remote
  /// media — the card is a local gradient, never auto-downloaded).
  Future<void> _viewSponsoredStory(SponsoredStory s) async {
    await _sponsoredTracker.recordView();
    await _refreshSponsoredCap();
    if (!mounted) return;
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(s.advertiserName),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              height: 120,
              width: double.infinity,
              decoration: BoxDecoration(
                gradient: VyroColors.brandGradient,
                borderRadius: BorderRadius.circular(14),
              ),
              alignment: Alignment.center,
              child: Text(
                s.advertiserName.isNotEmpty ? s.advertiserName[0] : 'V',
                style: const TextStyle(
                  fontSize: 48,
                  fontWeight: FontWeight.w800,
                  color: Colors.white,
                ),
              ),
            ),
            const SizedBox(height: 12),
            Text(s.title),
            const SizedBox(height: 8),
            Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: Colors.amber.shade100,
                borderRadius: BorderRadius.circular(10),
                border: Border.all(color: Colors.amber.shade700),
              ),
              child: Text(
                s.isDemo
                    ? 'Sponsored · Demo ad — no real advertiser'
                    : 'Sponsored',
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                  color: Colors.amber.shade900,
                ),
              ),
            ),
            if (s.linkUrl != null) ...[
              const SizedBox(height: 8),
              Text(s.linkUrl!, style: const TextStyle(fontSize: 12)),
            ],
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close'),
          ),
          if (s.linkUrl != null)
            FilledButton(
              onPressed: () async {
                await Clipboard.setData(ClipboardData(text: s.linkUrl!));
                if (!context.mounted) return;
                Navigator.pop(context);
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(
                      content: Text('Advertiser link copied to clipboard')),
                );
              },
              child: Text(s.callToAction),
            ),
        ],
      ),
    );
  }

  /// Simple way for a business to submit a sponsored story.
  /// Honest by design: the ad service is not live yet, so the
  /// submission is stored on this device only and is NOT sent anywhere.
  /// The sheet says so in plain language, and lists past submissions.
  Future<void> _showPromoteSheet() async {
    final nameCtrl = TextEditingController();
    final emailCtrl = TextEditingController();
    final linkCtrl = TextEditingController();
    final titleCtrl = TextEditingController();
    final store = SponsoredSubmissionsStore();
    final previous = await store.list();
    if (!mounted) return;
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
          left: 16,
          right: 16,
          top: 16,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Promote your business',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800)),
              const SizedBox(height: 8),
              const Text(
                'Sponsored stories appear between user stories, clearly '
                'labelled "Sponsored", at most one per 5 stories and at '
                'most 5 views per user per day.\n\nThe ad service is not '
                'live yet: your request is saved on this device only and '
                'is not sent anywhere. When the service launches, saved '
                'requests will be the first ones reviewed.',
              ),
              const SizedBox(height: 12),
              TextField(
                controller: nameCtrl,
                decoration: const InputDecoration(
                    labelText: 'Business name', hintText: 'e.g. Acme Bakery'),
              ),
              TextField(
                controller: emailCtrl,
                decoration: const InputDecoration(
                    labelText: 'Contact email',
                    hintText: 'you@business.example'),
                keyboardType: TextInputType.emailAddress,
              ),
              TextField(
                controller: linkCtrl,
                decoration: const InputDecoration(
                    labelText: 'Website link',
                    hintText: 'https://business.example'),
                keyboardType: TextInputType.url,
              ),
              TextField(
                controller: titleCtrl,
                decoration: const InputDecoration(
                    labelText: 'Story headline',
                    hintText: 'e.g. Fresh bread every morning'),
              ),
              const SizedBox(height: 12),
              FilledButton(
                onPressed: () async {
                  if (nameCtrl.text.trim().isEmpty ||
                      linkCtrl.text.trim().isEmpty ||
                      titleCtrl.text.trim().isEmpty) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                          content: Text(
                              'Business name, website link and headline are required.')),
                    );
                    return;
                  }
                  await store.save(SponsoredSubmission(
                    advertiserName: nameCtrl.text.trim(),
                    contactEmail: emailCtrl.text.trim(),
                    linkUrl: linkCtrl.text.trim(),
                    title: titleCtrl.text.trim(),
                    submittedAt: DateTime.now(),
                  ));
                  if (!context.mounted) return;
                  Navigator.pop(context);
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(
                        content: Text('Saved on this device. '
                            'Nothing was sent — the ad service is not live yet.')),
                  );
                },
                child: const Text('Save my sponsored-story request'),
              ),
              if (previous.isNotEmpty) ...[
                const SizedBox(height: 16),
                const Text('Your saved requests (this device only)',
                    style: TextStyle(fontWeight: FontWeight.w700)),
                ...previous.map((p) => ListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text(p.title),
                      subtitle: Text('${p.advertiserName} · ${p.linkUrl}'),
                    )),
              ],
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
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
                if (!mounted || users.isEmpty) return;
                final other = users.first as Map<String, dynamic>;
                final chat = await _api.createDm(other['id'] as String);
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
      builder: (context) => Padding(
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
                      hintText: 'Search messages…',
                      prefixIcon: Icon(Icons.search)),
                  onSubmitted: (_) => _search(),
                ),
              ),
              IconButton(icon: const Icon(Icons.search), onPressed: _search),
            ]),
            const SizedBox(height: 8),
            if (_searching) const CircularProgressIndicator(),
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
    );
  }

  Future<void> _startCall({required bool video}) async {
    if (_activeChat == null || _call == null) return;
    final peerId = await _peerUserId();
    if (peerId.isEmpty) {
      setState(() => _error = 'Cannot call: peer user not found');
      return;
    }
    try {
      await _call!.startCall(
          chatId: _activeChat!['id'] as String, peerId: peerId, video: video);
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

  Widget _buildChatBody() {
    final callState = _call?.state ?? CallState.idle;
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Container(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: dark
              ? const [Color(0xFF171226), Color(0xFF0F1115)]
              : const [Color(0xFFF1F4FF), Color(0xFFF6F7FB)],
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
        ),
      ),
      child: Column(
        children: [
          if (_incomingCall != null)
            Container(
              color: Colors.green.shade100,
              padding: const EdgeInsets.all(12),
              child: Row(children: [
                const Icon(Icons.call),
                const SizedBox(width: 8),
                const Expanded(child: Text('Incoming voice call…')),
                FilledButton(
                  onPressed: () async {
                    final call = _incomingCall!;
                    setState(() => _incomingCall = null);
                    await _call?.acceptCall(call);
                    setState(() {});
                  },
                  child: const Text('Accept'),
                ),
                const SizedBox(width: 8),
                OutlinedButton(
                  onPressed: () {
                    _call?.declineCall(_incomingCall!);
                    setState(() => _incomingCall = null);
                  },
                  child: const Text('Decline'),
                ),
              ]),
            ),
          if (callState == CallState.inCall || callState == CallState.outgoing)
            Container(
              color: Colors.blue.shade100,
              padding: const EdgeInsets.all(12),
              child: Row(children: [
                const Icon(Icons.call),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    callState == CallState.outgoing
                        ? 'Calling… (Opus 12 kbps)'
                        : 'In call — Opus 12 kbps · ~0.09 MB/min',
                  ),
                ),
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
                  onPressed: () async {
                    await _call?.endCall();
                    setState(() {});
                  },
                ),
              ]),
            ),
          if (_call?.video == true && _call?.remoteStream != null)
            SizedBox(
              height: 200,
              child: RTCVideoView(_remoteRenderer),
            ),
          Expanded(
            child: ListView.builder(
              itemCount: _messages.length,
              itemBuilder: (context, i) {
                final m = _messages[i] as Map<String, dynamic>;
                return _messageBubble(m);
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
              color: Colors.grey.shade200,
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
          Container(
            padding: const EdgeInsets.fromLTRB(10, 8, 10, 12),
            decoration: BoxDecoration(
              color: Theme.of(context).scaffoldBackgroundColor,
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.05),
                  blurRadius: 10,
                  offset: const Offset(0, -3),
                ),
              ],
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                IconButton(
                  icon: const Icon(Icons.add_circle_outline),
                  tooltip: 'Attach: photo, location, schedule, disappearing',
                  iconSize: 30,
                  color: VyroColors.purple,
                  onPressed: _showAttachSheet,
                ),
                Expanded(
                  child: Container(
                    decoration: BoxDecoration(
                      color:
                          Theme.of(context).colorScheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(26),
                    ),
                    child: Row(
                      children: [
                        Expanded(
                          child: TextField(
                            controller: _draft,
                            decoration: const InputDecoration(
                              hintText: 'Type a message…',
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
                            padding: const EdgeInsets.only(right: 8),
                            child: Chip(
                              label: Text(
                                  '⏱ ${_ttlSeconds >= 86400 ? '1d' : _ttlSeconds >= 3600 ? '1h' : '1m'}'),
                              visualDensity: VisualDensity.compact,
                              onDeleted: () => setState(() => _ttlSeconds = 0),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                VoiceRecordButton(
                  api: _api,
                  chatId: _activeChat!['id'] as String,
                  onSent: _refreshMessages,
                ),
                const SizedBox(width: 8),
                GradientIconButton(icon: Icons.send, onPressed: _send),
              ],
            ),
          ),
          if (_error != null)
            Text(_error!, style: const TextStyle(color: Colors.red)),
        ],
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

  Widget _messageBubble(Map<String, dynamic> m) {
    final mine = m['sender_id'] == _user!['id'];
    final kind = m['kind'] as String? ?? 'text';
    final status = m['status'] as String?;
    final ticks = !mine
        ? ''
        : status == 'read'
            ? ' ✓✓'
            : status == 'delivered'
                ? ' ✓✓'
                : ' ✓';
    final time = (m['created_at'] as String?) ?? '';
    final isE2ee = kind == 'e2ee';
    final content = isE2ee
        ? (_decrypted[m['id'] as String] ?? '🔒 Encrypted message')
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
              gradient: mine ? VyroColors.brandGradient : null,
              color: mine
                  ? null
                  : Theme.of(context).colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.only(
                topLeft: const Radius.circular(20),
                topRight: const Radius.circular(20),
                bottomLeft: Radius.circular(mine ? 20 : 6),
                bottomRight: Radius.circular(mine ? 6 : 20),
              ),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.06),
                  blurRadius: 6,
                  offset: const Offset(0, 2),
                ),
              ],
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 9),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (isE2ee)
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.lock,
                            size: 12,
                            color: mine ? Colors.white70 : Colors.grey),
                        const SizedBox(width: 2),
                        Text('E2EE',
                            style: TextStyle(
                                fontSize: 11,
                                color: mine ? Colors.white70 : Colors.grey)),
                      ],
                    ),
                  if (fwdLabel != null)
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.forward,
                            size: 12,
                            color: mine ? Colors.white70 : Colors.grey),
                        const SizedBox(width: 2),
                        Text(fwdLabel,
                            style: TextStyle(
                                fontSize: 11,
                                fontStyle: FontStyle.italic,
                                color: mine ? Colors.white70 : Colors.grey)),
                      ],
                    ),
                  if (replyText != null)
                    Container(
                      padding: const EdgeInsets.all(6),
                      margin: const EdgeInsets.only(bottom: 4),
                      decoration: BoxDecoration(
                        color: mine
                            ? Colors.white.withOpacity(0.18)
                            : Colors.black12,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Text(replyText,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                              fontSize: 12,
                              fontStyle: FontStyle.italic,
                              color: mine ? Colors.white : null)),
                    ),
                  DefaultTextStyle(
                    style: TextStyle(
                      fontSize: 15,
                      height: 1.35,
                      color: mine
                          ? Colors.white
                          : Theme.of(context).colorScheme.onSurface,
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
                          foregroundColor:
                              mine ? Colors.white70 : VyroColors.purple,
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
                          color: mine ? Colors.white : null,
                        ),
                      ),
                    ),
                  const SizedBox(height: 2),
                  Text(
                    '${time.length >= 19 ? time.substring(11, 16) : time}$ticks',
                    style: TextStyle(
                      fontSize: 11,
                      color: mine
                          ? Colors.white70
                          : status == 'read'
                              ? Colors.blue
                              : Colors.grey,
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
  Widget _messageContent(Map<String, dynamic> m, String kind, String content) {
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
    return Text(content);
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
