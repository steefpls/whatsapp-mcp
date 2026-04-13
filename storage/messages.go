package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Message represents a WhatsApp message.
type Message struct {
	ID          string
	ChatJID     string // Canonical JID
	SenderJID   string // Canonical JID
	Text        string
	Timestamp   time.Time
	IsFromMe    bool
	MessageType string

	// Quote-reply parent fields. Empty when this message is not a reply.
	QuotedMessageID string // Parent WhatsApp message ID
	QuotedSenderJID string // Canonical JID of the parent's sender
	QuotedText      string // Text/caption of the parent message
}

// MessageWithNames represents a message with sender and chat names from the database view.
type MessageWithNames struct {
	Message
	SenderPushName    string         // Current WhatsApp display name (from push_names table)
	SenderContactName string         // Current saved contact name (from chats table)
	ChatName          string         // Current chat name (for display)
	MediaMetadata     *MediaMetadata // Associated media metadata (null if no media)
	QuotedSenderName  string         // Resolved display name of the quoted-message sender (empty if no reply)
}

// MessageStore handles message operations on the database.
type MessageStore struct {
	db *sql.DB
}

// NewMessageStore creates a new message store instance.
func NewMessageStore(db *sql.DB) *MessageStore {
	return &MessageStore{db: db}
}

// SaveMessage saves a WhatsApp message to the database.
func (s *MessageStore) SaveMessage(msg Message) error {
	query := `
	INSERT OR REPLACE INTO messages
	(id, chat_jid, sender_jid, text, timestamp, is_from_me, message_type,
	 quoted_message_id, quoted_sender_jid, quoted_text)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		msg.ID,
		msg.ChatJID,
		msg.SenderJID,
		msg.Text,
		msg.Timestamp.Unix(),
		msg.IsFromMe,
		msg.MessageType,
		nullableString(msg.QuotedMessageID),
		nullableString(msg.QuotedSenderJID),
		nullableString(msg.QuotedText),
	)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// nullableString returns sql.NullString-style nil for empty strings so the
// quoted_* columns stay NULL when a message is not a quote-reply.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// EditHistory represents one edit event for a message.
type EditHistory struct {
	OldText  string
	NewText  string
	EditedAt time.Time
}

// UpdateMessageText updates only the text column of an existing message,
// preserving sender, timestamp, and all other metadata. Used by the @claude
// trigger to patch the local DB after a successful WhatsApp edit, since
// outbound edits don't propagate back through the normal message handler.
// Returns nil even if no row matched (caller's choice to log).
func (s *MessageStore) UpdateMessageText(id string, newText string) error {
	_, err := s.db.Exec(`UPDATE messages SET text = ? WHERE id = ?`, newText, id)
	if err != nil {
		return fmt.Errorf("failed to update message text: %w", err)
	}
	return nil
}

// SaveEditHistory records an edit event: the old text, new text, and when it happened.
func (s *MessageStore) SaveEditHistory(messageID, oldText, newText string, editedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO message_edits (message_id, old_text, new_text, edited_at) VALUES (?, ?, ?, ?)`,
		messageID, oldText, newText, editedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to save edit history: %w", err)
	}
	return nil
}

// GetEditHistory returns all edits for a message, oldest first.
func (s *MessageStore) GetEditHistory(messageID string) ([]EditHistory, error) {
	rows, err := s.db.Query(
		`SELECT old_text, new_text, edited_at FROM message_edits WHERE message_id = ? ORDER BY edited_at ASC`,
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get edit history: %w", err)
	}
	defer rows.Close()

	var edits []EditHistory
	for rows.Next() {
		var e EditHistory
		var ts int64
		if err := rows.Scan(&e.OldText, &e.NewText, &ts); err != nil {
			return nil, err
		}
		e.EditedAt = time.Unix(ts, 0)
		edits = append(edits, e)
	}
	return edits, rows.Err()
}

// GetEditHistoryForMessages returns edit histories for multiple message IDs.
// Returns a map of messageID -> []EditHistory.
func (s *MessageStore) GetEditHistoryForMessages(messageIDs []string) (map[string][]EditHistory, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	// build placeholder list
	placeholders := make([]string, len(messageIDs))
	args := make([]any, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT message_id, old_text, new_text, edited_at FROM message_edits WHERE message_id IN (%s) ORDER BY edited_at ASC`,
		strings.Join(placeholders, ","),
	)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get edit histories: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]EditHistory)
	for rows.Next() {
		var msgID string
		var e EditHistory
		var ts int64
		if err := rows.Scan(&msgID, &e.OldText, &e.NewText, &ts); err != nil {
			return nil, err
		}
		e.EditedAt = time.Unix(ts, 0)
		result[msgID] = append(result[msgID], e)
	}
	return result, rows.Err()
}

// SaveBulk saves multiple messages in a single transaction.
// This is optimized for history sync operations.
func (s *MessageStore) SaveBulk(messages []Message) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT OR REPLACE INTO messages
	(id, chat_jid, sender_jid, text, timestamp, is_from_me, message_type,
	 quoted_message_id, quoted_sender_jid, quoted_text)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, msg := range messages {
		_, err := stmt.Exec(
			msg.ID,
			msg.ChatJID,
			msg.SenderJID,
			msg.Text,
			msg.Timestamp.Unix(),
			msg.IsFromMe,
			msg.MessageType,
			nullableString(msg.QuotedMessageID),
			nullableString(msg.QuotedSenderJID),
			nullableString(msg.QuotedText),
		)

		if err != nil {
			return fmt.Errorf("failed to insert message %s: %w", msg.ID, err)
		}
	}

	return tx.Commit()

}

// SearchMessages searches messages by text content.
func (s *MessageStore) SearchMessages(q string, limit int) ([]Message, error) {
	query := `
	SELECT id, chat_jid, sender_jid, text, timestamp, is_from_me, message_type,
	       quoted_message_id, quoted_sender_jid, quoted_text
	FROM messages
	WHERE text LIKE ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := s.db.Query(query, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return s.scanMessages(rows)
}

// GetChatMessages retrieves messages from a specific chat.
func (s *MessageStore) GetChatMessages(chatJID string, limit int, offset int) ([]Message, error) {
	query := `
	SELECT id, chat_jid, sender_jid, text, timestamp, is_from_me, message_type,
	       quoted_message_id, quoted_sender_jid, quoted_text
	FROM messages
	WHERE chat_jid = ?
	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chatJID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMessages(rows)
}

// GetMessageByID retrieves a message by its ID.
// It returns nil if the message is not found.
func (s *MessageStore) GetMessageByID(messageID string) (*Message, error) {
	query := `
	SELECT id, chat_jid, sender_jid, text, timestamp, is_from_me, message_type,
	       quoted_message_id, quoted_sender_jid, quoted_text
	FROM messages
	WHERE id = ?
	`

	row := s.db.QueryRow(query, messageID)

	var msg Message
	var timestampUnix int64
	var quotedMessageID, quotedSenderJID, quotedText sql.NullString

	err := row.Scan(
		&msg.ID,
		&msg.ChatJID,
		&msg.SenderJID,
		&msg.Text,
		&timestampUnix,
		&msg.IsFromMe,
		&msg.MessageType,
		&quotedMessageID,
		&quotedSenderJID,
		&quotedText,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	msg.Timestamp = time.Unix(timestampUnix, 0)
	msg.QuotedMessageID = quotedMessageID.String
	msg.QuotedSenderJID = quotedSenderJID.String
	msg.QuotedText = quotedText.String

	return &msg, nil
}

// GetOldestMessage retrieves the oldest message from a specific chat.
// This is used for history sync requests.
func (s *MessageStore) GetOldestMessage(chatJID string) (*Message, error) {
	query := `
	SELECT id, chat_jid, sender_jid, text, timestamp, is_from_me, message_type,
	       quoted_message_id, quoted_sender_jid, quoted_text
	FROM messages
	WHERE chat_jid = ?
	ORDER BY timestamp ASC
	LIMIT 1
	`

	row := s.db.QueryRow(query, chatJID)

	var msg Message
	var timestampUnix int64
	var quotedMessageID, quotedSenderJID, quotedText sql.NullString

	err := row.Scan(
		&msg.ID,
		&msg.ChatJID,
		&msg.SenderJID,
		&msg.Text,
		&timestampUnix,
		&msg.IsFromMe,
		&msg.MessageType,
		&quotedMessageID,
		&quotedSenderJID,
		&quotedText,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	msg.Timestamp = time.Unix(timestampUnix, 0)
	msg.QuotedMessageID = quotedMessageID.String
	msg.QuotedSenderJID = quotedSenderJID.String
	msg.QuotedText = quotedText.String

	return &msg, nil
}

// GetChatMessagesOlderThan retrieves messages older than a specific timestamp.
// This is used for retrieving newly loaded messages from history sync.
func (s *MessageStore) GetChatMessagesOlderThan(chatJID string, timestamp time.Time, limit int) ([]MessageWithNames, error) {
	query := `
	SELECT id, chat_jid, sender_jid, sender_push_name, sender_contact_name, chat_name,
	       text, timestamp, is_from_me, message_type,
	       media_file_path, media_file_name, media_file_size, media_mime_type,
	       media_width, media_height, media_duration, media_download_status,
	       media_download_timestamp, media_download_error,
	       quoted_message_id, quoted_sender_jid, quoted_text, quoted_sender_name
	FROM messages_with_names
	WHERE chat_jid = ? AND timestamp < ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := s.db.Query(query, chatJID, timestamp.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMessagesWithNames(rows)
}

// GetChatMessagesWithNamesFiltered retrieves chat messages with advanced filtering.
func (s *MessageStore) GetChatMessagesWithNamesFiltered(
	chatJID string,
	limit int,
	beforeTimestamp *time.Time,
	afterTimestamp *time.Time,
	senderJID string,
) ([]MessageWithNames, error) {
	query := `
	SELECT id, chat_jid, sender_jid, sender_push_name, sender_contact_name, chat_name,
	       text, timestamp, is_from_me, message_type,
	       media_file_path, media_file_name, media_file_size, media_mime_type,
	       media_width, media_height, media_duration, media_download_status,
	       media_download_timestamp, media_download_error,
	       quoted_message_id, quoted_sender_jid, quoted_text, quoted_sender_name
	FROM messages_with_names
	WHERE chat_jid = ?
	`

	args := []any{chatJID}

	// add timestamp filters
	if beforeTimestamp != nil {
		query += " AND timestamp < ?"
		args = append(args, beforeTimestamp.Unix())
	}

	if afterTimestamp != nil {
		query += " AND timestamp > ?"
		args = append(args, afterTimestamp.Unix())
	}

	// add sender filter
	if senderJID != "" {
		query += " AND sender_jid = ?"
		args = append(args, senderJID)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMessagesWithNames(rows)
}

// scanMessages converts SQL rows into Message objects.
func (s *MessageStore) scanMessages(rows *sql.Rows) ([]Message, error) {
	var messages []Message

	for rows.Next() {
		var msg Message
		var timestampUnix int64
		var quotedMessageID, quotedSenderJID, quotedText sql.NullString

		err := rows.Scan(
			&msg.ID,
			&msg.ChatJID,
			&msg.SenderJID,
			&msg.Text,
			&timestampUnix,
			&msg.IsFromMe,
			&msg.MessageType,
			&quotedMessageID,
			&quotedSenderJID,
			&quotedText,
		)
		if err != nil {
			return nil, err
		}

		msg.Timestamp = time.Unix(timestampUnix, 0)
		msg.QuotedMessageID = quotedMessageID.String
		msg.QuotedSenderJID = quotedSenderJID.String
		msg.QuotedText = quotedText.String
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// SearchMessagesWithNamesFiltered searches messages with pattern matching and sender filtering.
// It uses GLOB patterns if useGlob is true, otherwise uses LIKE for fuzzy matching.
func (s *MessageStore) SearchMessagesWithNamesFiltered(
	query string,
	useGlob bool,
	senderJID string,
	limit int,
) ([]MessageWithNames, error) {
	var sqlQuery string
	var args []any

	// choose LIKE or GLOB based on pattern type
	if useGlob {
		sqlQuery = `
		SELECT id, chat_jid, sender_jid, sender_push_name, sender_contact_name, chat_name,
		       text, timestamp, is_from_me, message_type,
		       media_file_path, media_file_name, media_file_size, media_mime_type,
		       media_width, media_height, media_duration, media_download_status,
		       media_download_timestamp, media_download_error
		FROM messages_with_names
		WHERE text GLOB ?
		`
		args = append(args, query)
	} else {
		sqlQuery = `
		SELECT id, chat_jid, sender_jid, sender_push_name, sender_contact_name, chat_name,
		       text, timestamp, is_from_me, message_type,
		       media_file_path, media_file_name, media_file_size, media_mime_type,
		       media_width, media_height, media_duration, media_download_status,
		       media_download_timestamp, media_download_error
		FROM messages_with_names
		WHERE text LIKE ?
		`
		args = append(args, "%"+query+"%")
	}

	// add sender filter
	if senderJID != "" {
		sqlQuery += " AND sender_jid = ?"
		args = append(args, senderJID)
	}

	sqlQuery += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMessagesWithNames(rows)
}

// SearchMessagesWithNames searches messages and includes sender names from view
func (s *MessageStore) SearchMessagesWithNames(q string, limit int) ([]MessageWithNames, error) {
	query := `
	SELECT id, chat_jid, sender_jid, sender_push_name, sender_contact_name, chat_name,
	       text, timestamp, is_from_me, message_type,
	       media_file_path, media_file_name, media_file_size, media_mime_type,
	       media_width, media_height, media_duration, media_download_status,
	       media_download_timestamp, media_download_error,
	       quoted_message_id, quoted_sender_jid, quoted_text, quoted_sender_name
	FROM messages_with_names
	WHERE text LIKE ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := s.db.Query(query, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return s.scanMessagesWithNames(rows)
}

// GetChatMessagesWithNames gets chat messages and includes sender names from view
func (s *MessageStore) GetChatMessagesWithNames(chatJID string, limit int, offset int) ([]MessageWithNames, error) {
	query := `
	SELECT id, chat_jid, sender_jid, sender_push_name, sender_contact_name, chat_name,
	       text, timestamp, is_from_me, message_type,
	       media_file_path, media_file_name, media_file_size, media_mime_type,
	       media_width, media_height, media_duration, media_download_status,
	       media_download_timestamp, media_download_error,
	       quoted_message_id, quoted_sender_jid, quoted_text, quoted_sender_name
	FROM messages_with_names
	WHERE chat_jid = ?
	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chatJID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMessagesWithNames(rows)
}

// scanMessagesWithNames converts SQL rows into MessageWithNames objects.
func (s *MessageStore) scanMessagesWithNames(rows *sql.Rows) ([]MessageWithNames, error) {
	var messages []MessageWithNames

	for rows.Next() {
		var msg MessageWithNames
		var timestampUnix int64

		// media metadata fields (nullable)
		var mediaFilePath, mediaFileName, mediaMimeType sql.NullString
		var mediaFileSize sql.NullInt64
		var mediaWidth, mediaHeight, mediaDuration sql.NullInt64
		var mediaDownloadStatus, mediaDownloadError sql.NullString
		var mediaDownloadTimestamp sql.NullInt64

		// quote-reply parent fields (nullable)
		var quotedMessageID, quotedSenderJID, quotedText, quotedSenderName sql.NullString

		err := rows.Scan(
			&msg.ID,
			&msg.ChatJID,
			&msg.SenderJID,
			&msg.SenderPushName,
			&msg.SenderContactName,
			&msg.ChatName,
			&msg.Text,
			&timestampUnix,
			&msg.IsFromMe,
			&msg.MessageType,
			// media metadata fields
			&mediaFilePath,
			&mediaFileName,
			&mediaFileSize,
			&mediaMimeType,
			&mediaWidth,
			&mediaHeight,
			&mediaDuration,
			&mediaDownloadStatus,
			&mediaDownloadTimestamp,
			&mediaDownloadError,
			// quote-reply fields
			&quotedMessageID,
			&quotedSenderJID,
			&quotedText,
			&quotedSenderName,
		)
		if err != nil {
			return nil, err
		}

		msg.Timestamp = time.Unix(timestampUnix, 0)
		msg.QuotedMessageID = quotedMessageID.String
		msg.QuotedSenderJID = quotedSenderJID.String
		msg.QuotedText = quotedText.String
		msg.QuotedSenderName = quotedSenderName.String

		// populate media metadata if present
		if mediaFileName.Valid && mediaMimeType.Valid {
			meta := &MediaMetadata{
				MessageID:      msg.ID,
				FileName:       mediaFileName.String,
				FileSize:       mediaFileSize.Int64,
				MimeType:       mediaMimeType.String,
				DownloadStatus: "pending",
			}

			if mediaFilePath.Valid {
				meta.FilePath = mediaFilePath.String
			}
			if mediaWidth.Valid {
				w := int(mediaWidth.Int64)
				meta.Width = &w
			}
			if mediaHeight.Valid {
				h := int(mediaHeight.Int64)
				meta.Height = &h
			}
			if mediaDuration.Valid {
				d := int(mediaDuration.Int64)
				meta.Duration = &d
			}
			if mediaDownloadStatus.Valid {
				meta.DownloadStatus = mediaDownloadStatus.String
			}
			if mediaDownloadTimestamp.Valid {
				ts := time.Unix(mediaDownloadTimestamp.Int64, 0)
				meta.DownloadTimestamp = &ts
			}
			if mediaDownloadError.Valid {
				meta.DownloadError = mediaDownloadError.String
			}

			msg.MediaMetadata = meta
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}
