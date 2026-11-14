// VYRO mobile local-AI + call bitrate tests.
//
// The assistant, dictionary translation and smart replies are local
// and rule-based (no network, no cloud AI); these tests pin their
// real behaviour, including the honest coverage number. The call
// tests pin the Data Saver enforcement: Opus is capped at 12 kbps in
// the SDP, so a voice call can never negotiate more.
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/call.dart';
import 'package:vyro/data_saver.dart';
import 'package:vyro/local_ai.dart';

void main() {
  test('Assistant answers a greeting and help in both languages', () {
    final hi = assistantReply('hello');
    expect(hi.text, contains('Hello'));
    final help = assistantReply('help');
    expect(help.text, contains('VYRO'));
    final fa = assistantReply('سلام');
    expect(fa.lang, AiLang.fa);
    expect(fa.text, contains('سلام'));
    // Unknown general question: honest fallback, not a made-up answer.
    final other = assistantReply('what is the capital of France?');
    expect(other.text, contains('only know how VYRO works'));
  });

  test('Dictionary translation EN->FA with coverage', () {
    final tr = translateLocal('hello friend, how are you', AiLang.fa);
    expect(tr.text, contains('سلام'));
    expect(tr.text, contains('دوست'));
    expect(tr.coverage, greaterThan(0.8));
    expect(tr.knownWords, tr.totalWords);
    // Unknown words pass through and lower the coverage honestly.
    final partial = translateLocal('hello zxqwv', AiLang.fa);
    expect(partial.text, contains('zxqwv'));
    expect(partial.coverage, lessThan(1.0));
    // FA -> EN works the other way.
    final back = translateLocal('سلام دوست', AiLang.en);
    expect(back.text, contains('hello'));
    expect(back.text, contains('friend'));
  });

  test('Smart replies match the last message', () {
    expect(smartReplies('hello', AiLang.en).first, contains('Hello'));
    expect(smartReplies('how are you?', AiLang.en).first, contains("I'm good"));
    expect(smartReplies('thank you!', AiLang.en).first,
        contains("You're welcome"));
    expect(smartReplies('سلام', AiLang.fa).first, contains('سلام'));
    expect(smartReplies('', AiLang.en), isEmpty);
  });

  test('Call SDP caps Opus at 12 kbps and video at 150 kbps', () {
    const sdp = 'v=0\r\n'
        'm=audio 9 UDP/TLS/RTP/SAVPF 111\r\n'
        'a=rtpmap:111 opus/48000/2\r\n'
        'a=fmtp:111 minptime=10;useinbandfec=1\r\n'
        'm=video 9 UDP/TLS/RTP/SAVPF 96\r\n'
        'a=rtpmap:96 VP8/90000\r\n';
    final capped = CallController.capSdpBitrates(sdp, video: true);
    expect(capped, contains('maxaveragebitrate=12000'));
    expect(capped, contains('b=AS:12\r\n'));
    expect(capped, contains('b=AS:150\r\n'));
    // No cap is ever added twice.
    final twice = CallController.capSdpBitrates(capped, video: true);
    expect('maxaveragebitrate=12000'.allMatches(twice).length, 1);
  });

  test('Data Saver numbers: voice call 0.09 MB/min, video ~1.1 MB/min', () {
    // 12 kbps = 1.5 KB/s = 0.09 MB/min — far below WhatsApp 0.3-0.5.
    expect(DataSaver.voiceCallMbPerMin, closeTo(0.09, 0.01));
    expect(DataSaver.voiceCallMbPerMin, lessThan(0.15));
    expect(DataSaver.videoCallMbPerMin, closeTo(1.125, 0.01));
    // A 10,000-char text message is ~10 KB, not megabytes.
    expect(DataSaver.mbPerMinute(0), 0);
  });
}
