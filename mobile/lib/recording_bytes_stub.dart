// Fallback used only if neither dart:io nor dart:html is available
// (should not happen on any supported Flutter target).
import 'dart:typed_data';

Future<String> recordingOutputPath(String fileName) async =>
    throw UnsupportedError('Voice recording is not supported on this platform');

Future<Uint8List> readRecordedBytes(String path) async =>
    throw UnsupportedError('Voice recording is not supported on this platform');
