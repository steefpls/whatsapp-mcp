package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// TrustedUser represents a user authorized to trigger @claude.
type TrustedUser struct {
	JID     string
	AddedAt time.Time
}

// TrustedUserStore handles database operations for trusted users.
type TrustedUserStore struct {
	db *sql.DB
}

// NewTrustedUserStore creates a new trusted user store.
func NewTrustedUserStore(db *sql.DB) *TrustedUserStore {
	return &TrustedUserStore{db: db}
}

// AddTrustedUser adds a JID to the trusted users list.
func (s *TrustedUserStore) AddTrustedUser(jid string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO trusted_users (jid, added_at) VALUES (?, ?)`,
		jid, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to add trusted user %s: %w", jid, err)
	}
	return nil
}

// RemoveTrustedUser removes a JID from the trusted users list.
func (s *TrustedUserStore) RemoveTrustedUser(jid string) error {
	result, err := s.db.Exec(`DELETE FROM trusted_users WHERE jid = ?`, jid)
	if err != nil {
		return fmt.Errorf("failed to remove trusted user %s: %w", jid, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("trusted user %s not found", jid)
	}
	return nil
}

// ListTrustedUsers returns all trusted users ordered by when they were added.
func (s *TrustedUserStore) ListTrustedUsers() ([]TrustedUser, error) {
	rows, err := s.db.Query(`SELECT jid, added_at FROM trusted_users ORDER BY added_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list trusted users: %w", err)
	}
	defer rows.Close()

	var users []TrustedUser
	for rows.Next() {
		var jid string
		var addedAt int64
		if err := rows.Scan(&jid, &addedAt); err != nil {
			return nil, fmt.Errorf("failed to scan trusted user: %w", err)
		}
		users = append(users, TrustedUser{
			JID:     jid,
			AddedAt: time.Unix(addedAt, 0),
		})
	}
	return users, rows.Err()
}

// IsTrusted checks if a JID is in the trusted users list.
func (s *TrustedUserStore) IsTrusted(jid string) (bool, error) {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM trusted_users WHERE jid = ? LIMIT 1`, jid).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check trusted user %s: %w", jid, err)
	}
	return true, nil
}
