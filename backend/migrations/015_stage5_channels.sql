-- 015_stage5_channels.sql: Channels (Stage 5) — public/private
-- channels (is_public on chats; only meaningful for type='channel')
-- and per-user view records for channel posts (WhatsApp's view
-- counter: one row per viewer, so re-views never inflate the count).
ALTER TABLE chats ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT TRUE;
CREATE TABLE IF NOT EXISTS message_views (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_message_views_message ON message_views(message_id);
