// Smoke test for the real Safari app: it boots to the login screen.
// Note: the login page has a forever-repeating drift animation, so
// pumpAndSettle never settles — pump a few frames instead.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/local_ai.dart';
import 'package:vyro/main.dart';
import 'package:vyro/theme.dart';

void main() {
  testWidgets('Safari boots to the login screen', (WidgetTester tester) async {
    await tester.pumpWidget(const VyroApp());
    await tester.pump(const Duration(seconds: 1));
    await tester.pump(const Duration(seconds: 1));
    expect(find.text('Safari'), findsWidgets);
    expect(find.text('Log in'), findsOneWidget);
  });

  testWidgets('Assistant starter chips send prompts and show real replies',
      (WidgetTester tester) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    String? sentPrompt;
    String? reply;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: AssistantStarterSuggestions(
            controller: controller,
            prompts: const [
              'How do I create a group?',
              'How do I invite someone with an invite link?',
              'How do I make a video call?',
            ],
            onSend: () async {
              sentPrompt = controller.text;
              reply = assistantReply(sentPrompt!).text;
            },
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('How do I create a group?'), findsOneWidget);
    expect(find.text('How do I invite someone with an invite link?'),
        findsOneWidget);
    expect(find.text('How do I make a video call?'), findsOneWidget);

    await tester.tap(find.text('How do I create a group?'));
    await tester.pump();
    expect(sentPrompt, 'How do I create a group?');
    expect(reply, contains('New group / channel'));
  });

  testWidgets('Assistant welcome is localized for English, Persian and Arabic',
      (WidgetTester tester) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    const localizedContent = <(Locale, List<String>)>[
      (
        Locale('en'),
        [
          'Safari assistant',
          'Ask me about using Safari',
          'Runs on this device — rule-based, not cloud AI.',
          'How do I create a group?',
          'How do I invite someone with an invite link?',
          'How do I make a video call?',
        ],
      ),
      (
        Locale('fa'),
        [
          'دستیار سافاری',
          'دربارهٔ استفاده از سافاری از من بپرسید',
          'روی دستگاه شما و بر پایهٔ قوانین اجرا می‌شود، نه هوش مصنوعی ابری.',
          'چطور گروه بسازم؟',
          'چطور کسی را با لینک دعوت کنم؟',
          'چطور تماس تصویری برقرار کنم؟',
        ],
      ),
      (
        Locale('ar'),
        [
          'مساعد سفاري',
          'اسألني عن كيفية استخدام سفاري',
          'يعمل على جهازك وفق قواعد محددة، وليس ذكاءً اصطناعيًا سحابيًا.',
          'كيف أنشئ مجموعة؟',
          'كيف أدعو شخصًا برابط دعوة؟',
          'كيف أجري مكالمة فيديو؟',
        ],
      ),
    ];
    for (final (locale, expectedTexts) in localizedContent) {
      await tester.pumpWidget(
        MaterialApp(
          locale: locale,
          supportedLocales: const [Locale('en'), Locale('fa'), Locale('ar')],
          localizationsDelegates: const [
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          home: Builder(
            builder: (context) => Scaffold(
              appBar: AppBar(
                title:
                    Text(assistantLocalizedText(context, 'Safari assistant')),
              ),
              body: AssistantWelcomeContent(
                controller: controller,
                onSend: () async {},
                avatar: const Icon(Icons.smart_toy_outlined),
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      for (final text in expectedTexts) {
        expect(find.text(text), findsOneWidget);
      }
    }
  });

  testWidgets(
      'Assistant starter chips use legible Safari colors in both themes',
      (WidgetTester tester) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    Future<void> pumpFor(ThemeMode mode) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Theme(
            data: mode == ThemeMode.dark ? vyroDarkTheme() : vyroLightTheme(),
            child: Scaffold(
              body: AssistantStarterSuggestions(
                controller: controller,
                prompts: const ['Group?', 'Invite?', 'Video?'],
                onSend: () async {},
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      expect(
        Theme.of(tester.element(find.byType(Scaffold))).brightness,
        mode == ThemeMode.dark ? Brightness.dark : Brightness.light,
      );
      final chips = tester.widgetList<ActionChip>(find.byType(ActionChip));
      expect(chips, hasLength(3));
      for (final chip in chips) {
        expect(chip.side, isNotNull);
        expect(chip.side!.color, OrbitalColors.electricCyan.withOpacity(0.56));
        expect(
            chip.labelStyle!.color,
            mode == ThemeMode.dark
                ? OrbitalColors.stellarWhite
                : OrbitalColors.nebulaNavy);
      }
      expect(
        chips.first.backgroundColor!.computeLuminance(),
        mode == ThemeMode.dark ? lessThan(0.1) : greaterThan(0.8),
      );
    }

    await pumpFor(ThemeMode.light);
    await pumpFor(ThemeMode.dark);
  });

  testWidgets(
      'Chats search capsule suppresses the theme fill in light and dark',
      (WidgetTester tester) async {
    final controller = TextEditingController();
    final focusNode = FocusNode();
    addTearDown(controller.dispose);
    addTearDown(focusNode.dispose);

    for (final theme in [vyroLightTheme(), vyroDarkTheme()]) {
      await tester.pumpWidget(
        MaterialApp(
          home: Theme(
            data: theme,
            child: Scaffold(
              body: ChatSearchCapsule(
                controller: controller,
                focusNode: focusNode,
                hintText: 'Search chats',
                clearTooltip: 'Clear search',
                onChanged: (_) {},
                onClear: controller.clear,
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      expect(theme.inputDecorationTheme.filled, isTrue);
      final field = find.byType(TextField);
      final decorator = tester.widget<InputDecorator>(
        find.descendant(of: field, matching: find.byType(InputDecorator)),
      );
      expect(decorator.decoration.filled, isFalse);
      expect(decorator.decoration.fillColor, Colors.transparent);
    }
  });
}
