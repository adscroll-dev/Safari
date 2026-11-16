import 'dart:convert';
import 'dart:typed_data';
import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, TargetPlatform;
import 'package:http/http.dart' as http;

import 'data_saver.dart';

/// Minimal typed client for the VYRO Go backend.
/// Android emulator reaches the host at 10.0.2.2; everywhere else localhost.
/// (Uses defaultTargetPlatform instead of dart:io so the web build works.)
class ApiClient {
  ApiClient({String? baseUrl, this.usage})
      : baseUrl = baseUrl ?? _defaultBaseUrl();

  static String _defaultBaseUrl() {
    // Release builds can bake in the public API URL, e.g.:
    //   flutter build apk --dart-define=VYRO_API_URL=https://vyro-api-xxxx.onrender.com
    // Empty (dev default) keeps the emulator/localhost behaviour, so the
    // web build screenshotted against localhost:8080 is unaffected.
    const override = String.fromEnvironment('VYRO_API_URL', defaultValue: '');
    if (override.isNotEmpty) return override;
    if (defaultTargetPlatform == TargetPlatform.android) {
      return 'http://10.0.2.2:8080';
    }
    return 'http://localhost:8080';
  }

  final String baseUrl;
  String? token;

  /// Optional usage tracker (Data Saver meter). When set, every
  /// request's real byte counts are recorded per category.
  DataUsageTracker? usage;

  /// Data Saver is ON by default: the UI must not auto-download media.
  bool dataSaver = true;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (token != null) 'Authorization': 'Bearer $token',
      };

  Future<Map<String, dynamic>> _req(String method, String path,
      [Map<String, dynamic>? body]) async {
    final uri = Uri.parse('$baseUrl$path');
    final reqBytes = body == null ? 0 : utf8.encode(jsonEncode(body)).length;
    final res = method == 'GET'
        ? await http.get(uri, headers: _headers)
        : method == 'PUT'
            ? await http.put(uri,
                headers: _headers, body: jsonEncode(body ?? {}))
            : method == 'DELETE'
                ? await http.delete(uri,
                    headers: _headers, body: jsonEncode(body ?? {}))
                : await http.post(uri,
                    headers: _headers, body: jsonEncode(body ?? {}));
    usage?.add(UsageCategory.chat, reqBytes + res.bodyBytes.length);
    final data = jsonDecode(res.body) as Map<String, dynamic>;
    if (res.statusCode >= 400) {
      throw Exception(data['error'] ?? 'Request failed: ${res.statusCode}');
    }
    return data;
  }

  Future<List<dynamic>> _getList(String path) async {
    final res = await http.get(Uri.parse('$baseUrl$path'), headers: _headers);
    usage?.add(UsageCategory.chat, res.bodyBytes.length);
    if (res.statusCode >= 400) {
      throw Exception('Request failed: ${res.statusCode}');
    }
    return jsonDecode(res.body) as List<dynamic>;
  }

  /// Email verification (v0.6): register/login only send a 6-digit
  /// code; the token is issued by the verify step. register/login
  /// return the raw response: {"status":"code_sent", ...} or, on an
  /// older server, {"user":..., "token":...}.
  ///
  /// The username is chosen by the server from the display name
  /// (user direction 2026-11-14): the user never picks one. The
  /// register response carries the generated "username".
  Future<Map<String, dynamic>> register(
      String email, String displayName, String password) async {
    final out = await _req('POST', '/api/auth/register', {
      'email': email,
      'password': password,
      'display_name': displayName,
    });
    if (out['token'] != null) token = out['token'] as String;
    return out;
  }

  Future<Map<String, dynamic>> login(String email, String password) async {
    final out = await _req(
        'POST', '/api/auth/login', {'email': email, 'password': password});
    if (out['token'] != null) token = out['token'] as String;
    return out;
  }

  /// Verify a 6-digit code (register or login). Returns {user, token}.
  Future<Map<String, dynamic>> verifyRegister(String email, String code) async {
    final out = await _req(
        'POST', '/api/auth/verify-register', {'email': email, 'code': code});
    if (out['token'] != null) token = out['token'] as String;
    return out;
  }

  Future<Map<String, dynamic>> verifyLogin(String email, String code) async {
    final out = await _req(
        'POST', '/api/auth/verify-login', {'email': email, 'code': code});
    if (out['token'] != null) token = out['token'] as String;
    return out;
  }

  /// Resend the code (60s cooldown enforced server-side).
  Future<Map<String, dynamic>> resendCode(String email, String purpose) => _req(
      'POST', '/api/auth/resend-code', {'email': email, 'purpose': purpose});

  Future<List<dynamic>> listChats() => _getList('/api/chats');
  Future<List<dynamic>> listMessages(String chatId) =>
      _getList('/api/chats/$chatId/messages?limit=50');

  /// v0.4: [ttlSeconds] sends a disappearing message (60..604800).
  /// v0.7: [forwardedFrom]/[forwardedFromId] mark a forwarded message.
  Future<Map<String, dynamic>> sendMessage(String chatId, String content,
          {int? ttlSeconds,
          String? replyToId,
          String? forwardedFrom,
          String? forwardedFromId}) =>
      _req('POST', '/api/chats/$chatId/messages', {
        'content': content,
        if (ttlSeconds != null) 'ttl_seconds': ttlSeconds,
        if (replyToId != null) 'reply_to_id': replyToId,
        if (forwardedFrom != null) 'forwarded_from': forwardedFrom,
        if (forwardedFromId != null) 'forwarded_from_id': forwardedFromId,
      });

  /// v0.7: forward a message into another chat, keeping its original
  /// sender as attribution ("Forwarded from <name>"). Media keeps its
  /// kind + file fields, so nothing is re-uploaded. E2EE messages
  /// cannot be forwarded (the server rejects them: their ciphertext
  /// is only decryptable by the original chat's members).
  Future<Map<String, dynamic>> forwardMessage(
      String targetChatId, Map<String, dynamic> original) async {
    final kind = original['kind'] as String? ?? 'text';
    if (kind == 'e2ee') {
      throw Exception('Encrypted messages cannot be forwarded');
    }
    final fromName = (original['forwarded_from'] as String?)?.isNotEmpty == true
        ? original['forwarded_from'] as String
        : await _displayName(original['sender_id'] as String? ?? '');
    final fromId =
        (original['forwarded_from_id'] as String?)?.isNotEmpty == true
            ? original['forwarded_from_id'] as String
            : (original['sender_id'] as String? ?? '');
    return _req('POST', '/api/chats/$targetChatId/messages', {
      'content': original['content'] as String? ?? '',
      'kind': kind,
      if ((original['file_url'] as String?)?.isNotEmpty == true)
        'file_url': original['file_url'],
      if ((original['file_name'] as String?)?.isNotEmpty == true)
        'file_name': original['file_name'],
      if (original['file_size'] != null) 'file_size': original['file_size'],
      if ((original['mime_type'] as String?)?.isNotEmpty == true)
        'mime_type': original['mime_type'],
      // A forwarded location keeps its coordinates and label.
      if (original['lat'] != null) 'lat': original['lat'],
      if (original['lng'] != null) 'lng': original['lng'],
      if ((original['location_label'] as String?)?.isNotEmpty == true)
        'location_label': original['location_label'],
      'forwarded_from': fromName,
      'forwarded_from_id': fromId,
    });
  }

  /// Best-effort display name for a user id: username from the user
  /// list, falling back to the raw id. Used for forward attribution
  /// when the original message only carries a sender id.
  Future<String> _displayName(String userId) async {
    if (userId.isEmpty) return '';
    try {
      final users = await listUsers();
      for (final u in users) {
        final m = u as Map<String, dynamic>;
        if (m['id'] == userId) {
          final name = (m['display_name'] as String?)?.trim() ?? '';
          if (name.isNotEmpty) return name;
          return (m['username'] as String?) ?? userId;
        }
      }
    } catch (_) {}
    return userId;
  }

  /// v0.6 parity: create a group or channel (title + members).
  Future<Map<String, dynamic>> createChat(String type, List<String> memberIds,
          {String title = '', bool? public}) =>
      _req('POST', '/api/chats', {
        'type': type,
        'member_ids': memberIds,
        'title': title,
        if (public != null) 'public': public,
      });

  // Stage 9 — backup export (raw JSON) + storage report.
  Future<String> exportMyData() async {
    final uri = Uri.parse('$baseUrl/api/users/me/export');
    final res = await http.get(uri, headers: _headers);
    usage?.add(UsageCategory.chat, res.bodyBytes.length);
    if (res.statusCode >= 400) {
      final data = jsonDecode(res.body) as Map<String, dynamic>;
      throw Exception(data['error'] ?? 'Request failed: ${res.statusCode}');
    }
    return res.body;
  }

  Future<Map<String, dynamic>> storageStats() =>
      _req('GET', '/api/users/me/storage');

  // Stage 8 — change the account email (code to the new address).
  Future<Map<String, dynamic>> requestEmailChange(String newEmail) =>
      _req('POST', '/api/users/me/change-email', {'new_email': newEmail});
  Future<Map<String, dynamic>> verifyEmailChange(
          String newEmail, String code) =>
      _req('POST', '/api/users/me/change-email/verify',
          {'new_email': newEmail, 'code': code});

  // Stage 7 — Accounts: profile, block/report, linked devices.
  Future<Map<String, dynamic>> updateProfile(
          {String? displayName, String? bio, String? avatarUrl}) =>
      _req('PATCH', '/api/users/me', {
        if (displayName != null) 'display_name': displayName,
        if (bio != null) 'bio': bio,
        if (avatarUrl != null) 'avatar_url': avatarUrl,
      });
  Future<Map<String, dynamic>> reportUser(String userId, String reason) =>
      _req('POST', '/api/users/$userId/report', {'reason': reason});
  Future<void> blockUser(String userId) =>
      _req('POST', '/api/users/$userId/block', null);
  Future<void> unblockUser(String userId) =>
      _req('DELETE', '/api/users/$userId/block');
  Future<List<dynamic>> listBlocked() => _getList('/api/users/me/blocked');
  Future<List<dynamic>> listDevices() => _getList('/api/devices');
  Future<void> removeDevice(String deviceId) =>
      _req('DELETE', '/api/devices/$deviceId');

  // Stage 6 — Communities (WhatsApp parity).
  Future<List<dynamic>> listCommunities() => _getList('/api/communities');
  Future<Map<String, dynamic>> createCommunity(String name,
          {String description = ''}) =>
      _req('POST', '/api/communities',
          {'name': name, 'description': description});
  Future<Map<String, dynamic>> getCommunity(String id) =>
      _req('GET', '/api/communities/$id');
  Future<Map<String, dynamic>> updateCommunity(String id,
          {String? name, String? description}) =>
      _req('PATCH', '/api/communities/$id', {
        if (name != null) 'name': name,
        if (description != null) 'description': description,
      });
  Future<List<dynamic>> listCommunityMembers(String id) =>
      _getList('/api/communities/$id/members');
  Future<void> addCommunityMember(String id, String userId) =>
      _req('POST', '/api/communities/$id/members', {'user_id': userId});
  Future<void> removeCommunityMember(String id, String userId) =>
      _req('DELETE', '/api/communities/$id/members/$userId');
  Future<void> setCommunityMemberRole(String id, String userId, String role) =>
      _req('POST', '/api/communities/$id/members/$userId/role', {'role': role});
  Future<void> leaveCommunity(String id) =>
      _req('POST', '/api/communities/$id/leave', null);
  Future<Map<String, dynamic>> addSubgroup(String id,
          {String? chatId, String? title}) =>
      _req('POST', '/api/communities/$id/subgroups', {
        if (chatId != null) 'chat_id': chatId,
        if (title != null) 'title': title,
      });
  Future<void> removeSubgroup(String id, String chatId) =>
      _req('DELETE', '/api/communities/$id/subgroups/$chatId');

  // Stage 5 — Channels (WhatsApp parity).
  Future<List<dynamic>> searchChannels(String q) =>
      _getList('/api/channels/search?q=${Uri.encodeQueryComponent(q)}');
  Future<Map<String, dynamic>> followChannel(String chatId) =>
      _req('POST', '/api/chats/$chatId/follow', null);
  Future<void> unfollowChannel(String chatId) =>
      _req('POST', '/api/chats/$chatId/unfollow', null);
  Future<Map<String, dynamic>> updateChat(String chatId,
          {String? title, bool? public}) =>
      _req('PATCH', '/api/chats/$chatId', {
        if (title != null) 'title': title,
        if (public != null) 'public': public,
      });
  Future<Map<String, dynamic>> recordMessageView(String messageId) =>
      _req('POST', '/api/messages/$messageId/view', null);
  Future<Map<String, dynamic>> getMessageViews(String messageId) =>
      _req('GET', '/api/messages/$messageId/views');

  /// v0.6 parity: edit / delete a message. deleteMessage is
  /// delete-for-everyone (sender only); deleteMessageForMe hides a
  /// message only for the caller (any member).
  Future<Map<String, dynamic>> editMessage(String id, String content) =>
      _req('PUT', '/api/messages/$id', {'content': content});
  Future<Map<String, dynamic>> deleteMessage(String id) =>
      _req('DELETE', '/api/messages/$id');
  Future<Map<String, dynamic>> deleteMessageForMe(String id) =>
      _req('POST', '/api/messages/$id/delete-for-me');

  /// Stage 1: share a Safari user as a contact card. The server
  /// validates that the user exists; content is JSON {"user_id"}.
  Future<Map<String, dynamic>> sendContact(String chatId, String userId) =>
      _req('POST', '/api/chats/$chatId/messages', {
        'kind': 'contact',
        'content': jsonEncode({'user_id': userId})
      });

  /// v0.6 parity: reactions (POST {emoji} / DELETE ?emoji=).
  Future<Map<String, dynamic>> addReaction(String id, String emoji) =>
      _req('POST', '/api/messages/$id/reactions', {'emoji': emoji});
  Future<Map<String, dynamic>> removeReaction(String id, String emoji) => _req(
      'DELETE',
      '/api/messages/$id/reactions?emoji=${Uri.encodeComponent(emoji)}');

  /// v0.6 parity: message search.
  Future<List<dynamic>> searchMessages(String q) =>
      _getList('/api/messages/search?q=${Uri.encodeQueryComponent(q)}');

  /// v0.6 parity: group/channel members and admin roles.
  Future<List<dynamic>> listMembers(String chatId) =>
      _getList('/api/chats/$chatId/members');
  Future<Map<String, dynamic>> addMember(String chatId, String userId) =>
      _req('POST', '/api/chats/$chatId/members', {'user_id': userId});
  Future<Map<String, dynamic>> removeMember(String chatId, String userId) =>
      _req('DELETE', '/api/chats/$chatId/members/$userId');
  Future<Map<String, dynamic>> setMemberRole(
          String chatId, String userId, String role) =>
      _req('POST', '/api/chats/$chatId/members/$userId/role', {'role': role});

  /// Invite links for groups/channels (Telegram-style).
  Future<Map<String, dynamic>> createInvite(String chatId) =>
      _req('POST', '/api/chats/$chatId/invites');
  Future<List<dynamic>> listInvites(String chatId) =>
      _getList('/api/chats/$chatId/invites');
  Future<Map<String, dynamic>> revokeInvite(String chatId, String token) =>
      _req('DELETE', '/api/chats/$chatId/invites/$token');
  Future<Map<String, dynamic>> previewInvite(String token) =>
      _req('GET', '/api/invites/$token');
  Future<Map<String, dynamic>> joinViaInvite(String token) =>
      _req('POST', '/api/invites/$token/join');

  /// Pull the invite token out of a pasted link
  /// (https://…/join/<token>) or a bare token.
  static String extractInviteToken(String input) {
    final s = input.trim();
    final idx = s.lastIndexOf('/join/');
    if (idx >= 0) return s.substring(idx + 6).trim();
    return s;
  }

  /// v0.6 parity: stories (post + list scheduled + cancel scheduled).
  /// Stage 2: kinds photo/video/text/audio (text has no media_url).
  Future<Map<String, dynamic>> createStory(String mediaUrl,
          {String caption = '',
          String kind = 'photo',
          String textContent = '',
          String backgroundColor = ''}) =>
      _req('POST', '/api/stories', {
        'media_url': mediaUrl,
        'caption': caption,
        'kind': kind,
        'text_content': textContent,
        'background_color': backgroundColor,
      });
  Future<List<dynamic>> listStories() => _getList('/api/stories');

  /// Stage 2 (status gaps): delete, views, reactions, replies, privacy.
  Future<Map<String, dynamic>> deleteStory(String id) =>
      _req('DELETE', '/api/stories/$id');
  Future<Map<String, dynamic>> recordStoryView(String id) =>
      _req('POST', '/api/stories/$id/view');
  Future<List<dynamic>> listStoryViews(String id) =>
      _getList('/api/stories/$id/views');
  Future<List<dynamic>> listStoryReactions(String id) =>
      _getList('/api/stories/$id/reactions');
  Future<Map<String, dynamic>> reactToStory(String id, String emoji) =>
      _req('POST', '/api/stories/$id/reactions', {'emoji': emoji});
  Future<Map<String, dynamic>> deleteStoryReaction(String id) =>
      _req('DELETE', '/api/stories/$id/reactions');
  Future<List<dynamic>> listStoryReplies(String id) =>
      _getList('/api/stories/$id/replies');
  Future<Map<String, dynamic>> replyToStory(String id, String content) =>
      _req('POST', '/api/stories/$id/reply', {'content': content});
  Future<Map<String, dynamic>> getStatusPrivacy() =>
      _req('GET', '/api/users/me/status-privacy');
  Future<Map<String, dynamic>> setStatusPrivacy(String v) =>
      _req('PUT', '/api/users/me/status-privacy', {'status_privacy': v});

  /// Stage 3 (calls): record a finished/missed call (caller only)
  /// and list my call history, newest first.
  Future<Map<String, dynamic>> recordCall({
    required String chatId,
    String? calleeId,
    required String callType,
    required String status,
    required int durationSeconds,
  }) =>
      _req('POST', '/api/calls', {
        'chat_id': chatId,
        if (calleeId != null) 'callee_id': calleeId,
        'call_type': callType,
        'status': status,
        'duration_seconds': durationSeconds,
      });
  Future<List<dynamic>> listCalls() => _getList('/api/calls');

  // Stage 4 — Polls (WhatsApp parity).
  Future<Map<String, dynamic>> createPoll(String chatId, String question,
          List<String> options, bool allowMultiple) =>
      _req('POST', '/api/chats/$chatId/polls', {
        'question': question,
        'options': options,
        'allow_multiple': allowMultiple,
      });
  Future<List<dynamic>> listPolls(String chatId) =>
      _getList('/api/chats/$chatId/polls');
  Future<Map<String, dynamic>> getPoll(String pollId) =>
      _req('GET', '/api/polls/$pollId');
  Future<Map<String, dynamic>> votePoll(String pollId, List<int> indexes) =>
      _req('POST', '/api/polls/$pollId/vote', {'option_indexes': indexes});
  Future<void> deletePoll(String pollId) =>
      _req('DELETE', '/api/polls/$pollId', null);

  Future<List<dynamic>> listScheduled(String chatId) =>
      _getList('/api/chats/$chatId/scheduled');
  Future<Map<String, dynamic>> cancelScheduled(String id) =>
      _req('DELETE', '/api/scheduled/$id');

  /// v0.6 parity: send a media message (photo/video/file) pointing at
  /// an uploaded file. Data Saver: callers must compress first.
  Future<Map<String, dynamic>> sendMediaMessage(
          String chatId, Map<String, dynamic> up,
          {String content = '', String? kind}) =>
      _req('POST', '/api/chats/$chatId/messages', {
        'content': content,
        'kind': kind ?? up['kind'],
        'file_url': up['url'],
        'file_name': up['file_name'],
        'file_size': up['file_size'],
        'mime_type': up['mime_type'],
      });

  /// Last-seen privacy (Telegram/WhatsApp-style).
  Future<Map<String, dynamic>> getPresence(String userId) =>
      _req('GET', '/api/users/$userId/presence');
  Future<Map<String, dynamic>> getPrivacy() =>
      _req('GET', '/api/users/me/privacy');
  Future<Map<String, dynamic>> setPrivacy(String visibility) => _req(
      'PUT', '/api/users/me/privacy', {'last_seen_visibility': visibility});

  /// Stage 11 (Settings parity): one combined endpoint for every
  /// privacy setting — last seen, status, avatar, about, groups and
  /// read receipts. Partial update: only the fields present change.
  Future<Map<String, dynamic>> getPrivacySettings() =>
      _req('GET', '/api/users/me/privacy-settings');
  Future<Map<String, dynamic>> setPrivacySettings(Map<String, dynamic> patch) =>
      _req('PUT', '/api/users/me/privacy-settings', patch);

  /// Stage 11: delete my account and everything it owns. The
  /// server answers {"status": "deleted"}.
  Future<void> deleteAccount() => _req('DELETE', '/api/users/me');

  /// v0.6 E2EE key management (public keys only — private keys never
  /// leave the device).
  Future<Map<String, dynamic>> uploadIdentityKey(
          String deviceId,
          String identityKey,
          String signedPrekey,
          int signedPrekeyId,
          String signKey,
          String signedPrekeySig) =>
      _req('POST', '/api/keys/identity', {
        'device_id': deviceId,
        'identity_key': identityKey,
        'signed_prekey': signedPrekey,
        'signed_prekey_id': signedPrekeyId,
        'sign_key': signKey,
        'signed_prekey_sig': signedPrekeySig,
      });
  Future<Map<String, dynamic>> uploadPreKeys(
          String deviceId, List<Map<String, dynamic>> keys) =>
      _req('POST', '/api/keys/prekeys', {'device_id': deviceId, 'keys': keys});
  Future<Map<String, dynamic>> fetchKeyBundle(String userId) =>
      _req('GET', '/api/keys/$userId/bundle');
  Future<Map<String, dynamic>> fetchPreKey(String userId, String deviceId) =>
      _req('GET', '/api/keys/$userId/$deviceId/prekey');
  Future<Map<String, dynamic>> sendE2eeMessage(
          String chatId, String envelopeJson) =>
      _req('POST', '/api/chats/$chatId/messages',
          {'content': envelopeJson, 'kind': 'e2ee'});

  /// v0.4 bots: list my bots, create one (token shown once in the
  /// response — the server stores only its hash), delete, set webhook.
  Future<List<dynamic>> listBots() => _getList('/api/bots');
  Future<Map<String, dynamic>> createBot(String name, String username) =>
      _req('POST', '/api/bots', {'name': name, 'username': username});
  Future<Map<String, dynamic>> deleteBot(String id) =>
      _req('DELETE', '/api/bots/$id', null);
  Future<Map<String, dynamic>> setBotWebhook(String id, String url) =>
      _req('POST', '/api/bots/$id/webhook', {'webhook_url': url});

  /// One call per recipient device: the envelope is a 1:1 E2EE
  /// envelope carrying the sender key, encrypted under the 1:1
  /// session with that device.
  Future<Map<String, dynamic>> uploadSenderKey(String chatId,
          {required String senderDeviceId,
          required String keyId,
          required String forUserId,
          required String forDeviceId,
          required String ciphertext}) =>
      _req('POST', '/api/chats/$chatId/sender-keys', {
        'sender_device_id': senderDeviceId,
        'key_id': keyId,
        'for_user_id': forUserId,
        'for_device_id': forDeviceId,
        'ciphertext': ciphertext,
      });
  Future<List<dynamic>> listSenderKeys(String chatId, String deviceId) =>
      _getList('/api/chats/$chatId/sender-keys?device_id=$deviceId');

  /// v0.6 multi-device: register this device.
  Future<Map<String, dynamic>> registerDevice(
          String deviceId, String deviceName) =>
      _req('POST', '/api/devices', {
        'device_id': deviceId,
        'name': deviceName,
        'platform': 'android',
      });

  /// FCM push: register this device's Firebase token so the server can
  /// wake the app when a message arrives while it is closed. Best
  /// effort — the caller swallows failures (push must never break
  /// login). The server answers 503 when FCM is not configured.
  Future<Map<String, dynamic>> registerFcmToken(
          String token, String deviceId) =>
      _req('POST', '/api/push/fcm-token', {
        'token': token,
        'device_id': deviceId,
        'platform': 'android',
      });

  /// FCM push: unregister a token (e.g. on logout).
  Future<Map<String, dynamic>> unregisterFcmToken(String token) =>
      _req('DELETE', '/api/push/fcm-token', {'token': token});

  /// Download an uploaded file's bytes with auth (Data Saver:
  /// only called after the user taps Download, or when Data Saver
  /// is off). Returns the raw bytes; the caller records usage.
  Future<Uint8List> downloadFile(String fileUrl) async {
    final res =
        await http.get(Uri.parse('$baseUrl$fileUrl'), headers: _headers);
    if (res.statusCode >= 400) {
      throw Exception('Download failed: ${res.statusCode}');
    }
    usage?.add(UsageCategory.media, res.bodyBytes.length);
    return res.bodyBytes;
  }

  /// v0.4: mark a message read (blue ticks for the sender).
  Future<Map<String, dynamic>> markRead(String messageId) =>
      _req('POST', '/api/messages/$messageId/read');

  /// v0.4: send a location message (lat -90..90, lng -180..180).
  Future<Map<String, dynamic>> sendLocation(
          String chatId, double lat, double lng,
          {String label = ''}) =>
      _req('POST', '/api/chats/$chatId/messages', {
        'kind': 'location',
        'lat': lat,
        'lng': lng,
        'location_label': label,
      });

  /// v0.4: schedule a message for [sendAt] (ISO-8601).
  Future<Map<String, dynamic>> scheduleMessage(
          String chatId, String content, String sendAt) =>
      _req('POST', '/api/chats/$chatId/scheduled',
          {'content': content, 'send_at': sendAt});

  /// v0.4: pin / archive helpers.
  Future<Map<String, dynamic>> pinChat(String chatId) =>
      _req('POST', '/api/chats/$chatId/pin');
  Future<Map<String, dynamic>> unpinChat(String chatId) =>
      _req('POST', '/api/chats/$chatId/unpin');
  Future<Map<String, dynamic>> archiveChat(String chatId) =>
      _req('POST', '/api/chats/$chatId/archive');
  Future<Map<String, dynamic>> unarchiveChat(String chatId) =>
      _req('POST', '/api/chats/$chatId/unarchive');

  /// Stage 1: mute / unmute a chat (per-user; no push for it).
  Future<Map<String, dynamic>> muteChat(String chatId) =>
      _req('POST', '/api/chats/$chatId/mute');
  Future<Map<String, dynamic>> unmuteChat(String chatId) =>
      _req('POST', '/api/chats/$chatId/unmute');

  /// Stage 1: pin / unpin one message in a chat (pinned banner).
  Future<Map<String, dynamic>> pinMessage(String chatId, String messageId) =>
      _req('POST', '/api/chats/$chatId/pin-message', {'message_id': messageId});
  Future<Map<String, dynamic>> unpinMessage(String chatId) =>
      _req('DELETE', '/api/chats/$chatId/pin-message');
  Future<Map<String, dynamic>> setFolder(String chatId, String folder) =>
      _req('POST', '/api/chats/$chatId/folder', {'folder': folder});
  Future<Map<String, dynamic>> createDm(String otherUserId) =>
      _req('POST', '/api/chats', {
        'type': 'dm',
        'member_ids': [otherUserId]
      });
  Future<List<dynamic>> listUsers() => _getList('/api/users');

  /// v0.6: upload a file (e.g. a voice recording) via the real
  /// POST /api/uploads multipart endpoint. Returns the decoded body
  /// ({url, file_name, file_size, mime_type, kind}).
  Future<Map<String, dynamic>> uploadFile(
      Uint8List bytes, String fileName, String mimeType,
      {UsageCategory category = UsageCategory.media}) async {
    final req =
        http.MultipartRequest('POST', Uri.parse('$baseUrl/api/uploads'));
    if (token != null) req.headers['Authorization'] = 'Bearer $token';
    req.files
        .add(http.MultipartFile.fromBytes('file', bytes, filename: fileName));
    final res = await http.Response.fromStream(await req.send());
    final data = jsonDecode(res.body) as Map<String, dynamic>;
    if (res.statusCode >= 400) {
      throw Exception(data['error'] ?? 'Upload failed: ${res.statusCode}');
    }
    usage?.add(category, bytes.length + res.bodyBytes.length);
    return data;
  }

  /// v0.6: send a voice message (kind 'voice') pointing at an uploaded file.
  Future<Map<String, dynamic>> sendVoiceMessage(
          String chatId, String fileUrl, int fileSize, String fileName) =>
      _req('POST', '/api/chats/$chatId/messages', {
        'kind': 'voice',
        'content': '',
        'file_url': fileUrl,
        'file_size': fileSize,
        'file_name': fileName,
      });

  /// Auth header for players/loaders that fetch protected URLs
  /// (e.g. just_audio fetching /uploads/...).
  Map<String, String> get authHeaders => {
        if (token != null) 'Authorization': 'Bearer $token',
      };


  /// Stage 11: forget the token (after account deletion), so the
  /// app returns to the login screen and sends no more authed calls.
  void clearToken() => token = null;
}
