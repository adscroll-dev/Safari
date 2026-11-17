import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart' show kIsWeb;

import 'api.dart';

/// FCM (Firebase Cloud Messaging) push for the mobile app.
///
/// Flow: main() initialises Firebase and registers the background
/// handler before runApp. After a successful login, _finishLogin calls
/// [FcmPush.init]: it asks for notification permission, gets the
/// device's FCM registration token and POSTs it to /api/push/fcm-token.
/// The server then sends a notification to that token when a message
/// arrives while the app is closed or backgrounded.
///
/// What the user sees: when the app is backgrounded or killed, Android
/// shows the notification itself (the server sends a notification
/// payload, so no local-notification plugin is needed); tapping it
/// opens the right chat. While the app is open, messages arrive over
/// the WebSocket as usual and the foreground FCM message only triggers
/// the in-app banner callback — it never yanks the user away.
///
/// Privacy: the server never puts message content in a push — only
/// "New message from <name>" (or "New encrypted message") plus the
/// chat id in the data payload (see backend/internal/push/fcm.go).
///
/// Push is best-effort throughout: every step is wrapped so a missing
/// google-services.json, a denied permission, or a server without FCM
/// configured never breaks login or chat. Real delivery to a phone is
/// untested until a phone test — see the README limits.
@pragma('vm:entry-point')
Future<void> fcmBackgroundHandler(RemoteMessage message) async {
  // The OS shows the notification payload on its own when the app is
  // backgrounded or killed; Firebase just needs to be initialised in
  // this isolate for the plugin to work.
  await Firebase.initializeApp();
}

class FcmPush {
  static bool _started = false;

  /// Initialises Firebase and registers the background handler. Called
  /// from main() on mobile only — the web build uses Web Push (VAPID)
  /// instead and has no Firebase options configured.
  static Future<void> initFirebaseForMain() async {
    if (kIsWeb) return;
    try {
      await Firebase.initializeApp();
      FirebaseMessaging.onBackgroundMessage(fcmBackgroundHandler);
    } catch (_) {
      // Firebase not configured (e.g. google-services.json missing):
      // the app works without push; init() below will try once more
      // after login and no-op if it still fails.
    }
  }

  /// Requests notification permission, registers this device's FCM
  /// token with the backend, and wires tap-to-open-chat. Call once,
  /// after a successful login (the token endpoint needs auth).
  static Future<void> init({
    required ApiClient api,
    required String deviceId,
    required void Function(String chatId) onOpenChat,
    required void Function(String title, String body, String? chatId)
        onForegroundMessage,
  }) async {
    if (_started || kIsWeb) return;
    _started = true;
    try {
      // main() normally initialises Firebase already; if that failed,
      // try once more here.
      if (Firebase.apps.isEmpty) {
        await Firebase.initializeApp();
        FirebaseMessaging.onBackgroundMessage(fcmBackgroundHandler);
      }
      final messaging = FirebaseMessaging.instance;
      await messaging.requestPermission(
          alert: true, badge: true, sound: true);
      final token = await messaging.getToken();
      if (token != null) {
        await api.registerFcmToken(token, deviceId);
      }
      // FCM rotates tokens: keep the server's copy fresh.
      messaging.onTokenRefresh.listen((newToken) async {
        try {
          await api.registerFcmToken(newToken, deviceId);
        } catch (_) {
          // Best-effort: the old token stays registered server-side.
        }
      });
      // Foreground: the message is already in the chat over WebSocket;
      // show it as an in-app banner via the callback, never navigate.
      FirebaseMessaging.onMessage.listen((RemoteMessage message) {
        final title = message.notification?.title ?? 'Safari';
        final body = message.notification?.body ?? 'New message';
        final chatId = message.data['chat_id'];
        onForegroundMessage(
            title, body, chatId is String ? chatId : null);
      });
      // Background tap: open the chat the notification was about.
      FirebaseMessaging.onMessageOpenedApp.listen((RemoteMessage message) {
        final chatId = message.data['chat_id'];
        if (chatId is String && chatId.isNotEmpty) onOpenChat(chatId);
      });
      // Launched from a notification while terminated.
      final initial = await messaging.getInitialMessage();
      final initialChatId = initial?.data['chat_id'];
      if (initialChatId is String && initialChatId.isNotEmpty) {
        onOpenChat(initialChatId);
      }
    } catch (_) {
      // Push is best-effort: login and chat work fully without it.
      // Allow a retry on the next login.
      _started = false;
    }
  }
}
