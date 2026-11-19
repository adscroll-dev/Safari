// Smoke test for the real Safari app: it boots to the login screen.
// Note: the login page has a forever-repeating drift animation, so
// pumpAndSettle never settles — pump a few frames instead.
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/local_ai.dart';
import 'package:vyro/main.dart';

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
}
