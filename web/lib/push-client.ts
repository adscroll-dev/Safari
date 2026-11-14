// Web Push client (VAPID, v0.5). Free and standards-based: no FCM/APNs
// account. The service worker (public/sw.js) shows the notification;
// the server's payload never contains message content.
//
// Push needs a secure context (https or localhost) and a browser that
// supports PushManager. subscribe() returns a short status string for
// the UI; it never throws for "not supported" — it reports it.
import * as api from "./api";

function b64urlToUint8Array(b64url: string): Uint8Array {
  const pad = "=".repeat((4 - (b64url.length % 4)) % 4);
  const b64 = (b64url + pad).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(b64);
  const out = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

export function pushSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    window.isSecureContext
  );
}

export async function pushStatus(): Promise<"subscribed" | "unsubscribed" | "unsupported"> {
  if (!pushSupported()) return "unsupported";
  const reg = await navigator.serviceWorker.getRegistration();
  if (!reg) return "unsubscribed";
  const sub = await reg.pushManager.getSubscription();
  return sub ? "subscribed" : "unsubscribed";
}

export async function subscribePush(): Promise<string> {
  if (!pushSupported()) return "Push is not supported in this browser (needs https/localhost + PushManager).";
  const key = await api.getVapidPublicKey();
  if (!key.available || !key.public_key) return "Push is not configured on this server.";
  const reg = await navigator.serviceWorker.register("/sw.js");
  const sub = await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: b64urlToUint8Array(key.public_key),
  });
  const json = sub.toJSON();
  await api.subscribePush({
    endpoint: sub.endpoint,
    keys: { p256dh: json.keys?.p256dh ?? "", auth: json.keys?.auth ?? "" },
  });
  return "Notifications enabled — the server will push new-message alerts.";
}

export async function unsubscribePush(): Promise<string> {
  if (!pushSupported()) return "Push is not supported in this browser.";
  const reg = await navigator.serviceWorker.getRegistration();
  const sub = await reg?.pushManager.getSubscription();
  if (!sub) return "Not subscribed.";
  await api.unsubscribePush(sub.endpoint);
  await sub.unsubscribe();
  return "Notifications disabled.";
}

export async function testPush(): Promise<string> {
  const out = await api.testPush();
  return out.sent > 0 ? "Test notification sent." : `Test notification could not be delivered (failed: ${out.failed}).`;
}
