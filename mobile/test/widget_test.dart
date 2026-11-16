// Smoke test for the real Safari app: it boots to the login screen.
// Note: the login page has a forever-repeating drift animation, so
// pumpAndSettle never settles — pump a few frames instead.
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/main.dart';

void main() {
  testWidgets('Safari boots to the login screen', (WidgetTester tester) async {
    await tester.pumpWidget(const VyroApp());
    await tester.pump(const Duration(seconds: 1));
    await tester.pump(const Duration(seconds: 1));
    expect(find.text('Safari'), findsWidgets);
    expect(find.text('Log in'), findsOneWidget);
  });
}
