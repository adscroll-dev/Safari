-- Stage 1 chat gaps (WhatsApp parity): mute chat, pin message,
-- delete-for-me. Idempotent: safe to run on a database that already
-- has these columns/tables.
ALTER TABLE chat_settings
    ADD COLUMN IF NOT EXISTS muted BOOLEAN NOT NULL DEFAULT FALSE;

-- One pinned message per chat (WhatsApp-style pinned banner).
ALTER TABLE chats
    ADD COLUMN IF NOT EXISTS pinned_message_id UUID REFERENCES messages(id) ON DELETE SET NULL;

-- Delete-for-me: messages a user hid only for themself. The message
-- itself stays for everyone else; delete-for-everyone is still the
-- hard delete (messages.deleted_at).
CREATE TABLE IF NOT EXISTS message_deletions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (message_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_message_deletions_user ON message_deletions (user_id);
