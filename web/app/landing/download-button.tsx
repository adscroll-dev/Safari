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
// 2026-11-14 with aapt (package app.vyro, version 0.6.0).
// Rebuilt 2026-11-14 05:25 UTC — now includes the v0.7 code
// (forwarding with sender name, chat folders, PIN chat lock,
// message drafts, voice playback speed).
const apks: Apk[] = [
  {
    abi: "arm64-v8a",
    href: "/downloads/vyro-0.6.0-arm64-v8a.apk",
    size: "32.6 MB",
    bytes: "32,583,679 bytes",
    sha256:
      "f04a52e5f236de2e5b244564200d25749a7da1dcaf47c87f8aae9906a4b7de7f",
  },
  {
    abi: "armeabi-v7a",
    href: "/downloads/vyro-0.6.0-armeabi-v7a.apk",
    size: "25.7 MB",
    bytes: "25,694,715 bytes",
    sha256:
      "7d92bd53fcc4c7c521d34ec75e663b4b0a99f1b454c2fbcc58a40dfb66600098",
  },
  {
    abi: "x86_64",
    href: "/downloads/vyro-0.6.0-x86_64.apk",
    size: "35.1 MB",
    bytes: "35,119,876 bytes",
    sha256:
      "380d8628f997ff6c14ab04cb9fde6ed5b64c5b424341a5522cf0046e8db3d096",
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

export default function DownloadButton() {
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
        VYRO 0.6.0 — for your phone
      </p>
      <p style={{ margin: "0 0 12px", color: "#4b5563", fontSize: 14 }}>
        Detected for your phone: {apk.abi} · Size: {apk.size} ({apk.bytes})
      </p>
      <p style={{ margin: "0 0 12px" }}>
        <a href={apk.href} style={buttonStyle}>
          Download VYRO for Android
        </a>
      </p>
      <p style={{ margin: "0 0 4px", fontSize: 13, color: "#374151" }}>
        Verify the download (SHA-256):
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
