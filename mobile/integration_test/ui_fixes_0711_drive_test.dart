// Visual drive check for the Safari 0.7.11 polish round. The supervisor
// runs this against the real backend in Chrome; screenshots are saved by
// ui_fixes_0711_driver.dart under ~/workspace/site-shots/.
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:integration_test/integration_test.dart';
import 'package:vyro/local_ai.dart';
import 'package:vyro/main.dart';

const _helper = 'http://localhost:1099';
const _demo1Email = 'demo1@vyro.test';
const _demo2Email = 'demo2@vyro.test';
const _demoPassword = 'DemoPass123!';
const _faPrompts = [
  'چطور گروه بسازم؟',
  'چطور کسی را با لینک دعوت کنم؟',
  'چطور تماس تصویری برقرار کنم؟',
];

Future<Map<String, dynamic>> _getJson(String url) async {
  final res = await http.get(Uri.parse(url)).timeout(
        const Duration(seconds: 30),
      );
  if (res.statusCode != 200) {
    throw Exception('GET $url -> ${res.statusCode}');
  }
  return jsonDecode(res.body) as Map<String, dynamic>;
}

Future<void> _waitFor(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 90),
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
  final end = DateTime.now().add(const Duration(seconds: 90));
  while (DateTime.now().isBefore(end)) {
    try {
      final data = await _getJson(
        '$_helper/code?email=${Uri.encodeQueryComponent(email)}',
      );
      final code = data['code'] as String?;
      if (code != null && code.length == 6 && code != except) return code;
    } catch (_) {
      // The verification request may still be reaching the local helper.
    }
    await Future<void>.delayed(const Duration(milliseconds: 750));
  }
  throw Exception('Timed out waiting for a verification code for $email');
}

Future<String?> _previousCode(String email) async {
  try {
    final data = await _getJson(
      '$_helper/code?email=${Uri.encodeQueryComponent(email)}',
    );
    return data['code'] as String?;
  } catch (_) {
    return null;
  }
}

Future<void> _finishCodeStep(WidgetTester tester, String email,
    {String? except}) async {
  await _waitFor(tester, find.text('Check your email'), label: 'code screen');
  final code = await _waitForCode(email, except: except);
  final field = find.widgetWithText(TextField, '6-digit code');
  await tester.enterText(field, code);
  await tester.pump();
  var controller = tester.widget<TextField>(field).controller;
  if (controller?.text != code) {
    controller?.text = code;
    await tester.pump();
  }
  controller = tester.widget<TextField>(field).controller;
  if (controller?.text != code) {
    throw Exception('Code field read-back mismatch for $email');
  }
  await tester.tap(find.text('Verify and continue'));
  await _waitFor(tester, find.byIcon(Icons.smart_toy_outlined),
      label: 'signed-in app');
}

Future<void> _loginAs(
  WidgetTester tester,
  String email,
  String password,
) async {
  await _waitFor(tester, find.text('Log in'), label: 'login screen');
  final oldCode = await _previousCode(email);
  await tester.enterText(find.widgetWithText(TextField, 'Email'), email);
  await tester.enterText(find.widgetWithText(TextField, 'Password'), password);
  await tester.tap(find.text('Log in'));
  await tester.pump();
  await _finishCodeStep(tester, email, except: oldCode);
}

Future<void> _registerAs(
  WidgetTester tester,
  String email,
  String password,
) async {
  await _waitFor(tester, find.text('Log in'), label: 'registration start');
  await tester.tap(find.text('New to Safari? Create an account'));
  await tester.pump();
  final oldCode = await _previousCode(email);
  await tester.enterText(
      find.widgetWithText(TextField, 'Your name'), 'Empty Chats');
  await tester.enterText(find.widgetWithText(TextField, 'Email'), email);
  await tester.enterText(find.widgetWithText(TextField, 'Password'), password);
  await tester.tap(find.text('Create account'));
  await tester.pump();
  await _finishCodeStep(tester, email, except: oldCode);
}

/// Dismiss an open modal bottom sheet by tapping its scrim.
Future<void> _dismissSheet(WidgetTester tester) async {
  await tester.tapAt(const Offset(195, 120));
  await tester.pump(const Duration(milliseconds: 700));
}

Future<bool> _logoutFromSettings(WidgetTester tester) async {
  await tester.tap(find.byIcon(Icons.more_vert));
  await tester.pump(const Duration(milliseconds: 400));
  final settingsLabel =
      find.text('تنظیمات').evaluate().isNotEmpty ? 'تنظیمات' : 'Settings';
  await tester.tap(find.text(settingsLabel));
  await tester.pump();
  await _waitFor(tester, find.text(settingsLabel), label: 'Settings');
  final lists = find.byType(ListView);
  if (lists.evaluate().isNotEmpty) {
    await tester.drag(lists.first, const Offset(0, -4000));
    await tester.pump();
  }
  const labels = ['Log out', 'Logout', 'Sign out', 'Sign Out'];
  Finder? logout;
  for (final label in labels) {
    final candidate = find.text(label);
    if (candidate.evaluate().isNotEmpty) {
      logout = candidate;
      break;
    }
  }
  if (logout == null) {
    // This checkout currently has no logout control in Settings. Do not
    // simulate a logout or claim cross-account isolation was exercised.
    // ignore: avoid_print
    print('SKIPPED account-switch step: no logout affordance in Settings.');
    await tester.tap(find.byIcon(Icons.arrow_back));
    await tester.pump();
    return false;
  }
  await tester.tap(logout);
  await tester.pump();
  if (find.text('Log out').evaluate().isNotEmpty) {
    await tester.tap(find.text('Log out'));
    await tester.pump();
  }
  await _waitFor(tester, find.text('Log in'), label: 'logged-out screen');
  return true;
}

Future<void> _openAssistant(
  WidgetTester tester, {
  bool persian = false,
}) async {
  await tester.tap(find.byIcon(Icons.smart_toy_outlined));
  await tester.pump();
  await _waitFor(
      tester, find.text(persian ? 'دستیار سافاری' : 'Safari assistant'),
      label: 'Assistant title');
}

Future<void> _ensureLightMode(WidgetTester tester) async {
  Brightness brightness() =>
      Theme.of(tester.element(find.byType(Scaffold).first)).brightness;
  if (brightness() == Brightness.dark) {
    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pump(const Duration(milliseconds: 400));
    final label = find.text('حالت تاریک').evaluate().isNotEmpty
        ? 'حالت تاریک'
        : 'Dark mode';
    await tester.tap(find.text(label));
    await tester.pump();
  }
  expect(brightness(), Brightness.light);
}

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('drive Safari 0.7.11 UI fixes', (tester) async {
    await tester.pumpWidget(const VyroApp());

    // (a) Demo-user search filtering and the separate close X.
    await _waitFor(tester, find.text('Log in'), label: 'login screen');
    final config = await _getJson('$_helper/config');
    var currentEmail = config['email'] as String;
    await _loginAs(tester, currentEmail, config['password'] as String);
    await tester.tap(find.byIcon(Icons.search));
    await _waitFor(
      tester,
      find.widgetWithText(TextField, 'Search chats'),
      label: 'search capsule',
    );
    final searchField = find.widgetWithText(TextField, 'Search chats');
    await binding.takeScreenshot('chat-search');
    final searchTextField = tester.widget<TextField>(searchField);
    final searchController = searchTextField.controller!;
    final searchFocus = searchTextField.focusNode!;
    await tester.enterText(searchField, 'no-chat-match-0711');
    await _waitFor(tester, find.text('No matching chats'),
        label: 'filtered empty list');
    expect(searchController.text, 'no-chat-match-0711');
    expect(searchFocus.hasFocus, isTrue);
    await tester.tap(find.byTooltip('Close search'));
    await tester.pump();
    expect(searchController.text, isEmpty);
    expect(searchFocus.hasFocus, isFalse);
    expect(find.widgetWithText(TextField, 'Search chats'), findsNothing);

    // Natural English captures from the requested polish states.
    await tester.tap(find.byIcon(Icons.add_comment_outlined));
    await _waitFor(tester, find.text('New direct message'),
        label: 'English new-chat sheet');
    await tester.pump(const Duration(seconds: 1));
    await binding.takeScreenshot('newchat-sheet');
    await _dismissSheet(tester);
    await _openAssistant(tester);
    await _waitFor(tester, find.text('How do I create a group?'),
        label: 'English starter suggestions');
    await binding.takeScreenshot('assistant-chat');
    await tester.tap(find.byIcon(Icons.arrow_back));
    await tester.pump();

    // (b) Switch English -> Persian, capture all localized Assistant UI,
    // then tap each chip and validate the on-device answer.
    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.tap(find.text('Language'));
    await _waitFor(tester, find.text('گفت‌وگوها'), label: 'Persian locale');
    await tester.tap(find.byIcon(Icons.add_comment_outlined));
    await _waitFor(tester, find.text('پیام مستقیم جدید'),
        label: 'Persian new-chat sheet');
    await binding.takeScreenshot('newchat-sheet-fa');
    await _dismissSheet(tester);
    await _openAssistant(tester, persian: true);
    expect(find.text('دستیار سافاری'), findsOneWidget);
    expect(find.text('دربارهٔ استفاده از سافاری از من بپرسید'), findsOneWidget);
    expect(
      find.text(
          'روی دستگاه شما و بر پایهٔ قوانین اجرا می‌شود، نه هوش مصنوعی ابری.'),
      findsOneWidget,
    );
    for (final prompt in _faPrompts) {
      expect(find.text(prompt), findsOneWidget);
    }
    await binding.takeScreenshot('assistant-chat-fa');
    const answerMarkers = [
      'ایجاد گروه یا کانال',
      'پیوستن با لینک دعوت',
      'تماس تصویری'
    ];
    for (var i = 0; i < _faPrompts.length; i++) {
      final prompt = _faPrompts[i];
      await tester.tap(find.text(prompt));
      final answer = assistantReply(prompt).text;
      expect(answer, contains(answerMarkers[i]));
      await _waitFor(tester, find.text(answer),
          label: 'specific Persian answer for $prompt');
    }
    await tester.tap(find.byIcon(Icons.arrow_back));
    await tester.pump();

    // (c) The app has no Settings logout action in this checkout. Only
    // perform the real cross-account assertion if that UI is available.
    var historyIsolationVerified = false;
    if (currentEmail != _demo1Email) {
      final didLogout = await _logoutFromSettings(tester);
      if (didLogout) {
        await _loginAs(tester, _demo1Email, _demoPassword);
        currentEmail = _demo1Email;
      }
    }
    if (currentEmail == _demo1Email) {
      const marker = 'demo1-history-probe-0711';
      await _openAssistant(tester, persian: true);
      final input = find.byWidgetPredicate(
        (widget) =>
            widget is TextField &&
            widget.decoration?.hintText == 'دربارهٔ استفاده از سافاری بپرسید…',
      );
      await tester.enterText(input, marker);
      await tester.tap(find.byTooltip('ارسال'));
      await _waitFor(tester, find.text(marker),
          label: 'demo1 Assistant message');
      await tester.pump(const Duration(milliseconds: 300));
      await tester.tap(find.byIcon(Icons.arrow_back));
      await tester.pump();
      if (await _logoutFromSettings(tester)) {
        await _loginAs(tester, _demo2Email, _demoPassword);
        await _openAssistant(tester, persian: true);
        expect(find.text(marker), findsNothing);
        await tester.tap(find.byIcon(Icons.arrow_back));
        await tester.pump();
        if (await _logoutFromSettings(tester)) {
          await _loginAs(tester, _demo1Email, _demoPassword);
          await _openAssistant(tester, persian: true);
          await _waitFor(tester, find.text(marker),
              label: 'restored demo1 Assistant history');
          historyIsolationVerified = true;
          await tester.tap(find.byIcon(Icons.arrow_back));
          await tester.pump();
        }
      }
    }
    if (!historyIsolationVerified) {
      // ignore: avoid_print
      print('UNVERIFIED account history separation: add a Settings logout '
          'affordance (or adjust this harness) before treating step (c) as run.');
    }

    // (d) Light-mode search capture. Read the actual Scaffold theme first.
    await _ensureLightMode(tester);
    await tester.tap(find.byIcon(Icons.search));
    await _waitFor(tester, find.widgetWithText(TextField, 'جست‌وجوی گفت‌وگوها'),
        label: 'light-mode search capsule');
    await binding.takeScreenshot('chat-search-light');
    await tester.tap(find.byTooltip('بستن جست‌وجو'));
    await tester.pump();

    // (e) Register a fresh account only when the real logout UI permits
    // reaching registration without bypassing the app's session handling.
    if (await _logoutFromSettings(tester)) {
      final freshEmail =
          'empty-${DateTime.now().millisecondsSinceEpoch}@vyro.test';
      await _registerAs(tester, freshEmail, _demoPassword);
      await _waitFor(tester, find.text('گفت‌وگوها'),
          label: 'fresh account in Persian');
      await tester.tap(find.byIcon(Icons.add_comment_outlined));
      await _waitFor(tester, find.text('شروع گفت‌وگو'),
          label: 'empty Persian chats state behind new-chat sheet');
      await binding.takeScreenshot('newchat-sheet-empty-fa');
    } else {
      // No fake sign-out or registration: this checkout lacks the UI
      // affordance required for the fresh-account step.
      // ignore: avoid_print
      print('UNVERIFIED fresh-account sheet: Settings has no logout control.');
    }
  });
}
