"use client";

import { useEffect, useState } from "react";
import DownloadButton from "./download-button";
import { languages, translations, type LangCode } from "./translations";

const LANG_KEY = "vyro-site-lang";

function isLangCode(value: string): value is LangCode {
  return languages.some((l) => l.code === value);
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
        maxWidth: 880,
        margin: "0 auto",
        padding: "48px 20px",
        lineHeight: 1.8,
        color: "#111827",
        background: "#ffffff",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "flex-end",
          alignItems: "center",
          gap: 8,
          marginBottom: 16,
        }}
      >
        <label
          htmlFor="vyro-lang"
          style={{ fontSize: 14, color: "#4b5563", fontWeight: 600 }}
        >
          🌐 {t.languageLabel}
        </label>
        <select
          id="vyro-lang"
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
      <p style={{ fontSize: 20, margin: "0 0 8px" }}>{t.tagline}</p>
      <p>{t.intro}</p>

      <h2 style={{ marginTop: 48 }}>{t.downloadTitle}</h2>
      <p style={{ color: "#4b5563", fontSize: 14 }}>{t.versionLine}</p>

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

      <h2 style={{ marginTop: 48 }}>{t.featuresTitle}</h2>
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
        {t.footer}
      </p>

      <div
        style={{
          marginTop: 48,
          borderTop: "1px solid #e5e7eb",
          paddingTop: 32,
        }}
      >
        <h2>{t.aboutTitle}</h2>
        <p>{t.intro}</p>
        <ul>
          {t.features.map((f) => (
            <li key={f.title}>
              <strong>{f.title}:</strong> {f.desc}
            </li>
          ))}
        </ul>
      </div>
    </main>
  );
}
