import 'package:http/http.dart' as http;

/// Website link preview (Stage 8, WhatsApp-style).
///
/// For a text message that contains a link, fetch the page's
/// <title> (5 s timeout, 1 MB cap) and show it as a small card
/// above the message. Titles are cached in memory so scrolling
/// never refetches. A failed fetch returns null and the message
/// simply shows without a preview — never an error.
class LinkPreviewCache {
  final Map<String, String?> _cache = {};

  /// The first http(s) URL in [text], or null.
  static String? extractUrl(String text) {
    final m = RegExp(r'https?://[^\s]+').firstMatch(text);
    if (m == null) return null;
    // Trailing punctuation is usually not part of the link.
    return m.group(0)!.replaceAll(RegExp(r'[.,;:!?)]+$'), '');
  }

  /// Page title for [url] (cached). Null when the page cannot be
  /// fetched or has no title.
  Future<String?> titleFor(String url) async {
    if (_cache.containsKey(url)) return _cache[url];
    String? title;
    try {
      final res = await http
          .get(Uri.parse(url), headers: {'User-Agent': 'Safari/0.6'})
          .timeout(const Duration(seconds: 5));
      if (res.statusCode == 200 && res.body.length < 1024 * 1024) {
        final m = RegExp(r'<title[^>]*>(.*?)</title>',
                caseSensitive: false, dotAll: true)
            .firstMatch(res.body);
        if (m != null) {
          title = m
              .group(1)!
              .replaceAll(RegExp(r'\s+'), ' ')
              .replaceAll('&amp;', '&')
              .replaceAll('&lt;', '<')
              .replaceAll('&gt;', '>')
              .replaceAll('&quot;', '"')
              .replaceAll('&#39;', "'")
              .trim();
          if (title.isEmpty) title = null;
          if (title != null && title.length > 120) {
            title = '${title.substring(0, 117)}…';
          }
        }
      }
    } catch (_) {
      title = null;
    }
    _cache[url] = title;
    return title;
  }
}
