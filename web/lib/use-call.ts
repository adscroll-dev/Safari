// Voice-call hook (WebRTC, v0.5; TURN relay added in v0.6). Signaling goes through VYRO's
// WebSocket (call-offer / call-answer / call-ice / call-end, relayed
// verbatim by the server); media is peer-to-peer via a public STUN
// server, with the self-hosted coturn TURN server as a relay when
// configured (see NEXT_PUBLIC_TURN_URL below). Calls work when the two peers
// can reach each other directly, which covers most home/office NATs;
// symmetric NATs fall back to the TURN relay instead of failing.
// Audio-only in v0.5; video is not wired yet (see README).
import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "./api";

export type CallState = "idle" | "outgoing" | "incoming" | "in-call";

export interface IncomingCall {
  chatId: string;
  peerId: string;
  peerName: string;
  offer: RTCSessionDescriptionInit;
}

// ICE servers for calls (v0.6). Default is STUN-only, exactly as in
// v0.5. When the self-hosted TURN server is configured via env
// (docker-compose sets these when the `turn` service runs), it is
// appended so relayed calls work behind symmetric NATs.
//   NEXT_PUBLIC_TURN_URL        e.g. "turn:localhost:3478" (comma-separated for several)
//   NEXT_PUBLIC_TURN_USERNAME   e.g. "vyro"
//   NEXT_PUBLIC_TURN_CREDENTIAL e.g. $TURN_PASSWORD
function buildIceServers(): RTCIceServer[] {
  const servers: RTCIceServer[] = [{ urls: "stun:stun.l.google.com:19302" }];
  const turnUrls = (process.env.NEXT_PUBLIC_TURN_URL ?? "")
    .split(",")
    .map((u) => u.trim())
    .filter(Boolean);
  if (turnUrls.length > 0) {
    servers.push({
      urls: turnUrls,
      username: process.env.NEXT_PUBLIC_TURN_USERNAME || "vyro",
      credential: process.env.NEXT_PUBLIC_TURN_CREDENTIAL || "",
    });
  }
  return servers;
}

function getRtcConfig(): RTCConfiguration {
  return { iceServers: buildIceServers() };
}

// Built fresh per call from the current environment.
const RTC_CONFIG: RTCConfiguration = getRtcConfig();

export function useCall(
  sendSignal: (chatId: string, type: string, toUserId: string, payload: unknown) => void,
) {
  const [callState, setCallState] = useState<CallState>("idle");
  const [incoming, setIncoming] = useState<IncomingCall | null>(null);
  const [callError, setCallError] = useState("");
  const [callPeer, setCallPeer] = useState("");
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);
  const callChatRef = useRef("");
  const callPeerRef = useRef("");
  const remoteAudioRef = useRef<HTMLAudioElement | null>(null);

  const cleanup = useCallback(() => {
    pcRef.current?.close();
    pcRef.current = null;
    localStreamRef.current?.getTracks().forEach((t) => t.stop());
    localStreamRef.current = null;
    setCallState("idle");
    setIncoming(null);
  }, []);

  const makePC = useCallback(
    (chatId: string, peerId: string) => {
      const pc = new RTCPeerConnection(RTC_CONFIG);
      pcRef.current = pc;
      pc.onicecandidate = (e) => {
        if (e.candidate) sendSignal(chatId, "call-ice", peerId, e.candidate.toJSON());
      };
      pc.ontrack = (e) => {
        if (remoteAudioRef.current && e.streams[0]) {
          remoteAudioRef.current.srcObject = e.streams[0];
          void remoteAudioRef.current.play().catch(() => undefined);
        }
      };
      pc.onconnectionstatechange = () => {
        if (pc.connectionState === "connected") setCallState("in-call");
        if (pc.connectionState === "failed" || pc.connectionState === "disconnected") {
          setCallError("Call connection failed — the peers could not connect directly or via the TURN relay.");
          cleanup();
        }
      };
      return pc;
    },
    [sendSignal, cleanup],
  );

  const startCall = useCallback(
    async (chatId: string, peerId: string, peerName: string) => {
      setCallError("");
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
        localStreamRef.current = stream;
        callChatRef.current = chatId;
        callPeerRef.current = peerId;
        setCallPeer(peerName);
        const pc = makePC(chatId, peerId);
        stream.getTracks().forEach((t) => pc.addTrack(t, stream));
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        sendSignal(chatId, "call-offer", peerId, offer);
        setCallState("outgoing");
      } catch (e) {
        setCallError(e instanceof Error ? `Could not start call: ${e.message}` : "Could not start call");
      }
    },
    [makePC, sendSignal],
  );

  const acceptCall = useCallback(async () => {
    if (!incoming) return;
    setCallError("");
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
      localStreamRef.current = stream;
      callChatRef.current = incoming.chatId;
      callPeerRef.current = incoming.peerId;
      setCallPeer(incoming.peerName);
      const pc = makePC(incoming.chatId, incoming.peerId);
      stream.getTracks().forEach((t) => pc.addTrack(t, stream));
      await pc.setRemoteDescription(incoming.offer);
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      sendSignal(incoming.chatId, "call-answer", incoming.peerId, answer);
      setCallState("in-call");
      setIncoming(null);
    } catch (e) {
      setCallError(e instanceof Error ? `Could not answer call: ${e.message}` : "Could not answer call");
      cleanup();
    }
  }, [incoming, makePC, sendSignal, cleanup]);

  const declineCall = useCallback(() => {
    if (!incoming) return;
    sendSignal(incoming.chatId, "call-end", incoming.peerId, { reason: "declined" });
    setIncoming(null);
  }, [incoming, sendSignal]);

  const hangup = useCallback(() => {
    if (callChatRef.current && callPeerRef.current) {
      sendSignal(callChatRef.current, "call-end", callPeerRef.current, { reason: "hangup" });
    }
    cleanup();
  }, [sendSignal, cleanup]);

  // Handle one incoming signaling event from the WebSocket. Returns
  // true when the event was a call event.
  const handleCallEvent = useCallback(
    (ev: api.WSEvent): boolean => {
      if (!ev.type.startsWith("call-")) return false;
      const chatId = ev.chat_id ?? "";
      const peerId = ev.user_id ?? "";
      if (ev.type === "call-offer") {
        setIncoming({ chatId, peerId, peerName: peerId, offer: ev.payload as RTCSessionDescriptionInit });
        setCallState("incoming");
        return true;
      }
      if (ev.type === "call-answer" && pcRef.current) {
        void pcRef.current.setRemoteDescription(ev.payload as RTCSessionDescriptionInit).catch((e) => {
          setCallError(`Could not complete call: ${e instanceof Error ? e.message : "bad answer"}`);
          cleanup();
        });
        return true;
      }
      if (ev.type === "call-ice" && pcRef.current) {
        void pcRef.current.addIceCandidate(ev.payload as RTCIceCandidateInit).catch(() => undefined);
        return true;
      }
      if (ev.type === "call-end") {
        cleanup();
        return true;
      }
      return true;
    },
    [cleanup],
  );

  useEffect(() => () => cleanup(), [cleanup]);

  return {
    callState,
    incoming,
    callError,
    callPeer,
    remoteAudioRef,
    startCall,
    acceptCall,
    declineCall,
    hangup,
    handleCallEvent,
  };
}
