-- 013_stage3_calls.sql: Call history — one record per call, written by
-- the caller when the call ends (or is missed). Group calls have a NULL
-- callee_id; the chat is then the group.
CREATE TABLE IF NOT EXISTS call_records (
    id UUID PRIMARY KEY,
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    caller_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    callee_id UUID REFERENCES users(id) ON DELETE CASCADE,
    call_type TEXT NOT NULL DEFAULT 'voice'
        CHECK (call_type IN ('voice', 'video')),
    status TEXT NOT NULL DEFAULT 'missed'
        CHECK (status IN ('missed', 'ended')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    duration_seconds INT NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0)
);
CREATE INDEX IF NOT EXISTS idx_call_records_caller
    ON call_records(caller_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_call_records_callee
    ON call_records(callee_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_call_records_chat
    ON call_records(chat_id, started_at DESC);
