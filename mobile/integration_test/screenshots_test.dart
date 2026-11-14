// Screenshot capture for the VYRO download site (local test backend only).
//
// Drives the REAL Flutter app in Chrome: logs in as the seeded demo user
// through the real login + 6-digit-code screens, then screenshots the
// chat list, an open conversation, and the Stories tab. The verification
// code and demo credentials come from the local helper server on
// http://localhost:1099 (never hard-coded here, never committed).
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:integration_test/integration_test.dart';
import 'package:vyro/main.dart';

const _helper = 'http://localhost:1099';

Future<Map<String, dynamic>> _getJson(String url) async {
  final res = await http
      .get(Uri.parse(url))
      .timeout(const Duration(seconds: 15));
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

Future<String> _waitForCode(String email) async {
  final end = DateTime.now().add(const Duration(seconds: 30));
  while (DateTime.now().isBefore(end)) {
    try {
      final data = await _getJson('$_helper/code?email=$email');
      final code = data['code'] as String?;
      if (code != null && code.length == 6) return code;
    } catch (_) {
      // Not issued yet — keep polling.
    }
    await Future<void>.delayed(const Duration(milliseconds: 500));
  }
  throw Exception('Timed out waiting for a verification code for $email');
}

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('capture real app screenshots', (tester) async {
    // Phone-sized viewport is set by the driver (390x844).
    await tester.pumpWidget(const VyroApp());

    // 1) Login screen.
    await _waitFor(tester, find.text('Log in'), label: 'Log in button');
    final config = await _getJson('$_helper/config');
    final email = config['email'] as String;
    final password = config['password'] as String;

    await tester.enterText(
        find.widgetWithText(TextField, 'Email'), email);
    await tester.enterText(
        find.widgetWithText(TextField, 'Password'), password);
    await tester.tap(find.text('Log in'));
    await tester.pump();

    // 2) 6-digit code screen (code is emailed by the real backend).
    await _waitFor(tester, find.text('Check your email'),
        label: 'code screen');
    final code = await _waitForCode(email);
    await tester.enterText(
        find.widgetWithText(TextField, '6-digit code'), code);
    await tester.tap(find.text('Verify and continue'));
    await tester.pump();

    // 3) Chat list ("Your World").
    await _waitFor(tester, find.text('🌍 Your World'),
        label: 'chat list');
    await _waitFor(tester, find.text('Demo Two'), label: 'Demo Two chat');
    await _waitFor(tester, find.text('Weekend Plans'),
        label: 'Weekend Plans chat');
    // Let the world background settle and chats render fully.
    await tester.pump(const Duration(seconds: 2));
    await binding.takeScreenshot('chat-list');

    // 4) Open the DM conversation.
    await tester.tap(find.text('Demo Two'));
    await _waitFor(tester,
        find.text('Perfect for slow internet. See you tomorrow!'),
        label: 'last DM message');
    await tester.pump(const Duration(seconds: 1));
    await binding.takeScreenshot('chat');

    // 5) Back to the list, then the Stories tab.
    await tester.tap(find.byIcon(Icons.arrow_back));
    await _waitFor(tester, find.text('🌍 Your World'),
        label: 'chat list again');
    await tester.tap(find.text('Stories'));
    await _waitFor(tester, find.text('Morning in Kabul ☀️'),
        label: 'story row');
    await tester.pump(const Duration(seconds: 1));
    await binding.takeScreenshot('stories');
  });
}
