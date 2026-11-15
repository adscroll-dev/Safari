import 'package:flutter/services.dart';

/// System share sheet for a file (Stage 8, chat export).
///
/// Implemented as a tiny MethodChannel in MainActivity (ACTION_SEND
/// with a FileProvider content:// URI) instead of the share_plus
/// plugin, whose Android Gradle plugin requirement is newer than
/// this offline Gradle 8.3 build can resolve.
class ShareFile {
  static const _channel = MethodChannel('safari/share');

  /// Opens the Android share sheet with [path] attached.
  /// On other platforms this is a no-op that returns false.
  static Future<bool> shareFile(String path, {String? text}) async {
    try {
      await _channel.invokeMethod<void>('shareFile', {
        'path': path,
        if (text != null) 'text': text,
      });
      return true;
    } on MissingPluginException {
      return false;
    } on PlatformException {
      return false;
    }
  }
}
