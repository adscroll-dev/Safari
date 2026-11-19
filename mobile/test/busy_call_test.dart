import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:vyro/api.dart';

void main() {
  test('startCallCheck throws BusyException on 409', () async {
    final client = MockClient((request) async {
      expect(request.method, 'POST');
      expect(request.url.path, '/api/calls/start');
      final body = jsonDecode(request.body) as Map<String, dynamic>;
      expect(body['callee_id'], 'bob');
      expect(body['call_type'], 'voice');
      return http.Response(jsonEncode({'error': 'user is busy'}), 409);
    });
    final api = ApiClient(baseUrl: 'http://test', client: client);

    await expectLater(
      api.startCallCheck(
        chatId: 'dm',
        calleeId: 'bob',
        callType: 'voice',
      ),
      throwsA(isA<BusyException>()),
    );
  });
}
