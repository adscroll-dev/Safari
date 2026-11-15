-- 014_stage4_polls.sql: Group polls (Stage 4) — a poll lives in a
-- chat and appears in the chat flow as a message of kind "poll"
-- (content {"poll_id": "...", "question": "..."}). Votes are one
-- row per (poll, user, option); re-voting replaces the old votes.
CREATE TABLE IF NOT EXISTS polls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    question TEXT NOT NULL,
    options TEXT NOT NULL,
    allow_multiple BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS poll_votes (
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    option_index INT NOT NULL CHECK (option_index >= 0),
    PRIMARY KEY (poll_id, user_id, option_index)
);
CREATE INDEX IF NOT EXISTS idx_polls_chat ON polls(chat_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_poll_votes_poll ON poll_votes(poll_id);
