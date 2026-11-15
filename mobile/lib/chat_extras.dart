// Chat extras (v0.7): user-created chat folders, chat lock with a PIN,
// per-chat message drafts, forwarding labels, and voice playback speed.
//
// All persistence is on-device only (SharedPreferences) — nothing here
// makes a network request, so Data Saver is unaffected. Folder
// assignment itself reuses the existing POST /api/chats/{id}/folder
// endpoint (folder name max 32 chars, server-side).
import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// A forwarded message shows "Forwarded from <name>" above its body,
/// like Telegram/WhatsApp. Returns null for non-forwarded messages.
String? forwardedLabel(Map<String, dynamic> message) {
  final name = (message['forwarded_from'] as String?)?.trim() ?? '';
  if (name.isEmpty) return null;
  return 'Forwarded from $name';
}

/// Voice-message playback speeds, cycled 1x -> 1.5x -> 2x -> 1x.
/// Pure function so the cycle is unit-testable; the player itself
/// applies it with AudioPlayer.setSpeed (just_audio).
double nextPlaybackSpeed(double current) {
  if (current < 1.25) return 1.5;
  if (current < 1.75) return 2.0;
  return 1.0;
}

/// Formats a playback speed for the speed button (1x, 1.5x, 2x).
String formatPlaybackSpeed(double speed) {
  if (speed == 2.0) return '2x';
  if (speed == 1.5) return '1.5x';
  return '1x';
}

/// User-created chat folders (v0.7).
///
/// The folder *names* live on this device (SharedPreferences); which
/// chat is in which folder is stored server-side per chat via the
/// existing folder setting, so it survives reinstalls and follows the
/// user to other devices. Folder names follow the server limit:
/// 1..32 characters, unique (case-insensitive).
class ChatFolders {
  ChatFolders(this._prefs);
  final SharedPreferences _prefs;

  static const _key = 'vyro_chat_folders_v1';

  /// Validates a folder name. Returns an error string, or null when ok.
  static String? validateName(String name, List<String> existing) {
    final n = name.trim();
    if (n.isEmpty) return 'Folder name is required';
    if (n.length > 32) return 'Folder name too long (max 32 characters)';
    if (existing.any((e) => e.toLowerCase() == n.toLowerCase())) {
      return 'A folder with this name already exists';
    }
    return null;
  }

  List<String> list() {
    final raw = _prefs.getString(_key);
    if (raw == null || raw.isEmpty) return [];
    try {
      final decoded = jsonDecode(raw) as List;
      return decoded.map((e) => e.toString()).toList();
    } catch (_) {
      return [];
    }
  }

  Future<void> _save(List<String> folders) =>
      _prefs.setString(_key, jsonEncode(folders));

  /// Adds a folder. Returns an error string, or null on success.
  Future<String?> add(String name) async {
    final folders = list();
    final err = validateName(name, folders);
    if (err != null) return err;
    folders.add(name.trim());
    await _save(folders);
    return null;
  }

  /// Renames a folder. Returns an error string, or null on success.
  Future<String?> rename(String oldName, String newName) async {
    final folders = list();
    if (!folders.contains(oldName)) return 'Folder not found';
    final others = folders.where((f) => f != oldName).toList();
    final err = validateName(newName, others);
    if (err != null) return err;
    folders[folders.indexOf(oldName)] = newName.trim();
    await _save(folders);
    return null;
  }

  /// Deletes a folder name from this device. Chats that were in it
  /// keep their server-side folder value until reassigned; callers
  /// should clear those with setFolder(chatId, '').
  Future<void> remove(String name) async {
    final folders = list()..remove(name);
    await _save(folders);
  }
}

/// Chat lock with a PIN (v0.7, on-device only).
///
/// The PIN itself is never stored: only its SHA-256 hash (via the
/// audited `cryptography` package) plus a per-device salt. Locked chat
/// ids are stored as a plain set — locking hides content, it is not
/// encryption; the server still holds the messages. A locked chat
/// shows a lock screen until the PIN is entered for the session.
class ChatLock {
  ChatLock(this._prefs);
  final SharedPreferences _prefs;

  static const _hashKey = 'vyro_chat_lock_pin_hash_v1';
  static const _saltKey = 'vyro_chat_lock_pin_salt_v1';
  static const _lockedKey = 'vyro_locked_chats_v1';

  /// SHA-256(salt + PIN). Pure and testable.
  static Future<String> hashPin(String pin, String salt) async {
    final hash = await Sha256().hash(utf8.encode('$salt:$pin'));
    return base64Encode(hash.bytes);
  }

  /// Validates a PIN: exactly 4-8 digits, like WhatsApp/Telegram locks.
  static String? validatePin(String pin) {
    if (pin.length < 4 || pin.length > 8) {
      return 'PIN must be 4-8 digits';
    }
    if (!RegExp(r'^\d+$').hasMatch(pin)) {
      return 'PIN must contain digits only';
    }
    return null;
  }

  bool get hasPin => _prefs.getString(_hashKey) != null;

  /// Sets (or replaces) the PIN. Returns an error string, or null.
  Future<String?> setPin(String pin) async {
    final err = validatePin(pin);
    if (err != null) return err;
    var salt = _prefs.getString(_saltKey);
    salt ??= DateTime.now().microsecondsSinceEpoch.toRadixString(36);
    final hash = await hashPin(pin, salt);
    await _prefs.setString(_saltKey, salt);
    await _prefs.setString(_hashKey, hash);
    return null;
  }

  /// Checks a PIN against the stored hash. False when no PIN is set.
  Future<bool> verifyPin(String pin) async {
    final hash = _prefs.getString(_hashKey);
    final salt = _prefs.getString(_saltKey);
    if (hash == null || salt == null) return false;
    return await hashPin(pin, salt) == hash;
  }

  /// Removes the PIN and unlocks every chat.
  Future<void> removePin() async {
    await _prefs.remove(_hashKey);
    await _prefs.remove(_saltKey);
    await _prefs.remove(_lockedKey);
  }

  Set<String> lockedChatIds() {
    return _prefs.getStringList(_lockedKey)?.toSet() ?? {};
  }

  bool isLocked(String chatId) => lockedChatIds().contains(chatId);

  Future<void> lockChat(String chatId) async {
    final ids = lockedChatIds()..add(chatId);
    await _prefs.setStringList(_lockedKey, ids.toList());
  }

  Future<void> unlockChat(String chatId) async {
    final ids = lockedChatIds()..remove(chatId);
    await _prefs.setStringList(_lockedKey, ids.toList());
  }
}

/// Whole-app lock (Stage 8, WhatsApp-style).
///
/// Same salted SHA-256 PIN scheme as ChatLock (4-8 digits), but with
/// separate keys, so the app PIN and the chat PIN are independent.
/// Fingerprint is a UI preference here; the UI drives local_auth
/// and only unlocks after the platform check succeeds.
class AppLock {
  AppLock(this._prefs);
  final SharedPreferences _prefs;

  static const _hashKey = 'vyro_app_lock_pin_hash_v1';
  static const _saltKey = 'vyro_app_lock_pin_salt_v1';
  static const _enabledKey = 'vyro_app_lock_enabled_v1';
  static const _biometricKey = 'vyro_app_lock_biometric_v1';

  bool get hasPin => _prefs.getString(_hashKey) != null;

  /// The lock is armed only when a PIN exists AND the user enabled it.
  bool get enabled => (_prefs.getBool(_enabledKey) ?? false) && hasPin;

  bool get biometric => _prefs.getBool(_biometricKey) ?? false;

  /// Sets (or replaces) the PIN and arms the lock. Returns an error
  /// string, or null.
  Future<String?> setPin(String pin) async {
    final err = ChatLock.validatePin(pin);
    if (err != null) return err;
    var salt = _prefs.getString(_saltKey);
    salt ??= DateTime.now().microsecondsSinceEpoch.toRadixString(36);
    final hash = await ChatLock.hashPin(pin, salt);
    await _prefs.setString(_saltKey, salt);
    await _prefs.setString(_hashKey, hash);
    await _prefs.setBool(_enabledKey, true);
    return null;
  }

  /// Checks a PIN against the stored hash. False when no PIN is set.
  Future<bool> verifyPin(String pin) async {
    final hash = _prefs.getString(_hashKey);
    final salt = _prefs.getString(_saltKey);
    if (hash == null || salt == null) return false;
    return await ChatLock.hashPin(pin, salt) == hash;
  }

  /// Removes the PIN and disarms the lock entirely.
  Future<void> removePin() async {
    await _prefs.remove(_hashKey);
    await _prefs.remove(_saltKey);
    await _prefs.remove(_enabledKey);
    await _prefs.remove(_biometricKey);
  }

  Future<void> setEnabled(bool on) async {
    await _prefs.setBool(_enabledKey, on);
  }

  Future<void> setBiometric(bool on) async {
    await _prefs.setBool(_biometricKey, on);
  }
}

/// Per-chat message drafts (v0.7, on-device only).
///
/// The unsent text of each chat is saved as the user types (debounced
/// by the caller) and restored when the chat is opened again — like
/// Telegram's drafts. Empty text clears the draft.
class MessageDrafts {
  MessageDrafts(this._prefs);
  final SharedPreferences _prefs;

  static const _key = 'vyro_message_drafts_v1';

  Map<String, String> _read() {
    final raw = _prefs.getString(_key);
    if (raw == null || raw.isEmpty) return {};
    try {
      final decoded = jsonDecode(raw) as Map<String, dynamic>;
      return decoded.map((k, v) => MapEntry(k, v.toString()));
    } catch (_) {
      return {};
    }
  }

  String? draftFor(String chatId) => _read()[chatId];

  Future<void> save(String chatId, String text) async {
    final drafts = _read();
    if (text.trim().isEmpty) {
      drafts.remove(chatId);
    } else {
      drafts[chatId] = text;
    }
    await _prefs.setString(_key, jsonEncode(drafts));
  }

  Future<void> clear(String chatId) async {
    final drafts = _read()..remove(chatId);
    await _prefs.setString(_key, jsonEncode(drafts));
  }
}
