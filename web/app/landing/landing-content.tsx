"use client";

import { useEffect, useRef, useState } from "react";
import DownloadButton from "./download-button";
import { languages, translations, type LangCode } from "./translations";

const LANG_KEY = "vyro-site-lang";

function isLangCode(value: string): value is LangCode {
  return languages.some((l) => l.code === value);
}

// Reveal-on-scroll: lifts a block in when it enters the viewport.
// TRANSFORM-ONLY — content is never hidden with opacity: 0, so a
// section can never get stuck invisible if the observer fails.
// Pure IntersectionObserver, no libraries. Reduced-motion users see
// the content immediately (CSS forces .safari-reveal visible).
function Reveal({
  children,
  style,
}: {
  children: React.ReactNode;
  style?: React.CSSProperties;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    // Already in (or near) the viewport at mount — reveal at once.
    // (Tall viewports, short pages, restored scroll positions: the
    // observer may never get a chance to fire for these.)
    const rect = el.getBoundingClientRect();
    if (rect.top < window.innerHeight * 1.15 && rect.bottom > -80) {
      el.classList.add("safari-in");
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            el.classList.add("safari-in");
            io.disconnect();
          }
        }
      },
      { threshold: 0.1 }
    );
    io.observe(el);
    return () => io.disconnect();
  }, []);
  return (
    <div ref={ref} className="safari-reveal" style={style}>
      {children}
    </div>
  );
}

// Shared SVG gradient defs for the feature icons (defined once).
function IconDefs() {
  return (
    <svg
      width="0"
      height="0"
      style={{ position: "absolute" }}
      aria-hidden="true"
    >
      <defs>
        <linearGradient id="saf-ic-a" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#22d3ee" />
          <stop offset="100%" stopColor="#3b82f6" />
        </linearGradient>
        <linearGradient id="saf-ic-b" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#3b82f6" />
          <stop offset="100%" stopColor="#a78bfa" />
        </linearGradient>
      </defs>
    </svg>
  );
}

// 3D-style stroke icons for the feature cards (index = feature order).
function FeatureIcon({ index }: { index: number }) {
  const stroke = index % 2 === 0 ? "url(#saf-ic-a)" : "url(#saf-ic-b)";
  const common = {
    width: 26,
    height: 26,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke,
    strokeWidth: 1.8,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
  };
  switch (index) {
    case 0: // private messaging — chat bubble
      return (
        <svg {...common}>
          <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
        </svg>
      );
    case 1: // groups
      return (
        <svg {...common}>
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
          <circle cx="9" cy="7" r="4" />
          <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
          <path d="M16 3.13a4 4 0 0 1 0 7.75" />
        </svg>
      );
    case 2: // voice call
      return (
        <svg {...common}>
          <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
        </svg>
      );
    case 3: // video call
      return (
        <svg {...common}>
          <polygon points="23 7 16 12 23 17 23 7" />
          <rect x="1" y="5" width="15" height="14" rx="2" ry="2" />
        </svg>
      );
    case 4: // stories — add a story
      return (
        <svg {...common}>
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8v8M8 12h8" />
        </svg>
      );
    case 5: // privacy & security — shield + check
      return (
        <svg {...common}>
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
          <path d="M9 12l2 2 4-4" />
        </svg>
      );
    case 6: // bots — robot
      return (
        <svg {...common}>
          <rect x="5" y="8" width="14" height="11" rx="3" />
          <path d="M12 8V4" />
          <circle cx="12" cy="3" r="1" />
          <circle cx="9.5" cy="13" r="0.9" />
          <circle cx="14.5" cy="13" r="0.9" />
          <path d="M9 16.5h6" />
        </svg>
      );
    default: // ultra-low data — feather (light)
      return (
        <svg {...common}>
          <path d="M20.24 12.24a6 6 0 0 0-8.49-8.49L5 10.5V19h8.5z" />
          <path d="M16 8L2 22" />
          <path d="M17.5 15H9" />
        </svg>
      );
  }
}

// Real screenshots of the Safari Flutter app (captured from the actual
// app running against the local test backend; see
// mobile/integration_test/screenshots_test.dart). Shown in a dark
// phone frame with a screen glare — the app's real light interface
// glowing on the dark page.
function ShotFrame({
  src,
  alt,
  caption,
}: {
  src: string;
  alt: string;
  caption: string;
}) {
  return (
    <figure style={{ margin: 0, flex: "0 0 220px" }}>
      <div
        className="safari-shot safari-shot-3d"
        style={{
          borderRadius: 28,
          border: "6px solid #0f172a",
          background: "#f8fafc",
          overflow: "hidden",
          position: "relative",
          boxShadow:
            "0 30px 60px rgba(0,0,0,0.55), 0 6px 16px rgba(0,0,0,0.35), 0 0 40px rgba(34,211,238,0.12)",
        }}
      >
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={src}
          alt={alt}
          style={{ display: "block", width: "100%", height: "auto" }}
        />
        <div
          aria-hidden="true"
          style={{
            position: "absolute",
            inset: 0,
            background:
              "linear-gradient(115deg, rgba(255,255,255,0.22) 0%, rgba(255,255,255,0.05) 28%, transparent 45%)",
            pointerEvents: "none",
          }}
        />
      </div>
      <figcaption
        style={{
          textAlign: "center",
          color: "#cbd5e1",
          fontSize: 14,
          marginTop: 12,
        }}
      >
        {caption}
      </figcaption>
    </figure>
  );
}

// Midnight backdrop: deep blue/purple light blobs + drifting glass
// orbs and chat bubbles, floating behind the content.
function Backdrop() {
  return (
    <div
      aria-hidden="true"
      style={{
        position: "absolute",
        inset: 0,
        overflow: "hidden",
        pointerEvents: "none",
      }}
    >
      <div
        className="safari-blob"
        style={{
          width: 420,
          height: 420,
          top: -120,
          right: "-10%",
          background: "rgba(37,99,235,0.35)",
        }}
      />
      <div
        className="safari-blob"
        style={{
          width: 380,
          height: 380,
          top: 500,
          left: "-12%",
          background: "rgba(139,92,246,0.28)",
        }}
      />
      <div
        className="safari-blob"
        style={{
          width: 360,
          height: 360,
          top: 1500,
          right: "-8%",
          background: "rgba(34,211,238,0.22)",
        }}
      />
      <div
        className="safari-blob"
        style={{
          width: 300,
          height: 300,
          top: 2400,
          left: "-6%",
          background: "rgba(59,130,246,0.20)",
        }}
      />
      <div
        className="safari-orb safari-orb-a"
        style={{
          width: 90,
          height: 90,
          top: 120,
          left: "4%",
          borderColor: "rgba(34,211,238,0.5)",
          background: "rgba(34,211,238,0.08)",
        }}
      />
      <div
        className="safari-orb safari-orb-b"
        style={{
          width: 54,
          height: 54,
          top: 420,
          right: "6%",
          borderColor: "rgba(167,139,250,0.5)",
          background: "rgba(139,92,246,0.08)",
        }}
      />
      <div
        className="safari-orb safari-orb-a"
        style={{
          width: 40,
          height: 40,
          top: 1100,
          left: "8%",
          borderColor: "rgba(34,211,238,0.4)",
          background: "rgba(34,211,238,0.06)",
        }}
      />
      <div
        className="safari-bubble safari-orb-b"
        style={{
          width: 110,
          height: 110,
          top: 800,
          right: "3%",
          borderColor: "rgba(34,211,238,0.4)",
          background: "rgba(34,211,238,0.07)",
        }}
      />
      <div
        className="safari-bubble safari-orb-a"
        style={{
          width: 80,
          height: 80,
          top: 1900,
          left: "2%",
          borderColor: "rgba(167,139,250,0.4)",
          background: "rgba(139,92,246,0.07)",
        }}
      />
    </div>
  );
}

// Floating 3D messaging elements around the hero logo — decorative
// app UI (language-neutral: icons, initials, times, dots). They drift
// gently; two hide on small phones to keep the hero clean.
function HeroFloaters() {
  const card: React.CSSProperties = {
    position: "absolute",
    zIndex: 2,
    background:
      "linear-gradient(160deg, rgba(255,255,255,0.12), rgba(255,255,255,0.04))",
    border: "1px solid rgba(255,255,255,0.16)",
    borderRadius: 16,
    backdropFilter: "blur(12px)",
    WebkitBackdropFilter: "blur(12px)",
    boxShadow:
      "0 18px 36px rgba(0,0,0,0.5), inset 0 1px 0 rgba(255,255,255,0.18)",
    padding: 10,
    display: "flex",
    alignItems: "center",
    gap: 8,
  };
  const bar = (w: number, o = 1): React.CSSProperties => ({
    width: w,
    height: 6,
    borderRadius: 999,
    background: "rgba(255,255,255,0.35)",
    opacity: o,
  });
  return (
    <>
      {/* chat bubble card — top left */}
      <div
        className="safari-float-slow"
        style={{ ...card, top: "2%", left: "-2%", width: 132 }}
      >
        <span
          style={{
            width: 30,
            height: 30,
            borderRadius: "50%",
            background: "linear-gradient(135deg,#22d3ee,#7c3aed)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontWeight: 800,
            fontSize: 13,
            color: "#fff",
            flexShrink: 0,
          }}
        >
          S
        </span>
        <span
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 5,
            flex: 1,
          }}
        >
          <span style={bar(64)} />
          <span style={bar(44, 0.6)} />
        </span>
        <span style={{ color: "#38bdf8", fontWeight: 800, fontSize: 12 }}>
          ✓✓
        </span>
      </div>
      {/* notification card — top right (hidden on small phones) */}
      <div
        className="safari-float-slow-alt"
        style={{ ...card, top: "0%", right: "-2%", width: 118 }}
      >
        <span style={{ position: "relative", flexShrink: 0 }}>
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#67e8f9"
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
            <path d="M13.73 21a2 2 0 0 1-3.46 0" />
          </svg>
          <span
            style={{
              position: "absolute",
              top: -4,
              right: -4,
              width: 14,
              height: 14,
              borderRadius: "50%",
              background: "#f43f5e",
              color: "#fff",
              fontSize: 9,
              fontWeight: 800,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            1
          </span>
        </span>
        <span
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 5,
            flex: 1,
          }}
        >
          <span style={bar(56)} />
          <span style={bar(38, 0.6)} />
        </span>
      </div>
      {/* voice call card — bottom right */}
      <div
        className="safari-float-slow-alt"
        style={{ ...card, bottom: "4%", right: "0%", width: 138 }}
      >
        <span
          style={{
            width: 30,
            height: 30,
            borderRadius: "50%",
            background: "#16a34a",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
          }}
        >
          <svg
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#fff"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
          </svg>
        </span>
        <span
          style={{ display: "flex", alignItems: "flex-end", gap: 3, height: 18 }}
        >
          {[0, 0.15, 0.3, 0.45].map((d) => (
            <span
              key={d}
              className="safari-eq-bar"
              style={{
                width: 3,
                height: 18,
                borderRadius: 999,
                background: "#22d3ee",
                animationDelay: `${d}s`,
              }}
            />
          ))}
        </span>
        <span style={{ color: "#e2e8f0", fontSize: 12, fontWeight: 700 }}>
          02:34
        </span>
      </div>
      {/* group card — bottom left (hidden on small phones) */}
      <div
        className="safari-float-slow"
        style={{ ...card, bottom: "0%", left: "-2%", width: 128 }}
      >
        <span style={{ display: "flex", flexShrink: 0 }}>
          {["A", "M", "L"].map((ch, i) => (
            <span
              key={ch}
              style={{
                width: 26,
                height: 26,
                borderRadius: "50%",
                marginLeft: i === 0 ? 0 : -8,
                border: "2px solid rgba(4,6,15,0.9)",
                background:
                  i === 0
                    ? "linear-gradient(135deg,#22d3ee,#2563eb)"
                    : i === 1
                      ? "linear-gradient(135deg,#a78bfa,#7c3aed)"
                      : "linear-gradient(135deg,#34d399,#059669)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 11,
                fontWeight: 800,
                color: "#fff",
              }}
            >
              {ch}
            </span>
          ))}
        </span>
        <span
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 5,
            flex: 1,
          }}
        >
          <span style={bar(52)} />
          <span style={bar(34, 0.6)} />
        </span>
      </div>
    </>
  );
}

export default function LandingContent() {
  const [lang, setLang] = useState<LangCode>("en");

  useEffect(() => {
    try {
      const saved = window.localStorage.getItem(LANG_KEY);
      if (saved && isLangCode(saved)) setLang(saved);
    } catch {
      // localStorage unavailable — stay on English, the default.
    }
  }, []);

  const meta = languages.find((l) => l.code === lang) ?? languages[0];
  const t = translations[lang];

  useEffect(() => {
    document.documentElement.lang = lang;
    document.documentElement.dir = meta.dir;
  }, [lang, meta.dir]);

  function changeLang(code: LangCode) {
    setLang(code);
    try {
      window.localStorage.setItem(LANG_KEY, code);
    } catch {
      // ignore storage failures
    }
  }

  // Headline split: first half white, second half gradient.
  const headWords = t.tagline.split(" ");
  const headMid = Math.ceil(headWords.length / 2);
  const headA = headWords.slice(0, headMid).join(" ");
  const headB = headWords.slice(headMid).join(" ");

  return (
    <main
      dir={meta.dir}
      lang={lang}
      style={{
        fontFamily: "system-ui, sans-serif",
        background:
          "radial-gradient(1200px 600px at 80% -10%, rgba(59,130,246,0.25), transparent 60%), radial-gradient(900px 500px at 10% 0%, rgba(139,92,246,0.22), transparent 55%), linear-gradient(180deg,#04060f 0%,#070b1d 45%,#04060f 100%)",
        color: "#e2e8f0",
        lineHeight: 1.7,
        minHeight: "100vh",
        position: "relative",
      }}
    >
      <IconDefs />
      <Backdrop />

      {/* Top bar */}
      <header className="safari-nav">
        <div
          style={{
            maxWidth: 1080,
            margin: "0 auto",
            padding: "14px 20px",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            gap: 12,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <img
              src="/safari-logo.png"
              alt="Safari logo"
              width={36}
              height={36}
              style={{
                borderRadius: 10,
                display: "block",
                boxShadow:
                  "0 8px 18px rgba(0,0,0,0.5), 0 0 16px rgba(34,211,238,0.25)",
              }}
            />
            <strong
              style={{ fontSize: 19, letterSpacing: 1, color: "#f8fafc" }}
            >
              Safari
            </strong>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <label
              htmlFor="safari-lang"
              className="safari-nav-label"
              style={{ fontSize: 14, color: "#cbd5e1", fontWeight: 600 }}
            >
              🌐 {t.languageLabel}
            </label>
            <select
              id="safari-lang"
              value={lang}
              onChange={(e) => changeLang(e.target.value as LangCode)}
              style={{
                fontSize: 14,
                padding: "8px 12px",
                borderRadius: 999,
                border: "1px solid rgba(255,255,255,0.2)",
                background: "#0b1220",
                color: "#e2e8f0",
                cursor: "pointer",
              }}
            >
              {languages.map((l) => (
                <option key={l.code} value={l.code}>
                  {l.label}
                </option>
              ))}
            </select>
            <a
              href="#games"
              style={{
                padding: "9px 18px",
                fontSize: 14,
                color: "#67e8f9",
                textDecoration: "none",
                fontWeight: 700,
                border: "1px solid rgba(34,211,238,0.35)",
                borderRadius: 999,
              }}
            >
              🎮 {t.navGames}
            </a>
            <a
              href="#download"
              className="safari-btn-primary safari-nav-download"
              style={{ padding: "9px 18px", fontSize: 14 }}
            >
              {t.navDownload}
            </a>
          </div>
        </div>
      </header>

      <div
        style={{
          maxWidth: 1080,
          margin: "0 auto",
          padding: "24px 20px 64px",
          position: "relative",
          zIndex: 1,
        }}
      >
        {/* HERO */}
        <section className="safari-hero-grid">
          <div className="safari-hero-text" style={{ flex: "1 1 0", minWidth: 0 }}>
            <span
              style={{
                display: "inline-block",
                fontSize: 13,
                fontWeight: 700,
                letterSpacing: 1,
                padding: "6px 14px",
                borderRadius: 999,
                color: "#67e8f9",
                background: "rgba(34,211,238,0.1)",
                border: "1px solid rgba(34,211,238,0.35)",
                boxShadow: "0 0 24px rgba(34,211,238,0.15)",
              }}
            >
              {t.heroBadge}
            </span>
            <h1
              style={{
                fontSize: "clamp(40px, 6vw, 64px)",
                lineHeight: 1.05,
                fontWeight: 800,
                letterSpacing: -0.5,
                textTransform: "uppercase",
                color: "#f8fafc",
                margin: "20px 0 16px",
                textShadow: "0 4px 32px rgba(56,189,248,0.25)",
              }}
            >
              {headA}{" "}
              <span className="safari-grad-text">{headB}</span>
            </h1>
            <p
              style={{
                fontSize: 18,
                color: "#cbd5e1",
                margin: "0 0 28px",
                maxWidth: 520,
              }}
            >
              {t.heroSub}
            </p>
            <div
              style={{
                display: "flex",
                gap: 12,
                flexWrap: "wrap",
                alignItems: "center",
              }}
            >
              <a
                href="#download"
                className="safari-btn-primary"
                style={{ padding: "16px 30px", fontSize: 16 }}
              >
                ⬇ {t.downloadCta}
              </a>
              <a
                href="#features"
                className="safari-btn-ghost"
                style={{ padding: "12px 20px", fontSize: 14 }}
              >
                {t.exploreCta}
              </a>
            </div>
            <p style={{ fontSize: 13, color: "#c1cbda", marginTop: 18 }}>
              {t.free} · {t.requires} · 0.7.12 (Beta)
            </p>
          </div>

          {/* Logo stage: her ORIGINAL logo, exactly as she sent it —
              the thick glossy ring stays. A bright glint travels
              around the ring forever, like the iPhone Clock — that IS
              the ring rotating, never still. The whole 96px block is
              scaled up with a transform so the glint geometry is
              exactly preserved. */}
          <div
            className="safari-hero-stage"
            style={{
              position: "relative",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flex: "0 0 auto",
              width: "100%",
              maxWidth: 460,
            }}
          >
            <div
              aria-hidden="true"
              className="safari-glow-pulse"
              style={{
                position: "absolute",
                width: 340,
                height: 340,
                borderRadius: "50%",
                background:
                  "radial-gradient(circle, rgba(34,211,238,0.35), rgba(59,130,246,0.12) 45%, transparent 70%)",
                filter: "blur(20px)",
              }}
            />
            <div
              aria-hidden="true"
              style={{
                position: "absolute",
                width: 280,
                height: 280,
                borderRadius: "50%",
                background:
                  "radial-gradient(circle, rgba(139,92,246,0.3), transparent 65%)",
                filter: "blur(24px)",
                transform: "translate(60px, 70px)",
              }}
            />
            <div className="safari-hero-logo-wrap">
              <div className="safari-hero-logo">
                <div
                  style={{
                    position: "relative",
                    width: 150,
                    height: 112,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                >
                  <div
                    aria-hidden="true"
                    className="safari-ring safari-ring-back"
                  >
                    <div className="safari-ring-spin" />
                  </div>
                  <div className="safari-float">
                    <img
                      src="/safari-logo.png"
                      alt="Safari logo"
                      width={96}
                      height={96}
                      style={{
                        display: "block",
                        borderRadius: 24,
                        boxShadow:
                          "0 22px 44px rgba(0,0,0,0.5), 0 4px 10px rgba(0,0,0,0.3), 0 0 40px rgba(34,211,238,0.2)",
                      }}
                    />
                  </div>
                  <div
                    aria-hidden="true"
                    className="safari-ring safari-ring-front"
                  >
                    <div className="safari-ring-spin" />
                  </div>
                </div>
              </div>
            </div>
            <div
              aria-hidden="true"
              style={{
                position: "absolute",
                bottom: 18,
                left: "50%",
                transform: "translateX(-50%)",
                width: 200,
                height: 26,
                borderRadius: "50%",
                background: "rgba(0,0,0,0.55)",
                filter: "blur(14px)",
              }}
            />
            <HeroFloaters />
          </div>
        </section>

        {/* FEATURES */}
        <section id="features" style={{ marginTop: 72 }}>
          <Reveal>
            <h2
              style={{
                margin: "0 0 8px",
                fontSize: 32,
                color: "#f8fafc",
                textAlign: "center",
              }}
            >
              {t.featuresTitle}
            </h2>
            <p
              style={{
                margin: "0 0 28px",
                color: "#c1cbda",
                fontSize: 15,
                textAlign: "center",
              }}
            >
              {t.intro}
            </p>
          </Reveal>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit,minmax(240px,1fr))",
              gap: 16,
            }}
          >
            {t.features.map((f, i) => (
              <Reveal key={f.title}>
                <section
                  className="safari-glass-dark safari-feature-card"
                  style={{ padding: 22, height: "100%" }}
                >
                  <div className="safari-icon-tile" style={{ marginBottom: 14 }}>
                    <FeatureIcon index={i} />
                  </div>
                  <h3
                    style={{ margin: "0 0 6px", fontSize: 17, color: "#f8fafc" }}
                  >
                    {f.title}
                  </h3>
                  <p style={{ margin: 0, color: "#c1cbda", fontSize: 14 }}>
                    {f.desc}
                  </p>
                </section>
              </Reveal>
            ))}
          </div>
        </section>

        {/* APP SHOWCASE — real screenshots, nothing fabricated */}
        <section id="showcase" style={{ marginTop: 72 }}>
          <Reveal>
            <h2
              style={{
                margin: "0 0 8px",
                fontSize: 32,
                color: "#f8fafc",
                textAlign: "center",
              }}
            >
              {t.showcaseTitle}
            </h2>
            <p
              style={{
                margin: "0 0 28px",
                color: "#c1cbda",
                fontSize: 15,
                textAlign: "center",
              }}
            >
              {t.showcaseSub}
            </p>
          </Reveal>
          <Reveal>
            <div
              style={{
                display: "flex",
                gap: 24,
                overflowX: "auto",
                padding: "12px 4px 20px",
                justifyContent: "center",
              }}
            >
              <ShotFrame
                src="/screenshots/chat-list.png"
                alt="Safari chat list"
                caption={t.shotChatList}
              />
              <ShotFrame
                src="/screenshots/chat.png"
                alt="Safari conversation"
                caption={t.shotChat}
              />
              <ShotFrame
                src="/screenshots/stories.png"
                alt="Safari Stories"
                caption={t.shotStories}
              />
              <ShotFrame
                src="/screenshots/calls.png"
                alt="Safari calls"
                caption={t.shotCalls}
              />
              <ShotFrame
                src="/screenshots/communities.png"
                alt="Safari communities"
                caption={t.shotCommunities}
              />
            </div>
          </Reveal>
        </section>

        {/* SECURITY — only what is genuinely implemented */}
        <section id="security" style={{ marginTop: 72 }}>
          <Reveal>
            <div
              className="safari-glass-dark"
              style={{
                padding: 32,
                position: "relative",
                overflow: "hidden",
              }}
            >
              <div
                aria-hidden="true"
                style={{
                  position: "absolute",
                  width: 320,
                  height: 320,
                  borderRadius: "50%",
                  background:
                    "radial-gradient(circle, rgba(34,211,238,0.22), transparent 65%)",
                  filter: "blur(28px)",
                  top: -100,
                  right: "-5%",
                }}
              />
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                  textAlign: "center",
                  position: "relative",
                }}
              >
                <div
                  className="safari-icon-tile"
                  style={{
                    width: 72,
                    height: 72,
                    borderRadius: 22,
                    marginBottom: 16,
                    boxShadow:
                      "0 12px 28px rgba(0,0,0,0.4), 0 0 36px rgba(34,211,238,0.3), inset 0 1px 0 rgba(255,255,255,0.2)",
                  }}
                >
                  <svg
                    width="38"
                    height="38"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="url(#saf-ic-a)"
                    strokeWidth="1.6"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                    <path d="M9 12l2 2 4-4" />
                  </svg>
                </div>
                <h2 style={{ margin: "0 0 8px", fontSize: 32, color: "#f8fafc" }}>
                  {t.securityTitle}
                </h2>
                <p
                  style={{
                    margin: "0 0 28px",
                    color: "#c1cbda",
                    fontSize: 15,
                    maxWidth: 560,
                  }}
                >
                  {t.securitySub}
                </p>
              </div>
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fit,minmax(220px,1fr))",
                  gap: 16,
                  position: "relative",
                }}
              >
                {t.securityItems.map((s) => (
                  <div
                    key={s.title}
                    style={{
                      background: "rgba(255,255,255,0.05)",
                      border: "1px solid rgba(255,255,255,0.1)",
                      borderRadius: 18,
                      padding: 20,
                    }}
                  >
                    <h3
                      style={{
                        margin: "0 0 6px",
                        fontSize: 16,
                        color: "#f8fafc",
                      }}
                    >
                      🛡 {s.title}
                    </h3>
                    <p style={{ margin: 0, color: "#c1cbda", fontSize: 14 }}>
                      {s.desc}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </Reveal>
        </section>

        {/* Slow internet */}
        <Reveal>
          <section
            className="safari-glass-dark"
            style={{
              marginTop: 24,
              padding: 28,
              borderColor: "rgba(34,211,238,0.3)",
            }}
          >
            <h2 style={{ margin: "0 0 6px", fontSize: 22, color: "#f8fafc" }}>
              📶 {t.slowTitle}
            </h2>
            <p style={{ margin: 0, color: "#cbd5e1", fontSize: 15 }}>
              {t.slowDesc}
            </p>
          </section>
        </Reveal>

        {/* DOWNLOAD */}
        <section id="download" style={{ marginTop: 72 }}>
          <Reveal>
            <div className="safari-glass-dark" style={{ padding: 32 }}>
              <h2 style={{ margin: "0 0 8px", fontSize: 28, color: "#f8fafc" }}>
                {t.downloadTitle}
              </h2>
              <p style={{ color: "#c1cbda", fontSize: 14, margin: "0 0 20px" }}>
                {t.versionLine}
              </p>

              <DownloadButton
                strings={{
                  forYourPhone: t.forYourPhone,
                  detected: t.detected,
                  downloadButton: t.downloadButton,
                  verify: t.verify,
                }}
              />

              <p
                style={{
                  margin: "16px 0 0",
                  fontSize: 14,
                  color: "#fcd34d",
                  background: "rgba(245,158,11,0.1)",
                  border: "1px solid rgba(245,158,11,0.4)",
                  borderRadius: 12,
                  padding: 12,
                }}
              >
                ⚠️ {t.warning}
              </p>
            </div>
          </Reveal>
        </section>

        {/* GAMES — Safari City, the open-world Android game. Same
            download pattern as Safari itself: a static APK file under
            /downloads, one button, SHA-256 shown for verification.
            Only real facts — no ratings, counts or reviews. */}
        <section id="games" style={{ marginTop: 72 }}>
          <Reveal>
            <div className="safari-glass-dark" style={{ padding: 32 }}>
              <h2 style={{ margin: "0 0 8px", fontSize: 28, color: "#f8fafc" }}>
                🎮 {t.gamesTitle}
              </h2>
              <p
                style={{
                  margin: "0 0 4px",
                  fontSize: 20,
                  fontWeight: 800,
                  color: "#67e8f9",
                }}
              >
                Safari City
              </p>
              <p style={{ color: "#c1cbda", fontSize: 15, margin: "0 0 16px" }}>
                {t.gamesDesc}
              </p>
              <ul
                style={{
                  color: "#c1cbda",
                  margin: "0 0 20px",
                  paddingInlineStart: 20,
                  fontSize: 14,
                }}
              >
                {t.gamesFeatures.map((f) => (
                  <li key={f}>{f}</li>
                ))}
              </ul>
              <p style={{ margin: "0 0 12px", color: "#94a3b8", fontSize: 14 }}>
                {t.gamesVersionLine}
              </p>
              <p style={{ margin: "0 0 12px" }}>
                <a
                  href="/downloads/safari-city-0.15.0-arm64-v8a.apk"
                  className="safari-btn"
                  style={{
                    background:
                      "linear-gradient(90deg,#22d3ee,#2563eb 55%,#7c3aed)",
                    color: "#fff",
                    padding: "16px 32px",
                    borderRadius: 999,
                    textDecoration: "none",
                    fontWeight: 800,
                    fontSize: 17,
                    display: "inline-block",
                    boxShadow:
                      "0 16px 34px rgba(37,99,235,0.5), 0 0 24px rgba(34,211,238,0.35)",
                  }}
                >
                  {t.gamesDownloadButton}
                </a>
              </p>
              <p style={{ margin: "0 0 4px", fontSize: 13, color: "#cbd5e1" }}>
                {t.verify}
              </p>
              <code
                style={{
                  display: "block",
                  fontSize: 12,
                  wordBreak: "break-all",
                  background: "#0b1220",
                  border: "1px solid #1e293b",
                  borderRadius: 8,
                  padding: 10,
                  color: "#7dd3fc",
                }}
              >
                0d6121879f6f43be5d16f15b4d20a38d46bb1becca042a3710d24ecd925be221
              </code>
              <p style={{ margin: "16px 0 0", fontSize: 14, color: "#94a3b8" }}>
                {t.gamesNote}
              </p>
            </div>
          </Reveal>
        </section>

        {/* App info */}
        <Reveal>
          <h2
            style={{
              margin: "56px 0 16px",
              fontSize: 22,
              color: "#f8fafc",
            }}
          >
            {t.infoTitle}
          </h2>
        </Reveal>
        <Reveal>
          <section
            className="safari-glass-dark"
            style={{ overflow: "hidden" }}
          >
            {[
              [t.versionLabel, "0.7.12 (Beta)"],
              [t.sizeLabel, "35.3 MB · 28.4 MB · 37.8 MB"],
              [t.updated, t.updatedValue],
              [t.requires, "Android 6.0+"],
              [t.offeredBy, "Safari Team"],
            ].map(([label, value], i) => (
              <div
                key={label}
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  gap: 16,
                  padding: "12px 18px",
                  borderTop:
                    i === 0 ? "none" : "1px solid rgba(255,255,255,0.08)",
                  fontSize: 14,
                }}
              >
                <span style={{ color: "#c1cbda" }}>{label}</span>
                <span
                  style={{ fontWeight: 600, textAlign: "end", color: "#f1f5f9" }}
                >
                  {value}
                </span>
              </div>
            ))}
          </section>
        </Reveal>

        {/* About + footer */}
        <Reveal>
          <div className="safari-glass-dark" style={{ marginTop: 56, padding: 28 }}>
            <h2 style={{ fontSize: 20, color: "#f8fafc", margin: "0 0 8px" }}>
              {t.aboutTitle}
            </h2>
            <p style={{ color: "#cbd5e1", margin: "0 0 12px" }}>{t.intro}</p>
            <ul style={{ color: "#c1cbda", margin: 0, paddingInlineStart: 20 }}>
              {t.features.map((f) => (
                <li key={f.title}>
                  <strong style={{ color: "#e2e8f0" }}>{f.title}:</strong>{" "}
                  {f.desc}
                </li>
              ))}
            </ul>
          </div>
        </Reveal>

        <p
          style={{
            color: "#c1cbda",
            fontSize: 14,
            marginTop: 32,
            textAlign: "center",
          }}
        >
          {t.footer}
        </p>
        <p style={{ textAlign: "center", fontSize: 14, marginTop: 8 }}>
          <a
            href="/privacy-policy"
            style={{ color: "#67e8f9", textDecoration: "none" }}
          >
            {t.privacyLink}
          </a>
        </p>
      </div>
    </main>
  );
}
