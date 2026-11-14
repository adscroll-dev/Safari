-- VYRO — PostgreSQL migration 002: file attachments on messages (v0.3)
-- Apply with: psql "$DATABASE_URL" -f migrations/002_uploads.sql
-- Idempotent: safe to re-run. NewPostgresStore applies it on every start,
-- after 001_init.sql, so databases created by v0.1/v0.2 gain the columns.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_url  TEXT   NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_name TEXT   NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_size BIGINT NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS mime_type TEXT   NOT NULL DEFAULT '';
