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
// 2026-11-16 with aapt2 (package app.vyro, version 0.7.0, label
// "Safari"; live API https://vyro-api-wf4b.onrender.com present in
// libapp.so, no localhost URL). Rebuilt 2026-11-16 17:29 +0430 —
// v0.7.0: 3D animated registration + language picker, per-chat
// wallpapers (12 colours + 6 gradients), offline mode (cached chats,
// outbox queue).
const apks: Apk[] = [
  {
    abi: "arm64-v8a",
    href: "/downloads/safari-0.7.0-arm64-v8a.apk",
    size: "34.0 MB",
    bytes: "33,998,350 bytes",
    sha256:
      "5be99d03cb95c74c32b73c81df0e856d9ec2d4ace2c4c3967aacc76bf8323901",
  },
  {
    abi: "armeabi-v7a",
    href: "/downloads/safari-0.7.0-armeabi-v7a.apk",
    size: "27.1 MB",
    bytes: "27,109,363 bytes",
    sha256:
      "fa8099610cc878ed5d13d9e7b65288c739225cc0799ea51a597bf76b8e28d7de",
  },
  {
    abi: "x86_64",
    href: "/downloads/safari-0.7.0-x86_64.apk",
    size: "36.5 MB",
    bytes: "36,468,992 bytes",
    sha256:
      "872b7c4c0e9536510d0dae3a69614af5229ea48028c50cb25fe9a993fa94cebc",
  },
];

const buttonStyle: React.CSSProperties = {
  background: "linear-gradient(90deg,#25D366,#00A884)",
  color: "#fff",
  padding: "16px 32px",
  borderRadius: 999,
  textDecoration: "none",
  fontWeight: 800,
  fontSize: 17,
  display: "inline-block",
  boxShadow:
    "0 14px 28px rgba(0,168,132,0.45), 0 3px 8px rgba(0,0,0,0.2)",
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
        border: "2px solid #00A884",
        borderRadius: 20,
        padding: 20,
        background: "linear-gradient(135deg,#E6F7EF,#D9F2E4)",
        marginBottom: 16,
        boxShadow:
          "0 12px 28px rgba(0,0,0,0.12), 0 2px 6px rgba(0,0,0,0.08)",
      }}
    >
      <p style={{ margin: "0 0 4px", fontSize: 18, fontWeight: 700 }}>
        {strings.forYourPhone}
      </p>
      <p style={{ margin: "0 0 12px", color: "#4b5563", fontSize: 14 }}>
        {strings.detected} {apk.abi} · Size: {apk.size} ({apk.bytes})
      </p>
      <p style={{ margin: "0 0 12px" }}>
        <a href={apk.href} style={buttonStyle} className="safari-btn">
          {strings.downloadButton}
        </a>
      </p>
      <p style={{ margin: "0 0 4px", fontSize: 13, color: "#374151" }}>
        {strings.verify}
      </p>
      <code
        style={{
          display: "block",
          fontSize: 12,
          wordBreak: "break-all",
          background: "#ffffff",
          border: "1px solid #dbeafe",
          borderRadius: 8,
          padding: 10,
        }}
      >
        {apk.sha256}
      </code>
    </section>
  );
}
