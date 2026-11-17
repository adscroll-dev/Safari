"use client";

import { useEffect, useState } from "react";

type Apk = {
  abi: string;
  href: string;
  size: string;
  bytes: string;
  sha256: string;
};

// Per-ABI split APKs, arm64 first (most modern phones). Verified
// 2026-11-17 with aapt2 (package app.vyro, version 0.7.4, label
// "Safari"; live API https://vyro-api-wf4b.onrender.com present in
// each APK's own libapp.so, no dev URL). Rebuilt 2026-11-17 18:06
// +0430 — v0.7.4: stage 2 UI redesign (aurora glass, fixed Updates
// bottom cutoff, polished Chats, no DEBUG banner).
const apks: Apk[] = [
  {
    abi: "arm64-v8a",
    href: "/downloads/safari-0.7.4-arm64-v8a.apk",
    size: "34.9 MB",
    bytes: "34,929,282 bytes",
    sha256:
      "12dc3b56cf2986f5942cf447eba0ecfd03fad61c692ae972a9ac8c678b18fc77",
  },
  {
    abi: "armeabi-v7a",
    href: "/downloads/safari-0.7.4-armeabi-v7a.apk",
    size: "28.0 MB",
    bytes: "28,023,910 bytes",
    sha256:
      "a68e5a2fa7b96171ec2e7b15c75a268bb12f6e26ee007d0073085ffbe34be58a",
  },
  {
    abi: "x86_64",
    href: "/downloads/safari-0.7.4-x86_64.apk",
    size: "37.4 MB",
    bytes: "37,399,937 bytes",
    sha256:
      "5060305acf6999d335aad9c914c2aefbe9caa2a3f02bfc370fc335d6895f17fc",
  },
];

// Dark premium styling (2026-11-16 redesign): cyan-to-blue gradient
// button with glow, dark glass card. Functionality unchanged: exactly
// ONE auto-detected APK link, SHA-256 shown for verification.
const buttonStyle: React.CSSProperties = {
  background: "linear-gradient(90deg,#22d3ee,#2563eb 55%,#7c3aed)",
  color: "#fff",
  padding: "16px 32px",
  borderRadius: 999,
  textDecoration: "none",
  fontWeight: 800,
  fontSize: 17,
  display: "inline-block",
  boxShadow:
    "0 16px 34px rgba(37,99,235,0.5), 0 0 24px rgba(34,211,238,0.35)",
};

// Detect the visitor's phone processor (ABI) from the browser, so the
// page shows exactly ONE download button for their own phone — nobody
// has to choose between APKs. Default is arm64-v8a: the build for
// most modern phones.
function detectAbiFromUserAgent(): string {
  if (typeof navigator === "undefined") return "arm64-v8a";
  const ua = navigator.userAgent || "";
  if (/aarch64|arm64/i.test(ua)) return "arm64-v8a";
  if (/x86_64|amd64|win64|wow64|x64/i.test(ua)) return "x86_64";
  if (/armv7|armeabi/i.test(ua)) return "armeabi-v7a";
  return "arm64-v8a";
}

export default function DownloadButton({
  strings,
}: {
  strings: {
    forYourPhone: string;
    detected: string;
    downloadButton: string;
    verify: string;
  };
}) {
  const [abi, setAbi] = useState("arm64-v8a");

  useEffect(() => {
    const uaData = (
      navigator as unknown as {
        userAgentData?: {
          getHighEntropyValues: (
            hints: string[]
          ) => Promise<{ architecture?: string; bitness?: string }>;
        };
      }
    ).userAgentData;
    if (uaData?.getHighEntropyValues) {
      uaData
        .getHighEntropyValues(["architecture", "bitness"])
        .then((d) => {
          if (d.architecture === "arm") {
            setAbi(d.bitness === "32" ? "armeabi-v7a" : "arm64-v8a");
          } else if (d.architecture === "x86") {
            setAbi("x86_64");
          } else {
            setAbi(detectAbiFromUserAgent());
          }
        })
        .catch(() => setAbi(detectAbiFromUserAgent()));
    } else {
      setAbi(detectAbiFromUserAgent());
    }
  }, []);

  const apk = apks.find((a) => a.abi === abi) ?? apks[0];

  return (
    <section
      style={{
        border: "1px solid rgba(34,211,238,0.35)",
        borderRadius: 20,
        padding: 20,
        background:
          "linear-gradient(160deg, rgba(255,255,255,0.09), rgba(255,255,255,0.03))",
        marginBottom: 16,
        boxShadow:
          "0 24px 48px rgba(0,0,0,0.45), inset 0 1px 0 rgba(255,255,255,0.14)",
      }}
    >
      <p
        style={{
          margin: "0 0 4px",
          fontSize: 18,
          fontWeight: 700,
          color: "#f1f5f9",
        }}
      >
        {strings.forYourPhone}
      </p>
      <p style={{ margin: "0 0 12px", color: "#94a3b8", fontSize: 14 }}>
        {strings.detected} {apk.abi} · Size: {apk.size} ({apk.bytes})
      </p>
      <p style={{ margin: "0 0 12px" }}>
        <a href={apk.href} style={buttonStyle} className="safari-btn">
          {strings.downloadButton}
        </a>
      </p>
      <p style={{ margin: "0 0 4px", fontSize: 13, color: "#cbd5e1" }}>
        {strings.verify}
      </p>
      <code
        style={{
          display: "block",
          fontSize: 12,
          wordBreak: "break-all",
          background: "#0b1220",
          border: "1px solid #1e293b",
          borderRadius: 8,
          padding: 10,
          color: "#7dd3fc",
        }}
      >
        {apk.sha256}
      </code>
    </section>
  );
}
