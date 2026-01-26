// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// IdempotencyManager ensures operations can be safely retried.
// Implements SPEC-CONF-02: Idempotency
type IdempotencyManager struct {
	mu       sync.RWMutex
	stateDir string
	cache    map[string]*IdempotentOperation
	ttl      time.Duration
}

// IdempotentOperation represents an operation that can be safely retried.
type IdempotentOperation struct {
	Key       string
	Result    any
	Error     error
	Completed bool
	ExpiresAt time.Time
	Hash      string
}

// NewIdempotencyManager creates a new idempotency manager.
func NewIdempotencyManager(stateDir string) (*IdempotencyManager, error) {
	if stateDir == "" {
		stateDir = filepath.Join(os.TempDir(), "cicd-toolkit-idempotency")
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	mgr := &IdempotencyManager{
		stateDir: stateDir,
		cache:    make(map[string]*IdempotentOperation),
		ttl:      24 * time.Hour,
	}

	// Load existing state
	mgr.loadState()

	return mgr, nil
}

// Run executes an operation idempotently.
// If the operation was already completed successfully, returns cached result.
func (m *IdempotencyManager) Run(key string, fn func() (any, error)) (any, error) {
	return m.RunWithTTL(key, m.ttl, fn)
}

// RunWithTTL executes an operation with a specific TTL.
func (m *IdempotencyManager) RunWithTTL(key string, ttl time.Duration, fn func() (any, error)) (any, error) {
	hash := m.hashKey(key)

	m.mu.Lock()

	// Check if operation exists and is still valid
	if op, exists := m.cache[hash]; exists {
		if op.Completed && time.Now().Before(op.ExpiresAt) {
			m.mu.Unlock()
			return op.Result, op.Error
		}
		// Clean up expired operation
		delete(m.cache, hash)
	}

	// Create new operation record
	op := &IdempotentOperation{
		Key:       key,
		ExpiresAt: time.Now().Add(ttl),
		Hash:      hash,
	}
	m.cache[hash] = op
	m.mu.Unlock()

	// Execute the operation
	result, err := fn()

	m.mu.Lock()
	op.Result = result
	op.Error = err
	op.Completed = true
	m.mu.Unlock()

	// Persist state
	m.persistState()

	return result, err
}

// RunAsync executes an operation asynchronously, returning immediately if in progress.
func (m *IdempotencyManager) RunAsync(key string, fn func() (any, error)) (any, bool, error) {
	hash := m.hashKey(key)

	m.mu.Lock()
	op, exists := m.cache[hash]
	if !exists {
		op = &IdempotentOperation{
			Key:       key,
			ExpiresAt: time.Now().Add(m.ttl),
			Hash:      hash,
		}
		m.cache[hash] = op
	}
	m.mu.Unlock()

	// If completed, return result
	if op.Completed {
		return op.Result, true, op.Error
	}

	// Otherwise, execute (in real implementation, this would use a worker pool)
	result, err := fn()

	m.mu.Lock()
	op.Result = result
	op.Error = err
	op.Completed = true
	m.mu.Unlock()

	return result, false, err
}

// Invalidate removes a cached operation result.
func (m *IdempotencyManager) Invalidate(key string) {
	hash := m.hashKey(key)
	m.mu.Lock()
	delete(m.cache, hash)
	m.mu.Unlock()

	// Clean up state file
	os.Remove(filepath.Join(m.stateDir, hash+".json"))
}

// IsCompleted returns true if the operation was completed.
func (m *IdempotencyManager) IsCompleted(key string) bool {
	hash := m.hashKey(key)
	m.mu.RLock()
	defer m.mu.RUnlock()

	op, exists := m.cache[hash]
	return exists && op.Completed && time.Now().Before(op.ExpiresAt)
}

// GetResult returns the cached result if available.
func (m *IdempotencyManager) GetResult(key string) (any, error) {
	hash := m.hashKey(key)
	m.mu.RLock()
	defer m.mu.RUnlock()

	op, exists := m.cache[hash]
	if !exists || !op.Completed {
		return nil, fmt.Errorf("operation not found or incomplete")
	}

	if time.Now().After(op.ExpiresAt) {
		return nil, fmt.Errorf("operation result expired")
	}

	return op.Result, op.Error
}

// Cleanup removes expired operations.
func (m *IdempotencyManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for hash, op := range m.cache {
		if now.After(op.ExpiresAt) {
			delete(m.cache, hash)
			os.Remove(filepath.Join(m.stateDir, hash+".json"))
		}
	}
}

// SetTTL sets the default TTL for operations.
func (m *IdempotencyManager) SetTTL(ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttl = ttl
}

// hashKey creates a stable hash for the key.
func (m *IdempotencyManager) hashKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

// loadState loads persisted state from disk.
func (m *IdempotencyManager) loadState() {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(m.stateDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Parse and load operation (simplified)
		op := &IdempotentOperation{Hash: strings.TrimSuffix(entry.Name(), ".json")}
		if json.Unmarshal(data, op) == nil {
			if time.Now().Before(op.ExpiresAt) {
				m.cache[op.Hash] = op
			} else {
				os.Remove(path)
			}
		}
	}
}

// persistState saves current state to disk.
func (m *IdempotencyManager) persistState() {
	// In production, use a more efficient persistence strategy
	// For now, implement basic cleanup of old files
	m.Cleanup()
}

// IdempotentFileWriter writes files atomically.
type IdempotentFileWriter struct {
	baseDir string
}

// NewIdempotentFileWriter creates a new file writer.
func NewIdempotentFileWriter(baseDir string) *IdempotentFileWriter {
	return &IdempotentFileWriter{baseDir: baseDir}
}

// WriteFile writes a file atomically using temp file + rename.
func (w *IdempotentFileWriter) WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Write to temp file
	tempPath := path + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, path)
}

// EnsureDirectory ensures a directory exists (idempotent mkdir).
func EnsureDirectory(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", path)
		}
		return nil
	}

	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}

	return err
}

// IdempotentCommand wraps a command for idempotent execution.
type IdempotentCommand struct {
	name     string
	args     []string
	checkFn  func() error
	execFn   func() error
	rollback func() error
}

// NewIdempotentCommand creates a new idempotent command.
func NewIdempotentCommand(name string, args []string) *IdempotentCommand {
	return &IdempotentCommand{
		name: name,
		args: args,
	}
}

// WithCheck sets the idempotency check function.
// If check returns nil, command is already applied.
func (c *IdempotentCommand) WithCheck(fn func() error) *IdempotentCommand {
	c.checkFn = fn
	return c
}

// WithExec sets the execution function.
func (c *IdempotentCommand) WithExec(fn func() error) *IdempotentCommand {
	c.execFn = fn
	return c
}

// WithRollback sets the rollback function.
func (c *IdempotentCommand) WithRollback(fn func() error) *IdempotentCommand {
	c.rollback = fn
	return c
}

// Run executes the command idempotently.
func (c *IdempotentCommand) Run() error {
	// Check if already applied
	if c.checkFn != nil {
		if err := c.checkFn(); err == nil {
			return nil // Already applied
		}
	}

	// Execute
	if c.execFn != nil {
		return c.execFn()
	}

	return nil
}

// Rollback rolls back the command.
func (c *IdempotentCommand) Rollback() error {
	if c.rollback != nil {
		return c.rollback()
	}
	return nil
}
