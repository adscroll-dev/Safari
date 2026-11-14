-- 010_privacy.sql: per-user privacy settings (Telegram/WhatsApp-style
-- last-seen visibility: "everyone" or "nobody").
CREATE TABLE IF NOT EXISTS user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_seen_visibility TEXT NOT NULL DEFAULT 'everyone'
      CHECK (last_seen_visibility IN ('everyone', 'nobody'))
);
