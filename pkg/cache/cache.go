// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package cache provides caching for analysis results.
package cache

import (
	"context"
	"time"
)

// Cache is the cache interface.
// This will be fully implemented in SPEC-PERF-01.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// Entry represents a cache entry.
type Entry struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}
