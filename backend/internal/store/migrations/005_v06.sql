-- VYRO v0.6 migration.
--
-- 1. Group E2EE (Sender Keys): group_sender_keys stores, per group /
--    channel chat, one row per (sender device -> recipient device)
--    holding the sender's sender-key material encrypted under the
--    existing 1:1 E2EE session between those two devices (X3DH +
--    Double Ratchet, see internal/e2ee). The ciphertext column is an
--    opaque 1:1 envelope: the server stores and relays it, never
--    decrypts it, and cannot read group plaintext. Group message
--    bodies stay in messages.content as kind 'e2ee' envelopes, now in
--    the group-envelope shape (see e2ee.ValidateGroupEnvelope).
--    When a member is removed, rows addressed to them are deleted so
--    their devices cannot pick up sender keys afterwards; senders
--    rotate their sender key when membership changes (client-side).
-- 2. Multi-device sync: user_devices registers every device of an
--    account (device_id "default" is the first device). A peer fetches
--    a key bundle per device, and the server fans a message out to
--    every connected device of each member (see ws.Hub).
--    (The legacy devices table from 001_init.sql is unused by the
--    server and left untouched.)
-- 3. Web Push persistence: push_subscriptions moves subscriptions out
--    of process memory into PostgreSQL, so they survive a restart.
--    Payloads still never contain message content.
CREATE TABLE IF NOT EXISTS group_sender_keys (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id          UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sender_device_id TEXT NOT NULL,
    key_id           TEXT NOT NULL,
    for_user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    for_device_id    TEXT NOT NULL,
    ciphertext       TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chat_id, sender_user_id, sender_device_id, key_id, for_user_id, for_device_id)
);
CREATE INDEX IF NOT EXISTS idx_group_sender_keys_recipient
    ON group_sender_keys (chat_id, for_user_id, for_device_id);
CREATE INDEX IF NOT EXISTS idx_group_sender_keys_sender
    ON group_sender_keys (chat_id, sender_user_id, sender_device_id);

CREATE TABLE IF NOT EXISTS user_devices (
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id    TEXT NOT NULL,
    name         TEXT NOT NULL DEFAULT '',
    platform     TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, device_id)
);

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT NOT NULL,
    p256dh     TEXT NOT NULL,
    auth       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, endpoint)
);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user ON push_subscriptions (user_id);
