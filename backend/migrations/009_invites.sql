-- 009_invites.sql: Telegram-style invite links for groups and channels.
CREATE TABLE IF NOT EXISTS chat_invites (
    token TEXT PRIMARY KEY,
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_chat_invites_chat ON chat_invites(chat_id);
