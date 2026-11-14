import type { Metadata } from "next";
import LandingContent from "./landing-content";

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

export default function LandingPage() {
  return <LandingContent />;
}
