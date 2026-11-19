ALTER TABLE user_settings DROP CONSTRAINT IF EXISTS user_settings_status_privacy_check;
ALTER TABLE user_settings ADD CONSTRAINT user_settings_status_privacy_check
  CHECK (status_privacy IN ('everyone', 'nobody', 'except'));

CREATE TABLE IF NOT EXISTS status_exclusions (
    owner_id TEXT NOT NULL,
    excluded_user_id TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (owner_id, excluded_user_id)
);
CREATE INDEX IF NOT EXISTS idx_status_exclusions_excluded_user_id
    ON status_exclusions(excluded_user_id);
