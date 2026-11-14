import { NextResponse, type NextRequest } from "next/server";

// Public website is DOWNLOAD-ONLY (user rule, 2026-11-14): the messenger
// lives only inside the mobile app, no chatting in the browser.
// When DOWNLOAD_ONLY=1 (set on the public deployment), the site root
// redirects to the download landing page. Local development is unaffected.
export function middleware(req: NextRequest) {
  if (process.env.DOWNLOAD_ONLY === "1" && req.nextUrl.pathname === "/") {
    return NextResponse.redirect(new URL("/landing", req.url));
  }
  return NextResponse.next();
}

export const config = { matcher: ["/"] };
