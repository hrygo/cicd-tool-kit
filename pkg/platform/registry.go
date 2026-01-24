// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package platform

import (
	"sync"
)

// Registry manages platform adapters.
// This will be fully implemented in SPEC-PLAT-01.
type Registry struct {
	mu     sync.RWMutex
	platforms map[string]Platform
}

// NewRegistry creates a new platform registry.
func NewRegistry() *Registry {
	return &Registry{
		platforms: make(map[string]Platform),
	}
}

// Register registers a platform adapter.
func (r *Registry) Register(name string, p Platform) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.platforms[name] = p
}

// Get retrieves a platform by name.
func (r *Registry) Get(name string) (Platform, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.platforms[name]
	return p, ok
}

// Detect auto-detects the current platform.
func (r *Registry) Detect() (Platform, error) {
	// TODO: Implement per SPEC-PLAT-01
	// This will check environment variables and VCS info
	return nil, nil
}
