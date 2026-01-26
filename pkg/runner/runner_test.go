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

func TestNewRunner(t *testing.T) {
	r := runner.New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if r.State() != runner.StateUninitialized {
		t.Errorf("expected state %v, got %v", runner.StateUninitialized, r.State())
	}
}

func TestNewRunnerWithOptions(t *testing.T) {
	opts := &runner.Options{
		ConfigPath:      "custom-config.yaml",
		WorkDir:         "/tmp",
		GracefulTimeout: 10 * time.Second,
		Verbose:         true,
	}

	r := runner.NewWithOptions(opts)
	if r == nil {
		t.Fatal("NewWithOptions() returned nil")
	}
}

func TestRunnerBootstrap(t *testing.T) {
	r := runner.NewWithOptions(&runner.Options{
		WorkDir: "/tmp", // Use /tmp to avoid git repo check
	})

	ctx := context.Background()
	err := r.Bootstrap(ctx)
	// Bootstrap should succeed even without a config file (uses defaults)
	if err != nil {
		t.Errorf("Bootstrap() failed: %v", err)
	}

	if r.State() != runner.StateReady {
		t.Errorf("expected state %v, got %v", runner.StateReady, r.State())
	}
}

func TestRunnerBootstrapMetrics(t *testing.T) {
	r := runner.NewWithOptions(&runner.Options{
		WorkDir: "/tmp",
	})

	ctx := context.Background()
	_ = r.Bootstrap(ctx)

	metrics := r.BootstrapMetrics()
	if metrics == nil {
		t.Fatal("BootstrapMetrics() returned nil")
	}

	if metrics.TotalTime == 0 {
		t.Error("TotalTime should not be zero")
	}
}

func TestRunnerShutdown(t *testing.T) {
	r := runner.NewWithOptions(&runner.Options{
		WorkDir: "/tmp",
	})

	ctx := context.Background()
	_ = r.Bootstrap(ctx)

	err := r.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	if r.State() != runner.StateStopped {
		t.Errorf("expected state %v, got %v", runner.StateStopped, r.State())
	}
}

func TestRunnerStateTransitions(t *testing.T) {
	r := runner.New()

	// Initial state
	if r.State() != runner.StateUninitialized {
		t.Errorf("expected initial state %v, got %v", runner.StateUninitialized, r.State())
	}

	ctx := context.Background()

	// After bootstrap
	opts := runner.DefaultOptions()
	opts.WorkDir = "/tmp"
	r = runner.NewWithOptions(opts)
	_ = r.Bootstrap(ctx)
	if r.State() != runner.StateReady {
		t.Errorf("expected state after bootstrap %v, got %v", runner.StateReady, r.State())
	}

	// After shutdown
	_ = r.Shutdown(ctx)
	if r.State() != runner.StateStopped {
		t.Errorf("expected state after shutdown %v, got %v", runner.StateStopped, r.State())
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    runner.State
		expected string
	}{
		{runner.StateUninitialized, "uninitialized"},
		{runner.StateInitializing, "initializing"},
		{runner.StateReady, "ready"},
		{runner.StateRunning, "running"},
		{runner.StateShuttingDown, "shutting_down"},
		{runner.StateStopped, "stopped"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %v, want %v", tt.state, got, tt.expected)
		}
	}
}
