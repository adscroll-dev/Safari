// Smoke test for the real VYRO app: it boots to the login screen.
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/main.dart';

void main() {
  testWidgets('VYRO boots to the login screen', (WidgetTester tester) async {
    await tester.pumpWidget(const VyroApp());
    await tester.pumpAndSettle();
    expect(find.text('VYRO'), findsWidgets);
    expect(find.text('Log in'), findsOneWidget);
  });
}
