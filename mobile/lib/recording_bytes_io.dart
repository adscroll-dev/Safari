// Native (Android/iOS/desktop) implementation: recordings live in the
// system temp dir and are read back with dart:io.
import 'dart:io';
import 'dart:typed_data';

Future<String> recordingOutputPath(String fileName) async =>
    '${Directory.systemTemp.path}/$fileName';

Future<Uint8List> readRecordedBytes(String path) => File(path).readAsBytes();
