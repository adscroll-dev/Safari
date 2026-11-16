"use client";

import { useEffect, useState } from "react";
import DownloadButton from "./download-button";
import { languages, translations, type LangCode } from "./translations";

const LANG_KEY = "vyro-site-lang";

function isLangCode(value: string): value is LangCode {
  return languages.some((l) => l.code === value);
}

const GRADIENT = "linear-gradient(135deg,#2563eb,#7c3aed)";

// 3D landing page (matches the app's animated registration page):
// bright teal-to-emerald gradient, big drifting glass orbs and soft
// light blobs, frosted glass cards, the real app icon floating with
// a deep shadow, phone screenshots tilted in 3D. The download flow
// is unchanged: one auto-detected button, 11 languages, no fake
// ratings or download counts.

// Real screenshots of the Safari Flutter app (captured from the actual
// app running against the local test backend; see
// mobile/integration_test/screenshots_test.dart). Shown in the same
// phone-frame styling the CSS mockups used to have.
function ShotFrame({ src, alt }: { src: string; alt: string }) {
  return (
    <div
      className="safari-shot safari-shot-3d"
      style={{
        flex: "0 0 220px",
        borderRadius: 28,
        border: "6px solid #111827",
        background: "#f8fafc",
        overflow: "hidden",
        boxShadow:
          "0 30px 60px rgba(0,0,0,0.45), 0 6px 16px rgba(0,0,0,0.25)",
      }}
    >
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={src}
        alt={alt}
        style={{ display: "block", width: "100%", height: "auto" }}
      />
    </div>
  );
}

// Big bright glass orbs, soft light blobs and chat bubbles drifting
// behind the content — visible on a phone screen, not hidden at the
// edges.
function Orbs() {
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
          width: 260,
          height: 260,
          top: -40,
          right: -60,
          background: "rgba(220,255,235,0.55)",
        }}
      />
      <div
        className="safari-blob"
        style={{
          width: 220,
          height: 220,
          top: 620,
          left: -70,
          background: "rgba(255,255,255,0.35)",
        }}
      />
      <div
        className="safari-blob"
        style={{
          width: 240,
          height: 240,
          top: 1350,
          right: -80,
          background: "rgba(190,255,210,0.45)",
        }}
      />
      <div
        className="safari-orb safari-orb-a"
        style={{ width: 110, height: 110, top: 90, left: "5%" }}
      />
      <div
        className="safari-orb safari-orb-b"
        style={{ width: 64, height: 64, top: 300, right: "5%" }}
      />
      <div
        className="safari-orb safari-orb-a"
        style={{ width: 42, height: 42, top: 560, left: "12%" }}
      />
      <div
        className="safari-orb safari-orb-b"
        style={{ width: 88, height: 88, top: 1020, right: "7%" }}
      />
      <div
        className="safari-orb safari-orb-a"
        style={{ width: 52, height: 52, top: 1560, left: "6%" }}
      />
      <div
        className="safari-orb safari-orb-b"
        style={{ width: 36, height: 36, top: 1950, right: "10%" }}
      />
      <div
        className="safari-bubble safari-orb-a"
        style={{ width: 130, height: 130, top: 420, right: "2%" }}
      />
      <div
        className="safari-bubble safari-orb-b"
        style={{ width: 100, height: 100, top: 1250, left: "1%" }}
      />
    </div>
  );
}

// Frosted glass card: translucent white, blurred backdrop, bright top
// edge, deep layered shadow — real depth on the gradient.
const glassCard: React.CSSProperties = {
  background: "rgba(255,255,255,0.92)",
  backdropFilter: "blur(14px)",
  WebkitBackdropFilter: "blur(14px)",
  border: "1px solid rgba(255,255,255,0.95)",
  borderRadius: 28,
  boxShadow:
    "0 30px 60px rgba(0,0,0,0.30), 0 4px 12px rgba(0,0,0,0.18), inset 0 1px 0 rgba(255,255,255,0.9)",
};

const headingWhite: React.CSSProperties = {
  color: "#ffffff",
  textShadow: "0 2px 14px rgba(0,0,0,0.4)",
};

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

  return (
    <main
      dir={meta.dir}
      lang={lang}
      style={{
        fontFamily: "system-ui, sans-serif",
        background:
          "linear-gradient(180deg,#008069 0%,#00A884 42%,#25D366 100%)",
        color: "#111827",
        lineHeight: 1.7,
        minHeight: "100vh",
        position: "relative",
      }}
    >
      <Orbs />
      {/* Top bar */}
      <header
        style={{
          maxWidth: 960,
          margin: "0 auto",
          padding: "20px 20px 0",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 12,
          position: "relative",
          zIndex: 1,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <img
            src="/safari-logo.png"
            alt="Safari logo"
            width={40}
            height={40}
            style={{
              borderRadius: 11,
              display: "block",
              boxShadow:
                "0 10px 22px rgba(0,0,0,0.35), 0 2px 6px rgba(0,0,0,0.25)",
            }}
          />
          <strong
            style={{
              fontSize: 19,
              letterSpacing: 1,
              color: "#ffffff",
              textShadow: "0 2px 10px rgba(0,0,0,0.4)",
            }}
          >
            Safari
          </strong>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <label
            htmlFor="safari-lang"
            style={{
              fontSize: 14,
              color: "#ffffff",
              fontWeight: 600,
              textShadow: "0 1px 8px rgba(0,0,0,0.4)",
            }}
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
              border: "1px solid rgba(255,255,255,0.7)",
              background: "#ffffff",
              color: "#111827",
              cursor: "pointer",
              boxShadow: "0 8px 18px rgba(0,0,0,0.3)",
            }}
          >
            {languages.map((l) => (
              <option key={l.code} value={l.code}>
                {l.label}
              </option>
            ))}
          </select>
        </div>
      </header>

      <div
        style={{
          maxWidth: 960,
          margin: "0 auto",
          padding: "24px 20px 64px",
          position: "relative",
          zIndex: 1,
        }}
      >
        {/* App header card */}
        <section
          style={{
            ...glassCard,
            padding: 28,
          }}
        >
          <div
            style={{
              display: "flex",
              gap: 20,
              flexWrap: "wrap",
              alignItems: "flex-start",
            }}
          >
            {/* Planet-only logo (ring removed from the image) with the
                code-drawn ring sitting exactly ON the planet's own ring
                ellipse — the ring in the icon rotates forever, like the
                iPhone Clock, never still. */}
            <div
              style={{
                flexShrink: 0,
                position: "relative",
                width: 150,
                height: 112,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <div aria-hidden="true" className="safari-ring safari-ring-back">
                <div className="safari-ring-spin" />
              </div>
              <div className="safari-float">
                <img
                  src="/safari-logo-planet-no-ring.png"
                  alt="Safari logo"
                  width={96}
                  height={96}
                  style={{
                    display: "block",
                    borderRadius: 24,
                    boxShadow:
                      "0 22px 44px rgba(0,0,0,0.4), 0 4px 10px rgba(0,0,0,0.25)",
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
            <div style={{ flex: "1 1 280px", minWidth: 0 }}>
              <h1 style={{ fontSize: 34, margin: "0 0 2px" }}>Safari</h1>
              <p style={{ margin: "0 0 10px", color: "#4b5563", fontSize: 15 }}>
                {t.offeredBy}
              </p>
              <div
                style={{
                  display: "flex",
                  gap: 8,
                  flexWrap: "wrap",
                  marginBottom: 12,
                }}
              >
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 600,
                    padding: "4px 12px",
                    borderRadius: 999,
                    background: "#eef2ff",
                    color: "#3730a3",
                  }}
                >
                  {t.category}
                </span>
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 600,
                    padding: "4px 12px",
                    borderRadius: 999,
                    background: "#ecfdf5",
                    color: "#047857",
                  }}
                >
                  {t.free}
                </span>
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 600,
                    padding: "4px 12px",
                    borderRadius: 999,
                    background: "#f5f3ff",
                    color: "#7c3aed",
                  }}
                >
                  BETA
                </span>
              </div>
              <p style={{ fontSize: 19, margin: "0 0 6px", fontWeight: 600 }}>
                {t.tagline}
              </p>
              <p style={{ margin: 0, color: "#4b5563", fontSize: 15 }}>
                {t.intro}
              </p>
            </div>
          </div>
        </section>

        {/* Screenshots */}
        <h2 style={{ margin: "40px 0 16px", fontSize: 22, ...headingWhite }}>
          {t.screenshotsTitle}
        </h2>
        <div
          style={{
            display: "flex",
            gap: 20,
            overflowX: "auto",
            paddingBottom: 8,
          }}
        >
          <ShotFrame
            src="/screenshots/chat-list.png"
            alt="Safari chat list — Your World"
          />
          <ShotFrame
            src="/screenshots/chat.png"
            alt="Safari conversation"
          />
          <ShotFrame
            src="/screenshots/stories.png"
            alt="Safari Stories"
          />
        </div>

        {/* Download */}
        <section
          style={{
            ...glassCard,
            marginTop: 40,
            padding: 28,
          }}
        >
          <h2 style={{ margin: "0 0 8px", fontSize: 22 }}>
            {t.downloadTitle}
          </h2>
          <p style={{ color: "#4b5563", fontSize: 14, margin: "0 0 20px" }}>
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
              background: "#fffbeb",
              border: "1px solid #f59e0b",
              borderRadius: 12,
              padding: 12,
            }}
          >
            ⚠️ {t.warning}
          </p>
        </section>

        {/* Slow internet */}
        <section
          style={{
            marginTop: 24,
            borderRadius: 28,
            padding: 28,
            background:
              "linear-gradient(135deg,rgba(255,255,255,0.94),rgba(215,245,228,0.94))",
            backdropFilter: "blur(14px)",
            WebkitBackdropFilter: "blur(14px)",
            border: "1px solid rgba(255,255,255,0.95)",
            boxShadow:
              "0 30px 60px rgba(0,0,0,0.30), 0 4px 12px rgba(0,0,0,0.18), inset 0 1px 0 rgba(255,255,255,0.9)",
          }}
        >
          <h2 style={{ margin: "0 0 6px", fontSize: 22 }}>📶 {t.slowTitle}</h2>
          <p style={{ margin: 0, color: "#374151", fontSize: 15 }}>
            {t.slowDesc}
          </p>
        </section>

        {/* Features */}
        <h2 style={{ margin: "40px 0 16px", fontSize: 22, ...headingWhite }}>
          {t.featuresTitle}
        </h2>
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit,minmax(240px,1fr))",
            gap: 16,
          }}
        >
          {t.features.map((f) => (
            <section
              key={f.title}
              className="safari-feature"
              style={{
                background: "rgba(255,255,255,0.92)",
                backdropFilter: "blur(14px)",
                WebkitBackdropFilter: "blur(14px)",
                border: "1px solid rgba(255,255,255,0.95)",
                borderRadius: 20,
                padding: 18,
                boxShadow:
                  "0 18px 38px rgba(0,0,0,0.28), 0 3px 8px rgba(0,0,0,0.15), inset 0 1px 0 rgba(255,255,255,0.9)",
              }}
            >
              <h3 style={{ margin: "0 0 4px", fontSize: 17 }}>{f.title}</h3>
              <p style={{ margin: 0, color: "#4b5563", fontSize: 14 }}>
                {f.desc}
              </p>
            </section>
          ))}
        </div>

        {/* App info */}
        <h2 style={{ margin: "40px 0 16px", fontSize: 22, ...headingWhite }}>
          {t.infoTitle}
        </h2>
        <section
          style={{
            background: "rgba(255,255,255,0.92)",
            backdropFilter: "blur(14px)",
            WebkitBackdropFilter: "blur(14px)",
            border: "1px solid rgba(255,255,255,0.95)",
            borderRadius: 20,
            overflow: "hidden",
            boxShadow:
              "0 18px 38px rgba(0,0,0,0.28), 0 3px 8px rgba(0,0,0,0.15), inset 0 1px 0 rgba(255,255,255,0.9)",
          }}
        >
          {[
            [t.versionLabel, "0.7.1 (Beta)"],
            [t.sizeLabel, "34.1 MB · 27.2 MB · 36.5 MB"],
            [t.updated, "Nov 16, 2026"],
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
                borderTop: i === 0 ? "none" : "1px solid #f3f4f6",
                fontSize: 14,
              }}
            >
              <span style={{ color: "#6b7280" }}>{label}</span>
              <span style={{ fontWeight: 600, textAlign: "end" }}>{value}</span>
            </div>
          ))}
        </section>

        <p
          style={{
            color: "rgba(255,255,255,0.95)",
            fontSize: 14,
            marginTop: 32,
            textShadow: "0 1px 10px rgba(0,0,0,0.4)",
          }}
        >
          {t.footer}
        </p>

        <div
          style={{
            marginTop: 40,
            ...glassCard,
            padding: 28,
          }}
        >
          <h2 style={{ fontSize: 20 }}>{t.aboutTitle}</h2>
          <p>{t.intro}</p>
          <ul>
            {t.features.map((f) => (
              <li key={f.title}>
                <strong>{f.title}:</strong> {f.desc}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </main>
  );
}
