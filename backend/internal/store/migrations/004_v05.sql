-- VYRO v0.5 migration.
-- E2EE signing key columns on identity_keys (Ed25519 signing public
-- key + Ed25519 signature over the signed pre-key). Pre-v0.5 rows have
-- NULL here; the client treats a missing signature as "unverified
-- bundle" and must not use it for E2EE.
ALTER TABLE identity_keys ADD COLUMN IF NOT EXISTS sign_key BYTEA;
ALTER TABLE identity_keys ADD COLUMN IF NOT EXISTS signed_prekey_sig BYTEA;
