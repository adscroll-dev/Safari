"use client";

import { useEffect, useState } from "react";
import { languages, translations, type LangCode } from "../../landing/translations";
import { joinTranslations } from "../../landing/join-translations";

// Invite preview, as returned by GET /api/invites/{token}.
export type InvitePreview = {
  token: string;
  chat_id: string;
  title: string;
  type: string; // "group" | "channel"
  member_count: number;
  invited_by: string;
};

const LANG_KEY = "vyro-site-lang";

function isLangCode(value: string): value is LangCode {
  return languages.some((l) => l.code === value);
}

export default function JoinContent({ preview }: { preview: InvitePreview | null }) {
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
  const j = joinTranslations[lang];

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

  const isChannel = preview?.type === "channel";

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
        <a
          href="/landing"
          style={{ display: "flex", alignItems: "center", gap: 10, textDecoration: "none", color: "#111827" }}
        >
          <img src="/safari-logo.png" alt="Safari logo" width={36} height={36} style={{ borderRadius: 10 }} />
          <strong style={{ fontSize: 18, letterSpacing: 1 }}>Safari</strong>
        </a>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <label htmlFor="safari-lang" style={{ fontSize: 14, color: "#4b5563", fontWeight: 600 }}>
            🌐 {t.languageLabel}
          </label>
          <select
            id="safari-lang"
            value={lang}
            onChange={(e) => changeLang(e.target.value as LangCode)}
            style={{ fontSize: 14, padding: "6px 8px", borderRadius: 8, border: "1px solid #d1d5db", background: "#fff" }}
          >
            {languages.map((l) => (
              <option key={l.code} value={l.code}>
                {l.label}
              </option>
            ))}
          </select>
        </div>
      </header>

      {/* Invite card */}
      <section style={{ maxWidth: 560, margin: "0 auto", padding: "48px 20px" }}>
        <div
          style={{
            background: "#fff",
            border: "1px solid #e5e7eb",
            borderRadius: 16,
            padding: "32px 24px",
            textAlign: "center",
            boxShadow: "0 10px 30px rgba(37,99,235,0.08)",
          }}
        >
          {preview ? (
            <>
              <div style={{ fontSize: 48, lineHeight: 1 }}>{isChannel ? "📢" : "👥"}</div>
              <h1 style={{ fontSize: 24, margin: "16px 0 4px" }}>{preview.title}</h1>
              <p style={{ color: "#4b5563", margin: "0 0 8px" }}>
                {isChannel ? j.joinChannel : j.joinGroup}
              </p>
              <p style={{ color: "#4b5563", margin: "0 0 8px" }}>
                {j.membersLabel}: {preview.member_count}
              </p>
              {preview.invited_by ? (
                <p style={{ color: "#4b5563", margin: "0 0 24px" }}>
                  {j.invitedBy} @{preview.invited_by}
                </p>
              ) : null}
              <a
                href="/landing"
                style={{
                  display: "inline-block",
                  background: "#25d366",
                  color: "#fff",
                  fontWeight: 700,
                  fontSize: 16,
                  padding: "12px 28px",
                  borderRadius: 999,
                  textDecoration: "none",
                }}
              >
                {j.downloadToJoin}
              </a>
              <p style={{ color: "#6b7280", fontSize: 14, margin: "24px 0 0" }}>{j.haveApp}</p>
            </>
          ) : (
            <>
              <div style={{ fontSize: 48, lineHeight: 1 }}>🔗</div>
              <h1 style={{ fontSize: 24, margin: "16px 0 8px" }}>{j.invalidTitle}</h1>
              <p style={{ color: "#4b5563", margin: "0 0 24px" }}>{j.invalidBody}</p>
              <a
                href="/landing"
                style={{
                  display: "inline-block",
                  background: "#25d366",
                  color: "#fff",
                  fontWeight: 700,
                  fontSize: 16,
                  padding: "12px 28px",
                  borderRadius: 999,
                  textDecoration: "none",
                }}
              >
                {j.downloadToJoin}
              </a>
            </>
          )}
        </div>
      </section>
    </main>
  );
}
