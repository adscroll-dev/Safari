import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:vyro/api.dart';
import 'package:vyro/main.dart';

void main() {
  test('discoverCommunities and joinCommunity use real API paths', () async {
    final requests = <http.Request>[];
    final client = MockClient((request) async {
      requests.add(request);
      if (request.url.path.endsWith('/discover')) {
        return http.Response(jsonEncode([
          {
            'id': 'community-1',
            'name': 'Public Group',
            'member_count': 12,
          }
        ]), 200);
      }
      return http.Response(jsonEncode({'status': 'pending'}), 200);
    });
    final api = ApiClient(baseUrl: 'http://test', client: client);

    final communities = await api.discoverCommunities(limit: 25, offset: 5);
    expect(communities, hasLength(1));
    expect(requests.single.url.path, '/api/communities/discover');
    expect(requests.single.url.queryParameters['limit'], '25');
    expect(requests.single.url.queryParameters['offset'], '5');

    final joined = await api.joinCommunity('community-1');
    expect(joined['status'], 'pending');
    expect(requests.last.method, 'POST');
    expect(requests.last.url.path, '/api/communities/community-1/join');
  });

  testWidgets('DiscoverCommunityCard renders and handles join states',
      (tester) async {
    var joinCalls = 0;
    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: DiscoverCommunityCard(
          community: const {
            'id': 'community-1',
            'name': 'Public Group',
            'description': 'A real community',
            'member_count': 12,
          },
          joinState: null,
          inFlight: false,
          onJoin: () => joinCalls++,
        ),
      ),
    ));

    expect(find.text('Public Group'), findsOneWidget);
    expect(find.textContaining('12 members'), findsOneWidget);
    await tester.tap(find.text('Join'));
    await tester.pump();
    expect(joinCalls, 1);

    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: DiscoverCommunityCard(
          community: const {
            'id': 'community-1',
            'name': 'Public Group',
            'member_count': 12,
          },
          joinState: 'pending',
          inFlight: false,
          onJoin: () {},
        ),
      ),
    ));
    await tester.pump();
    expect(find.text('Request sent'), findsOneWidget);
    final textButton = tester.widget<TextButton>(find.byType(TextButton));
    expect(textButton.onPressed, isNull);
  });

  testWidgets('CommunityJoinRequests reviews requests and removes them',
      (tester) async {
    final requests = <http.Request>[];
    final client = MockClient((request) async {
      requests.add(request);
      if (request.url.path.endsWith('/join-requests')) {
        return http.Response(jsonEncode([
          {'user_id': 'user-1', 'display_name': 'Requester One'},
          {'user_id': 'user-2', 'display_name': 'Requester Two'},
        ]), 200);
      }
      return http.Response(jsonEncode({'status': 'ok'}), 200);
    });
    final api = ApiClient(baseUrl: 'http://test', client: client);

    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: CommunityJoinRequests(api: api, communityId: 'community-1'),
      ),
    ));
    await tester.pumpAndSettle();
    expect(find.text('Requester One'), findsOneWidget);
    expect(find.text('Requester Two'), findsOneWidget);

    await tester.tap(find.byTooltip('Approve').first);
    await tester.pumpAndSettle();
    expect(find.text('Requester One'), findsNothing);
    expect(requests.last.method, 'POST');
    expect(requests.last.url.path,
        '/api/communities/community-1/join-requests/user-1/approve');

    await tester.tap(find.byTooltip('Reject').first);
    await tester.pumpAndSettle();
    expect(find.text('Requester Two'), findsNothing);
    expect(requests.last.url.path,
        '/api/communities/community-1/join-requests/user-2/reject');
  });
}
