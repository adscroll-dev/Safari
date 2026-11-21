// Visual drive check for the Safari 0.7.10 UI fixes (branch
// copilot/app-ui-fixes-080). Drives the REAL Flutter app in Chrome:
// logs in as the seeded demo user through the real login + 6-digit-code
// screens (helper API on http://localhost:1099, same as
// screenshots_test.dart), then captures:
//   newchat-sheet.png   — new-chat menu, English (default locale)
//   chat-search.png     — Chats tab with the in-place search bar open
//                         (header magnifier must NOT also be visible)
//   assistant-chat.png  — Assistant chat open, three starter chips
//   newchat-sheet-fa.png— new-chat menu in PERSIAN locale (RTL)
// Screenshots are saved by the driver under ~/workspace/site-shots/.
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:integration_test/integration_test.dart';
import 'package:vyro/main.dart';

const _helper = 'http://localhost:1099';

Future<Map<String, dynamic>> _getJson(String url) async {
  final res = await http.get(Uri.parse(url)).timeout(
        const Duration(seconds: 15),
      );
  if (res.statusCode != 200) {
    throw Exception('GET $url -> ${res.statusCode}');
  }
  return jsonDecode(res.body) as Map<String, dynamic>;
}

Future<void> _waitFor(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 45),
  String? label,
}) async {
  final end = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(end)) {
    await tester.pump(const Duration(milliseconds: 400));
    if (finder.evaluate().isNotEmpty) return;
  }
  throw Exception('Timed out waiting for ${label ?? finder}');
}

Future<String> _waitForCode(String email, {String? except}) async {
  final end = DateTime.now().add(const Duration(seconds: 30));
  while (DateTime.now().isBefore(end)) {
    try {
      final data = await _getJson('$_helper/code?email=$email');
      final code = data['code'] as String?;
      if (code != null && code.length == 6 && code != except) return code;
    } catch (_) {
      // Not issued yet — keep polling.
    }
    await Future<void>.delayed(const Duration(milliseconds: 500));
  }
  throw Exception('Timed out waiting for a verification code for $email');
}

/// Dismiss the open modal bottom sheet by tapping the scrim above it.
Future<void> _dismissSheet(WidgetTester tester) async {
  await tester.tapAt(const Offset(195, 120));
  await tester.pump(const Duration(milliseconds: 700));
}

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('capture 0.7.10 UI-fix screenshots', (tester) async {
    await tester.pumpWidget(const VyroApp());

    // 1) Login screen.
    await _waitFor(tester, find.text('Log in'), label: 'Log in button');
    final config = await _getJson('$_helper/config');
    final email = config['email'] as String;
    final password = config['password'] as String;
    String? staleCode;
    try {
      final stale = await _getJson('$_helper/code?email=$email');
      staleCode = stale['code'] as String?;
    } catch (_) {}

    await tester.enterText(find.widgetWithText(TextField, 'Email'), email);
    await tester.enterText(
        find.widgetWithText(TextField, 'Password'), password);
    await tester.tap(find.text('Log in'));
    await tester.pump();

    // 2) 6-digit code screen.
    await _waitFor(tester, find.text('Check your email'),
        label: 'code screen');
    final code = await _waitForCode(email, except: staleCode);
    final codeFinder = find.widgetWithText(TextField, '6-digit code');
    await tester.enterText(codeFinder, code);
    await tester.pump();
    var ctrl = (codeFinder.evaluate().first.widget as TextField).controller;
    if (ctrl?.text != code) {
      ctrl?.text = code;
      await tester.pump();
    }
    ctrl = (codeFinder.evaluate().first.widget as TextField).controller;
    if (ctrl?.text != code) {
      throw Exception(
          'code field read-back mismatch: entered="${ctrl?.text}" want="$code"');
    }
    await tester.tap(find.text('Verify and continue'));
    await tester.pump();

    // 3) Chat list.
    await _waitFor(tester, find.text('Demo Two'), label: 'Demo Two chat');
    await tester.pump(const Duration(seconds: 2));

    // (b) New-chat sheet, English.
    await tester.tap(find.byIcon(Icons.add_comment_outlined));
    await _waitFor(tester, find.text('New direct message'),
        label: 'new-chat sheet (en)');
    await tester.pump(const Duration(seconds: 1));
    await binding.takeScreenshot('newchat-sheet');
    await _dismissSheet(tester);

    // (d) Chat search open — header magnifier must be hidden.
    await tester.tap(find.byIcon(Icons.search));
    await _waitFor(tester, find.widgetWithText(TextField, 'Search chats'),
        label: 'search capsule');
    await tester.pump(const Duration(seconds: 1));
    await binding.takeScreenshot('chat-search');
    // Close the search again via the X.
    await tester.tap(find.byIcon(Icons.close));
    await tester.pump(const Duration(milliseconds: 700));

    // (c) Assistant chat with the three starter chips.
    await tester.tap(find.byIcon(Icons.smart_toy_outlined));
    await _waitFor(tester, find.text('How do I create a group?'),
        label: 'assistant starter chip');
    await tester.pump(const Duration(seconds: 1));
    await binding.takeScreenshot('assistant-chat');
    // Back to the chat list.
    await tester.tap(find.byIcon(Icons.arrow_back));
    await _waitFor(tester, find.text('Demo Two'), label: 'chat list again');

    // (a) Persian locale LAST: ⋮ menu -> Language (en -> fa).
    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pump(const Duration(milliseconds: 500));
    await tester.tap(find.text('Language'));
    await _waitFor(tester, find.text('گفت‌وگوها'),
        label: 'Persian Chats tab');
    await tester.pump(const Duration(seconds: 1));
    await tester.tap(find.byIcon(Icons.add_comment_outlined));
    await _waitFor(tester, find.text('پیام مستقیم جدید'),
        label: 'new-chat sheet (fa)');
    await tester.pump(const Duration(seconds: 1));
    await binding.takeScreenshot('newchat-sheet-fa');
  });
}
