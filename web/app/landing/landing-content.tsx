"use client";

import { useEffect, useState } from "react";
import DownloadButton from "./download-button";
import { languages, translations, type LangCode } from "./translations";

const LANG_KEY = "vyro-site-lang";

function isLangCode(value: string): value is LangCode {
  return languages.some((l) => l.code === value);
}

const GRADIENT = "linear-gradient(135deg,#2563eb,#7c3aed)";

function PhoneFrame({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        flex: "0 0 220px",
        height: 420,
        borderRadius: 28,
        border: "6px solid #111827",
        background: "#f8fafc",
        overflow: "hidden",
        boxShadow: "0 10px 30px rgba(37,99,235,0.15)",
        display: "flex",
        flexDirection: "column",
      }}
    >
      {children}
    </div>
  );
}

function Avatar({ label, bg }: { label: string; bg: string }) {
  return (
    <div
      style={{
        width: 38,
        height: 38,
        borderRadius: "50%",
        background: bg,
        color: "#fff",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: 15,
        fontWeight: 700,
        flexShrink: 0,
      }}
    >
      {label}
    </div>
  );
}

function ChatRow({
  label,
  bg,
  name,
  preview,
  time,
  unread,
}: {
  label: string;
  bg: string;
  name: string;
  preview: string;
  time: string;
  unread?: number;
}) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "10px 12px",
        borderBottom: "1px solid #eef2f7",
      }}
    >
      <Avatar label={label} bg={bg} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 13,
            fontWeight: 700,
            color: "#111827",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          }}
        >
          {name}
        </div>
        <div
          style={{
            fontSize: 12,
            color: "#6b7280",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          }}
        >
          {preview}
        </div>
      </div>
      <div style={{ textAlign: "end", flexShrink: 0 }}>
        <div style={{ fontSize: 11, color: "#9ca3af" }}>{time}</div>
        {unread ? (
          <div
            style={{
              marginTop: 4,
              marginInlineStart: "auto",
              width: 18,
              height: 18,
              borderRadius: "50%",
              background: GRADIENT,
              color: "#fff",
              fontSize: 11,
              fontWeight: 700,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            {unread}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function Bubble({
  text,
  out,
  time,
}: {
  text: string;
  out?: boolean;
  time: string;
}) {
  return (
    <div
      style={{
        display: "flex",
        justifyContent: out ? "flex-end" : "flex-start",
        margin: "6px 12px",
      }}
    >
      <div
        style={{
          maxWidth: "80%",
          padding: "8px 12px",
          borderRadius: 16,
          borderBottomRightRadius: 4,
          background: out ? GRADIENT : "#ffffff",
          color: out ? "#ffffff" : "#111827",
          fontSize: 13,
          lineHeight: 1.5,
          boxShadow: "0 1px 2px rgba(0,0,0,0.06)",
        }}
      >
        {text}
        <span
          style={{
            fontSize: 10,
            opacity: 0.7,
            marginInlineStart: 8,
            whiteSpace: "nowrap",
          }}
        >
          {time}
        </span>
      </div>
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
          <div
            style={{
              width: 36,
              height: 36,
              borderRadius: 10,
              background: GRADIENT,
              color: "#fff",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontWeight: 800,
              fontSize: 20,
            }}
          >
            V
          </div>
          <strong style={{ fontSize: 18, letterSpacing: 1 }}>VYRO</strong>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
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
            <div
              style={{
                width: 96,
                height: 96,
                borderRadius: 24,
                background: GRADIENT,
                color: "#fff",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontWeight: 800,
                fontSize: 52,
                flexShrink: 0,
                boxShadow: "0 12px 28px rgba(124,58,237,0.35)",
              }}
            >
              V
            </div>
            <div style={{ flex: "1 1 280px", minWidth: 0 }}>
              <h1 style={{ fontSize: 34, margin: "0 0 2px" }}>VYRO</h1>
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
          <PhoneFrame>
            <div
              style={{
                background: GRADIENT,
                color: "#fff",
                padding: "14px 14px 10px",
                fontWeight: 700,
                fontSize: 16,
              }}
            >
              VYRO
            </div>
            <div style={{ padding: "10px 12px 4px" }}>
              <div
                style={{
                  background: "#e5e7eb",
                  borderRadius: 999,
                  padding: "6px 12px",
                  fontSize: 12,
                  color: "#6b7280",
                }}
              >
                🔍 Search
              </div>
            </div>
            <ChatRow
              label="A"
              bg="#2563eb"
              name="Amina"
              preview="See you tomorrow! 👋"
              time="16:20"
              unread={2}
            />
            <ChatRow
              label="S"
              bg="#7c3aed"
              name="Sara"
              preview="✓✓ The photos are here"
              time="15:41"
            />
            <ChatRow
              label="F"
              bg="#0891b2"
              name="Family ❤️"
              preview="Mom: dinner at 7"
              time="14:02"
              unread={5}
            />
            <ChatRow
              label="W"
              bg="#db2777"
              name="Work"
              preview="Ali: file sent 📎"
              time="11:18"
            />
          </PhoneFrame>

          <PhoneFrame>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: "12px",
                borderBottom: "1px solid #eef2f7",
                background: "#fff",
              }}
            >
              <Avatar label="A" bg="#2563eb" />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14, fontWeight: 700 }}>Amina</div>
                <div style={{ fontSize: 11, color: "#059669" }}>● online</div>
              </div>
              <span style={{ fontSize: 18 }}>📞</span>
              <span style={{ fontSize: 18 }}>📹</span>
            </div>
            <div style={{ flex: 1, paddingTop: 10, overflow: "hidden" }}>
              <Bubble text="Hi! Did you try VYRO?" time="16:18" />
              <Bubble
                text="Yes — so fast, and it uses almost no data! 🚀"
                out
                time="16:19 ✓✓"
              />
              <Bubble text="The voice call was so clear" time="16:20" />
              <div
                style={{
                  display: "flex",
                  justifyContent: "flex-end",
                  margin: "6px 12px",
                }}
              >
                <div
                  style={{
                    background: GRADIENT,
                    borderRadius: 16,
                    borderBottomRightRadius: 4,
                    padding: "10px 12px",
                    display: "flex",
                    alignItems: "center",
                    gap: 8,
                  }}
                >
                  <span style={{ color: "#fff", fontSize: 14 }}>▶</span>
                  <span
                    style={{
                      display: "flex",
                      gap: 2,
                      alignItems: "center",
                    }}
                  >
                    {[6, 12, 9, 15, 8, 13, 7].map((h, i) => (
                      <span
                        key={i}
                        style={{
                          width: 3,
                          height: h,
                          borderRadius: 2,
                          background: "#fff",
                          opacity: 0.85,
                        }}
                      />
                    ))}
                  </span>
                  <span style={{ color: "#fff", fontSize: 11 }}>0:24</span>
                </div>
              </div>
            </div>
            <div
              style={{
                margin: 10,
                background: "#fff",
                border: "1px solid #e5e7eb",
                borderRadius: 999,
                padding: "8px 14px",
                fontSize: 13,
                color: "#9ca3af",
              }}
            >
              Message…
            </div>
          </PhoneFrame>

          <PhoneFrame>
            <div
              style={{
                padding: "14px 12px 6px",
                fontWeight: 700,
                fontSize: 15,
                background: "#fff",
              }}
            >
              Stories
            </div>
            <div
              style={{
                display: "flex",
                gap: 12,
                padding: "8px 12px 14px",
                background: "#fff",
                borderBottom: "1px solid #eef2f7",
              }}
            >
              {["You", "Sara", "Ali", "Mina"].map((n, i) => (
                <div
                  key={n}
                  style={{ textAlign: "center", flexShrink: 0 }}
                >
                  <div
                    style={{
                      width: 52,
                      height: 52,
                      borderRadius: "50%",
                      padding: 3,
                      background:
                        i === 0 ? "#e5e7eb" : GRADIENT,
                    }}
                  >
                    <div
                      style={{
                        width: "100%",
                        height: "100%",
                        borderRadius: "50%",
                        background: "#fff",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontSize: 18,
                      }}
                    >
                      {i === 0 ? "＋" : "🙂"}
                    </div>
                  </div>
                  <div style={{ fontSize: 11, marginTop: 4 }}>{n}</div>
                </div>
              ))}
            </div>
            <div style={{ padding: 12 }}>
              <div
                style={{
                  background: "#fff",
                  border: "1px solid #e5e7eb",
                  borderRadius: 16,
                  padding: 14,
                }}
              >
                <div style={{ fontSize: 13, fontWeight: 700 }}>
                  📞 Voice call
                </div>
                <div style={{ fontSize: 12, color: "#6b7280" }}>
                  Amina · 12:34
                </div>
                <div
                  style={{
                    marginTop: 10,
                    height: 6,
                    borderRadius: 999,
                    background: "#e5e7eb",
                    overflow: "hidden",
                  }}
                >
                  <div
                    style={{
                      width: "35%",
                      height: "100%",
                      background: GRADIENT,
                    }}
                  />
                </div>
                <div style={{ fontSize: 11, color: "#059669", marginTop: 6 }}>
                  ● Data Saver on — 12 kbps
                </div>
              </div>
              <div
                style={{
                  marginTop: 12,
                  background: "#fff",
                  border: "1px solid #e5e7eb",
                  borderRadius: 16,
                  padding: 14,
                }}
              >
                <div style={{ fontSize: 13, fontWeight: 700 }}>
                  🔒 End-to-end encrypted
                </div>
                <div style={{ fontSize: 12, color: "#6b7280" }}>
                  Only you and Amina can read this chat.
                </div>
              </div>
            </div>
          </PhoneFrame>
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
            [t.sizeLabel, "32.6 MB · 25.7 MB · 35.1 MB"],
            [t.updated, "Nov 14, 2026"],
            [t.requires, "Android 6.0+"],
            [t.offeredBy, "VYRO Team"],
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
