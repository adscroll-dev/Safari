// Device-local E2EE storage for the web client (v0.5).
//
// Everything here lives in localStorage and NEVER goes to the server:
// the identity's private keys, the signed pre-key + one-time pre-keys'
// private halves, and the Double Ratchet sessions. Only PUBLIC key
// material is uploaded (see ensureDeviceKeys).
//
// Dev-grade storage: localStorage is readable by any script on this
// origin. A production client should use non-extractable WebCrypto
// keys or the OS keystore. The README says so.
import { b64encode, b64decode } from "./b64";
import * as e2ee from "./e2ee";
import * as api from "./api";

const ID_KEY = "vyro_e2ee_identity_v1";
const KEYS_KEY = "vyro_e2ee_keys_v1";
const SESSION_PREFIX = "vyro_e2ee_session_v1:";

export interface StoredIdentity {
  dhPriv: string;
  dhPub: string;
  signPrivJwk: { d: string; x: string };
  signPub: string;
}

export interface StoredKeys {
  spk: { id: number; priv: string; pub: string; signature: string };
  opks: { id: number; priv: string; pub: string }[];
}

function toIdentity(s: StoredIdentity): e2ee.Identity {
  return {
    dhPriv: b64decode(s.dhPriv),
    dhPub: b64decode(s.dhPub),
    signPrivJwk: s.signPrivJwk,
    signPub: b64decode(s.signPub),
  };
}

function fromIdentity(i: e2ee.Identity): StoredIdentity {
  return {
    dhPriv: b64encode(i.dhPriv),
    dhPub: b64encode(i.dhPub),
    signPrivJwk: i.signPrivJwk,
    signPub: b64encode(i.signPub),
  };
}

export function loadIdentity(): e2ee.Identity | null {
  try {
    const raw = localStorage.getItem(ID_KEY);
    if (!raw) return null;
    return toIdentity(JSON.parse(raw) as StoredIdentity);
  } catch {
    return null;
  }
}

export function loadKeys(): StoredKeys | null {
  try {
    const raw = localStorage.getItem(KEYS_KEY);
    return raw ? (JSON.parse(raw) as StoredKeys) : null;
  } catch {
    return null;
  }
}

export function loadSession(peerUserId: string): e2ee.Session | null {
  try {
    const raw = localStorage.getItem(SESSION_PREFIX + peerUserId);
    if (!raw) return null;
    return e2ee.Session.fromState(JSON.parse(raw) as e2ee.SessionState);
  } catch {
    return null;
  }
}

export function saveSession(peerUserId: string, s: e2ee.Session): void {
  try {
    localStorage.setItem(SESSION_PREFIX + peerUserId, JSON.stringify(s.toState()));
  } catch {
    // Storage full/blocked: session is lost on reload, but the chat
    // still works — the peer's next message re-initiates.
  }
}

// Generate the device identity + keys and publish the public halves.
// Runs once per device (when nothing is stored yet); afterwards the
// stored identity is reused. Returns null when E2EE is unsupported.
export async function ensureDeviceKeys(): Promise<{ identity: e2ee.Identity; keys: StoredKeys } | null> {
  if (!(await e2ee.e2eeSupported())) return null;
  const existing = loadIdentity();
  const existingKeys = loadKeys();
  if (existing && existingKeys) return { identity: existing, keys: existingKeys };
  const identity = await e2ee.generateIdentity();
  const spk = await e2ee.generateSignedPreKey(1, identity);
  const opks: e2ee.OneTimePreKey[] = [];
  for (let i = 1; i <= 20; i++) opks.push(await e2ee.generateOneTimePreKey(i));
  const keys: StoredKeys = {
    spk: { id: spk.id, priv: b64encode(spk.priv), pub: b64encode(spk.pub), signature: b64encode(spk.signature) },
    opks: opks.map((o) => ({ id: o.id, priv: b64encode(o.priv), pub: b64encode(o.pub) })),
  };
  await api.uploadIdentityKey(
    "default",
    b64encode(identity.dhPub),
    b64encode(spk.pub),
    spk.id,
    b64encode(identity.signPub),
    b64encode(spk.signature),
  );
  await api.uploadPreKeys(
    "default",
    opks.map((o) => ({ key_id: o.id, public_key: b64encode(o.pub) })),
  );
  localStorage.setItem(ID_KEY, JSON.stringify(fromIdentity(identity)));
  localStorage.setItem(KEYS_KEY, JSON.stringify(keys));
  return { identity, keys };
}

// The peer's identity public key, stored next to the session so the
// Safety Number never needs a second bundle fetch (each fetch burns
// one of the peer's one-time pre-keys).
const PEERKEY_PREFIX = "vyro_e2ee_peerkey_v1:";

export function loadPeerKey(peerUserId: string): string | null {
  try {
    return localStorage.getItem(PEERKEY_PREFIX + peerUserId);
  } catch {
    return null;
  }
}

export function savePeerKey(peerUserId: string, dhPub: string): void {
  try {
    localStorage.setItem(PEERKEY_PREFIX + peerUserId, dhPub);
  } catch {
    // ignore
  }
}

// Get (or start) the E2EE session with a peer for a 1:1 DM. Also
// returns the peer's identity public key (base64) so the UI can show
// the Safety Number.
export async function getSession(peerUserId: string): Promise<{ session: e2ee.Session; peerDhPub: string } | null> {
  const saved = loadSession(peerUserId);
  if (saved) {
    const peerDhPub = loadPeerKey(peerUserId);
    if (peerDhPub) return { session: saved, peerDhPub };
    // Old session without a stored peer key: one bundle fetch.
    const bundle = await api.fetchKeyBundle(peerUserId);
    savePeerKey(peerUserId, bundle.dh_pub);
    return { session: saved, peerDhPub: bundle.dh_pub };
  }
  const identity = loadIdentity();
  if (!identity) return null;
  const bundle = await api.fetchKeyBundle(peerUserId);
  if (!(await e2ee.verifyBundle(bundle))) throw new Error("Peer's key bundle failed signature verification");
  const session = await e2ee.initiateSession(identity, bundle);
  saveSession(peerUserId, session);
  savePeerKey(peerUserId, bundle.dh_pub);
  return { session, peerDhPub: bundle.dh_pub };
}

// Safety Number for the active DM: fingerprint of both identity keys.
export async function safetyNumber(peerUserId: string): Promise<string | null> {
  const identity = loadIdentity();
  if (!identity) return null;
  try {
    const bundle = await api.fetchKeyBundle(peerUserId);
    return await e2ee.fingerprint(identity.dhPub, b64decode(bundle.dh_pub));
  } catch {
    return null;
  }
}

// Decrypt one stored envelope. Handles the first message of a session
// (receive-init with our own SPK/OPK) and later messages alike.
export async function decryptMessage(peerUserId: string, envelopeJson: string): Promise<string> {
  const identity = loadIdentity();
  const keys = loadKeys();
  if (!identity || !keys) throw new Error("No E2EE identity on this device");
  const env = JSON.parse(envelopeJson) as e2ee.Envelope;
  let session = loadSession(peerUserId);
  if (!session && env.ik && env.ek) {
    const spk: e2ee.SignedPreKey = {
      id: keys.spk.id,
      priv: b64decode(keys.spk.priv),
      pub: b64decode(keys.spk.pub),
      signature: b64decode(keys.spk.signature),
    };
    let opk: e2ee.OneTimePreKey | null = null;
    if (env.opk) {
      const found = keys.opks.find((o) => o.id === env.opk);
      if (found) opk = { id: found.id, priv: b64decode(found.priv), pub: b64decode(found.pub) };
    }
    session = await e2ee.receiveInitSession(
      identity,
      spk,
      opk,
      b64decode(env.ik),
      b64decode(env.ek),
      peerUserId,
      "default",
    );
    saveSession(peerUserId, session);
  }
  if (!session) throw new Error("No E2EE session for this peer");
  const pt = await session.decrypt(env);
  saveSession(peerUserId, session);
  return pt;
}
