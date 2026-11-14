// Host-side driver for the VYRO screenshot integration test.
//
// Connects to the app running in Chrome, resizes the browser window to a
// phone viewport (390x844), then runs the integration test, saving each
// screenshot it requests as a PNG under web/public/screenshots/.
import 'dart:io';
import 'dart:math';

import 'package:flutter_driver/flutter_driver.dart';
import 'package:integration_test/integration_test_driver_extended.dart';

Future<void> main() async {
  final driver = await FlutterDriver.connect();
  // Phone-sized viewport so the screenshots look like the mobile app.
  final window = await driver.webDriver.window;
  await window.setSize(const Rectangle<int>(0, 0, 390, 844));
  await integrationDriver(
    driver: driver,
    onScreenshot: (name, bytes, [args]) async {
      final file = File(
          '/home/hatch/workspace/vyro/web/public/screenshots/$name.png');
      await file.create(recursive: true);
      await file.writeAsBytes(bytes);
      // ignore: avoid_print
      print('saved screenshot: ${file.path} (${bytes.length} bytes)');
      return true;
    },
  );
}
