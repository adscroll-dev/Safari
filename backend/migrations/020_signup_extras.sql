CREATE TABLE IF NOT EXISTS signup_invite_codes (
    code            TEXT PRIMARY KEY,
    inviter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_uses        INT NOT NULL DEFAULT 1 CHECK (max_uses BETWEEN 1 AND 20),
    uses            INT NOT NULL DEFAULT 0 CHECK (uses >= 0),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_signup_invite_codes_inviter
    ON signup_invite_codes (inviter_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS revoked_devices (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id  TEXT NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, device_id)
);

ALTER TABLE pending_users ADD COLUMN IF NOT EXISTS invite_code TEXT NOT NULL DEFAULT '';
ALTER TABLE pending_users ADD COLUMN IF NOT EXISTS device_id TEXT NOT NULL DEFAULT '';
ALTER TABLE pending_users ADD COLUMN IF NOT EXISTS device_name TEXT NOT NULL DEFAULT '';
ALTER TABLE pending_users ADD COLUMN IF NOT EXISTS platform TEXT NOT NULL DEFAULT '';
