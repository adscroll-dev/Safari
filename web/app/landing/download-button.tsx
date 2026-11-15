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
// 2026-11-15 with aapt2 (package app.vyro, version 0.6.0, label
// "Safari"). Rebuilt 2026-11-15 10:26 +0430 — Stage 1 chat gaps:
// mute chat, pin message, contact share, delete for me/everyone;
// live API https://vyro-api-wf4b.onrender.com.
const apks: Apk[] = [
  {
    abi: "arm64-v8a",
    href: "/downloads/safari-0.6.0-arm64-v8a.apk",
    size: "33.5 MB",
    bytes: "33,622,505 bytes",
    sha256:
      "3917a311e9f6ddb333cbaaf534617b444e6419120fbcfa74591da9b2dfe7be2d",
  },
  {
    abi: "armeabi-v7a",
    href: "/downloads/safari-0.6.0-armeabi-v7a.apk",
    size: "26.6 MB",
    bytes: "26,717,141 bytes",
    sha256:
      "961881a8abbce7cfe9b36376d50b5895edf1411f325ca3c6de6d41fc5adbc6df",
  },
  {
    abi: "x86_64",
    href: "/downloads/safari-0.6.0-x86_64.apk",
    size: "36.0 MB",
    bytes: "36,158,695 bytes",
    sha256:
      "64e98010874dcc20cf9ed78c867a874a2f1013412402314cdbf3a8b9632cd51b",
  },
];

const buttonStyle: React.CSSProperties = {
  background: "linear-gradient(90deg,#2563eb,#7c3aed)",
  color: "#fff",
  padding: "12px 24px",
  borderRadius: 999,
  textDecoration: "none",
  fontWeight: 700,
  display: "inline-block",
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
        border: "2px solid #2563eb",
        borderRadius: 16,
        padding: 20,
        background: "#eff6ff",
        marginBottom: 16,
      }}
    >
      <p style={{ margin: "0 0 4px", fontSize: 18, fontWeight: 700 }}>
        {strings.forYourPhone}
      </p>
      <p style={{ margin: "0 0 12px", color: "#4b5563", fontSize: 14 }}>
        {strings.detected} {apk.abi} · Size: {apk.size} ({apk.bytes})
      </p>
      <p style={{ margin: "0 0 12px" }}>
        <a href={apk.href} style={buttonStyle}>
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
