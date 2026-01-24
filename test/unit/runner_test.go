// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package unit_test

import (
	"context"
	"testing"

	"github.com/cicd-ai-toolkit/pkg/runner"
)

func TestNewRunner(t *testing.T) {
	r := runner.New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
}

func TestRunnerBootstrap(t *testing.T) {
	r := runner.New()
	err := r.Bootstrap(context.Background())
	if err != nil {
		t.Errorf("Bootstrap() failed: %v", err)
	}
}

func TestRunnerShutdown(t *testing.T) {
	r := runner.New()
	_ = r.Bootstrap(context.Background())
	err := r.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}
}
