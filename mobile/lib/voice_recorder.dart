// Voice messages in the Flutter client (v0.6).
//
// Recording uses the free `record` package; playback uses the free
// `just_audio` package. Both support Android, iOS and web. A recording
// is uploaded through the existing POST /api/uploads endpoint and sent
// as a real message with kind 'voice' — no fake UI: the mic button only
// appears when a chat is open, and failures are shown as errors.
//
// DATA SAVER (v0.6): recordings use the Opus encoder at 12 kbps
// (0.09 MB/min) instead of raw WAV (~256 kbps, ~1.9 MB/min) —
// roughly 20x less data per voice message. On platforms where Opus
// is unavailable the recorder falls back to AAC-LC at 16 kbps, and
// only then to WAV. The recorded bytes are counted in the usage
// meter under the "voice" category.
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:just_audio/just_audio.dart';
import 'package:record/record.dart';

import 'api.dart';
import 'chat_extras.dart';
import 'data_saver.dart';
import 'recording_bytes.dart';
import 'theme.dart';

/// Start a voice recording with the Data Saver encoder ladder (Opus
/// 12 kbps, then AAC-LC 16 kbps, then WAV). Returns the MIME type on
/// success, null when no encoder started. The caller stops the
/// recorder and reads the file (see readRecordedBytes). Shared by the
/// chat mic button and the voice-status flow.
Future<String?> startVoiceRecording(AudioRecorder recorder) async {
  final attempts = <(AudioEncoder, int, String, String)>[
    (
      AudioEncoder.opus,
      DataSaver.voiceMessageBitrateKbps * 1000,
      'ogg',
      'audio/ogg'
    ),
    (AudioEncoder.aacLc, 16000, 'm4a', 'audio/mp4'),
    (AudioEncoder.wav, 16000, 'wav', 'audio/wav'),
  ];
  for (final (encoder, bitRate, ext, mime) in attempts) {
    try {
      final fileName = 'voice-${DateTime.now().millisecondsSinceEpoch}.$ext';
      final path = await recordingOutputPath(fileName);
      await recorder.start(
        RecordConfig(
          encoder: encoder,
          bitRate: bitRate,
          sampleRate: 16000,
          numChannels: 1,
        ),
        path: path,
      );
      return mime;
    } catch (_) {
      // Try the next encoder.
    }
  }
  return null;
}

/// Mic button for the chat input row. Tap to start, tap again to stop
/// and send the recording as a voice message.
class VoiceRecordButton extends StatefulWidget {
  const VoiceRecordButton({
    super.key,
    required this.api,
    required this.chatId,
    required this.onSent,
  });

  final ApiClient api;
  final String chatId;
  final VoidCallback onSent;

  @override
  State<VoiceRecordButton> createState() => _VoiceRecordButtonState();
}

class _VoiceRecordButtonState extends State<VoiceRecordButton> {
  final _recorder = AudioRecorder();
  bool _recording = false;
  bool _busy = false;
  Timer? _timer;
  int _seconds = 0;
  String? _error;

  @override
  void dispose() {
    _timer?.cancel();
    _recorder.dispose();
    super.dispose();
  }

  Future<void> _toggle() async {
    if (_busy) return;
    if (_recording) {
      await _stopAndSend();
      return;
    }
    setState(() {
      _error = null;
      _busy = true;
    });
    try {
      if (!await _recorder.hasPermission()) {
        throw Exception('Microphone permission denied');
      }
      // Data Saver: Opus at 12 kbps (0.09 MB/min). Fall back to AAC-LC
      // 16 kbps, then WAV, on platforms without an Opus encoder.
      final started = await _startRecording();
      if (!started) throw Exception('Could not start recording');
      _seconds = 0;
      _timer = Timer.periodic(
        const Duration(seconds: 1),
        (_) => mounted ? setState(() => _seconds++) : null,
      );
      setState(() {
        _recording = true;
        _busy = false;
      });
    } catch (e) {
      setState(() {
        _busy = false;
        _error = e.toString();
      });
    }
  }

  String _mime = 'audio/ogg';

  /// Try Opus 12 kbps, then AAC-LC 16 kbps, then WAV. Returns true
  /// when the recorder started.
  Future<bool> _startRecording() async {
    final mime = await startVoiceRecording(_recorder);
    if (mime == null) return false;
    _mime = mime;
    return true;
  }

  Future<void> _stopAndSend() async {
    _timer?.cancel();
    setState(() => _busy = true);
    try {
      final path = await _recorder.stop();
      setState(() => _recording = false);
      if (path == null || path.isEmpty) {
        throw Exception('Recording produced no file');
      }
      final bytes = await readRecordedBytes(path);
      if (bytes.isEmpty) throw Exception('Recording is empty');
      final fileName = path.split('/').last.split('?').first;
      final up = await widget.api
          .uploadFile(bytes, fileName, _mime, category: UsageCategory.voice);
      await widget.api.sendVoiceMessage(
        widget.chatId,
        up['url'] as String,
        up['file_size'] as int,
        up['file_name'] as String,
      );
      widget.onSent();
    } catch (e) {
      if (mounted) {
        setState(() => _error = e.toString());
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Voice message failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String get _label {
    final m = (_seconds ~/ 60).toString().padLeft(2, '0');
    final s = (_seconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (_recording)
          Padding(
            padding: const EdgeInsets.only(right: 4),
            child: Text('● $_label',
                style: const TextStyle(color: Colors.red, fontSize: 13)),
          ),
        Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(
            color: dark ? WaColors.accentDark : WaColors.accent,
            shape: BoxShape.circle,
          ),
          child: IconButton(
            icon: Icon(
              _recording ? Icons.stop_circle_outlined : Icons.mic_outlined,
              color: Colors.white,
            ),
            tooltip: _recording
                ? 'Stop and send voice message'
                : 'Record voice message',
            iconSize: 28,
            onPressed: _busy ? null : _toggle,
          ),
        ),
        if (_error != null)
          const Icon(Icons.error_outline, color: Colors.red, size: 18),
      ],
    );
  }
}

/// Player for a received/sent voice message. Streams the file from
/// /uploads/... with the user's auth token.
class VoiceMessagePlayer extends StatefulWidget {
  const VoiceMessagePlayer({
    super.key,
    required this.api,
    required this.fileUrl,
    required this.fileName,
  });

  final ApiClient api;
  final String fileUrl;
  final String fileName;

  @override
  State<VoiceMessagePlayer> createState() => _VoiceMessagePlayerState();
}

class _VoiceMessagePlayerState extends State<VoiceMessagePlayer> {
  final _player = AudioPlayer();
  bool _ready = false;
  bool _playing = false;
  String? _error;
  // Playback speed (v0.7): cycles 1x -> 1.5x -> 2x -> 1x, like
  // WhatsApp/Telegram voice notes.
  double _speed = 1.0;

  Future<void> _cycleSpeed() async {
    final next = nextPlaybackSpeed(_speed);
    try {
      await _player.setSpeed(next);
    } catch (_) {
      // Speed change is best-effort on some platforms; the label
      // only updates when the player accepted it.
      return;
    }
    if (mounted) setState(() => _speed = next);
  }

  @override
  void initState() {
    super.initState();
    _player.playerStateStream.listen((state) {
      if (!mounted) return;
      setState(() => _playing = state.playing);
    });
    _player
        .setUrl(
      '${widget.api.baseUrl}${widget.fileUrl}',
      headers: widget.api.authHeaders,
    )
        .then((_) {
      if (mounted) setState(() => _ready = true);
    }).catchError((Object e) {
      if (mounted) setState(() => _error = e.toString());
      return null;
    });
  }

  @override
  void dispose() {
    _player.dispose();
    super.dispose();
  }

  String _fmt(Duration? d) {
    if (d == null) return '0:00';
    final m = d.inMinutes;
    final s = d.inSeconds % 60;
    return '$m:${s.toString().padLeft(2, '0')}';
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<Duration>(
      stream: _player.positionStream,
      builder: (context, pos) {
        final total = _player.duration;
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              icon: Icon(_playing ? Icons.pause : Icons.play_arrow),
              tooltip: _playing ? 'Pause' : 'Play voice message',
              onPressed: (_ready && _error == null)
                  ? () => _playing ? _player.pause() : _player.play()
                  : null,
            ),
            const Icon(Icons.audio_file_outlined, size: 18),
            const SizedBox(width: 4),
            Text(
              _error != null
                  ? 'Voice message unavailable'
                  : '${_fmt(pos.data)} / ${_fmt(total)}',
              style: const TextStyle(fontSize: 13),
            ),
            const SizedBox(width: 2),
            TextButton(
              onPressed: (_ready && _error == null) ? _cycleSpeed : null,
              style: TextButton.styleFrom(
                padding: const EdgeInsets.symmetric(horizontal: 6),
                minimumSize: const Size(0, 28),
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              child: Text(
                formatPlaybackSpeed(_speed),
                style: const TextStyle(
                    fontSize: 12, fontWeight: FontWeight.w700),
              ),
            ),
          ],
        );
      },
    );
  }
}
