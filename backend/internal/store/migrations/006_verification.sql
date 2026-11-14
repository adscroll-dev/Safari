-- VYRO v0.7 migration: email verification with 6-digit codes.
--
-- 1. email_codes: one row per (email, purpose) where purpose is
--    'register' or 'login'. Only the SHA-256 hash of the code is
--    stored (code_hash); the plain 6-digit code exists only in the
--    email sent to the user and is never logged or returned by the
--    API. attempts counts failed verifications against this code
--    (5 failures invalidate it); consumed_at marks a successful,
--    single-use verification; expires_at is 10 minutes after issue.
--    Saving a new code replaces the row, which is how a resend
--    invalidates the previous code.
-- 2. pending_users: a registration waiting for its code to be
--    verified. The real users row is created only on successful
--    verification; the password here is already bcrypt-hashed.
--    The row expires with its code (10 minutes) and is deleted on
--    verification.
CREATE TABLE IF NOT EXISTS email_codes (
    email       CITEXT NOT NULL,
    purpose     TEXT NOT NULL CHECK (purpose IN ('register', 'login')),
    code_hash   TEXT NOT NULL,
    attempts    INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (email, purpose)
);
CREATE INDEX IF NOT EXISTS idx_email_codes_expires ON email_codes (expires_at);

CREATE TABLE IF NOT EXISTS pending_users (
    email         CITEXT PRIMARY KEY,
    username      CITEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pending_users_expires ON pending_users (expires_at);
