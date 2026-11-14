// VYRO E2EE v1 — Dart client (mobile).
//
// A faithful mirror of web/lib/e2ee.ts (1:1) and backend/internal/e2ee
// (Go): X3DH + Double Ratchet over X25519 / HKDF-SHA256 / HMAC-SHA256 /
// AES-256-GCM, with Ed25519 signatures on the signed pre-key. Wire
// format, KDF labels ("vyro-ratchet-v1", "vyro-msg-v1",
// "vyro-x3dh-v1") and envelope JSON are identical to the Go and web
// sides, so envelopes produced here decrypt there and vice versa.
// Proven by test/e2ee_test.dart against the known vector in
// web/lib/e2ee-vector.json (Go-produced envelope decrypting to
// "سلام VYRO — E2EE test").
//
// Crypto comes only from the `cryptography` pub package (X25519,
// Ed25519, AES-256-GCM, HKDF, HMAC implementations) — no custom
// primitives and no hand-rolled crypto. Private keys never leave the
// device: the server only ever sees public keys and ciphertext.
// NOTE: this is the same construction as the web client, not libsignal,
// not Signal-interoperable, and not independently audited. Session and
// identity private keys are stored in SharedPreferences as base64 —
// dev-grade, same as the web client's localStorage storage; a
// production client should use the OS keystore.
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

const int protocolVersion = 1;
const int maxSkippedKeys = 1000;

final _x25519 = X25519();
final _ed25519 = Ed25519();
final _aesGcm = AesGcm.with256bits();
final _sha256 = Sha256();
final _hmacSha256 = Hmac(Sha256());

String b64(Uint8List b) => base64Encode(b);
Uint8List unb64(String s) => base64Decode(s);

Uint8List _concat(List<Uint8List> parts) {
  final out = Uint8List(parts.fold(0, (n, p) => n + p.length));
  var off = 0;
  for (final p in parts) {
    out.setAll(off, p);
    off += p.length;
  }
  return out;
}

bool _eq(Uint8List a, Uint8List b) {
  if (a.length != b.length) return false;
  var d = 0;
  for (var i = 0; i < a.length; i++) {
    d |= a[i] ^ b[i];
  }
  return d == 0;
}

/// Secure random bytes. (`Random` has no `nextBytes` — fill byte by byte.)
Uint8List _randomBytes(int n) {
  final r = Random.secure();
  return Uint8List.fromList(List.generate(n, (_) => r.nextInt(256)));
}

Future<Uint8List> _hmac(Uint8List key, Uint8List msg) async {
  final mac =
      await _hmacSha256.calculateMac(msg, secretKey: SecretKeyData(key));
  return Uint8List.fromList(mac.bytes);
}

/// HKDF-SHA256 (RFC 5869), mirroring Go's x/crypto/hkdf: empty salt =
/// HashLen zeros. Lengths used here are 32 or 64 bytes.
Future<Uint8List> _hkdf(
    Uint8List salt, Uint8List ikm, Uint8List info, int length) async {
  final saltEff = salt.isEmpty ? Uint8List(32) : salt;
  final prk = await _hmac(saltEff, ikm);
  final out = Uint8List(length);
  var t = Uint8List(0);
  var done = 0;
  var counter = 1;
  while (done < length) {
    t = await _hmac(
        prk,
        _concat([
          t,
          info,
          Uint8List.fromList([counter])
        ]));
    final take = t.length < length - done ? t.length : length - done;
    out.setRange(done, done + take, t);
    done += take;
    counter++;
  }
  return out;
}

Future<Uint8List> _dh(Uint8List priv, Uint8List pub, Uint8List ownPub) async {
  final kp = SimpleKeyPairData(priv,
      publicKey: SimplePublicKey(ownPub, type: KeyPairType.x25519),
      type: KeyPairType.x25519);
  final secret = await _x25519.sharedSecretKey(
      keyPair: kp,
      remotePublicKey: SimplePublicKey(pub, type: KeyPairType.x25519));
  return Uint8List.fromList(await secret.extractBytes());
}

Future<(Uint8List, Uint8List)> _x25519Keypair() async {
  final kp = await _x25519.newKeyPair();
  final pub = await kp.extractPublicKey();
  final priv = await kp.extractPrivateKeyBytes();
  return (Uint8List.fromList(priv), Uint8List.fromList(pub.bytes));
}

Future<Uint8List> _ed25519Sign(Uint8List seed, Uint8List msg) async {
  final kp = await _ed25519.newKeyPairFromSeed(seed);
  final sig = await _ed25519.sign(msg, keyPair: kp);
  return Uint8List.fromList(sig.bytes);
}

Future<bool> _ed25519Verify(Uint8List pub, Uint8List msg, Uint8List sig) async {
  final signature = Signature(sig,
      publicKey: SimplePublicKey(pub, type: KeyPairType.ed25519));
  return _ed25519.verify(msg, signature: signature);
}

Future<(Uint8List, Uint8List)> _kdfRk(Uint8List rk, Uint8List dhOut) async {
  final out = await _hkdf(
      rk, dhOut, Uint8List.fromList(utf8.encode('vyro-ratchet-v1')), 64);
  return (out.sublist(0, 32), out.sublist(32));
}

Future<(Uint8List, Uint8List)> _kdfCk(Uint8List ck) async => (
      await _hmac(ck, Uint8List.fromList([0x01])),
      await _hmac(ck, Uint8List.fromList([0x02])),
    );

Future<Uint8List> _msgKey(Uint8List mk) async =>
    _hkdf(mk, Uint8List(0), Uint8List.fromList(utf8.encode('vyro-msg-v1')), 32);

Future<Uint8List> _aeadEncrypt(
    Uint8List key, Uint8List nonce, Uint8List ad, Uint8List plaintext) async {
  final box = await _aesGcm.encrypt(plaintext,
      secretKey: SecretKeyData(key), nonce: nonce, aad: ad);
  return Uint8List.fromList([...box.cipherText, ...box.mac.bytes]);
}

Future<Uint8List> _aeadDecrypt(
    Uint8List key, Uint8List nonce, Uint8List ad, Uint8List ctWithTag) async {
  // cryptography's AesGcm expects the 16-byte tag separate from the
  // ciphertext (WebCrypto/Go append it); split it here.
  final ct = ctWithTag.sublist(0, ctWithTag.length - 16);
  final tag = ctWithTag.sublist(ctWithTag.length - 16);
  return Uint8List.fromList(await _aesGcm.decrypt(
      SecretBox(ct, nonce: nonce, mac: Mac(tag)),
      secretKey: SecretKeyData(key),
      aad: ad));
}

Uint8List _headerAd(String dhB64, int pn, int n) => Uint8List.fromList(
    utf8.encode(jsonEncode({'dh': dhB64, 'pn': pn, 'n': n})));

Future<String> fingerprint(Uint8List ikA, Uint8List ikB) async {
  final h = await _sha256.hash(_concat([ikA, ikB]));
  return h.bytes
      .sublist(0, 8)
      .map((b) => b.toRadixString(16).padLeft(2, '0'))
      .join();
}

// ------------------------------------------------------------ identities

class Identity {
  Identity(
      {required this.dhPriv,
      required this.dhPub,
      required this.signSeed,
      required this.signPub});

  Uint8List dhPriv; // 32 bytes X25519 private scalar
  Uint8List dhPub; // 32 bytes X25519 public
  Uint8List signSeed; // 32 bytes Ed25519 seed
  Uint8List signPub; // 32 bytes Ed25519 public

  Map<String, dynamic> toJson() => {
        'dh_priv': b64(dhPriv),
        'dh_pub': b64(dhPub),
        'sign_seed': b64(signSeed),
        'sign_pub': b64(signPub),
      };

  static Identity fromJson(Map<String, dynamic> j) => Identity(
        dhPriv: unb64(j['dh_priv'] as String),
        dhPub: unb64(j['dh_pub'] as String),
        signSeed: unb64(j['sign_seed'] as String),
        signPub: unb64(j['sign_pub'] as String),
      );
}

class SignedPreKey {
  SignedPreKey(
      {required this.id,
      required this.priv,
      required this.pub,
      required this.signature});

  int id;
  Uint8List priv;
  Uint8List pub;
  Uint8List signature; // Ed25519 signature over pub

  Map<String, dynamic> toJson() => {
        'id': id,
        'priv': b64(priv),
        'pub': b64(pub),
        'sig': b64(signature),
      };

  static SignedPreKey fromJson(Map<String, dynamic> j) => SignedPreKey(
        id: j['id'] as int,
        priv: unb64(j['priv'] as String),
        pub: unb64(j['pub'] as String),
        signature: unb64(j['sig'] as String),
      );
}

class OneTimePreKey {
  OneTimePreKey({required this.id, required this.priv, required this.pub});

  int id;
  Uint8List priv;
  Uint8List pub;

  Map<String, dynamic> toJson() => {
        'id': id,
        'priv': b64(priv),
        'pub': b64(pub),
      };

  static OneTimePreKey fromJson(Map<String, dynamic> j) => OneTimePreKey(
        id: j['id'] as int,
        priv: unb64(j['priv'] as String),
        pub: unb64(j['pub'] as String),
      );
}

class KeyBundle {
  KeyBundle(
      {required this.userId,
      required this.deviceId,
      required this.dhPub,
      required this.signPub,
      required this.spkPub,
      required this.spkId,
      required this.spkSig,
      this.oneTimePrekey});

  String userId;
  String deviceId;
  String dhPub;
  String signPub;
  String spkPub;
  int spkId;
  String spkSig;
  Map<String, dynamic>? oneTimePrekey; // {'id': int, 'pub': String}

  static KeyBundle fromJson(Map<String, dynamic> j) => KeyBundle(
        userId: j['user_id'] as String,
        deviceId: j['device_id'] as String,
        dhPub: j['dh_pub'] as String,
        signPub: j['sign_pub'] as String,
        spkPub: j['spk_pub'] as String,
        spkId: j['spk_id'] as int,
        spkSig: j['spk_sig'] as String,
        oneTimePrekey: j['one_time_prekey'] as Map<String, dynamic>?,
      );
}

Future<Identity> generateIdentity() async {
  final (dhPriv, dhPub) = await _x25519Keypair();
  final signKp = await _ed25519.newKeyPair();
  final signPub = await signKp.extractPublicKey();
  final signSeed = await signKp.extractPrivateKeyBytes();
  return Identity(
      dhPriv: dhPriv,
      dhPub: dhPub,
      signSeed: Uint8List.fromList(signSeed),
      signPub: Uint8List.fromList(signPub.bytes));
}

Future<SignedPreKey> generateSignedPreKey(int id, Identity ident) async {
  final (priv, pub) = await _x25519Keypair();
  final sig = await _ed25519Sign(ident.signSeed, pub);
  return SignedPreKey(id: id, priv: priv, pub: pub, signature: sig);
}

Future<OneTimePreKey> generateOneTimePreKey(int id) async {
  final (priv, pub) = await _x25519Keypair();
  return OneTimePreKey(id: id, priv: priv, pub: pub);
}

Future<bool> verifyBundle(KeyBundle bundle) async {
  try {
    final signPub = unb64(bundle.signPub);
    final spk = unb64(bundle.spkPub);
    final sig = unb64(bundle.spkSig);
    if (signPub.length != 32 || spk.length != 32 || sig.length != 64) {
      return false;
    }
    return _ed25519Verify(signPub, spk, sig);
  } catch (_) {
    return false;
  }
}

// ------------------------------------------------------------ envelopes

class Envelope {
  Envelope(
      {required this.dh,
      required this.pn,
      required this.n,
      required this.nonce,
      required this.ct,
      this.ik,
      this.ek,
      this.opk});

  String dh;
  int pn;
  int n;
  String nonce;
  String ct;
  String? ik;
  String? ek;
  int? opk;

  Map<String, dynamic> toJson() => {
        'v': protocolVersion,
        if (ik != null) 'ik': ik,
        if (ek != null) 'ek': ek,
        if (opk != null) 'opk': opk,
        'dh': dh,
        'pn': pn,
        'n': n,
        'nonce': nonce,
        'ct': ct,
      };

  static Envelope fromJson(Map<String, dynamic> j) => Envelope(
        dh: j['dh'] as String,
        pn: j['pn'] as int,
        n: j['n'] as int,
        nonce: j['nonce'] as String,
        ct: j['ct'] as String,
        ik: j['ik'] as String?,
        ek: j['ek'] as String?,
        opk: j['opk'] as int?,
      );

  String toJsonString() => jsonEncode(toJson());
  static Envelope fromJsonString(String s) =>
      Envelope.fromJson(jsonDecode(s) as Map<String, dynamic>);
}

// ------------------------------------------------------------ sessions

/// Live Double Ratchet state. Same JSON shape as the web client's
/// SessionState (field names included) so a session is readable on
/// both sides. Private scalars are base64; never send to the server.
class Session {
  Session(
      {required this.peerUserId,
      required this.peerDevice,
      required this.dhsPriv,
      required this.dhsPub,
      required this.dhrPub,
      required this.rk,
      this.cks,
      this.ckr,
      this.ns = 0,
      this.nr = 0,
      this.pn = 0,
      Map<String, Uint8List>? skipped,
      this.first = false,
      this.ekPub,
      this.opkId = 0,
      this.ownIkPub,
      this.spkPriv})
      : skipped = skipped ?? {};

  String peerUserId;
  String peerDevice;
  Uint8List dhsPriv;
  Uint8List dhsPub;
  Uint8List dhrPub;
  Uint8List rk;
  Uint8List? cks;
  Uint8List? ckr;
  int ns;
  int nr;
  int pn;
  Map<String, Uint8List> skipped;
  bool first;
  Uint8List? ekPub;
  int opkId;
  Uint8List? ownIkPub;
  Uint8List? spkPriv;

  Map<String, dynamic> toState() {
    final st = <String, dynamic>{
      'peer_user_id': peerUserId,
      'peer_device': peerDevice,
      'dhs_priv': b64(dhsPriv),
      'dhs_pub': b64(dhsPub),
      'dhr_pub': b64(dhrPub),
      'rk': b64(rk),
      'ns': ns,
      'nr': nr,
      'pn': pn,
      'first': first,
      'opk_id': opkId,
    };
    if (cks != null) st['cks'] = b64(cks!);
    if (ckr != null) st['ckr'] = b64(ckr!);
    if (ekPub != null) st['ek_pub'] = b64(ekPub!);
    if (ownIkPub != null) st['own_ik_pub'] = b64(ownIkPub!);
    if (spkPriv != null) st['spk_priv'] = b64(spkPriv!);
    if (skipped.isNotEmpty) {
      st['skipped'] = {for (final e in skipped.entries) e.key: b64(e.value)};
    }
    return st;
  }

  static Session fromState(Map<String, dynamic> st) {
    final s = Session(
      peerUserId: st['peer_user_id'] as String,
      peerDevice: st['peer_device'] as String,
      dhsPriv: unb64(st['dhs_priv'] as String),
      dhsPub: unb64(st['dhs_pub'] as String),
      dhrPub: unb64(st['dhr_pub'] as String),
      rk: unb64(st['rk'] as String),
      ns: st['ns'] as int,
      nr: st['nr'] as int,
      pn: st['pn'] as int,
      first: st['first'] as bool? ?? false,
      opkId: st['opk_id'] as int? ?? 0,
    );
    if (st['cks'] != null) s.cks = unb64(st['cks'] as String);
    if (st['ckr'] != null) s.ckr = unb64(st['ckr'] as String);
    if (st['ek_pub'] != null) s.ekPub = unb64(st['ek_pub'] as String);
    if (st['own_ik_pub'] != null) {
      s.ownIkPub = unb64(st['own_ik_pub'] as String);
    }
    if (st['spk_priv'] != null) s.spkPriv = unb64(st['spk_priv'] as String);
    if (st['skipped'] != null) {
      for (final e in (st['skipped'] as Map<String, dynamic>).entries) {
        s.skipped[e.key] = unb64(e.value as String);
      }
    }
    return s;
  }

  String _skippedKey(Uint8List dh, int n) => '${b64(dh)}:$n';

  Future<void> _skipTo(int n) async {
    while (nr < n) {
      if (ckr == null) throw Exception('no receiving chain');
      final (next, mk) = await _kdfCk(ckr!);
      if (skipped.length >= maxSkippedKeys) {
        throw Exception('too many skipped message keys');
      }
      skipped[_skippedKey(dhrPub, nr)] = mk;
      ckr = next;
      nr++;
    }
  }

  Future<String> _decryptWithKey(Uint8List mk, Envelope env) async {
    final key = await _msgKey(mk);
    final pt = await _aeadDecrypt(
        key, unb64(env.nonce), _headerAd(env.dh, env.pn, env.n), unb64(env.ct));
    return utf8.decode(pt);
  }

  Future<Envelope> encrypt(String plaintext) async {
    if (cks == null) {
      final dhOut = await _dh(dhsPriv, dhrPub, dhsPub);
      final (newRk, newCks) = await _kdfRk(rk, dhOut);
      rk = newRk;
      cks = newCks;
      pn = nr;
      ns = 0;
    }
    final (nextCk, mk) = await _kdfCk(cks!);
    cks = nextCk;
    final header = Envelope(dh: b64(dhsPub), pn: pn, n: ns, nonce: '', ct: '');
    ns++;
    final nonce = _randomBytes(12);
    final ct = await _aeadEncrypt(
        await _msgKey(mk),
        nonce,
        _headerAd(header.dh, header.pn, header.n),
        Uint8List.fromList(utf8.encode(plaintext)));
    header.nonce = b64(nonce);
    header.ct = b64(ct);
    if (first) {
      header.ik = b64(ownIkPub!);
      header.ek = b64(ekPub!);
      header.opk = opkId;
      first = false;
    }
    return header;
  }

  Future<String> decrypt(Envelope env) async {
    final dhPub = unb64(env.dh);
    if (dhPub.length != 32) throw Exception('invalid dh in envelope');
    final skipped = this.skipped[_skippedKey(dhPub, env.n)];
    if (skipped != null) {
      this.skipped.remove(_skippedKey(dhPub, env.n));
      return _decryptWithKey(skipped, env);
    }
    if (!_eq(dhPub, dhrPub)) {
      await _skipTo(env.pn);
      final dhOut = await _dh(dhsPriv, dhPub, dhsPub);
      final (newRk, newCkr) = await _kdfRk(rk, dhOut);
      rk = newRk;
      ckr = newCkr;
      dhrPub = dhPub;
      final (priv, pub) = await _x25519Keypair();
      dhsPriv = priv;
      dhsPub = pub;
      cks = null;
      nr = 0;
    }
    if (ckr == null) throw Exception('no receiving chain');
    while (nr < env.n) {
      final (next, mk) = await _kdfCk(ckr!);
      if (this.skipped.length >= maxSkippedKeys) {
        throw Exception('too many skipped message keys');
      }
      this.skipped[_skippedKey(dhrPub, nr)] = mk;
      ckr = next;
      nr++;
    }
    final (next, mk) = await _kdfCk(ckr!);
    ckr = next;
    nr++;
    return _decryptWithKey(mk, env);
  }
}

/// X3DH as the sender. Bundle must already be verified (verifyBundle).
Future<Session> initiateSession(Identity own, KeyBundle bundle) async {
  if (!await verifyBundle(bundle)) {
    throw Exception('bundle signature verification failed');
  }
  final (ekPriv, ekPub) = await _x25519Keypair();
  final peerIk = unb64(bundle.dhPub);
  final peerSpk = unb64(bundle.spkPub);
  final dh1 = await _dh(own.dhPriv, peerSpk, own.dhPub);
  final dh2 = await _dh(ekPriv, peerIk, ekPub);
  final dh3 = await _dh(ekPriv, peerSpk, ekPub);
  var secret = _concat([dh1, dh2, dh3]);
  var opkId = 0;
  if (bundle.oneTimePrekey != null) {
    final dh4 =
        await _dh(ekPriv, unb64(bundle.oneTimePrekey!['pub'] as String), ekPub);
    secret = _concat([secret, dh4]);
    opkId = bundle.oneTimePrekey!['id'] as int;
  }
  final sk = await _hkdf(Uint8List(32), secret,
      Uint8List.fromList(utf8.encode('vyro-x3dh-v1')), 64);
  final (dhsPriv, dhsPub) = await _x25519Keypair();
  return Session(
    peerUserId: bundle.userId,
    peerDevice: bundle.deviceId,
    dhsPriv: dhsPriv,
    dhsPub: dhsPub,
    dhrPub: peerSpk,
    rk: sk.sublist(0, 32),
    ckr: sk.sublist(32),
    first: true,
    ekPub: ekPub,
    opkId: opkId,
    ownIkPub: own.dhPub,
  );
}

/// X3DH as the receiver for the first message of a session.
Future<Session> receiveInitSession(
    Identity own,
    SignedPreKey spk,
    OneTimePreKey? opk,
    Uint8List senderIk,
    Uint8List senderEk,
    String peerUserId,
    String peerDevice) async {
  final dh1 = await _dh(spk.priv, senderIk, spk.pub);
  final dh2 = await _dh(own.dhPriv, senderEk, own.dhPub);
  final dh3 = await _dh(spk.priv, senderEk, spk.pub);
  var secret = _concat([dh1, dh2, dh3]);
  if (opk != null) {
    final dh4 = await _dh(opk.priv, senderEk, opk.pub);
    secret = _concat([secret, dh4]);
  }
  final sk = await _hkdf(Uint8List(32), secret,
      Uint8List.fromList(utf8.encode('vyro-x3dh-v1')), 64);
  return Session(
    peerUserId: peerUserId,
    peerDevice: peerDevice,
    dhsPriv: spk.priv,
    dhsPub: spk.pub,
    dhrPub: Uint8List(0),
    rk: sk.sublist(0, 32),
    ckr: sk.sublist(32),
    spkPriv: spk.priv,
  );
}

// ------------------------------------------------------------ group E2EE
//
// Sender Keys, mirroring backend/internal/e2ee/group.go exactly:
// chain ratchet (next = HMAC(ck, 0x01), msg key = HMAC(ck, 0x02)),
// message key -> HKDF "vyro-msg-v1" -> AES-256-GCM, header AD is the
// JSON of {key_id, sender_user, sender_device, n}, signature is
// Ed25519 over headerAD || nonce || ct.

class GroupDistribution {
  GroupDistribution(
      {required this.keyId,
      required this.chainKey,
      required this.signPub,
      required this.n});

  String keyId;
  Uint8List chainKey; // 32 bytes
  Uint8List signPub; // Ed25519 public key
  int n;

  Map<String, dynamic> toJson() => {
        'key_id': keyId,
        'chain_key': b64(chainKey),
        'sign_pub': b64(signPub),
        'n': n,
      };

  static GroupDistribution fromJson(Map<String, dynamic> j) =>
      GroupDistribution(
        keyId: j['key_id'] as String,
        chainKey: unb64(j['chain_key'] as String),
        signPub: unb64(j['sign_pub'] as String),
        n: j['n'] as int,
      );
}

class GroupEnvelope {
  GroupEnvelope(
      {required this.keyId,
      required this.senderUser,
      required this.senderDevice,
      required this.n,
      required this.nonce,
      required this.ct,
      required this.sig});

  String keyId;
  String senderUser;
  String senderDevice;
  int n;
  String nonce;
  String ct;
  String sig;

  Map<String, dynamic> toJson() => {
        'v': 1,
        'key_id': keyId,
        'sender_user': senderUser,
        'sender_device': senderDevice,
        'n': n,
        'nonce': nonce,
        'ct': ct,
        'sig': sig,
      };

  static GroupEnvelope fromJson(Map<String, dynamic> j) => GroupEnvelope(
        keyId: j['key_id'] as String,
        senderUser: j['sender_user'] as String,
        senderDevice: j['sender_device'] as String,
        n: j['n'] as int,
        nonce: j['nonce'] as String,
        ct: j['ct'] as String,
        sig: j['sig'] as String,
      );

  String toJsonString() => jsonEncode(toJson());
  static GroupEnvelope fromJsonString(String s) =>
      GroupEnvelope.fromJson(jsonDecode(s) as Map<String, dynamic>);

  Uint8List headerAd() => Uint8List.fromList(utf8.encode(jsonEncode({
        'key_id': keyId,
        'sender_user': senderUser,
        'sender_device': senderDevice,
        'n': n,
      })));

  Uint8List signedData() => _concat([headerAd(), unb64(nonce), unb64(ct)]);
}

class GroupSender {
  GroupSender(
      {required this.senderUserId,
      required this.senderDeviceId,
      required this.keyId,
      required this.chainKey,
      required this.signSeed,
      required this.signPub,
      this.n = 0});

  String senderUserId;
  String senderDeviceId;
  String keyId;
  Uint8List chainKey;
  Uint8List signSeed;
  Uint8List signPub;
  int n;

  static Future<GroupSender> generate(String userId, String deviceId) async {
    final ck = _randomBytes(32);
    final kp = await _ed25519.newKeyPair();
    final pub = await kp.extractPublicKey();
    final seed = await kp.extractPrivateKeyBytes();
    return GroupSender(
      senderUserId: userId,
      senderDeviceId: deviceId,
      keyId: 'sk-${DateTime.now().millisecondsSinceEpoch}',
      chainKey: ck,
      signSeed: Uint8List.fromList(seed),
      signPub: Uint8List.fromList(pub.bytes),
    );
  }

  GroupDistribution distribution() => GroupDistribution(
      keyId: keyId, chainKey: chainKey, signPub: signPub, n: n);

  Future<GroupEnvelope> encrypt(String plaintext) async {
    final (nextCk, mk) = await _kdfCk(chainKey);
    chainKey = nextCk;
    final env = GroupEnvelope(
        keyId: keyId,
        senderUser: senderUserId,
        senderDevice: senderDeviceId,
        n: n,
        nonce: '',
        ct: '',
        sig: '');
    n++;
    final nonce = _randomBytes(12);
    final ct = await _aeadEncrypt(await _msgKey(mk), nonce, env.headerAd(),
        Uint8List.fromList(utf8.encode(plaintext)));
    env.nonce = b64(nonce);
    env.ct = b64(ct);
    env.sig = b64(await _ed25519Sign(signSeed, env.signedData()));
    return env;
  }

  Map<String, dynamic> toJson() => {
        'sender_user': senderUserId,
        'sender_device': senderDeviceId,
        'key_id': keyId,
        'chain_key': b64(chainKey),
        'sign_seed': b64(signSeed),
        'sign_pub': b64(signPub),
        'n': n,
      };

  static GroupSender fromJson(Map<String, dynamic> j) => GroupSender(
        senderUserId: j['sender_user'] as String,
        senderDeviceId: j['sender_device'] as String,
        keyId: j['key_id'] as String,
        chainKey: unb64(j['chain_key'] as String),
        signSeed: unb64(j['sign_seed'] as String),
        signPub: unb64(j['sign_pub'] as String),
        n: j['n'] as int,
      );
}

class GroupReceiver {
  GroupReceiver(
      {required this.senderUserId,
      required this.senderDeviceId,
      required this.keyId,
      required this.chainKey,
      required this.signPub,
      required this.n})
      : skipped = {};

  String senderUserId;
  String senderDeviceId;
  String keyId;
  Uint8List chainKey;
  Uint8List signPub;
  int n;
  Map<int, Uint8List> skipped;

  static GroupReceiver fromDistribution(
          String senderUser, String senderDevice, GroupDistribution d) =>
      GroupReceiver(
          senderUserId: senderUser,
          senderDeviceId: senderDevice,
          keyId: d.keyId,
          chainKey: d.chainKey,
          signPub: d.signPub,
          n: d.n);

  Future<String> decrypt(GroupEnvelope env) async {
    if (env.keyId != keyId ||
        env.senderUser != senderUserId ||
        env.senderDevice != senderDeviceId) {
      throw Exception('envelope is not for this sender key');
    }
    final sig = unb64(env.sig);
    if (sig.length != 64) throw Exception('invalid sig');
    if (!await _ed25519Verify(signPub, env.signedData(), sig)) {
      throw Exception('signature verification failed');
    }
    final skippedMk = skipped[env.n];
    if (skippedMk != null) {
      skipped.remove(env.n);
      return _decryptWithKey(skippedMk, env);
    }
    if (env.n < n) throw Exception('message already decrypted (replay)');
    while (n < env.n) {
      final (nextCk, mk) = await _kdfCk(chainKey);
      if (skipped.length >= maxSkippedKeys) {
        throw Exception('too many skipped group message keys');
      }
      skipped[n] = mk;
      chainKey = nextCk;
      n++;
    }
    final (nextCk, mk) = await _kdfCk(chainKey);
    chainKey = nextCk;
    n++;
    return _decryptWithKey(mk, env);
  }

  Future<String> _decryptWithKey(Uint8List mk, GroupEnvelope env) async {
    final pt = await _aeadDecrypt(
        await _msgKey(mk), unb64(env.nonce), env.headerAd(), unb64(env.ct));
    return utf8.decode(pt);
  }

  Map<String, dynamic> toJson() => {
        'sender_user': senderUserId,
        'sender_device': senderDeviceId,
        'key_id': keyId,
        'chain_key': b64(chainKey),
        'sign_pub': b64(signPub),
        'n': n,
        'skipped': {
          for (final e in skipped.entries) e.key.toString(): b64(e.value)
        },
      };

  static GroupReceiver fromJson(Map<String, dynamic> j) {
    final r = GroupReceiver(
      senderUserId: j['sender_user'] as String,
      senderDeviceId: j['sender_device'] as String,
      keyId: j['key_id'] as String,
      chainKey: unb64(j['chain_key'] as String),
      signPub: unb64(j['sign_pub'] as String),
      n: j['n'] as int,
    );
    final sk = j['skipped'] as Map<String, dynamic>?;
    if (sk != null) {
      for (final e in sk.entries) {
        r.skipped[int.parse(e.key)] = unb64(e.value as String);
      }
    }
    return r;
  }
}
