import 'package:flutter/services.dart';

abstract interface class CallForegroundService {
  Future<void> start({
    required String chatId,
    required String peerName,
    required bool video,
  });

  Future<void> stop();
}

class MethodChannelCallForegroundService implements CallForegroundService {
  MethodChannelCallForegroundService({MethodChannel? channel})
      : _channel = channel ?? const MethodChannel('safari/call_service');

  final MethodChannel _channel;

  @override
  Future<void> start({
    required String chatId,
    required String peerName,
    required bool video,
  }) async {
    await _channel.invokeMethod<void>('startCall', {
      'chatId': chatId,
      'peerName': peerName,
      'video': video,
    });
  }

  @override
  Future<void> stop() async {
    await _channel.invokeMethod<void>('stopCall');
  }

  void setOpenCallHandler(void Function(String chatId)? handler) {
    _channel.setMethodCallHandler(handler == null
        ? null
        : (call) async {
            if (call.method == 'openCallScreen') {
              final chatId = call.arguments as String?;
              if (chatId != null && chatId.isNotEmpty) handler(chatId);
            }
          });
  }

  Future<String?> consumePendingNavigation() {
    return _channel.invokeMethod<String>('consumePendingCallNavigation');
  }
}
