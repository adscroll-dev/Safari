// Visual drive check for the Safari 0.7.12 polish round. The supervisor
// runs this against the real backend in Chrome; screenshots are saved by
// ui_fixes_0712_driver.dart under ~/workspace/site-shots/.
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:integration_test/integration_test.dart';
import 'package:vyro/local_ai.dart';
import 'package:vyro/main.dart';

const _helper = 'http://localhost:1099';
const _faIntro =
    'این دستیار روی دستگاه شما و بر اساس قوانین ازپیش‌تعریف‌شده کار می‌کند و به هوش مصنوعی ابری متصل نیست.';
const _enIntro = 'Runs on this device — rule-based, not cloud AI.';
const _faHint = 'سؤالت را درباره سافاری بنویس…';
const _enHint = 'Ask about Safari…';

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
  Duration timeout = const Duration(seconds: 120),
  String? label,
}) async {
  final end = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(end)) {
    await tester.pump(const Duration(milliseconds: 400));
    if (finder.evaluate().isNotEmpty) return;
  }
  throw Exception('Timed out waiting for ${label ?? finder}');
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

Future<String> _waitForCode(String email, {String? except}) async {
  final end = DateTime.now().add(const Duration(seconds: 120));
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

Future<void> _finishCodeStep(
  WidgetTester tester,
  String email, {
  String? except,
}) async {
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

Future<void> _dismissSheet(WidgetTester tester) async {
  await tester.tapAt(const Offset(195, 120));
  await tester.pump(const Duration(milliseconds: 700));
}

String _localeCode(WidgetTester tester) =>
    Localizations.localeOf(tester.element(find.byType(Scaffold).first))
        .languageCode;

Future<void> _waitForLocale(WidgetTester tester, String languageCode) async {
  final end = DateTime.now().add(const Duration(seconds: 120));
  while (DateTime.now().isBefore(end)) {
    await tester.pump(const Duration(milliseconds: 100));
    if (_localeCode(tester) == languageCode) return;
  }
  throw Exception('Timed out waiting for locale $languageCode');
}

/// Tap a popup-menu label ON its glyphs, not at the centre of its box.
///
/// PopupMenuItem stretches its Text to the menu width, so
/// `tester.tap(find.text(label))` taps the centre of a wide paragraph box —
/// empty space past the glyphs. On the web HTML renderer that hit-test
/// crashes in the engine (`Invalid argument: 128.78` from
/// `CanvasParagraph.getClosestGlyphInfoForOffset`: `clamp` with an inverted
/// glyph box) for both LTR and RTL labels. Tapping 24 px inside the
/// glyphs' edge (left edge for LTR, right edge for RTL) avoids it.
Future<void> _tapMenuLabel(WidgetTester tester, String label) async {
  final finder = find.text(label);
  await _waitFor(tester, finder, label: 'menu label $label');
  final topLeft = tester.getTopLeft(finder);
  final bottomRight = tester.getBottomRight(finder);
  final isRtl =
      Directionality.of(tester.element(finder)) == TextDirection.rtl;
  final x = isRtl ? bottomRight.dx - 24 : topLeft.dx + 24;
  final y = (topLeft.dy + bottomRight.dy) / 2;
  await tester.tapAt(Offset(x, y));
  await tester.pump(const Duration(milliseconds: 400));
}

Future<void> _setLanguage(WidgetTester tester, String languageCode) async {
  for (var attempt = 0;
      attempt < 3 && _localeCode(tester) != languageCode;
      attempt++) {
    final previous = _localeCode(tester);
    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pump(const Duration(milliseconds: 400));
    final label = switch (previous) {
      'fa' => 'زبان',
      'ar' => 'اللغة',
      _ => 'Language',
    };
    await _tapMenuLabel(tester, label);
    final nextLocale = switch (previous) {
      'en' => 'fa',
      'fa' => 'ar',
      _ => 'en',
    };
    await _waitForLocale(tester, nextLocale);
  }
  expect(_localeCode(tester), languageCode);
}

Future<void> _openAssistant(WidgetTester tester,
    {required String title}) async {
  await tester.tap(find.byIcon(Icons.smart_toy_outlined));
  await _waitFor(tester, find.text(title), label: 'Assistant title');
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
    await _tapMenuLabel(tester, label);
    await tester.pump();
  }
  expect(brightness(), Brightness.light);
}

Finder _assistantInput(String hint) => find.byWidgetPredicate(
      (widget) => widget is TextField && widget.decoration?.hintText == hint,
    );

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('drive Safari 0.7.12 UI fixes', (tester) async {
    await tester.pumpWidget(const VyroApp());
    await _waitFor(tester, find.text('Log in'), label: 'login screen');
    final config = await _getJson('$_helper/config');
    final email = config['email'] as String;
    await _loginAs(tester, email, config['password'] as String);
    if (find.byType(BottomSheet).evaluate().isNotEmpty) {
      await _dismissSheet(tester);
    }

    // (a) Ensure the expected English -> Persian switch, then check the
    // welcome copy and one-line input while NO Assistant history exists
    // yet (the welcome paragraph is only rendered before the first
    // message). Switch fa -> en -> fa here, while the intro is still
    // visible in both languages, so the language-switch check is a real
    // rendered-text check — not a mapping-only check.
    await _setLanguage(tester, 'en');
    await _setLanguage(tester, 'fa');
    await _openAssistant(tester, title: 'دستیار سافاری');
    expect(find.text(_faIntro), findsOneWidget);
    expect(
      find.text(
          'روی دستگاه شما و بر پایهٔ قوانین اجرا می‌شود، نه هوش مصنوعی ابری.'),
      findsNothing,
    );
    expect(_assistantInput(_faHint), findsOneWidget);
    await tester.tap(find.byIcon(Icons.arrow_back));
    await tester.pump();
    await _setLanguage(tester, 'en');
    await _openAssistant(tester, title: 'Safari assistant');
    expect(find.text(_enIntro), findsOneWidget);
    expect(_assistantInput(_enHint), findsOneWidget);
    expect(find.text(_faIntro), findsNothing);
    expect(_assistantInput(_faHint), findsNothing);
    await tester.tap(find.byIcon(Icons.arrow_back));
    await tester.pump();
    await _setLanguage(tester, 'fa');
    await _openAssistant(tester, title: 'دستیار سافاری');
    expect(find.text(_faIntro), findsOneWidget);
    expect(_assistantInput(_faHint), findsOneWidget);
    expect(find.text(_enIntro), findsNothing);
    expect(_assistantInput(_enHint), findsNothing);
    // Screenshot the WELCOME state (before any message): this is the only
    // state where the intro paragraph is rendered — once a message is
    // sent the welcome content is replaced by the conversation.
    await binding.takeScreenshot('assistant-chat-fa');
    const prompt = 'چطور گروه بسازم؟';
    await tester.tap(find.text(prompt));
    await _waitFor(tester, find.text(assistantReply(prompt).text),
        label: 'Persian Assistant reply');
    await tester.tap(find.byIcon(Icons.arrow_back));
    await tester.pump();

    // (b) Verify the translated Chats filters and both populated sections.
    await _waitFor(tester, find.text('کانال‌ها'), label: 'Channels filter');
    await _waitFor(tester, find.text('سنجاق‌شده'), label: 'Pinned header');
    await _waitFor(tester, find.text('گفت‌وگوها'), label: 'Chats header');
    expect(find.text('Channels'), findsNothing);
    expect(find.text('Pinned'), findsNothing);
    expect(find.text('Chats'), findsNothing);
    await binding.takeScreenshot('chat-list-fa');

    // (c) Read the actual Scaffold brightness and ensure light mode before
    // validating and capturing the search placeholder.
    await _ensureLightMode(tester);
    await tester.tap(find.byIcon(Icons.search));
    await _waitFor(tester, _assistantInput('جست‌وجوی گفت‌وگوها'),
        label: 'light-mode search capsule');
    final searchField =
        tester.widget<TextField>(_assistantInput('جست‌وجوی گفت‌وگوها'));
    expect(searchField.decoration!.hintStyle!.color, const Color(0xFFC1CBDA));
    await binding.takeScreenshot('chat-search-light');
    await tester.tap(find.byTooltip('بستن جست‌وجو'));
    await tester.pump();

    // (d) Language switch fa -> en -> fa was already verified with
    // rendered text in step (a), while the welcome intro was visible.
    // No further switch is needed here: once a conversation exists the
    // Assistant layout omits the welcome paragraph by design.
  });
}
