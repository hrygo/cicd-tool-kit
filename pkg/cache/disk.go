// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package cache

import (
	"context"
	"time"
)

// DiskCache is a disk-based cache.
// This will be fully implemented in SPEC-PERF-01.
type DiskCache struct {
	// TODO: Add disk cache fields
	path string
}

// NewDiskCache creates a new disk cache.
func NewDiskCache(path string) *DiskCache {
	return &DiskCache{
		path: path,
	}
}

// Get retrieves a value from disk cache.
func (d *DiskCache) Get(ctx context.Context, key string) ([]byte, error) {
	// TODO: Implement per SPEC-PERF-01
	return nil, ErrCacheMiss
}

// Set stores a value in disk cache.
func (d *DiskCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// TODO: Implement per SPEC-PERF-01
	return nil
}

// Delete removes a value from disk cache.
func (d *DiskCache) Delete(ctx context.Context, key string) error {
	// TODO: Implement per SPEC-PERF-01
	return nil
}

// Clear removes all entries from disk cache.
func (d *DiskCache) Clear(ctx context.Context) error {
	// TODO: Implement per SPEC-PERF-01
	return nil
}
