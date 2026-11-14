// VYRO E2EE client for mobile (v0.6).
//
// Owns the device identity, signed pre-key, one-time pre-keys, 1:1
// Double Ratchet sessions and group Sender Keys, and talks to the
// backend key/sender-key endpoints. Private keys are stored in
// SharedPreferences as base64 — dev-grade, same as the web client's
// localStorage storage; a production client should use the OS
// keystore. Private keys never leave the device.
//
// Wire format is identical to web/lib/e2ee.ts and
// backend/internal/e2ee, so a mobile client and a web client can
// exchange encrypted DMs (proven by test/e2ee_test.dart against the
// Go-produced vector in web/lib/e2ee-vector.json).
import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'api.dart';
import 'e2ee.dart';

class E2eeClient {
  E2eeClient(this.api, this.deviceId);

  final ApiClient api;
  final String deviceId;

  Identity? identity;
  SignedPreKey? signedPreKey;
  List<OneTimePreKey> oneTimePreKeys = [];
  final Map<String, Session> sessions = {};
  final Map<String, GroupSender> groupSenders = {};
  final Map<String, GroupReceiver> groupReceivers = {};

  bool get isReady => identity != null;
  bool _setupDone = false;

  static const String _prefsPrefix = 'vyro_e2ee_v1';

  /// Generate (or load) the device identity and publish public keys.
  /// Idempotent: safe to call on every login.
  Future<void> ensureSetup(String userId) async {
    if (_setupDone) return;
    final prefs = await SharedPreferences.getInstance();
    final stored = prefs.getString('$_prefsPrefix:identity:$deviceId');
    if (stored != null) {
      try {
        identity =
            Identity.fromJson(jsonDecode(stored) as Map<String, dynamic>);
        final spkRaw = prefs.getString('$_prefsPrefix:spk:$deviceId');
        if (spkRaw != null) {
          signedPreKey =
              SignedPreKey.fromJson(jsonDecode(spkRaw) as Map<String, dynamic>);
        }
        final opkRaw = prefs.getString('$_prefsPrefix:opks:$deviceId');
        if (opkRaw != null) {
          oneTimePreKeys = (jsonDecode(opkRaw) as List)
              .map((e) => OneTimePreKey.fromJson(e as Map<String, dynamic>))
              .toList();
        }
        final sessionsRaw = prefs.getString('$_prefsPrefix:sessions:$deviceId');
        if (sessionsRaw != null) {
          for (final e
              in (jsonDecode(sessionsRaw) as Map<String, dynamic>).entries) {
            sessions[e.key] =
                Session.fromState(e.value as Map<String, dynamic>);
          }
        }
      } catch (_) {
        identity = null;
      }
    }
    if (identity == null) {
      identity = await generateIdentity();
      signedPreKey = await generateSignedPreKey(1, identity!);
      oneTimePreKeys = [
        for (var i = 0; i < 10; i++) await generateOneTimePreKey(100 + i),
      ];
      await _persist(prefs);
      // Publish ONLY public key material.
      await api.uploadIdentityKey(
        deviceId,
        b64(identity!.dhPub),
        b64(signedPreKey!.pub),
        signedPreKey!.id,
        b64(identity!.signPub),
        b64(signedPreKey!.signature),
      );
      await api.uploadPreKeys(deviceId, [
        for (final k in oneTimePreKeys)
          {'key_id': k.id, 'public_key': b64(k.pub)},
      ]);
    }
    try {
      await api.registerDevice(deviceId, 'VYRO mobile');
    } catch (_) {
      // Device registry is best-effort; E2EE works without it.
    }
    _setupDone = true;
  }

  Future<void> _persist(SharedPreferences prefs) async {
    if (identity == null) return;
    await prefs.setString(
        '$_prefsPrefix:identity:$deviceId', jsonEncode(identity!.toJson()));
    if (signedPreKey != null) {
      await prefs.setString(
          '$_prefsPrefix:spk:$deviceId', jsonEncode(signedPreKey!.toJson()));
    }
    await prefs.setString('$_prefsPrefix:opks:$deviceId',
        jsonEncode([for (final k in oneTimePreKeys) k.toJson()]));
  }

  Future<void> persistSessions() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(
        '$_prefsPrefix:sessions:$deviceId',
        jsonEncode(
            {for (final e in sessions.entries) e.key: e.value.toState()}));
  }

  // ---------------------------------------------------------- 1:1 DMs

  /// Encrypt [plaintext] for a DM peer. Returns the envelope JSON to
  /// send as a kind:'e2ee' message.
  Future<String> encryptDm(String peerUserId, String plaintext) async {
    final session = await _sessionFor(peerUserId);
    final env = await session.encrypt(plaintext);
    await persistSessions();
    return env.toJsonString();
  }

  /// Decrypt an incoming kind:'e2ee' message in a DM. The envelope
  /// carries the sender's identity/ephemeral keys on the first
  /// message, so a session can be started from the envelope itself.
  Future<String> decryptDm(String senderUserId, String envelopeJson) async {
    final env = Envelope.fromJsonString(envelopeJson);
    var session = sessions[senderUserId];
    if (session == null && env.ik != null && env.ek != null) {
      if (identity == null || signedPreKey == null) {
        throw Exception('E2EE not set up on this device');
      }
      OneTimePreKey? opk;
      if (env.opk != null && env.opk! > 0) {
        for (final k in oneTimePreKeys) {
          if (k.id == env.opk) opk = k;
        }
      }
      session = await receiveInitSession(identity!, signedPreKey!, opk,
          unb64(env.ik!), unb64(env.ek!), senderUserId, '');
      sessions[senderUserId] = session;
    }
    if (session == null) throw Exception('No E2EE session for this peer');
    final text = await session.decrypt(env);
    await persistSessions();
    return text;
  }

  Future<Session> _sessionFor(String peerUserId) async {
    final existing = sessions[peerUserId];
    if (existing != null) return existing;
    if (identity == null) throw Exception('E2EE not set up on this device');
    final bundleJson = await api.fetchKeyBundle(peerUserId);
    final session =
        await initiateSession(identity!, KeyBundle.fromJson(bundleJson));
    sessions[peerUserId] = session;
    return session;
  }

  // ---------------------------------------------------------- groups

  /// Encrypt a group message with this device's sender key for
  /// [chatId]. Distributes the sender key to every other member
  /// device first (encrypted under each 1:1 session) when [members]
  /// is given and the key is new or rotated.
  Future<String> encryptGroup(
      String chatId, String plaintext, List<String> memberUserIds) async {
    var sender = groupSenders[chatId];
    var isNew = false;
    if (sender == null) {
      sender = await GroupSender.generate('', deviceId);
      groupSenders[chatId] = sender;
      isNew = true;
    }
    if (isNew) {
      // Distribute the sender key to every other member device.
      // We only know member user ids here; the server stores one
      // envelope per (user, device) and members fetch by device id.
      // The default device id is "default" (see backend devices).
      for (final memberId in memberUserIds) {
        if (memberId == _ownUserId) continue;
        try {
          final session = await _sessionFor(memberId);
          final dist = sender.distribution();
          final env = await session.encrypt(jsonEncode(dist.toJson()));
          await api.uploadSenderKey(chatId,
              senderDeviceId: deviceId,
              keyId: sender.keyId,
              forUserId: memberId,
              forDeviceId: 'default',
              ciphertext: env.toJsonString());
        } catch (_) {
          // One unreachable member must not block the group message;
          // they fetch the distribution when they come online.
        }
      }
      await persistSessions();
    }
    final env = await sender.encrypt(plaintext);
    return env.toJsonString();
  }

  /// Decrypt an incoming group envelope. Fetches this device's
  /// sender-key distributions first when the sender key is unknown.
  Future<String> decryptGroup(String chatId, String envelopeJson) async {
    final env = GroupEnvelope.fromJsonString(envelopeJson);
    final key = '${env.senderUser}:${env.senderDevice}:${env.keyId}';
    var receiver = groupReceivers[key];
    if (receiver == null) {
      // Fetch distributions addressed to this device and decrypt one.
      final dists = await api.listSenderKeys(chatId, deviceId);
      for (final raw in dists) {
        try {
          final map = raw as Map<String, dynamic>;
          final distEnv = Envelope.fromJsonString(map['ciphertext'] as String);
          final senderId = map['sender_user_id'] as String? ?? '';
          final plain = await decryptDm(senderId, distEnv.toJsonString());
          final dist = GroupDistribution.fromJson(
              jsonDecode(plain) as Map<String, dynamic>);
          if (dist.keyId == env.keyId) {
            receiver = GroupReceiver.fromDistribution(
                env.senderUser, env.senderDevice, dist);
            groupReceivers[key] = receiver;
            break;
          }
        } catch (_) {
          // Not our distribution (or not decryptable yet); keep trying.
        }
      }
    }
    if (receiver == null) {
      throw Exception('No sender key for this group message yet');
    }
    return receiver.decrypt(env);
  }

  /// Rotate the group sender key (call when membership changes).
  Future<void> rotateGroupKey(String chatId) async {
    groupSenders.remove(chatId);
  }

  String _ownUserId = '';
  void setOwnUserId(String id) => _ownUserId = id;
}

/// Data usage measured for a voice call at the capped Opus bitrate
/// (12 kbps = 0.09 MB/min). The call controller records actual
/// duration x negotiated bitrate into the tracker.
double callMegabytes(int kbps, Duration duration) =>
    kbps * 1000 / 8 * duration.inSeconds / 1e6;
