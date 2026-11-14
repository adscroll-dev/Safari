import "./globals.css";
import type { Metadata } from "next";

export const metadata: Metadata = {
  // Placeholder domain — replace metadataBase with the real domain before going live.
  metadataBase: new URL("https://vyro.example"),
  title: "Safari — Connect Beyond Limits | پیام‌رسان Safari",
  description:
    "Safari is a fast, secure, multi-platform messenger: chats, groups, channels, voice calls, stories and end-to-end encrypted DMs. Safari یک پیام‌رسان سریع، امن و چندسکویی است.",
  keywords: [
    "Safari",
    "messenger",
    "پیام‌رسان",
    "chat",
    "چت",
    "E2EE",
    "Telegram alternative",
    "WhatsApp alternative",
  ],
  openGraph: {
    title: "Safari — Connect Beyond Limits",
    description:
      "Fast, secure, multi-platform messenger. پیام‌رسان سریع، امن و چندسکویی.",
    type: "website",
    locale: "en_US",
    siteName: "Safari",
  },
  twitter: {
    card: "summary_large_image",
    title: "Safari — Connect Beyond Limits",
    description:
      "Fast, secure, multi-platform messenger. پیام‌رسان سریع، امن و چندسکویی.",
  },
  robots: { index: true, follow: true },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
