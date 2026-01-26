// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// ClassifyError classifies an error and returns a ClaudeError with fallback action.
func ClassifyError(err error) *ClaudeError {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Timeout errors - retryable
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
		return &ClaudeError{
			Code:      "TIMEOUT",
			Message:   msg,
			Retryable: true,
			Fallback:  FallbackRetry,
		}
	}

	// Rate limiting - retryable with backoff
	if strings.Contains(msg, "rate limit") || strings.Contains(msg, "429") {
		return &ClaudeError{
			Code:      "RATE_LIMITED",
			Message:   msg,
			Retryable: true,
			Fallback:  FallbackRetry,
		}
	}

	// Authentication errors - not retryable, skip
	if strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "authentication") {
		return &ClaudeError{
			Code:      "UNAUTHORIZED",
			Message:   msg,
			Retryable: false,
			Fallback:  FallbackSkip,
		}
	}

	// Server errors - retryable
	if strings.Contains(msg, "500") || strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") || strings.Contains(msg, "504") {
		return &ClaudeError{
			Code:      "SERVER_ERROR",
			Message:   msg,
			Retryable: true,
			Fallback:  FallbackRetry,
		}
	}

	// Content too large - not retryable, partial result
	if strings.Contains(msg, "too large") || strings.Contains(msg, "exceeds limit") ||
		strings.Contains(msg, "context length") {
		return &ClaudeError{
			Code:      "CONTENT_TOO_LARGE",
			Message:   msg,
			Retryable: false,
			Fallback:  FallbackPartial,
		}
	}

	// Claude binary not found - not retryable, skip
	if err == ErrClaudeNotFound {
		return &ClaudeError{
			Code:      "CLAUDE_NOT_FOUND",
			Message:   msg,
			Retryable: false,
			Fallback:  FallbackSkip,
		}
	}

	// Default: retryable
	return &ClaudeError{
		Code:      "UNKNOWN",
		Message:   msg,
		Retryable: true,
		Fallback:  FallbackRetry,
	}
}

// FallbackResult represents the result of a fallback action.
type FallbackResult struct {
	Skipped    bool
	SkipReason string
	Output     string
	Cached     bool
	Partial    bool
}

// FallbackHandler handles fallback actions when Claude fails.
type FallbackHandler struct {
	mu sync.RWMutex

	// Metrics
	totalFallbacks int64
	byAction       map[FallbackAction]int64
	byErrorCode    map[string]int64
}

// NewFallbackHandler creates a new fallback handler.
func NewFallbackHandler() *FallbackHandler {
	return &FallbackHandler{
		byAction:    make(map[FallbackAction]int64),
		byErrorCode: make(map[string]int64),
	}
}

// Handle executes the appropriate fallback action.
func (fh *FallbackHandler) Handle(ctx context.Context, err *ClaudeError, req *RunRequest) *FallbackResult {
	if err == nil {
		return nil
	}

	atomic.AddInt64(&fh.totalFallbacks, 1)

	fh.mu.Lock()
	fh.byAction[err.Fallback]++
	fh.byErrorCode[err.Code]++
	fh.mu.Unlock()

	switch err.Fallback {
	case FallbackRetry:
		// Handled by RetryExecutor
		return nil

	case FallbackSkip:
		return &FallbackResult{
			Skipped:    true,
			SkipReason: fmt.Sprintf("Claude API unavailable: %s - %s", err.Code, err.Message),
			Output:     "Analysis skipped due to API unavailability",
		}

	case FallbackCache:
		// TODO: Implement cache lookup
		return &FallbackResult{
			Skipped:    true,
			SkipReason: "Cache miss during fallback",
			Output:     "No cached result available",
		}

	case FallbackPartial:
		return &FallbackResult{
			Skipped:    false,
			Partial:    true,
			SkipReason: fmt.Sprintf("Returning partial results: %s", err.Message),
			Output:     "Partial analysis completed",
		}

	case FallbackFail:
		// Return nil to indicate failure should propagate
		return nil
	}

	return nil
}

// Metrics returns the fallback metrics (safe copy).
func (fh *FallbackHandler) Metrics() *FallbackMetrics {
	fh.mu.RLock()
	byAction := make(map[FallbackAction]int64, len(fh.byAction))
	for k, v := range fh.byAction {
		byAction[k] = v
	}
	byErrorCode := make(map[string]int64, len(fh.byErrorCode))
	for k, v := range fh.byErrorCode {
		byErrorCode[k] = v
	}
	fh.mu.RUnlock()

	return &FallbackMetrics{
		TotalFallbacks: atomic.LoadInt64(&fh.totalFallbacks),
		ByAction:       byAction,
		ByErrorCode:    byErrorCode,
	}
}

// FallbackMetrics contains fallback statistics.
type FallbackMetrics struct {
	TotalFallbacks int64
	ByAction       map[FallbackAction]int64
	ByErrorCode    map[string]int64
	FallbackRate   float64
}
