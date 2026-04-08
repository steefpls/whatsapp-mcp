-- Migration: 004_add_quoted_messages
-- Description: Add quote-reply parent fields to messages
-- Previous: 003_add_trusted_users
-- Version: 004
-- Created: 2026-04-08

-- WhatsApp quote-replies carry the parent message's id, sender, and content via
-- ContextInfo on every message subtype. Capture them so the @claude trigger and
-- MCP consumers can show what a message was actually replying to.
ALTER TABLE messages ADD COLUMN quoted_message_id TEXT;
ALTER TABLE messages ADD COLUMN quoted_sender_jid TEXT;
ALTER TABLE messages ADD COLUMN quoted_text TEXT;

-- recreate the view to surface the new columns plus a resolved quoted-sender name
DROP VIEW IF EXISTS messages_with_names;

CREATE VIEW messages_with_names AS
SELECT
    m.id,
    m.chat_jid,
    m.sender_jid,

    -- Get sender's current push name (WhatsApp display name)
    COALESCE(p.push_name, '') as sender_push_name,

    -- Get sender's current contact name (saved contact)
    COALESCE(c_sender.contact_name, '') as sender_contact_name,

    -- Get chat name (for display)
    COALESCE(
        c_chat.contact_name,  -- Saved contact name for DMs
        c_chat.push_name,     -- Push name for DMs or group name for groups
        m.chat_jid            -- Fallback to JID
    ) as chat_name,

    -- Original message fields
    m.text,
    m.timestamp,
    m.is_from_me,
    m.message_type,
    m.created_at,

    -- Media metadata fields (nullable)
    media.file_path as media_file_path,
    media.file_name as media_file_name,
    media.file_size as media_file_size,
    media.mime_type as media_mime_type,
    media.width as media_width,
    media.height as media_height,
    media.duration as media_duration,
    media.download_status as media_download_status,
    media.download_timestamp as media_download_timestamp,
    media.download_error as media_download_error,

    -- Quote-reply parent fields (nullable)
    m.quoted_message_id,
    m.quoted_sender_jid,
    m.quoted_text,

    -- Resolved name for the quoted-message sender (push name preferred, contact name fallback)
    COALESCE(qp.push_name, qc.contact_name, '') as quoted_sender_name
FROM messages m
LEFT JOIN push_names p ON m.sender_jid = p.jid
LEFT JOIN chats c_sender ON m.sender_jid = c_sender.jid
LEFT JOIN chats c_chat ON m.chat_jid = c_chat.jid
LEFT JOIN media_metadata media ON m.id = media.message_id
LEFT JOIN push_names qp ON m.quoted_sender_jid = qp.jid
LEFT JOIN chats qc ON m.quoted_sender_jid = qc.jid;
