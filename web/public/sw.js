// VYRO service worker — Web Push (VAPID), v0.5.
//
// The server's push payloads never contain message content (only
// "New message from <name>" / "New encrypted message"), so this worker
// simply shows what it receives. Clicking the notification opens the
// app. No analytics, no third-party code.
self.addEventListener("push", (event) => {
  let data = { title: "VYRO", body: "New message" };
  try {
    if (event.data) data = { ...data, ...event.data.json() };
  } catch {
    // Non-JSON payload: show the generic text.
  }
  event.waitUntil(
    self.registration.showNotification(data.title || "VYRO", {
      body: data.body || "New message",
      icon: "/logo.svg",
      badge: "/logo.svg",
      tag: "vyro-message",
    }),
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ("focus" in client) return client.focus();
      }
      if (self.clients.openWindow) return self.clients.openWindow("/");
      return undefined;
    }),
  );
});
