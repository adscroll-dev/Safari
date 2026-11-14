// VYRO E2EE v1 — TypeScript client (WebCrypto).
//
// A faithful mirror of backend/internal/e2ee (Go): X3DH + Double Ratchet
// over X25519 / HKDF-SHA256 / HMAC-SHA256 / AES-256-GCM, with Ed25519
// signatures on the signed pre-key. Wire format and all KDF labels are
// identical to the Go side, so envelopes produced here decrypt there
// and vice versa (proven by the interop test in web/lib/e2ee.interop.mjs).
//
// Keys are generated with the WebCrypto API only — no custom crypto and
// no third-party crypto library. Private keys never leave the device:
// the server only ever sees public keys and ciphertext envelopes.
//
// Browser support: X25519 in WebCrypto needs a current browser
// (Chrome 137+, Safari 26+, Firefox 144+). e2eeSupported() probes for
// it; the UI disables encrypted chat when it is missing instead of
// silently falling back to plaintext.
//
// Storage note: session/identity private keys are kept as raw 32-byte
// scalars (base64 in JSON) so they can be stored in localStorage.
// That is dev-grade — a production client should use non-extractable
// keys or the OS keystore. Never send private keys to the server.
import { b64encode, b64decode } from "./b64";

export const PROTOCOL_VERSION = 1;
const MAX_SKIPPED_KEYS = 1000;

export interface Bundle {
  user_id: string;
  device_id: string;
  dh_pub: string; // identity DH public key, base64
  sign_pub: string; // Ed25519 public key, base64
  spk_pub: string; // signed pre-key public, base64
  spk_id: number;
  spk_sig: string; // Ed25519 signature over SPK pub, base64
  one_time_prekey?: { id: number; pub: string } | null;
}

export interface Envelope {
  v: number;
  ik?: string; // sender identity DH pub, b64 (first message only)
  ek?: string; // sender ephemeral pub, b64 (first message only)
  opk?: number; // one-time pre-key id used (0 = none)
  dh: string; // sender ratchet pub, b64
  pn: number;
  n: number;
  nonce: string; // AES-GCM nonce, b64
  ct: string; // AES-GCM ciphertext (+16-byte tag), b64
}

export interface Identity {
  dhPriv: Uint8Array; // 32 bytes X25519 private scalar
  dhPub: Uint8Array; // 32 bytes X25519 public
  // Ed25519 signing key as a JWK (base64url d + x). WebCrypto rejects
  // a private JWK with only "d", so both halves are stored.
  signPrivJwk: { d: string; x: string };
  signPub: Uint8Array; // 32 bytes Ed25519 public
}

export interface SignedPreKey {
  id: number;
  priv: Uint8Array;
  pub: Uint8Array;
  signature: Uint8Array; // Ed25519 signature over pub
}

export interface OneTimePreKey {
  id: number;
  priv: Uint8Array;
  pub: Uint8Array;
}

// Session mirrors the Go Session JSON shape (field names included) so a
// session file is readable on both sides. Private scalars are base64.
export interface SessionState {
  peer_user_id: string;
  peer_device: string;
  dhs_priv: string;
  dhs_pub: string;
  dhr_pub: string;
  rk: string;
  cks?: string;
  ckr?: string;
  ns: number;
  nr: number;
  pn: number;
  skipped?: Record<string, string>;
  first?: boolean;
  ek_pub?: string;
  opk_id?: number;
  own_ik_pub?: string;
  spk_priv?: string;
}

// ------------------------------------------------------------ primitives

// ab returns an ArrayBuffer holding exactly this view's bytes. Passing
// u.buffer directly is a bug when u is a subarray (e.g. the halves of
// a 64-byte HKDF output): the whole underlying buffer would be used.
function ab(u: Uint8Array): ArrayBuffer {
  return u.buffer.slice(u.byteOffset, u.byteOffset + u.byteLength) as ArrayBuffer;
}

function concat(...parts: Uint8Array[]): Uint8Array {
  const out = new Uint8Array(parts.reduce((n, p) => n + p.length, 0));
  let off = 0;
  for (const p of parts) {
    out.set(p, off);
    off += p.length;
  }
  return out;
}

function eq(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  let d = 0;
  for (let i = 0; i < a.length; i++) d |= a[i] ^ b[i];
  return d === 0;
}

async function hmac(key: Uint8Array, msg: Uint8Array): Promise<Uint8Array> {
  const k = await crypto.subtle.importKey("raw", ab(key), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  const sig = await crypto.subtle.sign("HMAC", k, ab(msg));
  return new Uint8Array(sig);
}

// HKDF-SHA256 (RFC 5869), implemented over HMAC so the exact salt/info
// handling matches Go's x/crypto/hkdf (empty salt = HashLen zeros,
// empty IKM = empty string). Lengths used here are 32 or 64 bytes.
async function hkdf(salt: Uint8Array, ikm: Uint8Array, info: Uint8Array, length: number): Promise<Uint8Array> {
  const saltEff = salt.length === 0 ? new Uint8Array(32) : salt;
  const prk = await hmac(saltEff, ikm);
  const out = new Uint8Array(length);
  let t = new Uint8Array(0);
  let done = 0;
  let counter = 1;
  while (done < length) {
    t = await hmac(prk, concat(t, info, new Uint8Array([counter])));
    const take = Math.min(t.length, length - done);
    out.set(t.subarray(0, take), done);
    done += take;
    counter++;
  }
  return out;
}

function b64url(bytes: Uint8Array): string {
  return b64encode(bytes).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

async function importX25519Priv(priv: Uint8Array, pub: Uint8Array): Promise<CryptoKey> {
  return crypto.subtle.importKey(
    "jwk",
    { kty: "OKP", crv: "X25519", d: b64url(priv), x: b64url(pub) },
    { name: "X25519" },
    false,
    ["deriveBits"],
  );
}

async function importX25519Pub(pub: Uint8Array): Promise<CryptoKey> {
  return crypto.subtle.importKey("jwk", { kty: "OKP", crv: "X25519", x: b64url(pub) }, { name: "X25519" }, false, []);
}

async function dh(priv: Uint8Array, pub: Uint8Array, ownPub: Uint8Array): Promise<Uint8Array> {
  const privKey = await importX25519Priv(priv, ownPub);
  const pubKey = await importX25519Pub(pub);
  const bits = await crypto.subtle.deriveBits({ name: "X25519", public: pubKey }, privKey, 256);
  return new Uint8Array(bits);
}

async function x25519Keypair(): Promise<{ priv: Uint8Array; pub: Uint8Array }> {
  const kp = (await crypto.subtle.generateKey({ name: "X25519" }, true, ["deriveBits"])) as CryptoKeyPair;
  const privJwk = await crypto.subtle.exportKey("jwk", kp.privateKey);
  const pubJwk = await crypto.subtle.exportKey("jwk", kp.publicKey);
  return { priv: b64decode(privJwk.d as string, true), pub: b64decode(pubJwk.x as string, true) };
}

async function ed25519Sign(signPrivJwk: { d: string; x: string }, msg: Uint8Array): Promise<Uint8Array> {
  const privKey = await crypto.subtle.importKey(
    "jwk",
    { kty: "OKP", crv: "Ed25519", d: signPrivJwk.d, x: signPrivJwk.x },
    { name: "Ed25519" },
    false,
    ["sign"],
  );
  const sig = await crypto.subtle.sign({ name: "Ed25519" }, privKey, ab(msg));
  return new Uint8Array(sig);
}

async function ed25519Verify(pub: Uint8Array, msg: Uint8Array, sig: Uint8Array): Promise<boolean> {
  const pubKey = await crypto.subtle.importKey(
    "jwk",
    { kty: "OKP", crv: "Ed25519", x: b64url(pub) },
    { name: "Ed25519" },
    false,
    ["verify"],
  );
  return crypto.subtle.verify({ name: "Ed25519" }, pubKey, ab(sig), ab(msg));
}

async function kdfRK(rk: Uint8Array, dhOut: Uint8Array): Promise<[Uint8Array, Uint8Array]> {
  const out = await hkdf(rk, dhOut, new TextEncoder().encode("vyro-ratchet-v1"), 64);
  return [out.subarray(0, 32), out.subarray(32)];
}

async function kdfCK(ck: Uint8Array): Promise<[Uint8Array, Uint8Array]> {
  return [await hmac(ck, new Uint8Array([0x01])), await hmac(ck, new Uint8Array([0x02]))];
}

async function msgKey(mk: Uint8Array): Promise<Uint8Array> {
  return hkdf(mk, new Uint8Array(0), new TextEncoder().encode("vyro-msg-v1"), 32);
}

async function aeadEncrypt(key: Uint8Array, nonce: Uint8Array, ad: Uint8Array, plaintext: Uint8Array): Promise<Uint8Array> {
  const k = await crypto.subtle.importKey("raw", ab(key), "AES-GCM", false, ["encrypt"]);
  const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv: ab(nonce), additionalData: ab(ad), tagLength: 128 }, k, ab(plaintext));
  return new Uint8Array(ct);
}

async function aeadDecrypt(key: Uint8Array, nonce: Uint8Array, ad: Uint8Array, ct: Uint8Array): Promise<Uint8Array> {
  const k = await crypto.subtle.importKey("raw", ab(key), "AES-GCM", false, ["decrypt"]);
  const pt = await crypto.subtle.decrypt({ name: "AES-GCM", iv: ab(nonce), additionalData: ab(ad), tagLength: 128 }, k, ab(ct));
  return new Uint8Array(pt);
}

function headerAD(dhB64: string, pn: number, n: number): Uint8Array {
  // Must match Go's json.Marshal of {dh, pn, n} exactly (field order
  // dh, pn, n; no spaces).
  return new TextEncoder().encode(JSON.stringify({ dh: dhB64, pn, n }));
}

export async function fingerprint(ikA: Uint8Array, ikB: Uint8Array): Promise<string> {
  const h = await crypto.subtle.digest("SHA-256", ab(concat(ikA, ikB)));
  return Array.from(new Uint8Array(h).subarray(0, 8))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

// ------------------------------------------------------------ identities

export async function e2eeSupported(): Promise<boolean> {
  try {
    if (typeof crypto === "undefined" || !crypto.subtle) return false;
    await crypto.subtle.generateKey({ name: "X25519" }, false, ["deriveBits"]);
    await crypto.subtle.generateKey({ name: "Ed25519" }, false, ["sign"]);
    return true;
  } catch {
    return false;
  }
}

export async function generateIdentity(): Promise<Identity> {
  const dh = await x25519Keypair();
  const signKp = (await crypto.subtle.generateKey({ name: "Ed25519" }, true, ["sign", "verify"])) as CryptoKeyPair;
  const signJwk = await crypto.subtle.exportKey("jwk", signKp.privateKey);
  const signPubJwk = await crypto.subtle.exportKey("jwk", signKp.publicKey);
  return {
    dhPriv: dh.priv,
    dhPub: dh.pub,
    signPrivJwk: { d: signJwk.d as string, x: signJwk.x as string },
    signPub: b64decode(signPubJwk.x as string, true),
  };
}

export async function generateSignedPreKey(id: number, ident: Identity): Promise<SignedPreKey> {
  const kp = await x25519Keypair();
  const signature = await ed25519Sign(ident.signPrivJwk, kp.pub);
  return { id, priv: kp.priv, pub: kp.pub, signature };
}

export async function generateOneTimePreKey(id: number): Promise<OneTimePreKey> {
  const kp = await x25519Keypair();
  return { id, priv: kp.priv, pub: kp.pub };
}

export async function verifyBundle(bundle: Bundle): Promise<boolean> {
  try {
    const signPub = b64decode(bundle.sign_pub);
    const spk = b64decode(bundle.spk_pub);
    const sig = b64decode(bundle.spk_sig);
    if (signPub.length !== 32 || spk.length !== 32 || sig.length !== 64) return false;
    return await ed25519Verify(signPub, spk, sig);
  } catch {
    return false;
  }
}

// ------------------------------------------------------------ sessions

// Session is the live Double Ratchet state. Serialize it with
// serializeSession() for localStorage; never send it to the server.
export class Session {
  peerUserId!: string;
  peerDevice!: string;
  dhsPriv!: Uint8Array;
  dhsPub!: Uint8Array;
  dhrPub!: Uint8Array;
  rk!: Uint8Array;
  cks?: Uint8Array;
  ckr?: Uint8Array;
  ns = 0;
  nr = 0;
  pn = 0;
  skipped = new Map<string, Uint8Array>();
  first = false;
  ekPub?: Uint8Array;
  opkId = 0;
  ownIkPub?: Uint8Array;
  spkPriv?: Uint8Array;

  private constructor(init: Partial<Session>) {
    Object.assign(this, init);
  }

  static fromState(st: SessionState): Session {
    const s = new Session({
      peerUserId: st.peer_user_id,
      peerDevice: st.peer_device,
      dhsPriv: b64decode(st.dhs_priv),
      dhsPub: b64decode(st.dhs_pub),
      dhrPub: b64decode(st.dhr_pub),
      rk: b64decode(st.rk),
      ns: st.ns,
      nr: st.nr,
      pn: st.pn,
      first: !!st.first,
      opkId: st.opk_id ?? 0,
    });
    if (st.cks) s.cks = b64decode(st.cks);
    if (st.ckr) s.ckr = b64decode(st.ckr);
    if (st.ek_pub) s.ekPub = b64decode(st.ek_pub);
    if (st.own_ik_pub) s.ownIkPub = b64decode(st.own_ik_pub);
    if (st.spk_priv) s.spkPriv = b64decode(st.spk_priv);
    if (st.skipped) for (const [k, v] of Object.entries(st.skipped)) s.skipped.set(k, b64decode(v));
    return s;
  }

  toState(): SessionState {
    const st: SessionState = {
      peer_user_id: this.peerUserId,
      peer_device: this.peerDevice,
      dhs_priv: b64encode(this.dhsPriv),
      dhs_pub: b64encode(this.dhsPub),
      dhr_pub: b64encode(this.dhrPub),
      rk: b64encode(this.rk),
      ns: this.ns,
      nr: this.nr,
      pn: this.pn,
      first: this.first,
      opk_id: this.opkId,
    };
    if (this.cks) st.cks = b64encode(this.cks);
    if (this.ckr) st.ckr = b64encode(this.ckr);
    if (this.ekPub) st.ek_pub = b64encode(this.ekPub);
    if (this.ownIkPub) st.own_ik_pub = b64encode(this.ownIkPub);
    if (this.spkPriv) st.spk_priv = b64encode(this.spkPriv);
    if (this.skipped.size > 0) {
      st.skipped = {};
      for (const [k, v] of this.skipped) st.skipped[k] = b64encode(v);
    }
    return st;
  }

  private skippedKey(dh: Uint8Array, n: number): string {
    return `${b64encode(dh)}:${n}`;
  }

  private async skipTo(n: number): Promise<void> {
    while (this.nr < n) {
      if (!this.ckr) throw new Error("no receiving chain");
      const [next, mk] = await kdfCK(this.ckr);
      if (this.skipped.size >= MAX_SKIPPED_KEYS) throw new Error("too many skipped message keys");
      this.skipped.set(this.skippedKey(this.dhrPub, this.nr), mk);
      this.ckr = next;
      this.nr++;
    }
  }

  private async decryptWithKey(mk: Uint8Array, env: Envelope): Promise<string> {
    const key = await msgKey(mk);
    const pt = await aeadDecrypt(key, b64decode(env.nonce), headerAD(env.dh, env.pn, env.n), b64decode(env.ct));
    return new TextDecoder().decode(pt);
  }

  async encrypt(plaintext: string): Promise<Envelope> {
    if (!this.cks) {
      const dhOut = await dh(this.dhsPriv, this.dhrPub, this.dhsPub);
      const [rk, cks] = await kdfRK(this.rk, dhOut);
      this.rk = rk;
      this.cks = cks;
      this.pn = this.nr;
      this.ns = 0;
    }
    const [nextCk, mk] = await kdfCK(this.cks);
    this.cks = nextCk;
    const header: Envelope = { v: PROTOCOL_VERSION, dh: b64encode(this.dhsPub), pn: this.pn, n: this.ns, nonce: "", ct: "" };
    this.ns++;
    const nonce = crypto.getRandomValues(new Uint8Array(12));
    const ct = await aeadEncrypt(await msgKey(mk), nonce, headerAD(header.dh, header.pn, header.n), new TextEncoder().encode(plaintext));
    header.nonce = b64encode(nonce);
    header.ct = b64encode(ct);
    if (this.first) {
      header.ik = b64encode(this.ownIkPub as Uint8Array);
      header.ek = b64encode(this.ekPub as Uint8Array);
      header.opk = this.opkId;
      this.first = false;
    }
    return header;
  }

  async decrypt(env: Envelope): Promise<string> {
    if (env.v !== PROTOCOL_VERSION) throw new Error(`unsupported envelope version ${env.v}`);
    const dhPub = b64decode(env.dh);
    if (dhPub.length !== 32) throw new Error("invalid dh in envelope");
    const skipped = this.skipped.get(this.skippedKey(dhPub, env.n));
    if (skipped) {
      this.skipped.delete(this.skippedKey(dhPub, env.n));
      return this.decryptWithKey(skipped, env);
    }
    if (!eq(dhPub, this.dhrPub)) {
      await this.skipTo(env.pn);
      const dhOut = await dh(this.dhsPriv, dhPub, this.dhsPub);
      const [rk, ckr] = await kdfRK(this.rk, dhOut);
      this.rk = rk;
      this.ckr = ckr;
      this.dhrPub = dhPub;
      const kp = await x25519Keypair();
      this.dhsPriv = kp.priv;
      this.dhsPub = kp.pub;
      this.cks = undefined;
      this.nr = 0;
    }
    if (!this.ckr) throw new Error("no receiving chain");
    while (this.nr < env.n) {
      const [next, mk] = await kdfCK(this.ckr);
      if (this.skipped.size >= MAX_SKIPPED_KEYS) throw new Error("too many skipped message keys");
      this.skipped.set(this.skippedKey(this.dhrPub, this.nr), mk);
      this.ckr = next;
      this.nr++;
    }
    const [next, mk] = await kdfCK(this.ckr);
    this.ckr = next;
    this.nr++;
    return this.decryptWithKey(mk, env);
  }
}

// X3DH as the sender. Bundle must already be verified (verifyBundle).
export async function initiateSession(own: Identity, bundle: Bundle): Promise<Session> {
  if (!(await verifyBundle(bundle))) throw new Error("bundle signature verification failed");
  const ek = await x25519Keypair();
  const peerIK = b64decode(bundle.dh_pub);
  const peerSPK = b64decode(bundle.spk_pub);
  const dh1 = await dh(own.dhPriv, peerSPK, own.dhPub);
  const dh2 = await dh(ek.priv, peerIK, ek.pub);
  const dh3 = await dh(ek.priv, peerSPK, ek.pub);
  let secret = concat(dh1, dh2, dh3);
  let opkId = 0;
  if (bundle.one_time_prekey) {
    const dh4 = await dh(ek.priv, b64decode(bundle.one_time_prekey.pub), ek.pub);
    secret = concat(secret, dh4);
    opkId = bundle.one_time_prekey.id;
  }
  const sk = await hkdf(new Uint8Array(32), secret, new TextEncoder().encode("vyro-x3dh-v1"), 64);
  const dhs = await x25519Keypair();
  return Session.fromState({
    peer_user_id: bundle.user_id,
    peer_device: bundle.device_id,
    dhs_priv: b64encode(dhs.priv),
    dhs_pub: b64encode(dhs.pub),
    dhr_pub: b64encode(peerSPK),
    rk: b64encode(sk.subarray(0, 32)),
    ckr: b64encode(sk.subarray(32)),
    ns: 0,
    nr: 0,
    pn: 0,
    first: true,
    ek_pub: b64encode(ek.pub),
    opk_id: opkId,
    own_ik_pub: b64encode(own.dhPub),
  });
}

// X3DH as the receiver for the first message of a session. spk/opk are
// this device's keys (private parts); senderIK/senderEK come from the
// envelope's ik/ek fields.
export async function receiveInitSession(
  own: Identity,
  spk: SignedPreKey,
  opk: OneTimePreKey | null,
  senderIK: Uint8Array,
  senderEK: Uint8Array,
  peerUserId: string,
  peerDevice: string,
): Promise<Session> {
  const dh1 = await dh(spk.priv, senderIK, spk.pub);
  const dh2 = await dh(own.dhPriv, senderEK, own.dhPub);
  const dh3 = await dh(spk.priv, senderEK, spk.pub);
  let secret = concat(dh1, dh2, dh3);
  if (opk) {
    const dh4 = await dh(opk.priv, senderEK, opk.pub);
    secret = concat(secret, dh4);
  }
  const sk = await hkdf(new Uint8Array(32), secret, new TextEncoder().encode("vyro-x3dh-v1"), 64);
  return Session.fromState({
    peer_user_id: peerUserId,
    peer_device: peerDevice,
    dhs_priv: b64encode(spk.priv),
    dhs_pub: b64encode(spk.pub),
    dhr_pub: "",
    rk: b64encode(sk.subarray(0, 32)),
    ckr: b64encode(sk.subarray(32)),
    ns: 0,
    nr: 0,
    pn: 0,
    first: false,
    spk_priv: b64encode(spk.priv),
  });
}
