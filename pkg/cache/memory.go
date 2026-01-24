// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache is an in-memory cache.
// This will be fully implemented in SPEC-PERF-01.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*Entry
}

// NewMemoryCache creates a new memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]*Entry),
	}
}

// Get retrieves a value from cache.
func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.items[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, ErrCacheMiss
	}
	return entry.Value, nil
}

// Set stores a value in cache.
func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = &Entry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Delete removes a value from cache.
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

// Clear removes all entries from cache.
func (m *MemoryCache) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]*Entry)
	return nil
}

var ErrCacheMiss = &CacheError{Code: "CACHE_MISS"}
