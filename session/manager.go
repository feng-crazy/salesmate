package session

import (
	"fmt"
	"sync"
	"time"
)

// Session represents a conversation session
type Session struct {
	Key         string                 `json:"key"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Data        map[string]interface{} `json:"data"`
	Messages    []Message              `json:"messages"`
}

// Message represents a message in a session
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`      // "user", "assistant", "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionManager manages conversation sessions
type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
	baseDir  string
}

// NewSessionManager creates a new session manager
func NewSessionManager(baseDir string) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		baseDir:  baseDir,
	}
}

// GetOrCreateSession gets an existing session or creates a new one
func (sm *SessionManager) GetOrCreateSession(sessionKey string) *Session {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionKey]
	if !exists {
		session = &Session{
			Key:       sessionKey,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Data:      make(map[string]interface{}),
			Messages:  make([]Message, 0),
		}
		sm.sessions[sessionKey] = session
	}

	return session
}

// GetSession gets a session by key
func (sm *SessionManager) GetSession(sessionKey string) (*Session, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionKey]
	return session, exists
}

// SaveMessage saves a message to a session
func (sm *SessionManager) SaveMessage(sessionKey, role, content string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionKey]
	if !exists {
		return fmt.Errorf("session %s not found", sessionKey)
	}

	message := Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, message)
	session.UpdatedAt = time.Now()

	return nil
}

// GetMessageHistory gets message history for a session
func (sm *SessionManager) GetMessageHistory(sessionKey string, limit int) ([]Message, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionKey]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionKey)
	}

	startIdx := 0
	if len(session.Messages) > limit {
		startIdx = len(session.Messages) - limit
	}

	return session.Messages[startIdx:], nil
}

// ListSessions lists all active sessions
func (sm *SessionManager) ListSessions() []map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var sessions []map[string]interface{}
	for _, session := range sm.sessions {
		sessions = append(sessions, map[string]interface{}{
			"key":        session.Key,
			"created_at": session.CreatedAt,
			"updated_at": session.UpdatedAt,
			"message_count": len(session.Messages),
		})
	}

	return sessions
}

// ClearSession clears a session's messages
func (sm *SessionManager) ClearSession(sessionKey string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionKey]
	if !exists {
		return fmt.Errorf("session %s not found", sessionKey)
	}

	session.Messages = make([]Message, 0)
	session.UpdatedAt = time.Now()

	return nil
}

// UpdateSessionData updates session data
func (sm *SessionManager) UpdateSessionData(sessionKey string, data map[string]interface{}) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionKey]
	if !exists {
		return fmt.Errorf("session %s not found", sessionKey)
	}

	for k, v := range data {
		session.Data[k] = v
	}
	session.UpdatedAt = time.Now()

	return nil
}

// GetData retrieves session data
func (sm *SessionManager) GetData(sessionKey string) (map[string]interface{}, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionKey]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionKey)
	}

	// Create a copy to avoid race conditions
	dataCopy := make(map[string]interface{})
	for k, v := range session.Data {
		dataCopy[k] = v
	}

	return dataCopy, nil
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(sessionKey string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	_, exists := sm.sessions[sessionKey]
	if !exists {
		return fmt.Errorf("session %s not found", sessionKey)
	}

	delete(sm.sessions, sessionKey)
	return nil
}