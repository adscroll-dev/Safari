"use client";

import { useEffect, useState } from "react";
import DownloadButton from "./download-button";
import { languages, translations, type LangCode } from "./translations";

const LANG_KEY = "vyro-site-lang";

function isLangCode(value: string): value is LangCode {
  return languages.some((l) => l.code === value);
}

const GRADIENT = "linear-gradient(135deg,#2563eb,#7c3aed)";

// Real screenshots of the Safari Flutter app (captured from the actual
// app running against the local test backend; see
// mobile/integration_test/screenshots_test.dart). Shown in the same
// phone-frame styling the CSS mockups used to have.
function ShotFrame({ src, alt }: { src: string; alt: string }) {
  return (
    <div
      style={{
        flex: "0 0 220px",
        borderRadius: 28,
        border: "6px solid #111827",
        background: "#f8fafc",
        overflow: "hidden",
        boxShadow: "0 10px 30px rgba(37,99,235,0.15)",
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
        background: "#f8fafc",
        color: "#111827",
        lineHeight: 1.7,
        minHeight: "100vh",
      }}
    >
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
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <img
            src="/safari-logo.png"
            alt="Safari logo"
            width={36}
            height={36}
            style={{ borderRadius: 10 }}
          />
          <strong style={{ fontSize: 18, letterSpacing: 1 }}>Safari</strong>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <label
            htmlFor="safari-lang"
            style={{ fontSize: 14, color: "#4b5563", fontWeight: 600 }}
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
              border: "1px solid #d1d5db",
              background: "#ffffff",
              color: "#111827",
              cursor: "pointer",
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

      <div style={{ maxWidth: 960, margin: "0 auto", padding: "24px 20px 64px" }}>
        {/* App header card */}
        <section
          style={{
            background: "#ffffff",
            border: "1px solid #e5e7eb",
            borderRadius: 24,
            padding: 28,
            boxShadow: "0 4px 24px rgba(17,24,39,0.05)",
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
            <img
              src="/safari-logo.png"
              alt="Safari logo"
              width={96}
              height={96}
              style={{
                borderRadius: 24,
                flexShrink: 0,
                boxShadow: "0 12px 28px rgba(124,58,237,0.35)",
              }}
            />
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
        <h2 style={{ margin: "40px 0 16px", fontSize: 22 }}>
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
            marginTop: 40,
            background: "#ffffff",
            border: "1px solid #e5e7eb",
            borderRadius: 24,
            padding: 28,
            boxShadow: "0 4px 24px rgba(17,24,39,0.05)",
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
            borderRadius: 24,
            padding: 28,
            background: "linear-gradient(135deg,#eff6ff,#f5f3ff)",
            border: "1px solid #ddd6fe",
          }}
        >
          <h2 style={{ margin: "0 0 6px", fontSize: 22 }}>📶 {t.slowTitle}</h2>
          <p style={{ margin: 0, color: "#374151", fontSize: 15 }}>
            {t.slowDesc}
          </p>
        </section>

        {/* Features */}
        <h2 style={{ margin: "40px 0 16px", fontSize: 22 }}>
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
              style={{
                background: "#ffffff",
                border: "1px solid #e5e7eb",
                borderRadius: 16,
                padding: 18,
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
        <h2 style={{ margin: "40px 0 16px", fontSize: 22 }}>{t.infoTitle}</h2>
        <section
          style={{
            background: "#ffffff",
            border: "1px solid #e5e7eb",
            borderRadius: 16,
            overflow: "hidden",
          }}
        >
          {[
            [t.versionLabel, "0.6.0 (Beta)"],
            [t.sizeLabel, "33.4 MB · 26.5 MB · 35.9 MB"],
            [t.updated, "Nov 14, 2026"],
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

        <p style={{ color: "#6b7280", fontSize: 14, marginTop: 32 }}>
          {t.footer}
        </p>

        <div
          style={{
            marginTop: 40,
            borderTop: "1px solid #e5e7eb",
            paddingTop: 28,
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
