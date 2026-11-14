import type { Metadata } from "next";

// VYRO public website — DOWNLOAD PAGE ONLY.
// The messenger lives only inside the mobile app. There is no
// chatting/messaging in the browser on this site.
export const metadata: Metadata = {
  title: "VYRO — Connect Beyond Limits",
  description:
    "Download VYRO for Android — a fast, secure messenger with end-to-end encrypted chats, voice and video calls, stories and ultra-low data use. The messenger lives only inside the app.",
  openGraph: {
    title: "VYRO — Connect Beyond Limits",
    description:
      "Download VYRO for Android.",
    type: "website",
  },
};

const featuresEn = [
  { title: "Private & group chat", desc: "Direct messages, groups and channels with admin roles." },
  { title: "Real-time chat", desc: "Instant messages over WebSocket, read ticks and typing indicator." },
  { title: "Voice & video calls", desc: "Voice calls over WebRTC; video calls start at low quality (Data Saver)." },
  { title: "Photos, files & voice", desc: "Uploads up to 50 MB, image and file messages, voice messages." },
  { title: "24-hour stories", desc: "Stories and status, like the messengers you know." },
  { title: "End-to-end encryption", desc: "One-to-one DMs with real E2EE (X3DH + Double Ratchet); group E2EE with Sender Keys." },
  { title: "Bots", desc: "Bots with a token and webhook for automation." },
  { title: "English & Persian", desc: "English by default; RTL Persian and other languages available in Settings." },
  { title: "Ultra-low data use", desc: "Data Saver by default: voice calls at Opus 12 kbps (~0.09 MB/min), compressed photos, tap-to-download media." },
  { title: "Local AI assistant", desc: "On-device assistant, dictionary translation and smart replies — no cloud AI, no data leaves your device." },
];

const featuresFa = [
  { title: "چت خصوصی و گروه", desc: "گفت‌وگوی خصوصی، گروه و کانال با مدیریت مدیران." },
  { title: "چت ریل‌تایم", desc: "پیام آنی با WebSocket، تیک خوانده‌شدن و «در حال تایپ…»." },
  { title: "تماس صوتی و تصویری", desc: "تماس صوتی با WebRTC؛ تماس تصویری با کیفیت پایین شروع می‌شود (ذخیره داده)." },
  { title: "عکس، فایل و پیام صوتی", desc: "آپلود تا ۵۰ مگابایت و پیام تصویری و فایل." },
  { title: "استوری ۲۴ ساعته", desc: "استوری و استاتوس مثل پیام‌رسان‌های آشنا." },
  { title: "رمزنگاری سرتاسری", desc: "پیام‌های خصوصی دونفره با E2EE واقعی (X3DH + Double Ratchet)؛ رمزنگاری گروهی با کلید فرستنده." },
  { title: "ربات‌ها", desc: "ربات با توکن و webhook برای خودکارسازی." },
  { title: "انگلیسی و فارسی", desc: "انگلیسی پیش‌فرض است؛ فارسی راست‌چین و زبان‌های دیگر در تنظیمات در دسترس‌اند." },
  { title: "مصرف بسیار کم اینترنت", desc: "ذخیره داده به‌صورت پیش‌فرض: تماس صوتی با Opus ۱۲ کیلوبیت بر ثانیه (حدود ۰٫۰۹ مگابایت در دقیقه)، عکس‌های فشرده، دانلود با لمس." },
  { title: "دستیار هوش مصنوعی محلی", desc: "دستیار روی دستگاه، ترجمه با فرهنگ لغت و پاسخ‌های هوشمند — بدون هوش مصنوعی ابری، بدون خروج داده از دستگاه شما." },
];

type Apk = {
  abi: string;
  label: string;
  href: string;
  size: string;
  bytes: string;
  sha256: string;
  recommended?: boolean;
};

// Per-ABI split APKs, arm64 first (most modern phones). Verified
// 2026-11-14 with aapt (package app.vyro, version 0.6.0).
// Rebuilt 2026-11-14 05:25 UTC — now includes the v0.7 code
// (forwarding with sender name, chat folders, PIN chat lock,
// message drafts, voice playback speed).
const apks: Apk[] = [
  {
    abi: "arm64-v8a",
    label: "Most modern phones (recommended)",
    href: "/downloads/vyro-0.6.0-arm64-v8a.apk",
    size: "32.6 MB",
    bytes: "32,583,679 bytes",
    sha256: "f04a52e5f236de2e5b244564200d25749a7da1dcaf47c87f8aae9906a4b7de7f",
    recommended: true,
  },
  {
    abi: "armeabi-v7a",
    label: "Older 32-bit phones",
    href: "/downloads/vyro-0.6.0-armeabi-v7a.apk",
    size: "25.7 MB",
    bytes: "25,694,715 bytes",
    sha256: "7d92bd53fcc4c7c521d34ec75e663b4b0a99f1b454c2fbcc58a40dfb66600098",
  },
  {
    abi: "x86_64",
    label: "Intel-based devices / emulators",
    href: "/downloads/vyro-0.6.0-x86_64.apk",
    size: "35.1 MB",
    bytes: "35,119,876 bytes",
    sha256: "380d8628f997ff6c14ab04cb9fde6ed5b64c5b424341a5522cf0046e8db3d096",
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

export default function LandingPage() {
  return (
    <main
      dir="ltr"
      lang="en"
      style={{
        fontFamily: "system-ui, sans-serif",
        maxWidth: 880,
        margin: "0 auto",
        padding: "48px 20px",
        lineHeight: 1.8,
        color: "#111827",
        background: "#ffffff",
      }}
    >
      <p style={{ fontSize: 13, color: "#7c3aed", fontWeight: 700 }}>
        BETA — Beta version
      </p>
      <h1
        style={{
          fontSize: 44,
          margin: "8px 0",
          background: "linear-gradient(90deg,#2563eb,#7c3aed)",
          WebkitBackgroundClip: "text",
          backgroundClip: "text",
          color: "transparent",
        }}
      >
        VYRO
      </h1>
      <p style={{ fontSize: 20, margin: "0 0 8px" }}>
        Connect Beyond Limits
      </p>
      <p>
        VYRO is a fast, secure messenger for Android: private and group
        chat, channels, voice calls, stories, bots, and end-to-end
        encryption for one-to-one messages. This is a beta — some features
        are still on the way. The messenger lives only inside the app —
        download it below to start chatting.
      </p>

      <h2 style={{ marginTop: 48 }}>Download VYRO for Android</h2>
      <p style={{ color: "#4b5563", fontSize: 14 }}>
        Version 0.6.0 (Beta) · Requires Android 6.0+ (API 23) · Each phone
        downloads only the small build made for it — pick the one that
        matches your phone, or take the recommended one for most modern
        phones. Signed with the development key for this first test build;
        a proper release signature comes before wide distribution.
      </p>

      {apks.map((apk) => (
        <section
          key={apk.abi}
          style={{
            border: apk.recommended
              ? "2px solid #2563eb"
              : "1px solid #e5e7eb",
            borderRadius: 16,
            padding: 20,
            background: apk.recommended ? "#eff6ff" : "#ffffff",
            marginBottom: 16,
          }}
        >
          <p style={{ margin: "0 0 4px", fontSize: 18, fontWeight: 700 }}>
            VYRO 0.6.0 — {apk.abi}
            {apk.recommended ? " ★ Recommended" : ""}
          </p>
          <p style={{ margin: "0 0 12px", color: "#4b5563", fontSize: 14 }}>
            {apk.label} · Size: {apk.size} ({apk.bytes})
          </p>
          <p style={{ margin: "0 0 12px" }}>
            <a href={apk.href} style={buttonStyle}>
              Download VYRO 0.6.0 ({apk.abi})
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
      ))}

      <section
        style={{
          border: "1px solid #e5e7eb",
          borderRadius: 16,
          padding: 20,
          marginBottom: 16,
        }}
      >
        <p style={{ margin: "0 0 4px", fontSize: 17, fontWeight: 700 }}>
          VYRO 0.6.0 — all devices (fallback)
        </p>
        <p style={{ margin: "0 0 12px", color: "#4b5563", fontSize: 14 }}>
          One big file that works on every Android phone · Size: 96.3 MB
          (101,011,626 bytes). Use this only if none of the small builds
          above installs on your phone.
        </p>
        <p style={{ margin: "0 0 12px" }}>
          <a href="/downloads/vyro-0.6.0.apk" style={buttonStyle}>
            Download VYRO 0.6.0 (all devices)
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
          8a717aa83b26cf916b4f109aa16a2a3b27a84bd93a0fdf41e2b8ca6bc97fc592
        </code>
      </section>

      <p
        style={{
          margin: "16px 0 0",
          fontSize: 14,
          background: "#fffbeb",
          border: "1px solid #f59e0b",
          borderRadius: 12,
          padding: 12,
        }}
      >
        ⚠️ Download VYRO only from this official website. APKs from
        other sites are not safe and should not be installed.
      </p>

      <h2 style={{ marginTop: 48 }}>Features</h2>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit,minmax(240px,1fr))",
          gap: 16,
        }}
      >
        {featuresEn.map((f) => (
          <section
            key={f.title}
            style={{
              border: "1px solid #e5e7eb",
              borderRadius: 16,
              padding: 16,
            }}
          >
            <h3 style={{ margin: "0 0 4px", fontSize: 17 }}>{f.title}</h3>
            <p style={{ margin: 0, color: "#4b5563", fontSize: 14 }}>{f.desc}</p>
          </section>
        ))}
      </div>

      <p style={{ color: "#6b7280", fontSize: 14, marginTop: 32 }}>
        Free and open-source only, until VYRO reaches 10,000 registered
        users. The Android beta APK is available above; iOS is coming
        later.
      </p>

      <div dir="rtl" lang="fa" style={{ marginTop: 48, borderTop: "1px solid #e5e7eb", paddingTop: 32 }}>
        <h2>درباره وایرو</h2>
        <p>
          VYRO یک پیام‌رسان سریع، امن و چندسکویی است: چت خصوصی و گروه، کانال،
          تماس صوتی، استوری، ربات و رمزنگاری سرتاسری برای پیام‌های خصوصی.
          این نسخه هنوز آزمایشی (بتا) است و بعضی قابلیت‌ها هنوز در راه‌اند.
          پیام‌رسان فقط داخل اپ موبایل است — برای شروع، اپ را از بالای همین
          صفحه دانلود کنید.
        </p>
        <ul>
          {featuresFa.map((f) => (
            <li key={f.title}>
              <strong>{f.title}:</strong> {f.desc}
            </li>
          ))}
        </ul>
      </div>
    </main>
  );
}
