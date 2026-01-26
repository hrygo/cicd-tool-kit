// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner

import "errors"

// Exit codes as per SPEC-CORE-01
const (
	ExitSuccess         = 0   // Analysis completed
	ExitInfraError      = 1   // Infrastructure error (network, config)
	ExitClaudeError     = 2   // Claude error (API quota, overloaded)
	ExitTimeout         = 101 // Execution timed out
	ExitResourceLimit   = 102 // Resource limit exceeded
)

// Errors
var (
	ErrClaudeNotFound     = errors.New("claude binary not found in PATH")
	ErrProcessNotRunning  = errors.New("process is not running")
	ErrProcessAlreadyRun  = errors.New("process has already been started")
	ErrTimeout            = errors.New("execution timed out")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrShutdownTimeout    = errors.New("graceful shutdown timed out")
	ErrNotInitialized     = errors.New("runner not initialized")
	ErrSkillNotFound      = errors.New("skill not found")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrWorkspaceNotGit    = errors.New("workspace is not a git repository")
)

// ClaudeError represents a Claude API error with classification.
type ClaudeError struct {
	Code      string
	Message   string
	Retryable bool
	Fallback  FallbackAction
}

func (e *ClaudeError) Error() string {
	return e.Code + ": " + e.Message
}

// FallbackAction defines the action to take when Claude fails.
type FallbackAction string

const (
	FallbackRetry   FallbackAction = "retry"   // Retry the request
	FallbackSkip    FallbackAction = "skip"    // Skip, don't block CI
	FallbackCache   FallbackAction = "cache"   // Use cached result
	FallbackPartial FallbackAction = "partial" // Return partial results
	FallbackFail    FallbackAction = "fail"    // Block CI
)
