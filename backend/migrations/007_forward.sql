-- 007_forward: message forwarding attribution (v0.7).
-- A forwarded message keeps who it originally came from, so clients
-- can show "Forwarded from <name>" like Telegram/WhatsApp.
ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS forwarded_from TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS forwarded_from_id UUID;
