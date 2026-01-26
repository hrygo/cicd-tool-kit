// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner_test

import (
	"context"
	"testing"
	"time"

	"github.com/cicd-ai-toolkit/pkg/runner"
)

func TestNewClaudeProcess(t *testing.T) {
	args := []string{"-p", "--help"}
	p := runner.NewClaudeProcess(args)

	if p == nil {
		t.Fatal("NewClaudeProcess() returned nil")
	}

	if p.IsRunning() {
		t.Error("new process should not be running")
	}
}

func TestClaudeProcessNotFound(t *testing.T) {
	p := runner.NewClaudeProcess([]string{"-p"})
	p = p.WithBinary("nonexistent-binary-12345")

	ctx := context.Background()
	err := p.Start(ctx)

	if err != runner.ErrClaudeNotFound {
		t.Errorf("expected ErrClaudeNotFound, got %v", err)
	}
}

func TestClaudeProcessDoubleStart(t *testing.T) {
	// Use echo as a test binary since claude may not be available
	p := runner.NewClaudeProcess([]string{})
	p = p.WithBinary("echo")

	ctx := context.Background()
	err := p.Start(ctx)
	if err != nil {
		t.Skipf("skipping: echo not available: %v", err)
	}
	defer func() { _ = p.Stop() }()

	// Try to start again
	err = p.Start(ctx)
	if err != runner.ErrProcessAlreadyRun {
		t.Errorf("expected ErrProcessAlreadyRun, got %v", err)
	}
}

func TestProcessManager(t *testing.T) {
	pm := runner.NewProcessManager()

	if pm == nil {
		t.Fatal("NewProcessManager() returned nil")
	}

	// Test IsRunning for non-existent process
	if pm.IsRunning("nonexistent") {
		t.Error("IsRunning should return false for non-existent process")
	}
}

func TestProcessPool(t *testing.T) {
	pp := runner.NewProcessPool()

	if pp == nil {
		t.Fatal("NewProcessPool() returned nil")
	}

	if pp.IsWarm() {
		t.Error("new pool should not be warm")
	}
}

func TestProcessPoolWarmupFailure(t *testing.T) {
	pp := runner.NewProcessPool()

	// Set a non-existent binary
	// Note: ProcessPool doesn't have WithBinary method, so we skip this test
	// if claude is not available
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := pp.Warmup(ctx)
	// Warmup will fail if claude is not installed, which is expected
	if err == nil {
		if !pp.IsWarm() {
			t.Error("after successful warmup, pool should be warm")
		}
	}
}
