// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cicd-ai-toolkit/pkg/runner"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedCode  string
		expectedRetry bool
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedCode:  "",
			expectedRetry: false,
		},
		{
			name:          "timeout error",
			err:           errors.New("connection timeout"),
			expectedCode:  "TIMEOUT",
			expectedRetry: true,
		},
		{
			name:          "deadline exceeded",
			err:           errors.New("deadline exceeded"),
			expectedCode:  "TIMEOUT",
			expectedRetry: true,
		},
		{
			name:          "rate limit",
			err:           errors.New("rate limit exceeded"),
			expectedCode:  "RATE_LIMITED",
			expectedRetry: true,
		},
		{
			name:          "429 error",
			err:           errors.New("API returned 429"),
			expectedCode:  "RATE_LIMITED",
			expectedRetry: true,
		},
		{
			name:          "401 unauthorized",
			err:           errors.New("401 unauthorized"),
			expectedCode:  "UNAUTHORIZED",
			expectedRetry: false,
		},
		{
			name:          "authentication error",
			err:           errors.New("authentication failed"),
			expectedCode:  "UNAUTHORIZED",
			expectedRetry: false,
		},
		{
			name:          "500 server error",
			err:           errors.New("500 internal server error"),
			expectedCode:  "SERVER_ERROR",
			expectedRetry: true,
		},
		{
			name:          "502 bad gateway",
			err:           errors.New("502 bad gateway"),
			expectedCode:  "SERVER_ERROR",
			expectedRetry: true,
		},
		{
			name:          "503 unavailable",
			err:           errors.New("503 service unavailable"),
			expectedCode:  "SERVER_ERROR",
			expectedRetry: true,
		},
		{
			name:          "content too large",
			err:           errors.New("content too large"),
			expectedCode:  "CONTENT_TOO_LARGE",
			expectedRetry: false,
		},
		{
			name:          "context length exceeded",
			err:           errors.New("context length exceeded"),
			expectedCode:  "CONTENT_TOO_LARGE",
			expectedRetry: false,
		},
		{
			name:          "claude not found",
			err:           runner.ErrClaudeNotFound,
			expectedCode:  "CLAUDE_NOT_FOUND",
			expectedRetry: false,
		},
		{
			name:          "unknown error",
			err:           errors.New("some random error"),
			expectedCode:  "UNKNOWN",
			expectedRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classified := runner.ClassifyError(tt.err)

			if tt.err == nil {
				if classified != nil {
					t.Error("expected nil for nil error")
				}
				return
			}

			if classified == nil {
				t.Fatal("expected non-nil classification")
			}

			if classified.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, classified.Code)
			}

			if classified.Retryable != tt.expectedRetry {
				t.Errorf("expected retryable %v, got %v", tt.expectedRetry, classified.Retryable)
			}
		})
	}
}

func TestFallbackHandler(t *testing.T) {
	fh := runner.NewFallbackHandler()

	if fh == nil {
		t.Fatal("NewFallbackHandler() returned nil")
	}
}

func TestFallbackHandlerSkip(t *testing.T) {
	fh := runner.NewFallbackHandler()
	ctx := context.Background()

	err := &runner.ClaudeError{
		Code:      "UNAUTHORIZED",
		Message:   "test error",
		Retryable: false,
		Fallback:  runner.FallbackSkip,
	}

	result := fh.Handle(ctx, err, &runner.RunRequest{SkillName: "test"})

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Skipped {
		t.Error("expected Skipped to be true")
	}
	if result.SkipReason == "" {
		t.Error("expected SkipReason to be set")
	}
}

func TestFallbackHandlerRetry(t *testing.T) {
	fh := runner.NewFallbackHandler()
	ctx := context.Background()

	err := &runner.ClaudeError{
		Code:      "TIMEOUT",
		Message:   "test error",
		Retryable: true,
		Fallback:  runner.FallbackRetry,
	}

	result := fh.Handle(ctx, err, &runner.RunRequest{SkillName: "test"})

	// Retry fallback returns nil (handled by RetryExecutor)
	if result != nil {
		t.Error("expected nil result for retry fallback")
	}
}

func TestFallbackHandlerPartial(t *testing.T) {
	fh := runner.NewFallbackHandler()
	ctx := context.Background()

	err := &runner.ClaudeError{
		Code:      "CONTENT_TOO_LARGE",
		Message:   "test error",
		Retryable: false,
		Fallback:  runner.FallbackPartial,
	}

	result := fh.Handle(ctx, err, &runner.RunRequest{SkillName: "test"})

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Skipped {
		t.Error("expected Skipped to be false for partial")
	}
	if !result.Partial {
		t.Error("expected Partial to be true")
	}
}

func TestFallbackMetrics(t *testing.T) {
	fh := runner.NewFallbackHandler()
	ctx := context.Background()

	// Trigger some fallbacks
	fh.Handle(ctx, &runner.ClaudeError{
		Code:     "UNAUTHORIZED",
		Fallback: runner.FallbackSkip,
	}, &runner.RunRequest{})

	fh.Handle(ctx, &runner.ClaudeError{
		Code:     "CONTENT_TOO_LARGE",
		Fallback: runner.FallbackPartial,
	}, &runner.RunRequest{})

	metrics := fh.Metrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}

	if metrics.TotalFallbacks != 2 {
		t.Errorf("expected 2 total fallbacks, got %d", metrics.TotalFallbacks)
	}
}

func TestClaudeErrorError(t *testing.T) {
	err := &runner.ClaudeError{
		Code:    "TEST",
		Message: "test message",
	}

	expected := "TEST: test message"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
