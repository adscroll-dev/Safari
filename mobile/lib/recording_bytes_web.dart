// Web implementation: `record` ignores the output path on web and
// stop() returns a blob: URL, which is fetched back as bytes.
import 'dart:typed_data';
import 'package:http/http.dart' as http;

Future<String> recordingOutputPath(String fileName) async => fileName;

Future<Uint8List> readRecordedBytes(String path) async {
  final res = await http.get(Uri.parse(path));
  if (res.statusCode != 200) {
    throw Exception('Could not read recording (${res.statusCode})');
  }
  return res.bodyBytes;
}
