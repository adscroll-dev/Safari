import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:vyro/api.dart';
import 'package:vyro/main.dart';

void main() {
  testWidgets('status exclusions load, add, remove, and refresh from the API',
      (tester) async {
    final requests = <http.Request>[];
    var exclusions = <Map<String, dynamic>>[
      {
        'owner_id': 'me',
        'user_id': 'bob',
        'display_name': 'Bob Example',
        'created_at': '2026-01-01T00:00:00Z',
      },
    ];
    final client = MockClient((request) async {
      requests.add(request);
      if (request.method == 'GET' &&
          request.url.path == '/api/users/me/status-exclusions') {
        return http.Response(jsonEncode(exclusions), 200);
      }
      if (request.method == 'DELETE' &&
          request.url.path == '/api/users/me/status-exclusions/bob') {
        exclusions = [];
        return http.Response(jsonEncode({'status': 'ok'}), 200);
      }
      if (request.method == 'POST' &&
          request.url.path == '/api/users/me/status-exclusions') {
        expect(jsonDecode(request.body), {'user_id': 'bob'});
        exclusions = [
          {
            'owner_id': 'me',
            'user_id': 'bob',
            'display_name': 'Bob Example',
            'created_at': '2026-01-01T00:00:00Z',
          },
        ];
        return http.Response(jsonEncode(exclusions.single), 201);
      }
      return http.Response('{}', 404);
    });
    final api = ApiClient(baseUrl: 'http://test', client: client);

    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: StatusExclusionsSettings(
          api: api,
          currentUserId: 'me',
          users: const {
            'me': {'id': 'me', 'display_name': 'Myself'},
            'bob': {'id': 'bob', 'display_name': 'Bob Example'},
          },
        ),
      ),
    ));
    await tester.pumpAndSettle();
    expect(find.text('Bob Example'), findsOneWidget);

    await tester.tap(find.text('Remove'));
    await tester.pumpAndSettle();
    expect(find.text('Bob Example'), findsNothing);
    expect(
      requests.any((r) =>
          r.method == 'DELETE' &&
          r.url.path == '/api/users/me/status-exclusions/bob'),
      isTrue,
    );

    await tester.tap(find.text('Add people'));
    await tester.pumpAndSettle();
    expect(find.text('Myself'), findsNothing);
    await tester.tap(find.text('Bob Example'));
    await tester.pumpAndSettle();
    expect(find.text('Bob Example'), findsOneWidget);
    expect(
      requests.any((r) =>
          r.method == 'POST' &&
          r.url.path == '/api/users/me/status-exclusions'),
      isTrue,
    );
    expect(
      requests.where((r) =>
          r.method == 'GET' && r.url.path == '/api/users/me/status-exclusions'),
      hasLength(3),
    );
  });
}
