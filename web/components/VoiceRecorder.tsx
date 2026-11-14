"use client";
import { useEffect, useRef, useState } from "react";
import { fmtDuration } from "../lib/local-ai";

// v0.6: Voice message recording with the browser MediaRecorder API.
// No paid service, no server round-trip while recording: audio is
// captured locally and only sent (via the existing upload API) when
// the user taps send. Tap 🎤 to start, tap ■ to stop, then ✓ to send
// or ✕ to discard. Recordings under 1 second are discarded.
export function VoiceRecorder({
  disabled,
  onRecorded,
  onError,
}: {
  disabled?: boolean;
  onRecorded: (blob: Blob, durationSec: number) => void;
  onError: (msg: string) => void;
}) {
  const [recording, setRecording] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const [pending, setPending] = useState<{ blob: Blob; duration: number } | null>(null);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const startedAtRef = useRef(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      streamRef.current?.getTracks().forEach((t) => t.stop());
    };
  }, []);

  function pickMimeType(): string | undefined {
    const candidates = [
      "audio/webm;codecs=opus",
      "audio/webm",
      "audio/mp4",
      "audio/ogg;codecs=opus",
    ];
    for (const c of candidates) {
      if (typeof MediaRecorder !== "undefined" && MediaRecorder.isTypeSupported(c)) return c;
    }
    return undefined;
  }

  async function start() {
    if (recording || disabled) return;
    if (typeof navigator === "undefined" || !navigator.mediaDevices?.getUserMedia) {
      onError("Voice recording is not supported in this browser");
      return;
    }
    if (typeof MediaRecorder === "undefined") {
      onError("MediaRecorder is not supported in this browser");
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      chunksRef.current = [];
      const rec = new MediaRecorder(stream, pickMimeType() ? { mimeType: pickMimeType() } : undefined);
      rec.ondataavailable = (e) => { if (e.data.size > 0) chunksRef.current.push(e.data); };
      rec.onstop = () => {
        stream.getTracks().forEach((t) => t.stop());
        const duration = (Date.now() - startedAtRef.current) / 1000;
        const blob = new Blob(chunksRef.current, { type: rec.mimeType || "audio/webm" });
        if (duration >= 1 && blob.size > 0) {
          setPending({ blob, duration });
        }
        setRecording(false);
        setElapsed(0);
      };
      recorderRef.current = rec;
      startedAtRef.current = Date.now();
      rec.start(250);
      setRecording(true);
      timerRef.current = setInterval(() => {
        setElapsed((Date.now() - startedAtRef.current) / 1000);
      }, 250);
    } catch {
      onError("Microphone permission denied or unavailable");
    }
  }

  function stop() {
    if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null; }
    const rec = recorderRef.current;
    if (rec && rec.state !== "inactive") rec.stop();
    else { setRecording(false); setElapsed(0); }
  }

  function cancel() {
    setPending(null);
  }

  function send() {
    if (!pending) return;
    onRecorded(pending.blob, pending.duration);
    setPending(null);
  }

  if (pending) {
    return (
      <span style={{ display: "inline-flex", gap: 6, alignItems: "center" }}>
        <span style={{ fontSize: 13 }}>🎤 {fmtDuration(pending.duration)}</span>
        <button type="button" title="Send voice message" onClick={send} style={{ fontSize: 16, padding: "8px 12px" }}>✓</button>
        <button type="button" title="Discard recording" onClick={cancel} style={{ fontSize: 16, padding: "8px 12px" }}>✕</button>
      </span>
    );
  }

  return (
    <button
      type="button"
      title={disabled ? "Voice messages are not end-to-end encrypted in v0.6" : recording ? "Stop recording" : "Record a voice message"}
      disabled={disabled}
      onClick={recording ? stop : start}
      style={{
        fontSize: 20, padding: "10px 14px",
        background: recording ? "#dc2626" : undefined,
        color: recording ? "#fff" : undefined,
      }}
    >
      {recording ? `■ ${fmtDuration(elapsed)}` : "🎤"}
    </button>
  );
}
