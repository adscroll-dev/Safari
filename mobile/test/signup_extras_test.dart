import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vyro/api.dart';
import 'package:vyro/main.dart';

class _SignupApi extends ApiClient {
  _SignupApi() : super(baseUrl: 'http://test');

  String? registeredInviteCode;

  @override
  Future<Map<String, dynamic>> register(
    String email,
    String displayName,
    String password, {
    String? inviteCode,
    String? deviceId,
    String? deviceName,
    String? platform,
  }) async {
    registeredInviteCode = inviteCode;
    return {'status': 'code_sent', 'username': 'inviteuser'};
  }

  @override
  Future<Map<String, dynamic>> login(
    String email,
    String password, {
    String? deviceId,
    String? deviceName,
    String? platform,
  }) async =>
      {'status': 'code_sent'};

  @override
  Future<Map<String, dynamic>> verifyLogin(
    String email,
    String code, {
    String? deviceId,
    String? deviceName,
    String? platform,
  }) async {
    return {
      'token': 'device-token',
      'new_device': true,
      'user': {
        'id': 'test-user',
        'email': email,
        'username': 'testuser',
        'display_name': 'Test User',
      },
    };
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  Future<void> openAuth(WidgetTester tester, _SignupApi api) async {
    SharedPreferences.setMockInitialValues({});
    tester.view.physicalSize = const Size(800, 1400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(MaterialApp(
      home: HomePage(
        apiClient: api,
        onToggleTheme: () {},
        onToggleLang: () {},
      ),
    ));
    await tester.pump(const Duration(seconds: 1));
    await tester.pump(const Duration(seconds: 1));
  }

  testWidgets('invite field is register-only and is sent with signup',
      (tester) async {
    final api = _SignupApi();
    await openAuth(tester, api);

    expect(find.byKey(const ValueKey('signup-invite-code')), findsNothing);
    final toggle = find.text('New to Safari? Create an account');
    await tester.ensureVisible(toggle);
    await tester.tap(toggle);
    await tester.pump();
    expect(find.byKey(const ValueKey('signup-invite-code')), findsOneWidget);

    await tester.enterText(
        find.byKey(const ValueKey('signup-display-name')), 'Invite Test');
    await tester.enterText(
        find.byKey(const ValueKey('signup-email')), 'invite@example.com');
    await tester.enterText(
        find.byKey(const ValueKey('signup-password')), 'password-123');
    await tester.enterText(
        find.byKey(const ValueKey('signup-invite-code')), 'ABCD2345');
    await tester.tap(find.text('Create account'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(api.registeredInviteCode, 'ABCD2345');
    expect(find.byKey(const ValueKey('signup-invite-code')), findsNothing);
  });

  testWidgets('new-device login response displays a notice', (tester) async {
    final api = _SignupApi();
    await openAuth(tester, api);
    await tester.enterText(
        find.byKey(const ValueKey('signup-email')), 'login@example.com');
    await tester.enterText(
        find.byKey(const ValueKey('signup-password')), 'password-123');
    await tester.tap(find.text('Log in'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    await tester.enterText(find.byKey(const ValueKey('signup-code')), '123456');
    await tester.tap(find.text('Verify and continue'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(find.text('Signed in from a new device'), findsOneWidget);
    await tester.pump(const Duration(seconds: 11));
    await tester.pump();
  });
}
