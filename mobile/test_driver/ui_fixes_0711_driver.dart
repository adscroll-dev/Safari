// Host-side driver for the 0.7.11 UI-fix screenshot integration test.
// Saves screenshots into ~/workspace/site-shots/.
import 'dart:io';
import 'dart:math';

import 'package:flutter_driver/flutter_driver.dart';
import 'package:integration_test/integration_test_driver_extended.dart';

Future<void> main() async {
  final driver = await FlutterDriver.connect();
  final window = await driver.webDriver.window;
  await window.setSize(const Rectangle<int>(0, 0, 390, 844));
  await integrationDriver(
    driver: driver,
    onScreenshot: (name, bytes, [args]) async {
      final file = File('/home/hatch/workspace/site-shots/$name.png');
      await file.create(recursive: true);
      await file.writeAsBytes(bytes);
      // ignore: avoid_print
      print('saved screenshot: ${file.path} (${bytes.length} bytes)');
      return true;
    },
  );
}
