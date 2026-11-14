-- VYRO — PostgreSQL migration 003: v0.4 features (2026-11-12)
-- Idempotent: safe to re-run. NewPostgresStore applies it on every start,
-- after 001_init.sql and 002_uploads.sql.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS lat            DOUBLE PRECISION;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS lng            DOUBLE PRECISION;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_label TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS expires_at     TIMESTAMPTZ;
ALTER TABLE users    ADD COLUMN IF NOT EXISTS is_bot         BOOLEAN NOT NULL DEFAULT FALSE;

-- Read / delivery receipts (WhatsApp-style ticks).
CREATE TABLE IF NOT EXISTS message_receipts (
    message_id   UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delivered_at TIMESTAMPTZ,
    read_at      TIMESTAMPTZ,
    PRIMARY KEY (message_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_receipts_user ON message_receipts (user_id);

-- Per-user chat settings: pin / archive / folders (Telegram-style).
CREATE TABLE IF NOT EXISTS chat_settings (
    chat_id  UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pinned   BOOLEAN NOT NULL DEFAULT FALSE,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    folder   TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (chat_id, user_id)
);

-- Scheduled messages (sent later by the server dispatcher).
CREATE TABLE IF NOT EXISTS scheduled_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id         UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         TEXT NOT NULL DEFAULT '',
    kind            TEXT NOT NULL DEFAULT 'text',
    file_url        TEXT NOT NULL DEFAULT '',
    file_name       TEXT NOT NULL DEFAULT '',
    file_size       BIGINT NOT NULL DEFAULT 0,
    mime_type       TEXT NOT NULL DEFAULT '',
    reply_to_id     UUID REFERENCES messages(id) ON DELETE SET NULL,
    send_at         TIMESTAMPTZ NOT NULL,
    sent            BOOLEAN NOT NULL DEFAULT FALSE,
    sent_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_scheduled_due ON scheduled_messages (send_at) WHERE NOT sent;

-- Bots: token is stored only as a SHA-256 hash; the raw token is shown
-- once, at creation. Never store or log a raw bot token.
CREATE TABLE IF NOT EXISTS bots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    username    CITEXT NOT NULL UNIQUE,
    token_hash  TEXT NOT NULL UNIQUE,
    webhook_url TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
