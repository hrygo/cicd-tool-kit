// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner

// ProcessManager manages Claude subprocess lifecycle.
// This will be fully implemented in SPEC-CORE-01.
type ProcessManager struct {
	// TODO: Add fields as per SPEC-CORE-01
}

// NewProcessManager creates a new process manager.
func NewProcessManager() *ProcessManager {
	return &ProcessManager{}
}

// Start starts the Claude process.
func (pm *ProcessManager) Start() error {
	// TODO: Implement per SPEC-CORE-01
	return nil
}

// Stop stops the Claude process.
func (pm *ProcessManager) Stop() error {
	// TODO: Implement per SPEC-CORE-01
	return nil
}

// IsRunning checks if the process is running.
func (pm *ProcessManager) IsRunning() bool {
	// TODO: Implement per SPEC-CORE-01
	return false
}
