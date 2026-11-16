import type { Metadata } from "next";
import JoinContent, { type InvitePreview } from "./join-content";

// Invite links point here: https://vyro-web-8p4l.onrender.com/join/<token>.
// The page is a server component so the preview is fetched
// server-to-server from the API (no CORS issue), then rendered.
// No messaging happens in the browser — the site stays download-only;
// the page shows the invite and sends the visitor to the download page.
export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Join on Safari",
  description: "You were invited to join a group or channel on Safari. Download the app to join.",
};

const API_BASE = process.env.SAFARI_API_URL ?? "https://vyro-api-wf4b.onrender.com";

async function fetchPreview(token: string): Promise<InvitePreview | null> {
  try {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 8000);
    const res = await fetch(`${API_BASE}/api/invites/${encodeURIComponent(token)}`, {
      signal: ctrl.signal,
      cache: "no-store",
    });
    clearTimeout(timer);
    if (!res.ok) return null;
    const data = (await res.json()) as InvitePreview;
    if (!data || typeof data.title !== "string") return null;
    return data;
  } catch {
    return null;
  }
}

export default async function JoinPage({ params }: { params: { token: string } }) {
  const preview = await fetchPreview(params.token);
  return <JoinContent preview={preview} />;
}
