// VYRO mobile E2EE + Data Saver tests.
//
// Proves the Dart client speaks the same VYRO E2EE v1 protocol as the Go
// backend and the web client, using the Go-produced vector in
// ../web/lib/e2ee-vector.json (Alice's first message to Bob). Also pins
// the Data Saver numbers the user asked for: voice calls far below
// WhatsApp's ~0.3-0.5 MB/min, and photos compressed before upload.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vyro/data_saver.dart';
import 'package:vyro/e2ee.dart';
import 'package:vyro/e2ee_key_storage.dart';

void main() {
  test('E2EE key storage migrates legacy preferences before deleting them',
      () async {
    SharedPreferences.setMockInitialValues({'identity': 'legacy-private-key'});
    final preferences = await SharedPreferences.getInstance();
    final secure = _FakeSecureStorage();
    final storage = E2eeKeyStorage(
      preferences: preferences,
      secureStorage: secure,
    );

    expect(await storage.read('identity'), 'legacy-private-key');
    expect(secure.values['identity'], 'legacy-private-key');
    expect(preferences.containsKey('identity'), isFalse);
  });

  test('E2EE key storage preserves legacy keys and warns on secure failure',
      () async {
    SharedPreferences.setMockInitialValues({'identity': 'legacy-private-key'});
    final preferences = await SharedPreferences.getInstance();
    final warnings = <String>[];
    final storage = E2eeKeyStorage(
      preferences: preferences,
      secureStorage: _FakeSecureStorage(fail: true),
      onWarning: warnings.add,
    );

    expect(await storage.read('identity'), 'legacy-private-key');
    expect(preferences.getString('identity'), 'legacy-private-key');
    expect(warnings, hasLength(1));
    expect(warnings.single, contains('falling back'));
  });

  test('Go -> Dart: decrypt the Go-produced vector envelope', () async {
    final vec =
        jsonDecode(await File('../web/lib/e2ee-vector.json').readAsString())
            as Map<String, dynamic>;
    final bob = Identity(
      dhPriv: unb64(vec['bob_dh_priv'] as String),
      dhPub: unb64(vec['bob_dh_pub'] as String),
      // Go Ed25519 private keys are seed (32) + pub (32); the seed is first.
      signSeed: unb64(vec['bob_sign_priv'] as String).sublist(0, 32),
      signPub: unb64(vec['bob_sign_pub'] as String),
    );
    final spk = SignedPreKey(
      id: 1,
      priv: unb64(vec['bob_spk_priv'] as String),
      pub: unb64(vec['bob_spk_pub'] as String),
      signature: Uint8List(64), // not used by receiveInitSession
    );
    final envMap = vec['envelope'] as Map<String, dynamic>;
    final opk = OneTimePreKey(
      id: envMap['opk'] as int,
      priv: unb64(vec['bob_opk_priv'] as String),
      pub: unb64(vec['bob_opk_pub'] as String),
    );
    final sess = await receiveInitSession(
      bob,
      spk,
      opk,
      unb64(envMap['ik'] as String),
      unb64(envMap['ek'] as String),
      'alice',
      'default',
    );
    expect(await sess.decrypt(Envelope.fromJson(envMap)), vec['plaintext']);
  });

  test('Dart 1:1 round-trip: Alice initiates, Bob receives, Bob replies',
      () async {
    final alice = await generateIdentity();
    final bob = await generateIdentity();
    final bobSpk = await generateSignedPreKey(1, bob);
    final bundle = KeyBundle(
      userId: 'bob',
      deviceId: 'default',
      dhPub: b64(bob.dhPub),
      signPub: b64(bob.signPub),
      spkPub: b64(bobSpk.pub),
      spkId: bobSpk.id,
      spkSig: b64(bobSpk.signature),
    );
    final aliceSess = await initiateSession(alice, bundle);
    final env = await aliceSess.encrypt('سلام VYRO — Dart round-trip');
    final bobSess = await receiveInitSession(
        bob, bobSpk, null, unb64(env.ik!), unb64(env.ek!), 'alice', 'default');
    expect(await bobSess.decrypt(env), 'سلام VYRO — Dart round-trip');
    final reply = await bobSess.encrypt('reply from Bob');
    expect(await aliceSess.decrypt(reply), 'reply from Bob');
  });

  test('Group Sender Keys round-trip', () async {
    final sender = await GroupSender.generate('alice', 'default');
    final receiver = GroupReceiver.fromDistribution(
        'alice', 'default', sender.distribution());
    final env = await sender.encrypt('سلام گروه');
    expect(await receiver.decrypt(env), 'سلام گروه');
  });

  test('Data Saver: voice calls ~0.09 MB/min, photos shrink', () {
    expect(DataSaver.voiceCallMbPerMin, closeTo(0.09, 0.001));
    // Far below WhatsApp voice calls (~0.3-0.5 MB/min).
    expect(DataSaver.voiceCallMbPerMin, lessThan(0.12));
    expect(DataSaver.videoCallMbPerMin, closeTo(1.125, 0.01));
    // A noisy 2000x1500 phone photo compresses well below its PNG size.
    final photo = img.Image(width: 2000, height: 1500);
    for (var y = 0; y < 1500; y += 7) {
      for (var x = 0; x < 2000; x += 7) {
        photo.setPixelRgb(x, y, (x * 255 ~/ 2000), (y * 255 ~/ 1500), 128);
      }
    }
    final png = Uint8List.fromList(img.encodePng(photo));
    final compressed = DataSaver.compressImage(png);
    expect(compressed.length, lessThan(png.length));
  });
}

class _FakeSecureStorage implements SecureValueStorage {
  _FakeSecureStorage({this.fail = false});

  final bool fail;
  final Map<String, String> values = {};

  @override
  Future<String?> read(String key) async {
    if (fail) throw StateError('secure storage unavailable');
    return values[key];
  }

  @override
  Future<void> write(String key, String value) async {
    if (fail) throw StateError('secure storage unavailable');
    values[key] = value;
  }
}
