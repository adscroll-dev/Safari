"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "../lib/api";
import { dict, isRTL, type Lang } from "../lib/i18n";
import * as e2eeStore from "../lib/e2ee-store";
import * as e2ee from "../lib/e2ee";
import { b64decode } from "../lib/b64";
import * as pushClient from "../lib/push-client";
import { useCall } from "../lib/use-call";
import { VoiceRecorder } from "../components/VoiceRecorder";
import { assistantReply, translateLocal, detectLang, type Translation } from "../lib/local-ai";

// Delivery/read ticks (WhatsApp-style): ✓ sent, ✓✓ delivered, ✓✓ blue = read.
function Ticks({ m }: { m: api.Message }) {
  if (m.status === "read") return <span style={{ color: "#2563eb" }}>✓✓</span>;
  if (m.status === "delivered") return <span>✓✓</span>;
  return <span>✓</span>;
}

function formatSize(n?: number): string {
  if (!n) return "";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

// Renders a media message: fetches the stored file with the login token
// (GET /uploads/* requires auth) and shows an image / video / audio
// player, or a download button for other files. Text stays a caption.
function MediaContent({ m }: { m: api.Message }) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [loadError, setLoadError] = useState("");
  const isMedia = m.kind !== "text" && !!m.file_url;
  useEffect(() => {
    if (!isMedia || !m.file_url) return;
    let alive = true;
    api.fetchUploadBlobUrl(m.file_url)
      .then((u) => { if (alive) setBlobUrl(u); })
      .catch((e) => { if (alive) setLoadError(e instanceof Error ? e.message : "Could not load file"); });
    return () => { alive = false; if (blobUrl) URL.revokeObjectURL(blobUrl); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isMedia, m.file_url]);
  if (m.kind === "location" && m.lat !== undefined && m.lng !== undefined) {
    const mapUrl = `https://www.openstreetmap.org/?mlat=${m.lat}&mlon=${m.lng}#map=15/${m.lat}/${m.lng}`;
    return (
      <div>
        <div>📍 {m.location_label || "Location"}</div>
        <div style={{ fontSize: 12 }}>{m.lat.toFixed(5)}, {m.lng.toFixed(5)}</div>
        <a href={mapUrl} target="_blank" rel="noreferrer" style={{ color: "inherit", fontSize: 12 }}>Open map ↗</a>
        {m.content && <div>{m.content}</div>}
      </div>
    );
  }
  if (m.kind === "sticker" && blobUrl) {
    return (
      <div>
        <img src={blobUrl} alt={m.file_name ?? "sticker"} style={{ maxWidth: 140, borderRadius: 8 }} />
        {m.content && <div>{m.content}</div>}
      </div>
    );
  }
  if (!isMedia) return <div>{m.content}</div>;
  return (
    <div>
      {loadError && <div style={{ fontSize: 12, color: "red" }}>{loadError}</div>}
      {blobUrl && m.kind === "image" && (
        <img src={blobUrl} alt={m.file_name ?? "image"} style={{ maxWidth: "100%", borderRadius: 8 }} />
      )}
      {blobUrl && m.kind === "video" && (
        <video src={blobUrl} controls style={{ maxWidth: "100%", borderRadius: 8 }} />
      )}
      {blobUrl && m.kind === "voice" && <audio src={blobUrl} controls style={{ maxWidth: "100%" }} />}
      {blobUrl && m.kind === "file" && (
        <a href={blobUrl} download={m.file_name ?? undefined} style={{ color: "inherit" }}>
          📄 {m.file_name ?? "file"} {formatSize(m.file_size) && `(${formatSize(m.file_size)})`}
        </a>
      )}
      {!blobUrl && !loadError && <div style={{ fontSize: 12, opacity: 0.7 }}>Loading file…</div>}
      {m.content && <div>{m.content}</div>}
      {m.kind === "file" && m.file_name && <div style={{ fontSize: 11, opacity: 0.7 }}>{m.file_name}</div>}
    </div>
  );
}

export default function Home() {
  // English is the default language for new users; the choice is
  // saved (vyro_lang) so Persian and other languages stay available.
  const [lang, setLangState] = useState<Lang>(() => {
    if (typeof window === "undefined") return "en";
    const saved = window.localStorage.getItem("vyro_lang");
    return saved === "fa" ? "fa" : "en";
  });
  const setLang = (l: Lang) => {
    setLangState(l);
    try {
      window.localStorage.setItem("vyro_lang", l);
    } catch {
      // storage unavailable; language still applies for this session
    }
  };
  const [dark, setDark] = useState(false);
  const [user, setUser] = useState<api.User | null>(null);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [chats, setChats] = useState<api.Chat[]>([]);
  const [users, setUsers] = useState<api.User[]>([]);
  const [activeChat, setActiveChat] = useState<api.Chat | null>(null);
  const [messages, setMessages] = useState<api.Message[]>([]);
  const [draft, setDraft] = useState("");
  const [query, setQuery] = useState("");
  const [showUsers, setShowUsers] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [stories, setStories] = useState<api.Story[]>([]);
  const [typingUser, setTypingUser] = useState("");
  const [ttl, setTtl] = useState(0); // disappearing-message TTL in seconds (0 = off)
  const [archivedOpen, setArchivedOpen] = useState(false);
  // v0.5: E2EE (per-DM toggle), push, calls.
  const [e2eeSupported, setE2eeSupported] = useState<boolean | null>(null);
  const [e2eeChats, setE2eeChats] = useState<Record<string, boolean>>({});
  const [decrypted, setDecrypted] = useState<Record<string, string>>({});
  const [safetyNo, setSafetyNo] = useState("");
  const [pushState, setPushState] = useState("");
  const [pushNote, setPushNote] = useState("");
  // v0.6: local assistant + local dictionary translation (on-device only).
  const [assistantOpen, setAssistantOpen] = useState(false);
  const [assistantQ, setAssistantQ] = useState("");
  const [assistantLog, setAssistantLog] = useState<{ q: string; a: string }[]>([]);
  const [translations, setTranslations] = useState<Record<string, Translation>>({});
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const activeChatRef = useRef<api.Chat | null>(null);
  const userRef = useRef<api.User | null>(null);
  const lastTypingAt = useRef(0);
  const typingTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const t = dict[lang];

  useEffect(() => { userRef.current = user; }, [user]);

  const sendSignal = useCallback((chatId: string, type: string, toUserId: string, payload: unknown) => {
    api.sendCallSignal(wsRef.current, chatId, type, toUserId, payload);
  }, []);
  const call = useCall(sendSignal);

  function peerIdOf(chat: api.Chat | null): string | null {
    if (!chat || chat.type !== "dm" || !userRef.current) return null;
    return chat.member_ids.find((id) => id !== userRef.current!.id) ?? null;
  }

  // Decrypt one E2EE message into the cache (best effort; failures show
  // a lock placeholder in the bubble).
  async function decryptIntoCache(m: api.Message, peerId: string) {
    try {
      const pt = await e2eeStore.decryptMessage(peerId, m.content);
      setDecrypted((prev) => ({ ...prev, [m.id]: pt }));
    } catch {
      setDecrypted((prev) => ({ ...prev, [m.id]: "🔒 Encrypted message — could not decrypt on this device." }));
    }
  }

  async function enableE2EE(chat: api.Chat) {
    const peerId = peerIdOf(chat);
    if (!peerId) return;
    setError("");
    try {
      const got = await e2eeStore.getSession(peerId);
      if (!got) { setError("Encrypted chat is not available on this device."); return; }
      setE2eeChats((prev) => ({ ...prev, [chat.id]: true }));
      const ident = e2eeStore.loadIdentity();
      if (ident) {
        const fp = await e2ee.fingerprint(ident.dhPub, b64decode(got.peerDhPub));
        setSafetyNo(fp);
      }
      // Decrypt any already-loaded encrypted messages in this chat.
      messages.forEach((m) => { if (m.kind === "e2ee") void decryptIntoCache(m, peerId); });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not start encrypted chat (has the peer set up E2EE yet?)");
    }
  }

  useEffect(() => {
    document.documentElement.dataset.theme = dark ? "dark" : "light";
  }, [dark]);

  async function refreshChats() {
    try { setChats(await api.listChats()); } catch { /* not logged in yet */ }
  }

  useEffect(() => {
    api.me().then((u) => {
      setUser(u);
      refreshChats();
      api.listStories().then(setStories).catch(() => undefined);
      // v0.5: set up this device's E2EE identity (once) and read the
      // push subscription state. Both are best effort.
      void e2ee.e2eeSupported().then(async (ok) => {
        setE2eeSupported(ok);
        if (!ok) return;
        try { await e2eeStore.ensureDeviceKeys(); } catch { /* E2EE stays off */ }
      });
      void pushClient.pushStatus().then((s) => setPushState(s)).catch(() => undefined);
      wsRef.current = api.connectWS((ev) => {
        if (call.handleCallEvent(ev)) return;
        if (ev.type === "message" && ev.message) {
          const m = ev.message;
          setMessages((prev) => (activeChatRef.current && m.chat_id === activeChatRef.current.id ? [...prev, m] : prev));
          if (m.kind === "e2ee" && activeChatRef.current) {
            const peerId = peerIdOf(activeChatRef.current);
            if (peerId) void decryptIntoCache(m, peerId);
          }
          refreshChats();
        } else if (ev.type === "typing" && ev.chat_id === activeChatRef.current?.id) {
          setTypingUser(ev.user_id ?? "Someone");
          if (typingTimer.current) clearTimeout(typingTimer.current);
          typingTimer.current = setTimeout(() => setTypingUser(""), 3000);
        } else if ((ev.type === "message_read" || ev.type === "message_delivered") && ev.message_id) {
          const status = ev.type === "message_read" ? "read" : "delivered";
          setMessages((prev) => prev.map((m) => (m.id === ev.message_id ? { ...m, status } : m)));
        }
      });
    }).catch(() => undefined);
    return () => wsRef.current?.close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => { activeChatRef.current = activeChat; }, [activeChat]);

  useEffect(() => {
    if (!activeChat) return;
    api.listMessages(activeChat.id).then((msgs) => {
      setMessages(msgs);
      // Decrypt encrypted messages in this DM (if E2EE is on for it).
      const peerId = peerIdOf(activeChat);
      if (peerId && e2eeChats[activeChat.id]) {
        msgs.forEach((m) => { if (m.kind === "e2ee") void decryptIntoCache(m, peerId); });
      }
      // Opening a chat marks its newest message from others as read.
      const fromOthers = msgs.filter((m) => m.sender_id !== user?.id);
      const last = fromOthers[fromOthers.length - 1];
      if (last) void api.markRead(last.id).catch(() => undefined);
    }).catch((e) => setError(e.message));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeChat]);

  // v0.6: email verification — register/login send a 6-digit code;
  // no token until the code is verified.
  const [codeStep, setCodeStep] = useState(false);
  const [codeEmail, setCodeEmail] = useState("");
  const [codePurpose, setCodePurpose] = useState<"register" | "login">("login");
  const [code, setCode] = useState("");
  const [resendIn, setResendIn] = useState(0);

  useEffect(() => {
    if (resendIn <= 0) return;
    const t = setTimeout(() => setResendIn(resendIn - 1), 1000);
    return () => clearTimeout(t);
  }, [resendIn]);

  async function submitAuth(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const out = mode === "login" ? await api.login(email, password) : await api.register(email, username, password);
      if (out.codeSent) {
        setCodeEmail(out.email);
        setCodePurpose(out.purpose);
        setCodeStep(true);
        setResendIn(60);
        return;
      }
      setUser(out.user);
      await refreshChats();
    } catch (err) { setError(err instanceof Error ? err.message : "Auth failed"); }
  }

  async function submitCode(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!/^\d{6}$/.test(code.trim())) {
      setError("Enter the 6-digit code from your email");
      return;
    }
    try {
      const u = codePurpose === "register"
        ? await api.verifyRegister(codeEmail, code.trim())
        : await api.verifyLogin(codeEmail, code.trim());
      setCodeStep(false);
      setUser(u);
      await refreshChats();
    } catch (err) { setError(err instanceof Error ? err.message : "Verification failed"); }
  }

  async function resend() {
    if (resendIn > 0) return;
    try {
      await api.resendCode(codeEmail, codePurpose);
      setResendIn(60);
    } catch (err) { setError(err instanceof Error ? err.message : "Resend failed"); }
  }

  async function send(e: React.FormEvent) {
    e.preventDefault();
    if (!activeChat || !draft.trim()) return;
    const content = draft;
    setDraft("");
    try {
      // v0.5: when E2EE is on for this DM, encrypt locally and send
      // the ciphertext envelope; the server never sees plaintext.
      if (e2eeChats[activeChat.id]) {
        const peerId = peerIdOf(activeChat);
        if (!peerId) throw new Error("Encrypted chat needs a DM peer");
        const got = await e2eeStore.getSession(peerId);
        if (!got) throw new Error("No encrypted session for this peer");
        const env = await got.session.encrypt(content);
        e2eeStore.saveSession(peerId, got.session);
        const msg = await api.sendE2EEMessage(activeChat.id, JSON.stringify(env));
        setMessages((prev) => [...prev, msg]);
        setDecrypted((prev) => ({ ...prev, [msg.id]: content }));
        return;
      }
      const msg = await api.sendMessage(activeChat.id, content, ttl ? { ttlSeconds: ttl } : undefined);
      setMessages((prev) => [...prev, msg]);
    } catch (err) { setError(err instanceof Error ? err.message : "Send failed"); }
  }

  // Pick a file -> upload it (POST /api/uploads) -> send it as a media
  // message, using the text box as an optional caption.
  async function handleFile(file: File) {
    if (!activeChat) return;
    setUploading(true);
    setError("");
    try {
      const up = await api.uploadFile(file);
      const caption = draft;
      setDraft("");
      const msg = await api.sendMediaMessage(activeChat.id, up, caption);
      setMessages((prev) => [...prev, msg]);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  // v0.6: send a recorded voice message — upload the MediaRecorder
  // blob (POST /api/uploads) then send it as kind "voice". Voice
  // messages are NOT end-to-end encrypted (the upload is plaintext
  // to the server), so the recorder is disabled in encrypted chats.
  async function handleVoice(blob: Blob, durationSec: number) {
    if (!activeChat) return;
    setUploading(true);
    setError("");
    try {
      const ext = blob.type.includes("mp4") ? "m4a" : blob.type.includes("ogg") ? "ogg" : "webm";
      const file = new File([blob], `voice-${Date.now()}.${ext}`, { type: blob.type || "audio/webm" });
      const up = await api.uploadFile(file);
      const msg = await api.sendVoiceMessage(activeChat.id, up, durationSec);
      setMessages((prev) => [...prev, msg]);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not send voice message");
    } finally {
      setUploading(false);
    }
  }

  // v0.6: ask the LOCAL assistant (rule-based, on-device — no network).
  function askAssistant() {
    const q = assistantQ.trim();
    if (!q) return;
    const { text } = assistantReply(q);
    setAssistantLog((prev) => [...prev, { q, a: text }]);
    setAssistantQ("");
  }

  // v0.6: translate a text message with the LOCAL dictionary
  // (English <-> Persian). Tapping again hides the translation.
  function toggleTranslate(m: api.Message) {
    if (translations[m.id]) {
      setTranslations((prev) => {
        const next = { ...prev };
        delete next[m.id];
        return next;
      });
      return;
    }
    const to = detectLang(m.content) === "fa" ? "en" : "fa";
    setTranslations((prev) => ({ ...prev, [m.id]: translateLocal(m.content, to) }));
  }

  function onDraftChange(v: string) {
    setDraft(v);
    if (!activeChat) return;
    const now = Date.now();
    if (now - lastTypingAt.current > 2000) {
      lastTypingAt.current = now;
      api.sendTyping(wsRef.current, activeChat.id);
    }
  }

  // Share the device's current location (browser geolocation).
  async function shareLocation() {
    if (!activeChat || !navigator.geolocation) { setError("Geolocation is not available in this browser"); return; }
    navigator.geolocation.getCurrentPosition(async (pos) => {
      try {
        const msg = await api.sendLocation(activeChat.id, pos.coords.latitude, pos.coords.longitude);
        setMessages((prev) => [...prev, msg]);
      } catch (err) { setError(err instanceof Error ? err.message : "Could not share location"); }
    }, () => setError("Location permission denied"));
  }

  // Schedule the draft to be sent later (minutes from now).
  async function scheduleDraft() {
    if (!activeChat || !draft.trim()) return;
    const minutes = Number(prompt("Send in how many minutes?", "60"));
    if (!minutes || minutes <= 0) return;
    try {
      const sendAt = new Date(Date.now() + minutes * 60000).toISOString();
      await api.scheduleMessage(activeChat.id, draft, sendAt);
      setDraft("");
      setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Could not schedule message"); }
  }

  // Post a photo/video as a 24h story.
  async function postStory(file: File) {
    setUploading(true);
    try {
      const up = await api.uploadFile(file);
      const story = await api.createStory(up.url);
      setStories((prev) => [story, ...prev]);
    } catch (err) { setError(err instanceof Error ? err.message : "Could not post story"); }
    finally { setUploading(false); }
  }

  async function openUsers() {
    setShowUsers(true);
    try { setUsers(await api.listUsers()); } catch (e) { setError(e instanceof Error ? e.message : "Failed to load users"); }
  }

  async function startDM(other: api.User) {
    try {
      const chat = await api.createChat("dm", [other.id]);
      setChats((prev) => [chat, ...prev.filter((c) => c.id !== chat.id)]);
      setActiveChat(chat);
      setShowUsers(false);
    } catch (err) { setError(err instanceof Error ? err.message : "Could not start chat"); }
  }

  async function doSearch(q: string) {
    setQuery(q);
    if (!q.trim()) { if (activeChat) setMessages(await api.listMessages(activeChat.id)); return; }
    try { setMessages(await api.searchMessages(q)); } catch { /* ignore */ }
  }

  if (!user) {
    return (
      <main dir={isRTL(lang) ? "rtl" : "ltr"} style={{ maxWidth: 420, margin: "10vh auto", padding: 24, background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 16 }}>
        <h1 style={{ margin: 0, background: "linear-gradient(135deg,#2563eb,#7c3aed)", WebkitBackgroundClip: "text", color: "transparent" }}>VYRO</h1>
        <p style={{ color: "var(--muted)" }}>{t.tagline}</p>
        <div style={{ display: "flex", gap: 8, marginBottom: 12 }}>
          <button onClick={() => setLang(lang === "en" ? "fa" : "en")}>{lang === "en" ? "فارسی" : "English"}</button>
          <button onClick={() => setDark(!dark)}>{t.theme}: {dark ? "Dark" : "Light"}</button>
        </div>
        {codeStep ? (
          <>
            <p>We sent a 6-digit verification code to <b>{codeEmail}</b>. It expires in 10 minutes. Never share it with anyone.</p>
            <form onSubmit={submitCode} style={{ display: "grid", gap: 8 }}>
              <input placeholder="6-digit code" value={code} onChange={(e) => setCode(e.target.value)} required inputMode="numeric" maxLength={6} style={{ textAlign: "center", fontSize: 22, letterSpacing: 8 }} />
              <button type="submit">Verify and continue</button>
            </form>
            <button onClick={resend} disabled={resendIn > 0} style={{ marginTop: 8 }}>
              {resendIn > 0 ? `Resend code in ${resendIn}s` : "Resend code"}
            </button>
            <button onClick={() => setCodeStep(false)} style={{ marginTop: 8 }}>Back to login</button>
          </>
        ) : (
          <>
            <form onSubmit={submitAuth} style={{ display: "grid", gap: 8 }}>
              <input placeholder={t.email} value={email} onChange={(e) => setEmail(e.target.value)} required type="email" />
              {mode === "register" && <input placeholder={t.username} value={username} onChange={(e) => setUsername(e.target.value)} required minLength={3} />}
              <input placeholder={t.password} value={password} onChange={(e) => setPassword(e.target.value)} required type="password" minLength={8} />
              <button type="submit">{mode === "login" ? t.login : t.register}</button>
            </form>
            <button onClick={() => setMode(mode === "login" ? "register" : "login")} style={{ marginTop: 8 }}>
              {mode === "login" ? t.register : t.login}
            </button>
          </>
        )}
        {error && <p style={{ color: "red" }}>{error}</p>}
        <p style={{ color: "var(--muted)", fontSize: 13 }}>Backend must be running on http://localhost:8080 — see ../README.md</p>
      </main>
    );
  }

  return (
    <main dir={isRTL(lang) ? "rtl" : "ltr"} style={{ display: "grid", gridTemplateColumns: "280px 1fr", height: "100vh" }}>
      <aside style={{ borderInlineEnd: "1px solid var(--border)", padding: 12, overflow: "auto", background: "var(--surface)" }}>
        <h2 style={{ margin: "0 0 4px" }}>VYRO</h2>
        <p style={{ color: "var(--muted)", margin: "0 0 12px", fontSize: 13 }}>{user.display_name} · @{user.username}</p>
        <div style={{ display: "flex", gap: 6, marginBottom: 12 }}>
          <button onClick={openUsers}>{t.users}</button>
          <button onClick={() => setLang(lang === "en" ? "fa" : "en")}>{lang === "en" ? "فارسی" : "English"}</button>
          <button onClick={() => setDark(!dark)}>{dark ? "☀️" : "🌙"}</button>
          <button onClick={() => { api.logout(); location.reload(); }}>{t.logout}</button>
        </div>
        {/* v0.5: Web Push (VAPID) — free browser notifications. */}
        <div style={{ display: "flex", gap: 6, alignItems: "center", marginBottom: 12, fontSize: 13 }}>
          <button
            onClick={async () => {
              const note = pushState === "subscribed"
                ? await pushClient.unsubscribePush()
                : await pushClient.subscribePush();
              setPushNote(note);
              setPushState(await pushClient.pushStatus().catch(() => ""));
            }}
          >
            {pushState === "subscribed" ? "🔔 Notifications on" : "🔕 Enable notifications"}
          </button>
          {pushState === "subscribed" && (
            <button onClick={async () => setPushNote(await pushClient.testPush())}>Test</button>
          )}
        </div>
        {pushNote && <p style={{ fontSize: 12, color: "var(--muted)", margin: "0 0 8px" }}>{pushNote}</p>}
        {e2eeSupported === false && (
          <p style={{ fontSize: 12, color: "var(--muted)", margin: "0 0 8px" }}>🔒 Encrypted chat is not supported in this browser (needs X25519 WebCrypto).</p>
        )}
        <h3>{t.chats}</h3>
        {/* Stories strip: 24h status, like WhatsApp Status / Telegram stories. */}
        <div style={{ display: "flex", gap: 8, overflowX: "auto", paddingBottom: 8, marginBottom: 8 }}>
          <label style={{ flexShrink: 0, width: 56, textAlign: "center", cursor: "pointer", fontSize: 12 }}>
            <span style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 48, height: 48, borderRadius: "50%", background: "var(--border)", fontSize: 20, margin: "0 auto 2px" }}>＋</span>
            Story
            <input type="file" accept="image/*,video/*" style={{ display: "none" }} onChange={(e) => { const f = e.target.files?.[0]; if (f) void postStory(f); }} />
          </label>
          {stories.map((st) => (
            <div key={st.id} style={{ flexShrink: 0, width: 56, textAlign: "center", fontSize: 11, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              <span style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 48, height: 48, borderRadius: "50%", background: "linear-gradient(135deg,#2563eb,#7c3aed)", color: "#fff", fontSize: 18, margin: "0 auto 2px" }}>◉</span>
              {st.caption || "Story"}
            </div>
          ))}
        </div>
        {chats.length === 0 && <p style={{ color: "var(--muted)" }}>{t.noChats}</p>}
        {chats.filter((c) => archivedOpen || !c.archived).map((c) => (
          <button key={c.id} onClick={() => setActiveChat(c)} style={{ display: "flex", alignItems: "center", gap: 10, width: "100%", textAlign: "start", padding: "10px 8px", marginBottom: 4, background: activeChat?.id === c.id ? "var(--border)" : "transparent", border: "1px solid var(--border)", borderRadius: 12 }}>
            <span style={{ flexShrink: 0, display: "flex", alignItems: "center", justifyContent: "center", width: 44, height: 44, borderRadius: "50%", background: "linear-gradient(135deg,#2563eb,#7c3aed)", color: "#fff", fontWeight: 700, fontSize: 18 }}>
              {(c.title || c.type).charAt(0).toUpperCase()}
            </span>
            <span style={{ flex: 1, minWidth: 0 }}>
              <span style={{ display: "block", fontWeight: 600, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {c.type === "group" ? "👥 " : c.type === "channel" ? "📣 " : ""}{c.title || c.type}
                {c.pinned ? " 📌" : ""}{c.folder ? ` · 📁 ${c.folder}` : ""}
              </span>
              <span style={{ display: "block", fontSize: 12, color: "var(--muted)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {c.type === "dm" ? "Direct message" : c.type === "group" ? "Group" : "Channel"}
                {c.archived ? " · archived" : ""}
              </span>
            </span>
          </button>
        ))}
        {chats.some((c) => c.archived) && (
          <button onClick={() => setArchivedOpen(!archivedOpen)} style={{ marginTop: 4 }}>
            {archivedOpen ? "Hide archived" : `Archived (${chats.filter((c) => c.archived).length})`}
          </button>
        )}
        {showUsers && (
          <div>
            <h3>{t.users}</h3>
            {users.map((u) => (
              <button key={u.id} onClick={() => startDM(u)} style={{ display: "block", width: "100%", textAlign: "start", padding: 8, marginBottom: 4 }}>
                {u.display_name} (@{u.username})
              </button>
            ))}
          </div>
        )}
      </aside>
      <section style={{ display: "flex", flexDirection: "column", padding: 16 }}>
        {activeChat ? (
          <>
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8 }}>
              <span style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 40, height: 40, borderRadius: "50%", background: "linear-gradient(135deg,#2563eb,#7c3aed)", color: "#fff", fontWeight: 700 }}>
                {(activeChat.title || activeChat.type).charAt(0).toUpperCase()}
              </span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 700, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{activeChat.title || activeChat.type}</div>
                <div style={{ fontSize: 12, color: "var(--muted)" }}>{typingUser ? `${typingUser} is typing…` : activeChat.type}{e2eeChats[activeChat.id] ? " · 🔒 End-to-end encrypted" : ""}{safetyNo && e2eeChats[activeChat.id] ? ` · Safety № ${safetyNo}` : ""}</div>
              </div>
              {/* v0.5: E2EE toggle + voice call (DMs only). */}
              {activeChat.type === "dm" && (
                <button
                  title={e2eeSupported === false ? "Encrypted chat is not supported in this browser" : e2eeChats[activeChat.id] ? "End-to-end encrypted — click to turn off" : "Turn on end-to-end encryption"}
                  disabled={e2eeSupported === false}
                  onClick={async () => {
                    if (e2eeChats[activeChat.id]) {
                      setE2eeChats((prev) => ({ ...prev, [activeChat.id]: false }));
                      setSafetyNo("");
                    } else {
                      await enableE2EE(activeChat);
                    }
                  }}
                >
                  {e2eeChats[activeChat.id] ? "🔒" : "🔓"}
                </button>
              )}
              {activeChat.type === "dm" && (
                <button
                  title="Voice call (audio only in v0.5)"
                  disabled={call.callState !== "idle"}
                  onClick={() => {
                    const peerId = peerIdOf(activeChat);
                    if (peerId) void call.startCall(activeChat.id, peerId, activeChat.title || "peer");
                  }}
                >
                  📞
                </button>
              )}
              <button title={t.assistant} onClick={() => setAssistantOpen(!assistantOpen)}>🤖</button>
              <button title={activeChat.pinned ? "Unpin chat" : "Pin chat"} onClick={async () => { const c = activeChat.pinned ? await api.unpinChat(activeChat.id) : await api.pinChat(activeChat.id); setChats((p) => p.map((x) => (x.id === c.id ? c : x))); setActiveChat(c); }}>📌</button>
              <button title={activeChat.archived ? "Unarchive chat" : "Archive chat"} onClick={async () => { const c = activeChat.archived ? await api.unarchiveChat(activeChat.id) : await api.archiveChat(activeChat.id); setChats((p) => p.map((x) => (x.id === c.id ? c : x))); setActiveChat(c); }}>🗄️</button>
            </div>
            <input placeholder={t.search} value={query} onChange={(e) => doSearch(e.target.value)} style={{ marginBottom: 12, padding: 8 }} />
            {/* v0.5: incoming / active call bar. */}
            {call.callState === "incoming" && call.incoming && (
              <div style={{ display: "flex", gap: 8, alignItems: "center", padding: 10, marginBottom: 8, border: "1px solid var(--border)", borderRadius: 12, background: "var(--surface)" }}>
                <span>📞 Incoming voice call…</span>
                <button onClick={() => void call.acceptCall()}>Accept</button>
                <button onClick={() => call.declineCall()}>Decline</button>
              </div>
            )}
            {(call.callState === "outgoing" || call.callState === "in-call") && (
              <div style={{ display: "flex", gap: 8, alignItems: "center", padding: 10, marginBottom: 8, border: "1px solid var(--border)", borderRadius: 12, background: "var(--surface)" }}>
                <span>{call.callState === "outgoing" ? `📞 Calling ${call.callPeer}…` : `📞 In call with ${call.callPeer}`}</span>
                <button onClick={() => call.hangup()}>Hang up</button>
              </div>
            )}
            {call.callError && <p style={{ color: "red", fontSize: 13 }}>{call.callError}</p>}
            {/* v0.6: local assistant panel — rule-based, on-device only. */}
            {assistantOpen && (
              <div style={{ padding: 10, marginBottom: 8, border: "1px solid var(--border)", borderRadius: 12, background: "var(--surface)" }}>
                <div style={{ fontSize: 12, color: "var(--muted)", marginBottom: 8 }}>🤖 {t.assistant} — {t.assistantNote}</div>
                <div style={{ maxHeight: 180, overflow: "auto", display: "flex", flexDirection: "column", gap: 6, marginBottom: 8 }}>
                  {assistantLog.length === 0 && <div style={{ fontSize: 13, color: "var(--muted)" }}>Ask how to use VYRO — e.g. “how do voice messages work?”</div>}
                  {assistantLog.map((entry, i) => (
                    <div key={i} style={{ fontSize: 13 }}>
                      <div><b>You:</b> {entry.q}</div>
                      <div><b>Assistant:</b> {entry.a}</div>
                    </div>
                  ))}
                </div>
                <form
                  onSubmit={(e) => { e.preventDefault(); askAssistant(); }}
                  style={{ display: "flex", gap: 6 }}
                >
                  <input value={assistantQ} onChange={(e) => setAssistantQ(e.target.value)} placeholder={t.askAssistant} style={{ flex: 1, padding: "8px 10px" }} />
                  <button type="submit">{t.send}</button>
                </form>
              </div>
            )}
            <audio ref={call.remoteAudioRef} autoPlay style={{ display: "none" }} />
            <div style={{ flex: 1, overflow: "auto", display: "flex", flexDirection: "column", gap: 8 }}>
              {messages.map((m) => (
                <div key={m.id} style={{ alignSelf: m.sender_id === user.id ? "flex-end" : "flex-start", maxWidth: "70%", padding: "8px 12px", borderRadius: 12, background: m.sender_id === user.id ? "var(--bubble-me)" : "var(--surface)", color: m.sender_id === user.id ? "#fff" : "var(--text)", border: "1px solid var(--border)" }}>
                  {m.kind === "e2ee" ? (
                    <div>{decrypted[m.id] ?? "🔒 Encrypted message…"}</div>
                  ) : (
                    <MediaContent m={m} />
                  )}
                  {/* v0.6: local dictionary translation for text messages. */}
                  {m.kind === "text" && translations[m.id] && (
                    <div style={{ marginTop: 6, paddingTop: 6, borderTop: "1px dashed var(--border)", fontSize: 13 }}>
                      <div>{translations[m.id].text}</div>
                      <div style={{ fontSize: 11, opacity: 0.7 }}>
                        {t.translating} · {translations[m.id].knownWords}/{translations[m.id].totalWords} words known ({Math.round(translations[m.id].coverage * 100)}%) — unknown words are left unchanged
                      </div>
                    </div>
                  )}
                  <div style={{ fontSize: 11, opacity: 0.7 }}>{new Date(m.created_at).toLocaleTimeString()}{m.edited_at ? " · edited" : ""}{m.sender_id === user.id ? <> <Ticks m={m} /></> : ""}</div>
                  {m.kind === "text" && (
                    <div style={{ display: "flex", gap: 6, marginTop: 4 }}>
                      <button onClick={() => toggleTranslate(m)}>🌐 {translations[m.id] ? "Hide translation" : t.translate}</button>
                    </div>
                  )}
                  {m.sender_id === user.id && (
                    <div style={{ display: "flex", gap: 6, marginTop: 4 }}>
                      <button onClick={async () => { const next = prompt("Edit message", m.content); if (next) { const u = await api.editMessage(m.id, next); setMessages((p) => p.map((x) => (x.id === m.id ? u : x))); } }}>Edit</button>
                      <button onClick={async () => { await api.deleteMessage(m.id); setMessages((p) => p.filter((x) => x.id !== m.id)); }}>Delete</button>
                    </div>
                  )}
                </div>
              ))}
            </div>
            <form onSubmit={send} style={{ display: "flex", gap: 8, marginTop: 12, alignItems: "center" }}>
              <input
                ref={fileInputRef}
                type="file"
                style={{ display: "none" }}
                onChange={(e) => { const f = e.target.files?.[0]; if (f) void handleFile(f); }}
              />
              <button
                type="button"
                title={t.attach}
                disabled={uploading}
                onClick={() => fileInputRef.current?.click()}
                style={{ fontSize: 20, padding: "10px 14px" }}
              >
                {uploading ? t.uploading : "📎"}
              </button>
              <button type="button" title="Share location" onClick={shareLocation} style={{ fontSize: 20, padding: "10px 14px" }}>📍</button>
              {/* v0.6: voice recording (MediaRecorder). Disabled in E2EE
                  chats: voice uploads are not end-to-end encrypted. */}
              <VoiceRecorder
                disabled={uploading || !!e2eeChats[activeChat.id]}
                onRecorded={(blob, duration) => void handleVoice(blob, duration)}
                onError={(msg) => setError(msg)}
              />
              <input value={draft} onChange={(e) => onDraftChange(e.target.value)} placeholder={t.typeMessage} style={{ flex: 1, padding: "12px 14px", fontSize: 16 }} />
              <select value={ttl} onChange={(e) => setTtl(Number(e.target.value))} title="Disappearing message" style={{ padding: "10px 6px" }}>
                <option value={0}>⏱ Off</option>
                <option value={60}>⏱ 1 min</option>
                <option value={3600}>⏱ 1 hour</option>
                <option value={86400}>⏱ 1 day</option>
              </select>
              <button type="button" title="Schedule message" onClick={scheduleDraft} style={{ fontSize: 18, padding: "10px 12px" }}>🕐</button>
              <button type="submit" style={{ fontSize: 16, padding: "12px 18px" }}>{t.send}</button>
            </form>
          </>
        ) : (
          <p style={{ color: "var(--muted)" }}>{t.noChats}</p>
        )}
        {error && <p style={{ color: "red" }}>{error}</p>}
      </section>
    </main>
  );
}
