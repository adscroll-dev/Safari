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
    size: "33.4 MB",
    bytes: "33,425,019 bytes",
    sha256:
      "8cfb77d3fe91cc3c18a28b65a521efdc63c8ac4218f080f78e8924964c5ab67e",
  },
  {
    abi: "armeabi-v7a",
    href: "/downloads/safari-0.6.0-armeabi-v7a.apk",
    size: "26.5 MB",
    bytes: "26,536,037 bytes",
    sha256:
      "d5c085637ec64782b8704b3c4274d8bb6d50553290a4dd6e17730397d25c35b1",
  },
  {
    abi: "x86_64",
    href: "/downloads/safari-0.6.0-x86_64.apk",
    size: "36.0 MB",
    bytes: "35,961,213 bytes",
    sha256:
      "df201cc3146c0ed6ab57c4332b8b30ace6216aee97faec1f8b382b5f0b8b0f2c",
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
