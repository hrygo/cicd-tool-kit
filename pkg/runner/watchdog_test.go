// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cicd-ai-toolkit/pkg/runner"
)

func TestDefaultRetryPolicy(t *testing.T) {
	policy := runner.DefaultRetryPolicy()

	if policy.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", policy.MaxRetries)
	}
	if policy.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay 1s, got %v", policy.InitialDelay)
	}
	if policy.MaxDelay != 10*time.Second {
		t.Errorf("expected MaxDelay 10s, got %v", policy.MaxDelay)
	}
	if policy.Multiplier != 2.0 {
		t.Errorf("expected Multiplier 2.0, got %f", policy.Multiplier)
	}
}

func TestRetryExecutorCalculateDelay(t *testing.T) {
	policy := &runner.RetryPolicy{
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
	re := runner.NewRetryExecutor(policy)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 10 * time.Second}, // Capped at MaxDelay
		{6, 10 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		delay := re.CalculateDelay(tt.attempt)
		if delay != tt.expected {
			t.Errorf("CalculateDelay(%d) = %v, want %v", tt.attempt, delay, tt.expected)
		}
	}
}

func TestRetryExecutorSuccess(t *testing.T) {
	policy := &runner.RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond, // Fast for tests
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
	}
	re := runner.NewRetryExecutor(policy)

	callCount := 0
	err := re.Execute(context.Background(), func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryExecutorRetryThenSuccess(t *testing.T) {
	policy := &runner.RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
	}
	re := runner.NewRetryExecutor(policy)

	callCount := 0
	err := re.Execute(context.Background(), func() error {
		callCount++
		if callCount < 3 {
			return errors.New("timeout error") // Retryable
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryExecutorNonRetryableError(t *testing.T) {
	policy := &runner.RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
	}
	re := runner.NewRetryExecutor(policy)

	callCount := 0
	err := re.Execute(context.Background(), func() error {
		callCount++
		return errors.New("401 unauthorized") // Non-retryable
	})

	if err == nil {
		t.Error("expected error for non-retryable error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call for non-retryable error, got %d", callCount)
	}
}

func TestNewWatchdog(t *testing.T) {
	w := runner.NewWatchdog()
	if w == nil {
		t.Fatal("NewWatchdog() returned nil")
	}
	if w.IsRunning() {
		t.Error("new watchdog should not be running")
	}
}

func TestWatchdogTimeout(t *testing.T) {
	w := runner.NewWatchdog().
		WithCheckInterval(10 * time.Millisecond).
		WithTimeout(50 * time.Millisecond)

	ctx := context.Background()
	err := w.Watch(ctx)

	if !errors.Is(err, runner.ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
}

func TestWatchdogContextCancel(t *testing.T) {
	w := runner.NewWatchdog().
		WithCheckInterval(10 * time.Millisecond).
		WithTimeout(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := w.Watch(ctx)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestWatchdogOnTimeout(t *testing.T) {
	called := false
	w := runner.NewWatchdog().
		WithCheckInterval(10 * time.Millisecond).
		WithTimeout(50 * time.Millisecond).
		OnTimeout(func() {
			called = true
		})

	ctx := context.Background()
	_ = w.Watch(ctx)

	if !called {
		t.Error("OnTimeout callback was not called")
	}
}
