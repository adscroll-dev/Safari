import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/call.dart';
import 'package:vyro/call_foreground_service.dart';
import 'package:vyro/data_saver.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('safari/call_service_test');
  final calls = <MethodCall>[];

  setUp(() {
    calls.clear();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
      calls.add(call);
      return null;
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('call service maps call start and end to native channel methods',
      () async {
    final service = MethodChannelCallForegroundService(channel: channel);
    await service.start(
      chatId: 'chat-1',
      peerName: 'Alex',
      video: true,
    );
    await service.stop();

    expect(calls.map((call) => call.method), ['startCall', 'stopCall']);
    expect(calls.first.arguments, {
      'chatId': 'chat-1',
      'peerName': 'Alex',
      'video': true,
    });
  });

  test('declining an incoming call stops the foreground service', () async {
    final service = MethodChannelCallForegroundService(channel: channel);
    final controller = CallController(
      sendSignal: (_, __, ___, ____) {},
      usage: DataUsageTracker(),
      foregroundService: service,
    );
    final incoming = controller.handleOffer('chat-1', 'peer-1', {
      'sdp': 'offer',
      'type': 'offer',
    });

    await controller.declineCall(incoming);

    expect(calls.map((call) => call.method), ['stopCall']);
    controller.dispose();
  });

  test('hanging up stops the foreground service', () async {
    final controller = CallController(
      sendSignal: (_, __, ___, ____) {},
      usage: DataUsageTracker(),
      foregroundService: MethodChannelCallForegroundService(channel: channel),
    );

    await controller.endCall();

    expect(calls.map((call) => call.method), ['stopCall']);
    controller.dispose();
  });
}
