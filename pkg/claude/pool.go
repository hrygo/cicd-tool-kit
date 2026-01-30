// Package claude provides session pool for managing persistent Claude CLI sessions
// Based on production best practices from docs/BEST_PRACTICE_CLI_AGENT.md
package claude

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultSessionTTL is the default time-to-live for sessions
	DefaultSessionTTL = 24 * time.Hour
	// DefaultMaxSessions is the default maximum concurrent sessions
	DefaultMaxSessions = 10
	// CleanupInterval is the interval between cleanup runs
	CleanupInterval = 5 * time.Minute
	// MaxRetryBackoff is the maximum backoff time for retries
	MaxRetryBackoff = 30 * time.Second
	// MaxScannerCapacity is the maximum line size for stream parsing
	MaxScannerCapacity = 1024 * 1024 // 1MB
)

// SessionPool manages multiple Claude sessions with explicit ID strategy
// Implements the "Explicit ID Strategy" from docs/BEST_PRACTICE_CLI_AGENT.md section 7.2
// NOTE: SessionPool is safe for concurrent use.
type SessionPool struct {
	sessions map[string]*PooledSession
	mu       sync.RWMutex
	baseDir  string
	ttl      time.Duration
	done     chan struct{}    // Signals goroutines to stop
	cleanupWg sync.WaitGroup  // Waits for cleanup goroutine to finish
	started  sync.Once        // Ensures cleanup goroutine starts only once
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
		TTL:         DefaultSessionTTL,
		MaxSessions: DefaultMaxSessions,
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

	pool := &SessionPool{
		sessions: make(map[string]*PooledSession),
		baseDir:  config.BaseDir,
		ttl:      config.TTL,
		done:     make(chan struct{}),
	}

	// Start cleanup goroutine once
	pool.startCleanup()

	return pool, nil
}

// startCleanup starts the cleanup goroutine using sync.Once
func (p *SessionPool) startCleanup() {
	p.cleanupWg.Add(1)
	go func() {
		defer p.cleanupWg.Done()
		p.cleanupLoop()
	}()
}

// GetOrCreate gets an existing session by ID or creates a new one
// Implements the Explicit ID Strategy:
// - If sessionID exists: resume with --resume flag
// - If sessionID is new: create with --session-id flag
//
// NOTE: This method acquires locks in a specific order to avoid deadlock.
// It always releases the pool lock before acquiring individual session locks.
func (p *SessionPool) GetOrCreate(ctx context.Context, sessionID string) (*PooledSession, error) {
	// First, check if session exists under read lock
	p.mu.RLock()
	pooled, exists := p.sessions[sessionID]
	p.mu.RUnlock()

	if exists && pooled != nil {
		// Acquire session lock outside of pool lock to avoid deadlock
		pooled.Lock.Lock()

		// Double-check under session lock
		if pooled.Active {
			pooled.LastUsed = time.Now()
			pooled.Lock.Unlock()
			return pooled, nil
		}

		// Session exists but inactive, need to create new one
		pooled.Lock.Unlock()
	}

	// Create new session
	session, err := NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	newPooled := &PooledSession{
		ID:        sessionID,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		Session:   session,
		Active:    true,
	}

	// Add to pool under write lock
	p.mu.Lock()
	p.sessions[sessionID] = newPooled
	p.mu.Unlock()

	return newPooled, nil
}

// CreateNew creates a new session with a generated UUID
func (p *SessionPool) CreateNew(ctx context.Context) (*PooledSession, error) {
	sessionID := uuid.New().String()
	return p.GetOrCreate(ctx, sessionID)
}

// Get returns an existing session by ID, or error if not found
// NOTE: Acquires session lock; caller must not hold pool lock when calling.
func (p *SessionPool) Get(sessionID string) (*PooledSession, error) {
	p.mu.RLock()
	pooled, exists := p.sessions[sessionID]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if !pooled.Active {
		return nil, fmt.Errorf("session inactive: %s (created at %v, last used %v)",
			sessionID, pooled.CreatedAt.Format(time.RFC3339), pooled.LastUsed.Format(time.RFC3339))
	}

	pooled.Lock.Lock()
	// Note: Caller is responsible for unlocking via returned *PooledSession
	pooled.LastUsed = time.Now()

	return pooled, nil
}

// Remove removes a session from the pool and closes it
func (p *SessionPool) Remove(sessionID string) error {
	p.mu.Lock()
	pooled, exists := p.sessions[sessionID]
	if !exists {
		p.mu.Unlock()
		return nil
	}
	delete(p.sessions, sessionID)
	p.mu.Unlock()

	// Close session outside of pool lock
	pooled.Lock.Lock()
	pooled.Active = false
	if pooled.Session != nil {
		if err := pooled.Session.Close(); err != nil {
			log.Printf("[WARNING] failed to close session %s: %v", sessionID, err)
		}
	}
	pooled.Lock.Unlock()

	// Clean up session files
	sessionPath := filepath.Join(p.baseDir, sessionID)
	if err := os.RemoveAll(sessionPath); err != nil {
		log.Printf("[WARNING] failed to remove session directory %s: %v", sessionPath, err)
	}

	return nil
}

// cleanupLoop runs periodic cleanup of expired sessions
// This goroutine runs until the pool is closed.
func (p *SessionPool) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.done:
			return // Graceful shutdown
		}
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
				if err := pooled.Session.Close(); err != nil {
					log.Printf("[WARNING] cleanup: failed to close session %s: %v", id, err)
				}
			}
			delete(p.sessions, id)

			// Clean up session files
			sessionPath := filepath.Join(p.baseDir, id)
			if err := os.RemoveAll(sessionPath); err != nil {
				log.Printf("[WARNING] cleanup: failed to remove session directory %s: %v", sessionPath, err)
			}
		}

		pooled.Lock.Unlock()
	}
}

// Close closes all sessions in the pool and stops the cleanup goroutine
func (p *SessionPool) Close() error {
	// Signal cleanup goroutine to stop
	close(p.done)

	// Wait for cleanup goroutine to finish
	p.cleanupWg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for id, pooled := range p.sessions {
		pooled.Lock.Lock()
		pooled.Active = false
		if pooled.Session != nil {
			if err := pooled.Session.Close(); err != nil {
				log.Printf("[ERROR] failed to close session %s during pool shutdown: %v", id, err)
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
		TotalSessions:  len(p.sessions),
		ActiveSessions: active,
		BaseDir:        p.baseDir,
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
			if backoff > MaxRetryBackoff {
				backoff = MaxRetryBackoff
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
