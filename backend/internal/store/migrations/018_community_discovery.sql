ALTER TABLE communities ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public','private'));
ALTER TABLE communities ADD COLUMN IF NOT EXISTS join_policy TEXT NOT NULL DEFAULT 'open' CHECK (join_policy IN ('open','approval','invite_only'));
ALTER TABLE communities ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT FALSE;
CREATE TABLE IF NOT EXISTS community_join_requests (
    community_id UUID NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (community_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_communities_discover ON communities(visibility, is_archived, created_at DESC);
