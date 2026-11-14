import type { Metadata } from "next";
import LandingContent from "./landing-content";

// Safari public website — DOWNLOAD PAGE ONLY.
// The messenger lives only inside the mobile app. There is no
// chatting/messaging in the browser on this site.
export const metadata: Metadata = {
  title: "Safari — Connect Beyond Limits",
  description:
    "Download Safari for Android — a fast, secure messenger with end-to-end encrypted chats, voice and video calls, stories and ultra-low data use. The messenger lives only inside the app.",
  openGraph: {
    title: "Safari — Connect Beyond Limits",
    description:
      "Download Safari for Android.",
    type: "website",
  },
};

export default function LandingPage() {
  return <LandingContent />;
}
