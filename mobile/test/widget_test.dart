// Smoke test for the real Safari app: it boots to the login screen.
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/main.dart';

void main() {
  testWidgets('Safari boots to the login screen', (WidgetTester tester) async {
    await tester.pumpWidget(const VyroApp());
    await tester.pumpAndSettle();
    expect(find.text('Safari'), findsWidgets);
    expect(find.text('Log in'), findsOneWidget);
  });
}
