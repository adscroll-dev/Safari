-- 017_stage11_settings.sql: WhatsApp Settings parity (Stage 11).
-- Read receipts, profile-photo / about / groups privacy, on the
-- same user_settings table as last-seen (010) and status (012).
-- "everyone" is the default everywhere; "nobody" hides the thing
-- from everyone except the user themself. Read receipts off means
-- the user sends no read receipts AND sees no blue ticks.
ALTER TABLE user_settings
  ADD COLUMN IF NOT EXISTS read_receipts BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS avatar_privacy TEXT NOT NULL DEFAULT 'everyone'
    CHECK (avatar_privacy IN ('everyone', 'nobody')),
  ADD COLUMN IF NOT EXISTS about_privacy TEXT NOT NULL DEFAULT 'everyone'
    CHECK (about_privacy IN ('everyone', 'nobody')),
  ADD COLUMN IF NOT EXISTS groups_privacy TEXT NOT NULL DEFAULT 'everyone'
    CHECK (groups_privacy IN ('everyone', 'nobody'));
