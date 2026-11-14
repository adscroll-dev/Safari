// VYRO v0.5 LIVE end-to-end test against a real server (http://localhost:8080).
// Covers: E2EE DM over HTTP (server stores ciphertext only), WebSocket
// call signaling between two live clients, and Web Push delivery to a
// local stand-in push endpoint (decrypted with the subscription keys).
//
// Run (from backend/):
//   1. Build + start the server with Postgres and Redis, e.g.:
//        go build -o /tmp/vyro-server ./cmd/server
//        PORT=8080 DATABASE_URL="postgres://vyro:vyro@localhost:5432/vyro_live?sslmode=disable" \
//          REDIS_URL="redis://127.0.0.1:6379" /tmp/vyro-server
//   2. Compile the TS E2EE client once (it is imported below):
//        (cd ../web && npx tsc lib/e2ee.ts lib/b64.ts --outDir /tmp/e2ee-interop \
//          --module commonjs --target es2022 --lib es2022,dom --skipLibCheck)
//   3. node scripts/live-test.cjs
// Re-runnable: users log in when already registered, Alice's device
// keys persist in /tmp/vyro-live-keys.json, and the push subscription
// is removed at the end.
const http = require("http");
const { b64decode, b64encode } = require("/tmp/e2ee-interop/b64.js");
const e2ee = require("/tmp/e2ee-interop/e2ee.js");
const subtle = require("crypto").webcrypto.subtle;

const API = "http://localhost:8080";
let failures = 0;
function check(name, ok, extra = "") {
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${extra ? " — " + extra : ""}`);
  if (!ok) failures++;
}

async function req(path, token, method = "GET", body) {
  const res = await fetch(API + path, {
    method,
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: "Bearer " + token } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status}: ${JSON.stringify(data)}`);
  return data;
}

// ---- Web Push aes128gcm decryption (same scheme as Go push test) ----
async function hmac(key, msg) {
  const k = await subtle.importKey("raw", key, { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  return new Uint8Array(await subtle.sign("HMAC", k, msg));
}
async function hkdfFull(salt, ikm, info, len) {
  const prk = await hmac(salt.length ? salt : new Uint8Array(32), ikm);
  const cat = new Uint8Array(info.length + 1);
  cat.set(info, 0);
  cat[info.length] = 1;
  return (await hmac(prk, cat)).subarray(0, len);
}
async function decryptPush(body, subPrivJwk, subPub, auth) {
  const salt = body.subarray(0, 16);
  const senderPub = body.subarray(21, 86);
  const ct = body.subarray(86);
  const priv = await subtle.importKey("jwk", subPrivJwk, { name: "ECDH", namedCurve: "P-256" }, false, ["deriveBits"]);
  const pub = await subtle.importKey("jwk", { kty: "EC", crv: "P-256", x: Buffer.from(senderPub.subarray(1, 33)).toString("base64url"), y: Buffer.from(senderPub.subarray(33)).toString("base64url") }, { name: "ECDH", namedCurve: "P-256" }, false, []);
  const secret = new Uint8Array(await subtle.deriveBits({ name: "ECDH", public: pub }, priv, 256));
  const info = new Uint8Array(Buffer.concat([Buffer.from("WebPush: info\x00"), Buffer.from(subPub), Buffer.from(senderPub)]));
  const ikm = await hkdfFull(auth, secret, info, 32);
  const cek = await hkdfFull(salt, ikm, Buffer.from("Content-Encoding: aes128gcm\x00"), 16);
  const nonce = await hkdfFull(salt, ikm, Buffer.from("Content-Encoding: nonce\x00"), 12);
  const key = await subtle.importKey("raw", cek, "AES-GCM", false, ["decrypt"]);
  const raw = new Uint8Array(await subtle.decrypt({ name: "AES-GCM", iv: nonce }, key, ct));
  let end = raw.length;
  while (end > 0 && raw[end - 1] === 0) end--;
  if (raw[end - 1] !== 0x02) throw new Error("no 0x02 delimiter in push payload");
  return Buffer.from(raw.subarray(0, end - 1)).toString("utf8");
}

async function main() {
  // 1. Register two users (log in instead when re-running).
  async function registerOrLogin(email, username) {
    try {
      return await req("/api/auth/register", null, "POST", { email, username, password: "password-123", display_name: username });
    } catch (e) {
      if (String(e.message).includes("409")) {
        return await req("/api/auth/login", null, "POST", { email, password: "password-123" });
      }
      throw e;
    }
  }
  const alice = await registerOrLogin("live-alice@example.com", "livealice");
  const bob = await registerOrLogin("live-bob@example.com", "livebob");
  check("register alice+bob", !!alice.token && !!bob.token);

  // 2. E2EE: Alice publishes keys (persisted across runs in
  // /tmp/vyro-live-keys.json — a real device keeps its keys, and the
  // server still holds the ones from the first run).
  const KEYFILE = "/tmp/vyro-live-keys.json";
  let aliceIdent, spk, opks;
  if (require("fs").existsSync(KEYFILE)) {
    const saved = JSON.parse(require("fs").readFileSync(KEYFILE, "utf8"));
    aliceIdent = {
      dhPriv: b64decode(saved.dhPriv), dhPub: b64decode(saved.dhPub),
      signPrivJwk: saved.signPrivJwk, signPub: b64decode(saved.signPub),
    };
    spk = { id: 1, priv: b64decode(saved.spkPriv), pub: b64decode(saved.spkPub), signature: b64decode(saved.spkSig) };
    opks = saved.opks.map((o) => ({ id: o.id, priv: b64decode(o.priv), pub: b64decode(o.pub) }));
  } else {
    aliceIdent = await e2ee.generateIdentity();
    spk = await e2ee.generateSignedPreKey(1, aliceIdent);
    opks = [];
    for (let i = 1; i <= 20; i++) opks.push(await e2ee.generateOneTimePreKey(i));
    await req("/api/keys/identity", alice.token, "POST", {
      device_id: "default", identity_key: b64encode(aliceIdent.dhPub), signed_prekey: b64encode(spk.pub),
      signed_prekey_id: 1, sign_key: b64encode(aliceIdent.signPub), signed_prekey_sig: b64encode(spk.signature),
    });
    await req("/api/keys/prekeys", alice.token, "POST", { device_id: "default", keys: opks.map((o) => ({ key_id: o.id, public_key: b64encode(o.pub) })) });
    require("fs").writeFileSync(KEYFILE, JSON.stringify({
      dhPriv: b64encode(aliceIdent.dhPriv), dhPub: b64encode(aliceIdent.dhPub),
      signPrivJwk: aliceIdent.signPrivJwk, signPub: b64encode(aliceIdent.signPub),
      spkPriv: b64encode(spk.priv), spkPub: b64encode(spk.pub), spkSig: b64encode(spk.signature),
      opks: opks.map((o) => ({ id: o.id, priv: b64encode(o.priv), pub: b64encode(o.pub) })),
    }));
  }
  const bundle = await req(`/api/keys/${alice.user.id}/bundle`, bob.token);
  check("bundle verifies", await e2ee.verifyBundle(bundle));
  const bobIdent = await e2ee.generateIdentity();
  const bobSess = await e2ee.initiateSession(bobIdent, bundle);
  const plaintext = "live E2EE سلام — the server must not read this";
  const env = await bobSess.encrypt(plaintext);
  const chat = await req("/api/chats", bob.token, "POST", { type: "dm", member_ids: [alice.user.id] });
  const sent = await req(`/api/chats/${chat.id}/messages`, bob.token, "POST", { content: JSON.stringify(env), kind: "e2ee" });
  check("e2ee message stored", sent.kind === "e2ee");

  // 3. Alice reads: server stored ciphertext only; she decrypts it.
  const msgs = await req(`/api/chats/${chat.id}/messages`, alice.token);
  const stored = msgs.find((m) => m.id === sent.id);
  check("server stored ciphertext, not plaintext", stored && !stored.content.includes("سلام") && !stored.content.includes("server must not read"));
  const aliceSess = await e2ee.receiveInitSession(
    aliceIdent, spk, opks.find((o) => o.id === JSON.parse(stored.content).opk) || null,
    b64decode(JSON.parse(stored.content).ik), b64decode(JSON.parse(stored.content).ek), bob.user.id, "default",
  );
  const dec = await aliceSess.decrypt(JSON.parse(stored.content));
  check("alice decrypts live e2ee message", dec === plaintext, JSON.stringify(dec));

  // 4. WebSocket call signaling between two live clients.
  const wsUrl = (tok) => "ws://localhost:8080/ws?token=" + tok;
  const aliceWs = new WebSocket(wsUrl(alice.token));
  const bobWs = new WebSocket(wsUrl(bob.token));
  await Promise.all([
    new Promise((r) => (aliceWs.onopen = r)),
    new Promise((r) => (bobWs.onopen = r)),
  ]);
  // Join the DM room (both sides already belong, but subscribe explicitly).
  aliceWs.send(JSON.stringify({ type: "subscribe", chat_id: chat.id }));
  bobWs.send(JSON.stringify({ type: "subscribe", chat_id: chat.id }));
  await new Promise((r) => setTimeout(r, 200));
  const bobGot = new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("bob never got the offer")), 5000);
    bobWs.onmessage = (e) => {
      const ev = JSON.parse(e.data);
      if (ev.type === "call-offer") { clearTimeout(timer); resolve(ev); }
    };
  });
  aliceWs.send(JSON.stringify({ type: "call-offer", chat_id: chat.id, to_user_id: bob.user.id, payload: { type: "offer", sdp: "live-fake-offer" } }));
  const offer = await bobGot;
  check("live call-offer relayed verbatim", offer.payload && offer.payload.sdp === "live-fake-offer" && offer.user_id === alice.user.id, JSON.stringify(offer.payload));
  const aliceGot = new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("alice never got the answer")), 5000);
    aliceWs.onmessage = (e) => {
      const ev = JSON.parse(e.data);
      if (ev.type === "call-answer") { clearTimeout(timer); resolve(ev); }
    };
  });
  bobWs.send(JSON.stringify({ type: "call-answer", chat_id: chat.id, to_user_id: alice.user.id, payload: { type: "answer", sdp: "live-fake-answer" } }));
  const answer = await aliceGot;
  check("live call-answer relayed verbatim", answer.payload && answer.payload.sdp === "live-fake-answer");

  // 5. Web Push: local stand-in endpoint, real subscription keys.
  const pushKp = await subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, true, ["deriveBits"]);
  const pushPubJwk = await subtle.exportKey("jwk", pushKp.publicKey);
  const pushPrivJwk = await subtle.exportKey("jwk", pushKp.privateKey);
  const subPub = new Uint8Array([0x04, ...Buffer.from(pushPubJwk.x, "base64url"), ...Buffer.from(pushPubJwk.y, "base64url")]);
  const auth = new Uint8Array(16);
  require("crypto").randomFillSync(auth);
  const received = [];
  const pushSrv = http.createServer((rq, rs) => {
    const chunks = [];
    rq.on("data", (c) => chunks.push(c));
    rq.on("end", () => { received.push({ headers: rq.headers, body: Buffer.concat(chunks) }); rs.writeHead(201); rs.end(); });
  });
  await new Promise((r) => pushSrv.listen(0, r));
  const pushPort = pushSrv.address().port;
  const b64url = (b) => Buffer.from(b).toString("base64url");
  await req("/api/push/subscribe", bob.token, "POST", {
    endpoint: `http://127.0.0.1:${pushPort}/push`,
    keys: { p256dh: b64url(subPub), auth: b64url(auth) },
  });
  const testRes = await req("/api/push/test", bob.token, "POST");
  check("push test endpoint reports delivery", testRes.sent === 1 && testRes.failed === 0, JSON.stringify(testRes));
  await new Promise((r) => setTimeout(r, 300));
  check("push stand-in received one request", received.length === 1, `got ${received.length}`);
  if (received.length === 1) {
    check("push has VAPID auth header", !!received[0].headers.authorization);
    const body = await decryptPush(new Uint8Array(received[0].body), pushPrivJwk, subPub, auth);
    check("push payload decrypts to test notification", body.includes("Test notification"), body);
  }
  // Offline push: Bob closes WS, Alice sends a plaintext message, Bob's
  // push endpoint must get "New message from Live Alice".
  bobWs.close();
  await new Promise((r) => setTimeout(r, 300));
  await req(`/api/chats/${chat.id}/messages`, alice.token, "POST", { content: "hello while you are offline" });
  await new Promise((r) => setTimeout(r, 500));
  check("offline message triggered a push", received.length === 2, `got ${received.length}`);
  if (received.length === 2) {
    const body = await decryptPush(new Uint8Array(received[1].body), pushPrivJwk, subPub, auth);
    check("offline push body names the sender, not the content", body.includes("livealice") && !body.includes("hello while you are offline"), body);
  }

  aliceWs.close();
  // Unsubscribe so a re-run does not accumulate this run's dead endpoint.
  await req("/api/push/subscribe", bob.token, "DELETE", { endpoint: "http://127.0.0.1:" + pushPort + "/push" }).catch(() => undefined);
  pushSrv.close();
  console.log(failures === 0 ? "\nALL LIVE CHECKS PASSED" : `\n${failures} LIVE CHECK(S) FAILED`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => { console.error("LIVE TEST ERROR:", e); process.exit(1); });
