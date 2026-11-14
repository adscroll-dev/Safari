-- VYRO migration 008: FCM tokens (Firebase Cloud Messaging, mobile push).
--
-- One row per (user, FCM registration token). The mobile app POSTs its
-- token to /api/push/fcm-token after login; the server sends a
-- notification to each token of a member who did not receive a message
-- over WebSocket. Tokens are deleted when FCM reports them as
-- unregistered (app uninstalled / token revoked). Payloads never
-- contain message content (see internal/push/fcm.go).
CREATE TABLE IF NOT EXISTS fcm_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL,
    device_id  TEXT NOT NULL DEFAULT '',
    platform   TEXT NOT NULL DEFAULT 'android',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, token)
);
CREATE INDEX IF NOT EXISTS idx_fcm_tokens_user ON fcm_tokens (user_id);
