// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package claude

import (
	"context"
)

// Process manages the Claude subprocess.
// This will be fully implemented in SPEC-CORE-01.
type Process struct {
	binary string   //nolint:unused // TODO: Add process management fields (SPEC-CORE-01)
	args   []string //nolint:unused // TODO: Add process management fields (SPEC-CORE-01)
}

// NewProcess creates a new Claude process.
func NewProcess() *Process {
	return &Process{}
}

// Start starts the Claude process.
func (p *Process) Start(ctx context.Context) error {
	// TODO: Implement per SPEC-CORE-01
	return nil
}

// Stop stops the Claude process.
func (p *Process) Stop(ctx context.Context) error {
	// TODO: Implement per SPEC-CORE-01
	return nil
}

// IsRunning checks if Claude is running.
func (p *Process) IsRunning() bool {
	// TODO: Implement per SPEC-CORE-01
	return false
}

// Write writes to Claude's stdin.
func (p *Process) Write(data []byte) error {
	// TODO: Implement per SPEC-CORE-01
	return nil
}

// Read reads from Claude's stdout.
func (p *Process) Read() ([]byte, error) {
	// TODO: Implement per SPEC-CORE-01
	return nil, nil
}
