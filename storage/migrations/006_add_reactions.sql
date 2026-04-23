-- Migration: 006_add_reactions
-- Description: Track reactions (emoji responses) on messages

CREATE TABLE IF NOT EXISTS reactions (
    target_message_id TEXT NOT NULL,
    reactor_jid TEXT NOT NULL,
    emoji TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    PRIMARY KEY (target_message_id, reactor_jid)
);

CREATE INDEX IF NOT EXISTS idx_reactions_target ON reactions(target_message_id);
