package storage

import (
	"fmt"
	"strings"
	"time"
)

// Reaction represents an emoji reaction on a message.
type Reaction struct {
	TargetMessageID string
	ReactorJID      string
	Emoji           string
	Timestamp       time.Time
}

// UpsertReaction inserts or replaces a reaction. A reaction is uniquely
// identified by (target_message_id, reactor_jid); a reactor can only have one
// active reaction per message, and sending a new emoji replaces the old one.
// Older-timestamp updates are ignored so out-of-order delivery doesn't clobber
// a newer state.
func (s *MessageStore) UpsertReaction(r Reaction) error {
	_, err := s.db.Exec(
		`INSERT INTO reactions (target_message_id, reactor_jid, emoji, timestamp)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(target_message_id, reactor_jid) DO UPDATE SET
		     emoji = excluded.emoji,
		     timestamp = excluded.timestamp
		 WHERE excluded.timestamp >= reactions.timestamp`,
		r.TargetMessageID, r.ReactorJID, r.Emoji, r.Timestamp.Unix(),
	)
	return err
}

// DeleteReaction removes a reactor's reaction from a message. Used when a
// reaction event arrives with an empty emoji (reaction removal).
func (s *MessageStore) DeleteReaction(targetMessageID, reactorJID string, timestamp time.Time) error {
	// guard against out-of-order delivery: only delete if the removal is at
	// least as recent as the stored reaction
	_, err := s.db.Exec(
		`DELETE FROM reactions
		 WHERE target_message_id = ? AND reactor_jid = ? AND timestamp <= ?`,
		targetMessageID, reactorJID, timestamp.Unix(),
	)
	return err
}

// GetReactionsForMessages returns all reactions grouped by target message ID.
// Reactions within each message are ordered by timestamp (oldest first).
func (s *MessageStore) GetReactionsForMessages(messageIDs []string) (map[string][]Reaction, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]any, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT target_message_id, reactor_jid, emoji, timestamp
		 FROM reactions
		 WHERE target_message_id IN (%s)
		 ORDER BY timestamp ASC`,
		strings.Join(placeholders, ","),
	)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get reactions: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]Reaction)
	for rows.Next() {
		var r Reaction
		var ts int64
		if err := rows.Scan(&r.TargetMessageID, &r.ReactorJID, &r.Emoji, &ts); err != nil {
			return nil, err
		}
		r.Timestamp = time.Unix(ts, 0)
		result[r.TargetMessageID] = append(result[r.TargetMessageID], r)
	}
	return result, rows.Err()
}
