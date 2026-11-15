// VYRO voice/video calls in the Flutter client (v0.6).
//
// Signaling goes through the backend WebSocket exactly like the web
// client (call-offer / call-answer / call-ice / call-end, relayed
// verbatim by the server). Media is WebRTC peer-to-peer with a public
// STUN server, plus the self-hosted coturn TURN relay when configured
// (see kTurnUrl — empty by default, STUN-only, same as web).
//
// DATA SAVER (user requirement: VYRO must use far less data than
// Telegram/WhatsApp):
// - Voice calls: Opus is capped at 12 kbps via SDP
//   (maxaveragebitrate=12000). 12 kbps = 0.09 MB/min, vs WhatsApp
//   voice calls at ~0.3-0.5 MB/min and Telegram at ~0.4 MB/min.
//   The cap is enforced by rewriting the SDP before it is sent, so
//   the peer cannot negotiate a higher audio bitrate.
// - Video calls start at 320x240 @ ~150 kbps (~1.1 MB/min) and the
//   user can drop to audio-only at any time (audioOnly toggle).
// - Call bytes are estimated from negotiated bitrate x duration and
//   recorded in the DataUsageTracker, so the Settings meter shows
//   real call data use.
// - No call is ever faked: the call button only appears in a DM with
//   another user, and a call that cannot connect shows the error.
import 'dart:async';

import 'package:flutter_webrtc/flutter_webrtc.dart';

import 'data_saver.dart';

/// Default ICE servers. TURN is empty by default (STUN-only, like the
/// web client); set kTurnUrl/kTurnUser/kTurnCredential to point at the
/// self-hosted coturn relay from docker-compose.
const String kStunUrl = 'stun:stun.l.google.com:19302';
const String kTurnUrl = ''; // e.g. 'turn:your-server:3478'
const String kTurnUser = 'vyro';
const String kTurnCredential = '';

enum CallState { idle, outgoing, incoming, inCall }

class IncomingCall {
  IncomingCall(
      {required this.chatId,
      required this.peerId,
      required this.offer,
      this.group = false});
  final String chatId;
  final String peerId;
  final Map<String, dynamic> offer;

  /// True for a group-call invite (Stage 3): accepting joins the mesh.
  final bool group;
}

/// Sends one signaling frame over the existing WebSocket.
typedef SendSignal = void Function(
    String chatId, String type, String toUserId, Map<String, dynamic> payload);

class CallController {
  CallController({required this.sendSignal, required this.usage});

  final SendSignal sendSignal;
  final DataUsageTracker usage;

  RTCPeerConnection? _pc;
  MediaStream? _localStream;
  MediaStream? _remoteStream;
  CallState state = CallState.idle;
  String peerId = '';
  String chatId = '';
  bool video = false;
  bool audioOnly = false;
  DateTime? _callStartedAt;
  final int _negotiatedAudioKbps = DataSaver.voiceCallBitrateKbps;
  int _negotiatedVideoKbps = DataSaver.videoCallBitrateKbps;
  final _stateController = StreamController<CallState>.broadcast();
  final _remoteStreamController = StreamController<MediaStream?>.broadcast();

  // Stage 3: in-call controls.
  bool micMuted = false;
  bool cameraOff = false;
  bool _iceRestarted = false;

  // Stage 3: group calls (mesh) — one peer connection per member.
  bool isGroup = false;
  final Map<String, RTCPeerConnection> _groupPcs = {};
  final Map<String, MediaStream?> _groupRemoteStreams = {};
  final Set<String> _groupPeers = {};
  final Map<String, Map<String, dynamic>> _pendingGroupOffers = {};
  final _groupStreamsController = StreamController<void>.broadcast();

  Stream<CallState> get onState => _stateController.stream;
  Stream<MediaStream?> get onRemoteStream => _remoteStreamController.stream;
  MediaStream? get remoteStream => _remoteStream;

  /// Fires whenever a group member's remote stream appears or leaves.
  Stream<void> get onGroupStreams => _groupStreamsController.stream;

  /// Remote streams of the group call, keyed by peer user id.
  Map<String, MediaStream?> get groupRemoteStreams =>
      Map.unmodifiable(_groupRemoteStreams);

  bool get connected => state == CallState.inCall;
  DateTime? get callStartedAt => _callStartedAt;
  int get durationSeconds => _callStartedAt == null
      ? 0
      : DateTime.now().difference(_callStartedAt!).inSeconds;

  Map<String, dynamic> _iceServers() {
    final servers = <Map<String, dynamic>>[
      {'urls': kStunUrl},
    ];
    if (kTurnUrl.isNotEmpty) {
      servers.add({
        'urls': kTurnUrl,
        'username': kTurnUser,
        'credential': kTurnCredential,
      });
    }
    return {'iceServers': servers};
  }

  /// Cap Opus at 12 kbps and video at 150 kbps in the SDP, so the
  /// call can never negotiate more than the Data Saver budget.
  /// This is the enforcement point for the low-data requirement.
  /// Idempotent: calling it twice adds nothing twice.
  static String capSdpBitrates(String sdp, {bool video = false}) {
    var out = sdp;
    // Opus: find its payload type from the rtpmap line, then cap the
    // fmtp line for that payload type (the fmtp line itself never
    // mentions "opus").
    final rtpmap = RegExp(r'a=rtpmap:(\d+) opus/48000').firstMatch(out);
    if (rtpmap != null) {
      final pt = rtpmap.group(1)!;
      const cap = 'maxaveragebitrate=${DataSaver.voiceCallBitrateKbps * 1000}';
      if (!out.contains(cap)) {
        final fmtp = RegExp('a=fmtp:$pt ([^\\r\\n]*)');
        if (fmtp.hasMatch(out)) {
          out = out.replaceFirstMapped(fmtp, (m) {
            final params = m.group(1)!;
            return 'a=fmtp:$pt $params;$cap;useinbandfec=1';
          });
        } else {
          // No fmtp line for Opus yet: add one after the rtpmap line.
          final rtpLine = RegExp('a=rtpmap:$pt opus/48000[^\\r\\n]*\\r?\\n')
              .firstMatch(out);
          if (rtpLine != null) {
            out = out.replaceFirst(rtpLine.group(0)!,
                '${rtpLine.group(0)}a=fmtp:$pt $cap;useinbandfec=1\r\n');
          }
        }
      }
    }
    // b=AS lines as a second cap (audio and video), added once only.
    if (!out.contains('b=AS:${DataSaver.voiceCallBitrateKbps}\r\n') &&
        !out.contains('b=AS:${DataSaver.voiceCallBitrateKbps}\n')) {
      out = out.replaceFirstMapped(RegExp(r'(m=audio \d+ [^\r\n]*\r?\n)'),
          (m) => '${m.group(1)}b=AS:${DataSaver.voiceCallBitrateKbps}\r\n');
    }
    if (video &&
        !out.contains('b=AS:${DataSaver.videoCallBitrateKbps}\r\n') &&
        !out.contains('b=AS:${DataSaver.videoCallBitrateKbps}\n')) {
      out = out.replaceFirstMapped(RegExp(r'(m=video \d+ [^\r\n]*\r?\n)'),
          (m) => '${m.group(1)}b=AS:${DataSaver.videoCallBitrateKbps}\r\n');
    }
    return out;
  }

  /// Watch a peer connection's ICE state and restart ICE once if it
  /// fails (Stage 3 reconnect): a brief network drop should not kill
  /// the call.
  void _watchIce(RTCPeerConnection pc) {
    pc.onIceConnectionState = (state) {
      if (state == RTCIceConnectionState.RTCIceConnectionStateFailed &&
          !_iceRestarted) {
        _iceRestarted = true;
        unawaited(pc.restartIce());
      }
    };
  }

  Future<RTCPeerConnection> _newPc() async {
    final pc = await createPeerConnection(_iceServers(), {
      'mandatory': {},
      'optional': [
        {'DtlsSrtpKeyAgreement': true},
      ],
    });
    pc.onIceCandidate = (candidate) {
      if (candidate.candidate != null && chatId.isNotEmpty) {
        sendSignal(chatId, 'call-ice', peerId, candidate.toMap());
      }
    };
    pc.onTrack = (event) {
      if (event.streams.isNotEmpty) {
        _remoteStream = event.streams[0];
        _remoteStreamController.add(_remoteStream);
      }
    };
    pc.onAddStream = (stream) {
      _remoteStream = stream;
      _remoteStreamController.add(_remoteStream);
    };
    _watchIce(pc);
    return pc;
  }

  /// One peer connection for a single member of a group call
  /// (Stage 3, mesh): ICE and remote streams are routed per peer.
  Future<RTCPeerConnection> _newPcForPeer(String peer) async {
    final pc = await createPeerConnection(_iceServers(), {
      'mandatory': {},
      'optional': [
        {'DtlsSrtpKeyAgreement': true},
      ],
    });
    pc.onIceCandidate = (candidate) {
      if (candidate.candidate != null && chatId.isNotEmpty) {
        sendSignal(chatId, 'call-ice', peer, candidate.toMap());
      }
    };
    pc.onTrack = (event) {
      if (event.streams.isNotEmpty) {
        _groupRemoteStreams[peer] = event.streams[0];
        _groupStreamsController.add(null);
      }
    };
    pc.onAddStream = (stream) {
      _groupRemoteStreams[peer] = stream;
      _groupStreamsController.add(null);
    };
    _watchIce(pc);
    return pc;
  }

  Future<MediaStream> _getLocalMedia({required bool video}) async {
    final constraints = <String, dynamic>{
      'audio': true,
      'video': video
          ? {
              'mandatory': {
                'minWidth': '320',
                'minHeight': '240',
                'maxWidth': '320',
                'maxHeight': '240',
                'minFrameRate': '15',
                'maxFrameRate': '15',
              },
              'facingMode': 'user',
              'optional': [],
            }
          : false,
    };
    return navigator.mediaDevices.getUserMedia(constraints);
  }

  /// Start an outgoing call (audio, or video if [video] is true).
  Future<void> startCall(
      {required String chatId,
      required String peerId,
      bool video = false}) async {
    this.chatId = chatId;
    this.peerId = peerId;
    this.video = video;
    audioOnly = !video;
    _iceRestarted = false;
    _pc = await _newPc();
    _localStream = await _getLocalMedia(video: video);
    _localStream!.getTracks().forEach(_pc!.addTrack);
    final offer = await _pc!.createOffer(
        {'offerToReceiveAudio': 1, 'offerToReceiveVideo': video ? 1 : 0});
    final capped = capSdpBitrates(offer.sdp ?? '', video: video);
    await _pc!.setLocalDescription(RTCSessionDescription(capped, offer.type));
    state = CallState.outgoing;
    _stateController.add(state);
    sendSignal(chatId, 'call-offer', peerId, {
      'sdp': capped,
      'type': offer.type,
      'video': video,
    });
  }

  /// Handle an incoming call-offer (called from the WS event loop).
  /// Returns the incoming call for the UI to accept/decline.
  IncomingCall handleOffer(
      String chatId, String peerId, Map<String, dynamic> offer) {
    this.chatId = chatId;
    this.peerId = peerId;
    video = offer['video'] == true;
    state = CallState.incoming;
    _stateController.add(state);
    return IncomingCall(chatId: chatId, peerId: peerId, offer: offer);
  }

  Future<void> acceptCall(IncomingCall call) async {
    _pc = await _newPc();
    _iceRestarted = false;
    _localStream = await _getLocalMedia(video: video);
    _localStream!.getTracks().forEach(_pc!.addTrack);
    await _pc!.setRemoteDescription(RTCSessionDescription(
        call.offer['sdp'] as String, call.offer['type'] as String));
    final answer = await _pc!.createAnswer(
        {'offerToReceiveAudio': 1, 'offerToReceiveVideo': video ? 1 : 0});
    final capped = capSdpBitrates(answer.sdp ?? '', video: video);
    await _pc!.setLocalDescription(RTCSessionDescription(capped, answer.type));
    _callStartedAt = DateTime.now();
    state = CallState.inCall;
    _stateController.add(state);
    sendSignal(call.chatId, 'call-answer', call.peerId, {
      'sdp': capped,
      'type': answer.type,
      'video': video,
    });
  }

  void declineCall(IncomingCall call) {
    sendSignal(call.chatId, 'call-end', call.peerId, {'reason': 'declined'});
    state = CallState.idle;
    _stateController.add(state);
  }

  // ---- Stage 3: group calls (mesh) ----

  /// Start a group call: one peer connection per member, each offered
  /// directly via the existing to_user_id relay.
  Future<void> startGroupCall(
      {required String chatId,
      required List<String> peerIds,
      bool video = false}) async {
    this.chatId = chatId;
    isGroup = true;
    this.video = video;
    audioOnly = !video;
    _iceRestarted = false;
    _groupPeers.addAll(peerIds);
    _localStream = await _getLocalMedia(video: video);
    for (final peer in peerIds) {
      final pc = await _newPcForPeer(peer);
      _groupPcs[peer] = pc;
      _localStream!.getTracks().forEach(pc.addTrack);
      final offer = await pc.createOffer(
          {'offerToReceiveAudio': 1, 'offerToReceiveVideo': video ? 1 : 0});
      final capped = capSdpBitrates(offer.sdp ?? '', video: video);
      await pc.setLocalDescription(RTCSessionDescription(capped, offer.type));
      sendSignal(chatId, 'call-offer', peer, {
        'sdp': capped,
        'type': offer.type,
        'video': video,
        'group': true,
      });
    }
    state = CallState.outgoing;
    _stateController.add(state);
  }

  /// Handle an incoming group-call offer. The offer is parked until
  /// the user accepts; media starts in acceptGroupCall.
  IncomingCall handleGroupOffer(
      String chatId, String peerId, Map<String, dynamic> offer) {
    this.chatId = chatId;
    isGroup = true;
    video = offer['video'] == true;
    _pendingGroupOffers[peerId] = offer;
    state = CallState.incoming;
    _stateController.add(state);
    return IncomingCall(
        chatId: chatId, peerId: peerId, offer: offer, group: true);
  }

  /// Accept a group call: answer the peer who invited us, then offer
  /// to every other member (mesh) so all pairs connect.
  Future<void> acceptGroupCall(
      IncomingCall call, List<String> otherMemberIds) async {
    _iceRestarted = false;
    _localStream ??= await _getLocalMedia(video: video);
    _groupPeers.addAll(otherMemberIds);
    _groupPeers.add(call.peerId);
    // Answer the peer who invited us.
    final pc = await _newPcForPeer(call.peerId);
    _groupPcs[call.peerId] = pc;
    _localStream!.getTracks().forEach(pc.addTrack);
    await pc.setRemoteDescription(RTCSessionDescription(
        call.offer['sdp'] as String, call.offer['type'] as String));
    final answer = await pc.createAnswer(
        {'offerToReceiveAudio': 1, 'offerToReceiveVideo': video ? 1 : 0});
    final capped = capSdpBitrates(answer.sdp ?? '', video: video);
    await pc.setLocalDescription(RTCSessionDescription(capped, answer.type));
    sendSignal(call.chatId, 'call-answer', call.peerId, {
      'sdp': capped,
      'type': answer.type,
      'video': video,
      'group': true,
    });
    // Offer to the remaining members (mesh).
    for (final peer in otherMemberIds) {
      if (peer == call.peerId || _groupPcs.containsKey(peer)) continue;
      final p2 = await _newPcForPeer(peer);
      _groupPcs[peer] = p2;
      _localStream!.getTracks().forEach(p2.addTrack);
      final offer = await p2.createOffer(
          {'offerToReceiveAudio': 1, 'offerToReceiveVideo': video ? 1 : 0});
      final cappedOffer = capSdpBitrates(offer.sdp ?? '', video: video);
      await p2.setLocalDescription(
          RTCSessionDescription(cappedOffer, offer.type));
      sendSignal(call.chatId, 'call-offer', peer, {
        'sdp': cappedOffer,
        'type': offer.type,
        'video': video,
        'group': true,
      });
    }
    _callStartedAt = DateTime.now();
    state = CallState.inCall;
    _stateController.add(state);
  }

  /// Decline a group-call invite: only the inviter is told.
  void declineGroupCall(IncomingCall call) {
    sendSignal(call.chatId, 'call-end', call.peerId, {'reason': 'declined'});
    _pendingGroupOffers.remove(call.peerId);
    if (_groupPcs.isEmpty) {
      isGroup = false;
      state = CallState.idle;
      _stateController.add(state);
    }
  }

  Future<void> handleGroupAnswer(
      String peerId, Map<String, dynamic> answer) async {
    final pc = _groupPcs[peerId];
    if (pc == null) return;
    await pc.setRemoteDescription(RTCSessionDescription(
        answer['sdp'] as String, answer['type'] as String));
    _callStartedAt ??= DateTime.now();
    state = CallState.inCall;
    _stateController.add(state);
  }

  Future<void> handleGroupIce(
      String peerId, Map<String, dynamic> candidate) async {
    final pc = _groupPcs[peerId];
    if (pc == null) return;
    await pc.addCandidate(RTCIceCandidate(
      candidate['candidate'] as String?,
      candidate['sdpMid'] as String?,
      candidate['sdpMLineIndex'] as int?,
    ));
  }

  /// End a group call for us: one broadcast call-end reaches every
  /// other member of the room.
  Future<void> endGroupCall({String reason = 'hangup'}) async {
    if (chatId.isNotEmpty) {
      sendSignal(chatId, 'call-end', '', {'reason': reason});
    }
    await _recordUsage();
    await _cleanupGroup();
    state = CallState.idle;
    _stateController.add(state);
  }

  /// One peer left a group call. When nobody is left, the call ends
  /// for us too.
  Future<void> handleRemoteEndForPeer(String peerId) async {
    final pc = _groupPcs.remove(peerId);
    try {
      await pc?.close();
    } catch (_) {}
    _groupRemoteStreams.remove(peerId);
    _groupPeers.remove(peerId);
    _groupStreamsController.add(null);
    if (_groupPcs.isEmpty && isGroup) {
      await _recordUsage();
      await _cleanupGroup();
      state = CallState.idle;
      _stateController.add(state);
    }
  }

  Future<void> _cleanupGroup() async {
    for (final pc in _groupPcs.values) {
      try {
        await pc.close();
      } catch (_) {}
    }
    _groupPcs.clear();
    _groupRemoteStreams.clear();
    _groupPeers.clear();
    _pendingGroupOffers.clear();
    await _cleanup();
  }

  // ---- Stage 3: in-call controls ----

  /// Mute/unmute the microphone mid-call.
  void toggleMute() {
    micMuted = !micMuted;
    for (final track in _localStream?.getAudioTracks() ?? const []) {
      track.enabled = !micMuted;
    }
  }

  /// Turn the camera on/off mid-call.
  void toggleCamera() {
    cameraOff = !cameraOff;
    for (final track in _localStream?.getVideoTracks() ?? const []) {
      track.enabled = !cameraOff;
    }
  }

  /// Switch between front and back camera mid-call.
  Future<bool> switchCamera() async {
    final tracks = _localStream?.getVideoTracks() ?? const [];
    if (tracks.isEmpty) return false;
    return Helper.switchCamera(tracks.first);
  }

  Future<void> handleAnswer(Map<String, dynamic> answer) async {
    if (_pc == null) return;
    await _pc!.setRemoteDescription(RTCSessionDescription(
        answer['sdp'] as String, answer['type'] as String));
    _callStartedAt ??= DateTime.now();
    state = CallState.inCall;
    _stateController.add(state);
  }

  Future<void> handleIce(Map<String, dynamic> candidate) async {
    if (_pc == null) return;
    await _pc!.addCandidate(RTCIceCandidate(
      candidate['candidate'] as String?,
      candidate['sdpMid'] as String?,
      candidate['sdpMLineIndex'] as int?,
    ));
  }

  /// Drop video mid-call and continue audio-only (Data Saver fallback).
  Future<void> dropToAudioOnly() async {
    if (_pc == null || _localStream == null) return;
    for (final track in _localStream!.getVideoTracks()) {
      track.enabled = false;
      await track.stop();
    }
    audioOnly = true;
    video = false;
    _negotiatedVideoKbps = 0;
  }

  Future<void> endCall({String reason = 'hangup'}) async {
    if (chatId.isNotEmpty && peerId.isNotEmpty) {
      sendSignal(chatId, 'call-end', peerId, {'reason': reason});
    }
    await _recordUsage();
    await _cleanup();
    state = CallState.idle;
    _stateController.add(state);
  }

  /// Remote peer ended the call.
  Future<void> handleRemoteEnd() async {
    await _recordUsage();
    await _cleanup();
    state = CallState.idle;
    _stateController.add(state);
  }

  Future<void> _recordUsage() async {
    if (_callStartedAt == null) return;
    final seconds = DateTime.now().difference(_callStartedAt!).inSeconds;
    if (seconds <= 0) return;
    final kbps = _negotiatedAudioKbps + (audioOnly ? 0 : _negotiatedVideoKbps);
    await usage.add(UsageCategory.calls, (kbps * 1000 / 8 * seconds).round());
    _callStartedAt = null;
  }

  Future<void> _cleanup() async {
    try {
      await _localStream?.dispose();
    } catch (_) {}
    try {
      await _pc?.close();
    } catch (_) {}
    _localStream = null;
    _remoteStream = null;
    _pc = null;
    chatId = '';
    peerId = '';
    isGroup = false;
    micMuted = false;
    cameraOff = false;
    _iceRestarted = false;
  }

  void dispose() {
    _stateController.close();
    _remoteStreamController.close();
    _groupStreamsController.close();
  }
}
