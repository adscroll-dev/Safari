// Platform-specific helpers for voice recordings (v0.6).
//
// `record` returns a path from stop(): a file path on native platforms,
// a blob: URL on web. These helpers hide that difference so the rest of
// the app never imports dart:io (which would break the web build).
export 'recording_bytes_stub.dart'
    if (dart.library.io) 'recording_bytes_io.dart'
    if (dart.library.html) 'recording_bytes_web.dart';
