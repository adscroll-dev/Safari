import type { Metadata } from "next";
import DownloadButton from "./download-button";

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
        Version 0.6.0 (Beta) · Requires Android 6.0+ (API 23) · The site
        detects your phone automatically and downloads the build made
        for it — just tap the button below. Signed with the development
        key for this first test build; a proper release signature comes
        before wide distribution.
      </p>

      <DownloadButton />

      {/*
        Fallback "all devices" APK removed 2026-11-14: the 96.3 MB fat APK
        (vyro-0.6.0.apk) is over GitHub's 100 MB file limit and is not in
        the repo, so its button would 404 on the live site. The three
        per-device builds above are the downloads.
      */}

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
