// Sponsored stories — Safari earns money from day one through ads in the
// Stories section.
//
// Honesty rules for this file:
//  * The built-in inventory is DEMO content only (fictional businesses,
//    clearly labelled "Sponsored · Demo"). It is never presented as a
//    real advertiser and earns no real revenue.
//  * No fake metrics: no fake view/click counts, no fake revenue.
//  * Sponsored media is never auto-downloaded (Data Saver): demo entries
//    carry no remote media at all, so they cost zero bytes.
//  * Frequency cap: users are never spammed — at most one sponsored
//    story per 5 user stories in the feed, at most 3 sponsored views per
//    app session, at most 5 sponsored views per day (persisted).
//  * Business submissions are stored ON THIS DEVICE ONLY until the ad
//    service exists; nothing is sent anywhere and nothing is claimed
//    to be live. The submission sheet says so in plain language.
import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

/// How often sponsored stories may appear.
class SponsoredCaps {
  /// Insert one sponsored story after every [userStoriesPerSponsored]
  /// user stories in the feed.
  static const int userStoriesPerSponsored = 5;

  /// Maximum sponsored story views per app session.
  static const int maxViewsPerSession = 3;

  /// Maximum sponsored story views per calendar day (persisted).
  static const int maxViewsPerDay = 5;

  /// Maximum sponsored slots shown in one feed build.
  static const int maxSlotsPerFeed = 3;
}

/// One sponsored story (an ad slot inside Stories).
class SponsoredStory {
  const SponsoredStory({
    required this.id,
    required this.advertiserName,
    required this.title,
    this.linkUrl,
    this.callToAction = 'Learn more',
    this.isDemo = true,
  });

  final String id;
  final String advertiserName;
  final String title;

  /// Advertiser's website. Shown as text; the app copies it to the
  /// clipboard on tap (no in-app browser is wired yet).
  final String? linkUrl;
  final String callToAction;

  /// True for the built-in placeholder inventory. Real advertisers
  /// (once the ad service exists) will have isDemo == false.
  final bool isDemo;

  Map<String, dynamic> toJson() => {
        'id': id,
        'advertiserName': advertiserName,
        'title': title,
        'linkUrl': linkUrl,
        'callToAction': callToAction,
        'isDemo': isDemo,
      };

  factory SponsoredStory.fromJson(Map<String, dynamic> json) =>
      SponsoredStory(
        id: json['id'] as String,
        advertiserName: json['advertiserName'] as String,
        title: json['title'] as String,
        linkUrl: json['linkUrl'] as String?,
        callToAction: json['callToAction'] as String? ?? 'Learn more',
        isDemo: json['isDemo'] as bool? ?? true,
      );
}

/// Demo inventory: fictional businesses, clearly labelled as demo.
/// Nothing here is a real advertiser and nothing earns revenue.
const List<SponsoredStory> demoSponsoredStories = [
  SponsoredStory(
    id: 'demo-1',
    advertiserName: 'Demo Bakery',
    title: 'Fresh bread every morning — demo ad',
    linkUrl: 'https://example.com/demo-bakery',
  ),
  SponsoredStory(
    id: 'demo-2',
    advertiserName: 'Demo Mountain Tours',
    title: 'Guided hikes, small groups — demo ad',
    linkUrl: 'https://example.com/demo-tours',
    callToAction: 'See trips',
  ),
  SponsoredStory(
    id: 'demo-3',
    advertiserName: 'Safari',
    title: 'Safari — Connect Beyond Limits. Share Safari with a friend.',
    linkUrl: 'https://example.com',
    callToAction: 'Share Safari',
  ),
];

/// One feed entry: either a user story (index into the user list) or a
/// sponsored story.
sealed class StoryFeedItem {
  const StoryFeedItem();
}

class UserStoryItem extends StoryFeedItem {
  const UserStoryItem(this.index);
  final int index;
}

class SponsoredStoryItem extends StoryFeedItem {
  const SponsoredStoryItem(this.story);
  final SponsoredStory story;
}

/// Interleave sponsored stories into the user-story feed, honouring the
/// frequency cap. Pure function (no I/O) so it is unit-testable.
///
/// Rules:
///  * sponsored stories are never first;
///  * one sponsored slot after every [SponsoredCaps.userStoriesPerSponsored]
///    user stories;
///  * at most [SponsoredCaps.maxSlotsPerFeed] slots per feed;
///  * [sponsoredAvailable] is false when the daily/session cap is hit —
///    then no sponsored stories are inserted at all.
List<StoryFeedItem> buildStoryFeed(
  int userStoryCount,
  List<SponsoredStory> sponsored, {
  bool sponsoredAvailable = true,
}) {
  final feed = <StoryFeedItem>[];
  var slotsUsed = 0;
  var sponsoredIdx = 0;
  for (var i = 0; i < userStoryCount; i++) {
    feed.add(UserStoryItem(i));
    final afterGroup =
        (i + 1) % SponsoredCaps.userStoriesPerSponsored == 0;
    if (sponsoredAvailable &&
        sponsored.isNotEmpty &&
        afterGroup &&
        slotsUsed < SponsoredCaps.maxSlotsPerFeed) {
      feed.add(SponsoredStoryItem(sponsored[sponsoredIdx % sponsored.length]));
      sponsoredIdx++;
      slotsUsed++;
    }
  }
  return feed;
}

/// Tracks sponsored-story views against the daily + session caps.
/// Daily counts live in SharedPreferences (keyed by calendar date);
/// the session count is in-memory only.
class SponsoredCapTracker {
  static const String _prefsKey = 'vyro_sponsored_views_v1';

  int _sessionViews = 0;

  int get sessionViews => _sessionViews;

  /// True when another sponsored story may be shown right now.
  Future<bool> canShow() async {
    if (_sessionViews >= SponsoredCaps.maxViewsPerSession) return false;
    return await _todayViews() < SponsoredCaps.maxViewsPerDay;
  }

  /// Record one sponsored story view (call only when one was shown).
  Future<void> recordView() async {
    _sessionViews++;
    final prefs = await SharedPreferences.getInstance();
    final today = _todayKey();
    final raw = prefs.getString(_prefsKey);
    final counts = raw == null
        ? <String, int>{}
        : Map<String, int>.from(
            (jsonDecode(raw) as Map).map(
              (k, v) => MapEntry(k as String, (v as num).toInt()),
            ),
          );
    counts[today] = (counts[today] ?? 0) + 1;
    // Keep only the last 7 days of counters.
    final keys = counts.keys.toList()..sort();
    for (final k in keys.take(keys.length > 7 ? keys.length - 7 : 0)) {
      counts.remove(k);
    }
    await prefs.setString(_prefsKey, jsonEncode(counts));
  }

  Future<int> _todayViews() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_prefsKey);
    if (raw == null) return 0;
    final counts = jsonDecode(raw) as Map;
    return (counts[_todayKey()] as num?)?.toInt() ?? 0;
  }

  String _todayKey() {
    final now = DateTime.now();
    return '${now.year}-${now.month.toString().padLeft(2, '0')}-'
        '${now.day.toString().padLeft(2, '0')}';
  }
}

/// A business's request to run a sponsored story.
///
/// Stored ON THIS DEVICE ONLY (SharedPreferences) until the Safari ad
/// service exists. Nothing is uploaded, emailed, or sent anywhere, and
/// the submission sheet tells the business exactly that.
class SponsoredSubmission {
  const SponsoredSubmission({
    required this.advertiserName,
    required this.contactEmail,
    required this.linkUrl,
    required this.title,
    required this.submittedAt,
  });

  final String advertiserName;
  final String contactEmail;
  final String linkUrl;
  final String title;
  final DateTime submittedAt;

  Map<String, dynamic> toJson() => {
        'advertiserName': advertiserName,
        'contactEmail': contactEmail,
        'linkUrl': linkUrl,
        'title': title,
        'submittedAt': submittedAt.toIso8601String(),
      };

  factory SponsoredSubmission.fromJson(Map<String, dynamic> json) =>
      SponsoredSubmission(
        advertiserName: json['advertiserName'] as String,
        contactEmail: json['contactEmail'] as String,
        linkUrl: json['linkUrl'] as String,
        title: json['title'] as String,
        submittedAt: DateTime.parse(json['submittedAt'] as String),
      );
}

class SponsoredSubmissionsStore {
  static const String _prefsKey = 'vyro_sponsored_submissions_v1';

  Future<List<SponsoredSubmission>> list() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_prefsKey);
    if (raw == null) return const [];
    return (jsonDecode(raw) as List)
        .map((e) => SponsoredSubmission.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<void> save(SponsoredSubmission submission) async {
    final prefs = await SharedPreferences.getInstance();
    final existing = [...await list()];
    existing.add(submission);
    await prefs.setString(
      _prefsKey,
      jsonEncode(existing.map((s) => s.toJson()).toList()),
    );
  }
}
