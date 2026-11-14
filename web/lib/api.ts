// Minimal typed client for the VYRO Go backend.
// No fake data: every function calls the real API and throws on error.
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface User { id: string; email: string; username: string; display_name: string; avatar_url?: string; is_online: boolean }
export interface Chat {
  id: string; type: "dm" | "group" | "channel"; title?: string; member_ids: string[];
  created_at: string; roles?: Record<string, string>;
  pinned?: boolean; archived?: boolean; folder?: string;
}
export interface Message {
  id: string; chat_id: string; sender_id: string; content: string; kind: string;
  reply_to_id?: string; edited_at?: string; reactions?: Record<string, string[]>; created_at: string;
  file_url?: string; file_name?: string; file_size?: number; mime_type?: string;
  // v0.4: location, delivery/read ticks, disappearing messages.
  lat?: number; lng?: number; location_label?: string;
  delivered_to?: string[]; read_by?: string[];
  status?: "sent" | "delivered" | "read";
  expires_at?: string;
}
export interface Story {
  id: string; user_id: string; media_url: string; caption?: string;
  created_at: string; expires_at: string;
}
export interface ScheduledMessage {
  id: string; chat_id: string; sender_id: string; content: string; kind: string;
  send_at: string; sent: boolean; created_at: string;
}

export interface UploadResult {
  url: string; file_name: string; file_size: number; mime_type: string; kind: string;
}

function token(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("vyro_token");
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(token() ? { Authorization: `Bearer ${token()}` } : {}), ...(init?.headers ?? {}) },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

// v0.6: email verification. register/login only send a 6-digit code;
// the token is issued by the verify step. Both return the raw
// response: {status:"code_sent",...} or, on an older server,
// {user, token}.
export interface AuthCodeSent {
  status: "code_sent"; email: string; purpose: "register" | "login"; expires_in_seconds: number;
}
export interface AuthResult { user: User; token: string }

export async function register(email: string, username: string, password: string) {
  const out = await req<AuthCodeSent | AuthResult>("/api/auth/register", {
    method: "POST", body: JSON.stringify({ email, username, password, display_name: username }),
  });
  if ("token" in out) {
    localStorage.setItem("vyro_token", out.token);
    return { user: out.user, codeSent: false as const };
  }
  return { user: null, codeSent: true as const, email: out.email, purpose: out.purpose };
}

export async function login(email: string, password: string) {
  const out = await req<AuthCodeSent | AuthResult>("/api/auth/login", {
    method: "POST", body: JSON.stringify({ email, password }),
  });
  if ("token" in out) {
    localStorage.setItem("vyro_token", out.token);
    return { user: out.user, codeSent: false as const };
  }
  return { user: null, codeSent: true as const, email: out.email, purpose: out.purpose };
}

export async function verifyRegister(email: string, code: string) {
  const out = await req<AuthResult>("/api/auth/verify-register", {
    method: "POST", body: JSON.stringify({ email, code }),
  });
  localStorage.setItem("vyro_token", out.token);
  return out.user;
}

export async function verifyLogin(email: string, code: string) {
  const out = await req<AuthResult>("/api/auth/verify-login", {
    method: "POST", body: JSON.stringify({ email, code }),
  });
  localStorage.setItem("vyro_token", out.token);
  return out.user;
}

export async function resendCode(email: string, purpose: "register" | "login") {
  return req<AuthCodeSent>("/api/auth/resend-code", {
    method: "POST", body: JSON.stringify({ email, purpose }),
  });
}

export function logout() { localStorage.removeItem("vyro_token"); }

export const listChats = () => req<Chat[]>("/api/chats");
// v0.4: receipts, presence, stories, scheduled, pin/archive/folder, admin, bots.
export const markRead = (messageId: string) =>
  req<Message>(`/api/messages/${messageId}/read`, { method: "POST" });
export const getPresence = (userId: string) =>
  req<{ user_id: string; online: boolean; last_seen?: string }>(`/api/users/${userId}/presence`);
export const sendLocation = (chatId: string, lat: number, lng: number, label = "") =>
  req<Message>(`/api/chats/${chatId}/messages`, {
    method: "POST",
    body: JSON.stringify({ kind: "location", lat, lng, location_label: label }),
  });
export const sendSticker = (chatId: string, up: UploadResult) =>
  req<Message>(`/api/chats/${chatId}/messages`, {
    method: "POST",
    body: JSON.stringify({ kind: "sticker", file_url: up.url, file_name: up.file_name, file_size: up.file_size, mime_type: up.mime_type }),
  });
export const listStories = (userId = "") =>
  req<Story[]>(userId ? `/api/stories?user_id=${encodeURIComponent(userId)}` : "/api/stories");
export const createStory = (mediaUrl: string, caption = "") =>
  req<Story>("/api/stories", { method: "POST", body: JSON.stringify({ media_url: mediaUrl, caption }) });
export const scheduleMessage = (chatId: string, content: string, sendAt: string) =>
  req<ScheduledMessage>(`/api/chats/${chatId}/scheduled`, {
    method: "POST", body: JSON.stringify({ content, send_at: sendAt }),
  });
export const listScheduled = (chatId: string) =>
  req<ScheduledMessage[]>(`/api/chats/${chatId}/scheduled`);
export const cancelScheduled = (id: string) =>
  req<{ status: string }>(`/api/scheduled/${id}`, { method: "DELETE" });
export const pinChat = (chatId: string) => req<Chat>(`/api/chats/${chatId}/pin`, { method: "POST" });
export const unpinChat = (chatId: string) => req<Chat>(`/api/chats/${chatId}/unpin`, { method: "POST" });
export const archiveChat = (chatId: string) => req<Chat>(`/api/chats/${chatId}/archive`, { method: "POST" });
export const unarchiveChat = (chatId: string) => req<Chat>(`/api/chats/${chatId}/unarchive`, { method: "POST" });
export const setFolder = (chatId: string, folder: string) =>
  req<Chat>(`/api/chats/${chatId}/folder`, { method: "POST", body: JSON.stringify({ folder }) });
export const listMembers = (chatId: string) =>
  req<{ user_id: string; role: string }[]>(`/api/chats/${chatId}/members`);
export const addMember = (chatId: string, userId: string) =>
  req<{ status: string }>(`/api/chats/${chatId}/members`, { method: "POST", body: JSON.stringify({ user_id: userId }) });
export const removeMember = (chatId: string, userId: string) =>
  req<{ status: string }>(`/api/chats/${chatId}/members/${userId}`, { method: "DELETE" });
export const setMemberRole = (chatId: string, userId: string, role: string) =>
  req<{ status: string }>(`/api/chats/${chatId}/members/${userId}/role`, { method: "POST", body: JSON.stringify({ role }) });
export const listUsers = (search = "") => req<User[]>(`/api/users?search=${encodeURIComponent(search)}`);
export const listMessages = (chatId: string) => req<Message[]>(`/api/chats/${chatId}/messages?limit=50`);
export const sendMessage = (chatId: string, content: string, opts?: { ttlSeconds?: number }) =>
  req<Message>(`/api/chats/${chatId}/messages`, {
    method: "POST",
    body: JSON.stringify({ content, ...(opts?.ttlSeconds ? { ttl_seconds: opts.ttlSeconds } : {}) }),
  });

// Upload one file to the backend (POST /api/uploads, multipart field "file").
// Uses fetch directly, not req(): req() forces a JSON Content-Type, which
// would break the multipart boundary. Returns the stored file's URL + kind.
export async function uploadFile(file: File): Promise<UploadResult> {
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`${API}/api/uploads`, {
    method: "POST",
    headers: token() ? { Authorization: `Bearer ${token()}` } : {},
    body: form,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Upload failed: ${res.status}`);
  }
  return res.json() as Promise<UploadResult>;
}

// Send a media message pointing at a file already stored by uploadFile.
// `content` is the optional caption.
export const sendMediaMessage = (chatId: string, up: UploadResult, content = "") =>
  req<Message>(`/api/chats/${chatId}/messages`, {
    method: "POST",
    body: JSON.stringify({
      content, kind: up.kind, file_url: up.url,
      file_name: up.file_name, file_size: up.file_size, mime_type: up.mime_type,
    }),
  });

// Fetch an uploaded file with the login token and return a blob URL for
// <img>/<video>/<audio> or download. /uploads/* requires auth, so a plain
// src="/uploads/..." would load without the Authorization header.
export async function fetchUploadBlobUrl(fileUrl: string): Promise<string> {
  const res = await fetch(`${API}${fileUrl}`, {
    headers: token() ? { Authorization: `Bearer ${token()}` } : {},
  });
  if (!res.ok) throw new Error(`Could not load file: ${res.status}`);
  const blob = await res.blob();
  return URL.createObjectURL(blob);
}
// Send a voice message pointing at a file already stored by uploadFile.
// The kind is forced to "voice" (not the upload's detected kind):
// MediaRecorder audio (e.g. audio/webm) is not always sniffed as audio
// by the server's content-type detection, which would label it "file".
export const sendVoiceMessage = (chatId: string, up: UploadResult, durationSec: number) =>
  req<Message>(`/api/chats/${chatId}/messages`, {
    method: "POST",
    body: JSON.stringify({
      content: `🎤 Voice message · ${Math.floor(durationSec / 60)}:${Math.round(durationSec % 60).toString().padStart(2, "0")}`,
      kind: "voice", file_url: up.url,
      file_name: up.file_name, file_size: up.file_size, mime_type: up.mime_type,
    }),
  });

export const createChat = (type: "dm" | "group" | "channel", memberIds: string[], title = "") =>
  req<Chat>("/api/chats", { method: "POST", body: JSON.stringify({ type, member_ids: memberIds, title }) });
export const editMessage = (id: string, content: string) =>
  req<Message>(`/api/messages/${id}`, { method: "PUT", body: JSON.stringify({ content }) });
export const deleteMessage = (id: string) =>
  req<{ status: string }>(`/api/messages/${id}`, { method: "DELETE" });
export const searchMessages = (q: string) =>
  req<Message[]>(`/api/messages/search?q=${encodeURIComponent(q)}`);
export const me = () => req<User>("/api/users/me");

export interface WSEvent {
  type: string; chat_id?: string; message?: Message; message_id?: string;
  user_id?: string; online?: boolean; typing?: boolean;
  // v0.5: WebRTC signaling payload (SDP offer/answer or ICE candidate),
  // relayed verbatim by the server; to_user_id limits it to one peer.
  payload?: unknown; to_user_id?: string;
}

export function connectWS(onEvent: (ev: WSEvent) => void): WebSocket | null {
  const t = token();
  if (!t) return null;
  const ws = new WebSocket(`${API.replace(/^http/, "ws")}/ws?token=${encodeURIComponent(t)}`);
  ws.onmessage = (e) => { try { onEvent(JSON.parse(e.data)); } catch { /* ignore malformed frames */ } };
  return ws;
}

// Tell the other members of a chat that this user is typing.
export function sendTyping(ws: WebSocket | null, chatId: string) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: "typing", chat_id: chatId }));
  }
}

// Join an additional chat room over the existing socket (e.g. a chat
// created after connecting).
export function subscribeChat(ws: WebSocket | null, chatId: string) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: "subscribe", chat_id: chatId }));
  }
}

// v0.5: send a WebRTC signaling frame (call-offer / call-answer /
// call-ice / call-end) to the other member(s) of a chat room. The
// payload (SDP / ICE candidate) is relayed verbatim by the server.
export function sendCallSignal(ws: WebSocket | null, chatId: string, type: string, toUserId: string, payload: unknown) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type, chat_id: chatId, to_user_id: toUserId, payload }));
  }
}

// ---- v0.5: E2EE key management -------------------------------------
// The client uploads only PUBLIC key material. Private keys never
// leave the device. See lib/e2ee.ts for the protocol.
export interface KeyBundle {
  user_id: string; device_id: string; dh_pub: string; sign_pub: string;
  spk_pub: string; spk_id: number; spk_sig: string;
  one_time_prekey?: { id: number; pub: string } | null;
}
export const uploadIdentityKey = (deviceId: string, identityKey: string, signedPrekey: string, signedPrekeyId: number, signKey: string, signedPrekeySig: string) =>
  req("/api/keys/identity", {
    method: "POST",
    body: JSON.stringify({
      device_id: deviceId, identity_key: identityKey, signed_prekey: signedPrekey,
      signed_prekey_id: signedPrekeyId, sign_key: signKey, signed_prekey_sig: signedPrekeySig,
    }),
  });
export const uploadPreKeys = (deviceId: string, keys: { key_id: number; public_key: string }[]) =>
  req<{ status: string; count: number }>("/api/keys/prekeys", {
    method: "POST", body: JSON.stringify({ device_id: deviceId, keys }),
  });
export const fetchKeyBundle = (userId: string) => req<KeyBundle>(`/api/keys/${userId}/bundle`);
export const sendE2EEMessage = (chatId: string, envelopeJson: string) =>
  req<Message>(`/api/chats/${chatId}/messages`, {
    method: "POST", body: JSON.stringify({ content: envelopeJson, kind: "e2ee" }),
  });

// ---- v0.5: Web Push (VAPID) ----------------------------------------
// Free, standards-based push. The subscription is the browser's
// PushSubscription JSON; the server stores only the endpoint + keys.
export const getVapidPublicKey = () =>
  req<{ available: boolean; public_key?: string; subject?: string }>("/api/push/vapid-public-key");
export const subscribePush = (sub: { endpoint: string; keys: { p256dh: string; auth: string } }) =>
  req<{ status: string }>("/api/push/subscribe", { method: "POST", body: JSON.stringify(sub) });
export const unsubscribePush = (endpoint: string) =>
  req<{ status: string }>("/api/push/subscribe", { method: "DELETE", body: JSON.stringify({ endpoint }) });
export const testPush = () =>
  req<{ sent: number; failed: number }>("/api/push/test", { method: "POST" });
