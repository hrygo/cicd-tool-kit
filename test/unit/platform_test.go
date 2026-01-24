// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package unit_test

import (
	"testing"

	"github.com/cicd-ai-toolkit/pkg/platform"
)

func TestNewGitHub(t *testing.T) {
	p := platform.NewGitHub()
	if p == nil {
		t.Fatal("NewGitHub() returned nil")
	}
	if p.Name() != "github" {
		t.Errorf("Expected name 'github', got '%s'", p.Name())
	}
}

func TestNewGitLab(t *testing.T) {
	p := platform.NewGitLab()
	if p == nil {
		t.Fatal("NewGitLab() returned nil")
	}
	if p.Name() != "gitlab" {
		t.Errorf("Expected name 'gitlab', got '%s'", p.Name())
	}
}

func TestNewGitee(t *testing.T) {
	p := platform.NewGitee()
	if p == nil {
		t.Fatal("NewGitee() returned nil")
	}
	if p.Name() != "gitee" {
		t.Errorf("Expected name 'gitee', got '%s'", p.Name())
	}
}

func TestRegistry(t *testing.T) {
	reg := platform.NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	gh := platform.NewGitHub()
	reg.Register("github", gh)

	p, ok := reg.Get("github")
	if !ok {
		t.Fatal("Registry.Get() returned not found")
	}
	if p.Name() != "github" {
		t.Errorf("Expected name 'github', got '%s'", p.Name())
	}
}
