package claude

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// sessionMaxAge is the maximum age of a session before it's expired.
const sessionMaxAge = 24 * time.Hour

// ChatSession tracks a Claude CLI session for a specific chat.
type ChatSession struct {
	SessionID       string    // Claude CLI session UUID
	NewestMsgTime   time.Time // Timestamp of the newest message included in the last prompt
	LastUsed        time.Time // Wall clock time of last invocation
	LastInputTokens int       // Input-token count from the last turn (used to decide compaction)
	JustCompacted   bool      // True if this session was created by a compaction this turn — prevents back-to-back compaction loops
}

// SessionManager manages per-chat Claude sessions and per-chat processing locks.
type SessionManager struct {
	mu        sync.Mutex
	sessions  map[string]*ChatSession // keyed by chat JID

	chatMu    sync.Mutex
	chatLocks map[string]*sync.Mutex // per-chat processing mutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:  make(map[string]*ChatSession),
		chatLocks: make(map[string]*sync.Mutex),
	}
}

// GetChatLock returns (or lazily creates) the per-chat mutex for serializing
// @claude processing. Goroutines for the same chat block on this lock,
// naturally forming a queue.
func (sm *SessionManager) GetChatLock(chatJID string) *sync.Mutex {
	sm.chatMu.Lock()
	defer sm.chatMu.Unlock()

	if lock, ok := sm.chatLocks[chatJID]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	sm.chatLocks[chatJID] = lock
	return lock
}

// GetSession returns the current session for a chat, or nil if none exists
// or if the session has expired.
func (sm *SessionManager) GetSession(chatJID string) *ChatSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[chatJID]
	if !ok {
		return nil
	}

	// expire old sessions
	if time.Since(session.LastUsed) > sessionMaxAge {
		delete(sm.sessions, chatJID)
		return nil
	}

	return session
}

// SetSession stores or updates the session for a chat.
func (sm *SessionManager) SetSession(chatJID string, session *ChatSession) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[chatJID] = session
}

// ClearSession removes the session for a chat.
func (sm *SessionManager) ClearSession(chatJID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, chatJID)
}

// NewSessionID generates a new UUID for a Claude CLI session.
func NewSessionID() string {
	return uuid.New().String()
}
