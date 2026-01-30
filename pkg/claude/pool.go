// Package claude provides session pool for managing persistent Claude CLI sessions
// Based on production best practices from docs/BEST_PRACTICE_CLI_AGENT.md
package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionPool manages multiple Claude sessions with explicit ID strategy
// Implements the "Explicit ID Strategy" from docs/BEST_PRACTICE_CLI_AGENT.md section 7.2
type SessionPool struct {
	sessions map[string]*PooledSession
	mu       sync.RWMutex
	baseDir  string
	ttl      time.Duration
}

// PooledSession represents a session in the pool with metadata
type PooledSession struct {
	ID        string
	CreatedAt time.Time
	LastUsed  time.Time
	Lock      sync.Mutex
	Session   Session
	Active    bool
}

// PoolConfig contains configuration for the session pool
type PoolConfig struct {
	// BaseDir is the directory for storing session data
	BaseDir string

	// TTL is the time-to-live for idle sessions
	TTL time.Duration

	// MaxSessions is the maximum number of concurrent sessions
	MaxSessions int
}

// DefaultPoolConfig returns sensible defaults for session pool
func DefaultPoolConfig() PoolConfig {
	// Use XDG cache directory or fallback to .cache
	baseDir := os.Getenv("XDG_CACHE_HOME")
	if baseDir == "" {
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, ".cache")
	}
	sessionDir := filepath.Join(baseDir, "cicd-toolkit", "sessions")

	return PoolConfig{
		BaseDir:     sessionDir,
		TTL:         24 * time.Hour, // Default: 24 hours
		MaxSessions: 10,              // Maximum concurrent sessions
	}
}

// NewSessionPool creates a new session pool with the given configuration
func NewSessionPool(config PoolConfig) (*SessionPool, error) {
	if config.BaseDir == "" {
		config = DefaultPoolConfig()
	}

	// Ensure session directory exists
	if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &SessionPool{
		sessions: make(map[string]*PooledSession),
		baseDir:  config.BaseDir,
		ttl:      config.TTL,
	}, nil
}

// GetOrCreate gets an existing session by ID or creates a new one
// Implements the Explicit ID Strategy:
// - If sessionID exists: resume with --resume flag
// - If sessionID is new: create with --session-id flag
func (p *SessionPool) GetOrCreate(ctx context.Context, sessionID string) (*PooledSession, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if session exists
	if pooled, exists := p.sessions[sessionID]; exists {
		pooled.Lock.Lock()
		defer pooled.Lock.Unlock()

		if pooled.Active {
			pooled.LastUsed = time.Now()
			return pooled, nil
		}
	}

	// Create new session
	session, err := NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	pooled := &PooledSession{
		ID:        sessionID,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		Session:   session,
		Active:    true,
	}

	p.sessions[sessionID] = pooled

	// Start cleanup goroutine
	go p.cleanupLoop()

	return pooled, nil
}

// CreateNew creates a new session with a generated UUID
func (p *SessionPool) CreateNew(ctx context.Context) (*PooledSession, error) {
	sessionID := uuid.New().String()
	return p.GetOrCreate(ctx, sessionID)
}

// Get returns an existing session by ID, or error if not found
func (p *SessionPool) Get(sessionID string) (*PooledSession, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pooled, exists := p.sessions[sessionID]
	if !exists || !pooled.Active {
		return nil, fmt.Errorf("session not found or inactive: %s", sessionID)
	}

	pooled.Lock.Lock()
	defer pooled.Lock.Unlock()
	pooled.LastUsed = time.Now()

	return pooled, nil
}

// Remove removes a session from the pool and closes it
func (p *SessionPool) Remove(sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pooled, exists := p.sessions[sessionID]
	if !exists {
		return nil
	}

	pooled.Lock.Lock()
	defer pooled.Lock.Unlock()

	pooled.Active = false
	if pooled.Session != nil {
		_ = pooled.Session.Close()
	}

	delete(p.sessions, sessionID)

	// Clean up session files
	sessionPath := filepath.Join(p.baseDir, sessionID)
	_ = os.RemoveAll(sessionPath)

	return nil
}

// cleanupLoop runs periodic cleanup of expired sessions
func (p *SessionPool) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanup()
	}
}

// cleanup removes expired sessions from the pool
func (p *SessionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for id, pooled := range p.sessions {
		pooled.Lock.Lock()

		// Check if session is expired
		if now.Sub(pooled.LastUsed) > p.ttl {
			pooled.Active = false
			if pooled.Session != nil {
				_ = pooled.Session.Close()
			}
			delete(p.sessions, id)
		}

		pooled.Lock.Unlock()
	}
}

// Close closes all sessions in the pool
func (p *SessionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for id, pooled := range p.sessions {
		pooled.Lock.Lock()
		pooled.Active = false
		if pooled.Session != nil {
			if err := pooled.Session.Close(); err != nil {
				lastErr = err
			}
		}
		delete(p.sessions, id)
		pooled.Lock.Unlock()
	}

	return lastErr
}

// Stats returns pool statistics
func (p *SessionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	active := 0
	for _, pooled := range p.sessions {
		if pooled.Active {
			active++
		}
	}

	return PoolStats{
		TotalSessions: len(p.sessions),
		ActiveSessions: active,
		BaseDir:       p.baseDir,
	}
}

// PoolStats contains statistics about the session pool
type PoolStats struct {
	TotalSessions  int
	ActiveSessions int
	BaseDir        string
}

// ExecuteWithRetry executes a Claude command with retry logic
// Implements the retry mechanism from docs/BEST_PRACTICE_CLI_AGENT.md
func (p *PooledSession) ExecuteWithRetry(ctx context.Context, opts ExecuteOptions, maxRetries int) (*Output, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check if session is still active
		if !p.Active {
			return nil, fmt.Errorf("session is inactive")
		}

		p.Lock.Lock()
		output, err := p.Session.Execute(ctx, opts)
		p.Lock.Unlock()

		if err == nil {
			p.LastUsed = time.Now()
			return output, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Exponential backoff
		if attempt < maxRetries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return nil, fmt.Errorf("execution failed after %d retries: %w", maxRetries, lastErr)
}

// GetSessionDir returns the directory path for a session's data
func (p *SessionPool) GetSessionDir(sessionID string) string {
	return filepath.Join(p.baseDir, sessionID)
}

// IsSessionActive checks if a session is currently active
func (p *SessionPool) IsSessionActive(sessionID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pooled, exists := p.sessions[sessionID]
	return exists && pooled.Active
}
