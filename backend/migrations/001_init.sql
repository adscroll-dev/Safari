-- VYRO — PostgreSQL schema v0.1 (001_init.sql)
-- Apply with: psql "$DATABASE_URL" -f migrations/001_init.sql
-- Or via docker-compose (the postgres service mounts this file on first boot).
--
-- Notes:
-- * Passwords are stored ONLY as bcrypt hashes (password_hash). Never plaintext.
-- * Messages in v0.1 are plaintext (dev build). The E2EE build will store
--   ciphertext envelopes in messages.content_ciphertext / content_type='e2ee'
--   — see docs/SECURITY.md. The columns below already reserve that shape.
-- * IDs use gen_random_uuid() (pgcrypto). Enable the extension first.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------------------------------------------------------------- users
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         CITEXT NOT NULL UNIQUE,
    username      CITEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL,
    bio           TEXT NOT NULL DEFAULT '',
    avatar_url    TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,              -- bcrypt hash only
    is_online     BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
-- CITEXT requires the citext extension on some setups; if unavailable,
-- replace CITEXT with TEXT and rely on the application-level lower() checks.

-- ------------------------------------------------- identity keys (E2EE)
-- One row per device per user. Public keys only — private keys NEVER
-- leave the device. See docs/SECURITY.md (X3DH / Signal protocol).
CREATE TABLE IF NOT EXISTS identity_keys (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id        TEXT NOT NULL,
    identity_key     BYTEA NOT NULL,          -- public identity key
    signed_prekey    BYTEA NOT NULL,
    signed_prekey_id INT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, device_id)
);
CREATE TABLE IF NOT EXISTS one_time_prekeys (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id  TEXT NOT NULL,
    key_id     INT NOT NULL,
    public_key BYTEA NOT NULL,
    used       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, device_id, key_id)
);

-- ---------------------------------------------------------------- chats
CREATE TABLE IF NOT EXISTS chats (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT NOT NULL CHECK (type IN ('dm','group','channel')),
    title      TEXT NOT NULL DEFAULT '',
    owner_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chat_members (
    chat_id   UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member','admin','owner')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_chat_members_user ON chat_members (user_id);

-- ------------------------------------------------------------- messages
CREATE TABLE IF NOT EXISTS messages (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id             UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind                TEXT NOT NULL DEFAULT 'text'
                        CHECK (kind IN ('text','image','video','file','voice','location','e2ee')),
    content             TEXT NOT NULL DEFAULT '',   -- v0.1 plaintext; E2EE: empty / envelope ref
    content_ciphertext  BYTEA,                     -- E2EE ciphertext envelope (future)
    reply_to_id         UUID REFERENCES messages(id) ON DELETE SET NULL,
    edited_at           TIMESTAMPTZ,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_messages_chat_created ON messages (chat_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_content_trgm ON messages USING gin (content gin_trgm_ops);
-- NOTE: idx_messages_content_trgm needs pg_trgm (CREATE EXTENSION pg_trgm).
-- If pg_trgm is unavailable, drop that index; search falls back to ILIKE.

CREATE TABLE IF NOT EXISTS reactions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, user_id, emoji)
);

-- -------------------------------------------------------------- stories
CREATE TABLE IF NOT EXISTS stories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_url  TEXT NOT NULL,
    caption    TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_stories_user ON stories (user_id, created_at DESC);

-- ------------------------------------------------- blocks and reports
CREATE TABLE IF NOT EXISTS blocks (
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE TABLE IF NOT EXISTS reports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN ('user','message','chat')),
    target_id   UUID NOT NULL,
    reason      TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','reviewed','actioned','dismissed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports (status, created_at DESC);

-- ------------------------------------------------------------- devices
CREATE TABLE IF NOT EXISTS devices (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name  TEXT NOT NULL DEFAULT '',
    platform     TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, device_name)
);

-- Updated-at trigger helper
CREATE OR REPLACE FUNCTION vyro_touch_updated_at() RETURNS trigger AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_updated ON users;
CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION vyro_touch_updated_at();
DROP TRIGGER IF EXISTS trg_chats_updated ON chats;
CREATE TRIGGER trg_chats_updated BEFORE UPDATE ON chats
  FOR EACH ROW EXECUTE FUNCTION vyro_touch_updated_at();
DROP TRIGGER IF EXISTS trg_messages_updated ON messages;
CREATE TRIGGER trg_messages_updated BEFORE UPDATE ON messages
  FOR EACH ROW EXECUTE FUNCTION vyro_touch_updated_at();
