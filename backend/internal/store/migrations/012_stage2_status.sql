-- 012_stage2_status.sql: Status (Updates) gaps — text/audio stories,
-- viewers, reactions, replies, per-user status privacy.
ALTER TABLE stories ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'photo'
  CHECK (kind IN ('photo', 'video', 'text', 'audio'));
ALTER TABLE stories ADD COLUMN IF NOT EXISTS text_content TEXT NOT NULL DEFAULT '';
ALTER TABLE stories ADD COLUMN IF NOT EXISTS background_color TEXT NOT NULL DEFAULT '';

ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS status_privacy TEXT NOT NULL DEFAULT 'everyone'
  CHECK (status_privacy IN ('everyone', 'nobody'));

CREATE TABLE IF NOT EXISTS story_views (
    story_id UUID REFERENCES stories(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (story_id, user_id)
);
CREATE TABLE IF NOT EXISTS story_reactions (
    story_id UUID REFERENCES stories(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    emoji TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (story_id, user_id)
);
CREATE TABLE IF NOT EXISTS story_replies (
    id UUID PRIMARY KEY,
    story_id UUID REFERENCES stories(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_story_replies_story ON story_replies(story_id);
