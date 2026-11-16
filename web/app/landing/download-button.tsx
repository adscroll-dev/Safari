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
// 2026-11-16 with aapt2 (package app.vyro, version 0.6.0, label
// "Safari"; "Delete my account", "Chat wallpaper", "Read receipts",
// "Blocked contacts" strings present in all three). Rebuilt
// 2026-11-16 06:46 +0430 — Stage 11 Settings UI: privacy page
// (read receipts, avatar/about/groups), blocked contacts, delete
// account, chat wallpaper; live API
// https://vyro-api-wf4b.onrender.com.
const apks: Apk[] = [
  {
    abi: "arm64-v8a",
    href: "/downloads/safari-0.6.0-arm64-v8a.apk",
    size: "33.9 MB",
    bytes: "33,932,234 bytes",
    sha256:
      "595e2e16708c644d284b44f6ec88ea31dadd1f1ba1454e125798ec133970f76c",
  },
  {
    abi: "armeabi-v7a",
    href: "/downloads/safari-0.6.0-armeabi-v7a.apk",
    size: "27.1 MB",
    bytes: "27,076,028 bytes",
    sha256:
      "6e12d47891c1f195d411ddbb2dd648d560927b2606a1020e2941525f84d4576b",
  },
  {
    abi: "x86_64",
    href: "/downloads/safari-0.6.0-x86_64.apk",
    size: "36.5 MB",
    bytes: "36,468,427 bytes",
    sha256:
      "ef22cbf86e6316c8c7319dadacdb0286dc8ac9c2e4c0a181a6460dc3c4395a21",
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
