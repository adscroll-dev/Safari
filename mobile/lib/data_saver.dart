// VYRO Data Saver (v0.6 mobile) — the user requires VYRO to use FAR
// less internet data than Telegram and WhatsApp, in every part.
//
// Design (all numbers are enforced here, not just promised):
// - Data Saver is ON by default. When on, no photo/video/file/voice is
//   auto-downloaded: the UI shows a tap-to-download placeholder with the
//   file size, and nothing is fetched until the user taps it.
// - Voice messages are recorded with the Opus encoder at 12 kbps
//   (see voice_recorder.dart). 12 kbps = 0.09 MB/min, vs WhatsApp voice
//   calls at roughly 0.3-0.5 MB/min and Telegram at ~0.4 MB/min.
// - Voice calls cap Opus at 12 kbps (SDP munging in call.dart):
//   ~0.09 MB/min. Video starts at 320x240 @ ~150 kbps (~1.1 MB/min)
//   with an audio-only fallback.
// - Images are downscaled to at most 1280 px and re-encoded as JPEG
//   quality 70 before upload (pure-Dart `image` package), so a 4 MB
//   phone photo becomes ~150-300 KB on the wire.
// - Text/chat sync uses the existing WebSocket (no polling) and
//   50-message pages; no wasteful re-fetching.
//
// DataUsageTracker counts real bytes (API up/down, uploads, downloads,
// call bytes estimated from negotiated bitrate x duration) per
// category, persisted in SharedPreferences, reset monthly. The
// Settings screen shows "MB used this month" per activity so the
// saving is visible and measurable.
import 'dart:convert';
import 'dart:typed_data';

import 'package:image/image.dart' as img;
import 'package:shared_preferences/shared_preferences.dart';

/// Measured reference points (see mobile/README.md for method):
/// - WhatsApp voice call: ~0.3-0.5 MB/min (published measurements).
/// - Telegram voice call: ~0.4 MB/min. VYRO target: <= 0.12 MB/min.
class DataSaver {
  static const int voiceCallBitrateKbps = 12; // Opus, audio calls
  static const int videoCallBitrateKbps = 150; // starting video bitrate
  static const int voiceMessageBitrateKbps = 12; // Opus voice messages
  static const int maxImageDimension = 1280; // px, longest side
  static const int imageJpegQuality = 70;

  /// Bytes per minute at a given bitrate: kbps * 1000 / 8 * 60.
  static double mbPerMinute(int kbps) => kbps * 1000 / 8 * 60 / 1e6;

  /// MB/min for a VYRO voice call at the capped Opus bitrate.
  static double get voiceCallMbPerMin => mbPerMinute(voiceCallBitrateKbps);

  /// MB/min for a VYRO video call at the starting bitrate.
  static double get videoCallMbPerMin => mbPerMinute(videoCallBitrateKbps);

  /// Compress image bytes for upload: downscale to at most
  /// [maxImageDimension] px on the longest side, re-encode JPEG q70.
  /// Returns the original bytes unchanged when decoding fails or the
  /// image is already smaller than the compressed form.
  static Uint8List compressImage(Uint8List bytes) {
    final decoded = img.decodeImage(bytes);
    if (decoded == null) return bytes;
    img.Image out = decoded;
    if (decoded.width > maxImageDimension ||
        decoded.height > maxImageDimension) {
      out = img.copyResize(decoded,
          width: decoded.width >= decoded.height ? maxImageDimension : null,
          height: decoded.height > decoded.width ? maxImageDimension : null);
    }
    final compressed =
        Uint8List.fromList(img.encodeJpg(out, quality: imageJpegQuality));
    return compressed.length < bytes.length ? compressed : bytes;
  }
}

/// Usage categories shown in the Settings data meter.
enum UsageCategory { chat, media, voice, calls }

extension UsageCategoryLabel on UsageCategory {
  String get label => switch (this) {
        UsageCategory.chat => 'Chat (text & sync)',
        UsageCategory.media => 'Photos, files & stories',
        UsageCategory.voice => 'Voice messages',
        UsageCategory.calls => 'Voice & video calls',
      };
  String get key => name;
}

/// Counts real bytes per category for the current calendar month.
/// Persisted in SharedPreferences; a new month starts at zero.
class DataUsageTracker {
  static const String _prefsKey = 'vyro_data_usage_v1';
  static const String _monthKey = 'vyro_data_usage_month';

  final Map<UsageCategory, int> _bytes = {
    for (final c in UsageCategory.values) c: 0,
  };
  String _month = '';

  String get month => _month;
  int bytesFor(UsageCategory c) => _bytes[c] ?? 0;
  int get totalBytes => _bytes.values.fold(0, (a, b) => a + b);
  double mbFor(UsageCategory c) => bytesFor(c) / 1e6;
  double get totalMb => totalBytes / 1e6;

  static String _currentMonth() {
    final now = DateTime.now();
    return '${now.year}-${now.month.toString().padLeft(2, '0')}';
  }

  Future<void> load() async {
    final prefs = await SharedPreferences.getInstance();
    final month = _currentMonth();
    if (prefs.getString(_monthKey) != month) {
      _month = month;
      await prefs.setString(_monthKey, month);
      await prefs.setString(_prefsKey, jsonEncode({}));
      return;
    }
    _month = month;
    final raw = prefs.getString(_prefsKey);
    if (raw != null && raw.isNotEmpty) {
      final map = jsonDecode(raw) as Map<String, dynamic>;
      for (final c in UsageCategory.values) {
        _bytes[c] = (map[c.key] as num?)?.toInt() ?? 0;
      }
    }
  }

  Future<void> _save() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(
        _prefsKey,
        jsonEncode(
            {for (final c in UsageCategory.values) c.key: _bytes[c] ?? 0}));
  }

  /// Add [bytes] to a category and persist. Call this from the API
  /// client, upload/download paths and the call controller.
  Future<void> add(UsageCategory category, int bytes) async {
    if (bytes <= 0) return;
    _bytes[category] = (_bytes[category] ?? 0) + bytes;
    await _save();
  }

  /// Test helper / Settings "reset" — clears all counters.
  Future<void> reset() async {
    for (final c in UsageCategory.values) {
      _bytes[c] = 0;
    }
    await _save();
  }
}
