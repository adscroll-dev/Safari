import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vyro/sponsored_stories.dart';

const _demo = [
  SponsoredStory(id: 'd1', advertiserName: 'Demo A', title: 'Ad A'),
  SponsoredStory(id: 'd2', advertiserName: 'Demo B', title: 'Ad B'),
];

void main() {
  group('buildStoryFeed', () {
    test('no sponsored stories when there are no user stories', () {
      expect(buildStoryFeed(0, _demo), isEmpty);
    });

    test('no sponsored slot before the first group of 5', () {
      final feed = buildStoryFeed(4, _demo);
      expect(feed.length, 4);
      expect(feed.every((i) => i is UserStoryItem), isTrue);
    });

    test('one sponsored slot after every 5 user stories', () {
      final feed = buildStoryFeed(10, _demo);
      expect(feed.whereType<SponsoredStoryItem>().length, 2);
      // Sponsored is never first and never before the 5th user story.
      expect(feed.first, isA<UserStoryItem>());
      expect(feed[5], isA<SponsoredStoryItem>());
      expect(feed[11], isA<SponsoredStoryItem>());
    });

    test('at most 3 sponsored slots per feed', () {
      final feed = buildStoryFeed(25, _demo);
      expect(feed.whereType<SponsoredStoryItem>().length, 3);
      expect(feed.whereType<UserStoryItem>().length, 25);
    });

    test('no sponsored stories when the cap is hit', () {
      final feed = buildStoryFeed(10, _demo, sponsoredAvailable: false);
      expect(feed.whereType<SponsoredStoryItem>(), isEmpty);
      expect(feed.length, 10);
    });

    test('sponsored stories rotate through the inventory', () {
      final feed = buildStoryFeed(15, _demo);
      final ids =
          feed.whereType<SponsoredStoryItem>().map((i) => i.story.id).toList();
      expect(ids, ['d1', 'd2', 'd1']);
    });
  });

  group('SponsoredCapTracker', () {
    test('session cap blocks after 3 views', () async {
      SharedPreferences.setMockInitialValues({});
      final tracker = SponsoredCapTracker();
      expect(await tracker.canShow(), isTrue);
      await tracker.recordView();
      await tracker.recordView();
      await tracker.recordView();
      expect(tracker.sessionViews, 3);
      expect(await tracker.canShow(), isFalse);
    });

    test('daily cap blocks after 5 views across trackers', () async {
      SharedPreferences.setMockInitialValues({});
      // Session cap is 3, so use two trackers to reach the daily cap.
      final first = SponsoredCapTracker();
      for (var i = 0; i < 3; i++) {
        await first.recordView();
      }
      final second = SponsoredCapTracker();
      expect(await second.canShow(), isTrue);
      await second.recordView();
      await second.recordView();
      expect(await second.canShow(), isFalse);
    });
  });

  group('SponsoredSubmission', () {
    test('json round-trip keeps every field', () {
      final sub = SponsoredSubmission(
        advertiserName: 'Acme',
        contactEmail: 'ads@acme.example',
        linkUrl: 'https://acme.example',
        title: 'Acme sale',
        submittedAt: DateTime.utc(2026, 11, 12, 10, 0),
      );
      final back = SponsoredSubmission.fromJson(sub.toJson());
      expect(back.advertiserName, 'Acme');
      expect(back.contactEmail, 'ads@acme.example');
      expect(back.linkUrl, 'https://acme.example');
      expect(back.title, 'Acme sale');
      expect(back.submittedAt, DateTime.utc(2026, 11, 12, 10, 0));
    });

    test('store saves and lists submissions on-device', () async {
      SharedPreferences.setMockInitialValues({});
      final store = SponsoredSubmissionsStore();
      expect(await store.list(), isEmpty);
      await store.save(SponsoredSubmission(
        advertiserName: 'Acme',
        contactEmail: 'ads@acme.example',
        linkUrl: 'https://acme.example',
        title: 'Acme sale',
        submittedAt: DateTime.utc(2026, 11, 12, 10, 0),
      ));
      final listed = await store.list();
      expect(listed.length, 1);
      expect(listed.first.advertiserName, 'Acme');
    });
  });

  group('demo inventory', () {
    test('every demo entry is flagged demo and labelled as such', () {
      expect(demoSponsoredStories, isNotEmpty);
      for (final s in demoSponsoredStories) {
        expect(s.isDemo, isTrue);
        expect(s.advertiserName, isNotEmpty);
        expect(s.title, isNotEmpty);
      }
    });
  });
}
