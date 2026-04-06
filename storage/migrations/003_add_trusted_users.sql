-- Trusted users table for @claude trigger access control
CREATE TABLE IF NOT EXISTS trusted_users (
    jid TEXT PRIMARY KEY,      -- WhatsApp JID (canonical format)
    added_at INTEGER NOT NULL  -- Unix timestamp when added
);
