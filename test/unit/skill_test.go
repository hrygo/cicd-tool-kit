// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package unit_test

import (
	"testing"

	"github.com/cicd-ai-toolkit/pkg/skill"
)

func TestNewRegistry(t *testing.T) {
	reg := skill.NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestRegistryRegister(t *testing.T) {
	reg := skill.NewRegistry()

	s := &skill.Skill{
		Name:    "test",
		Version: "1.0.0",
	}
	reg.Register(s)

	got, ok := reg.Get("test")
	if !ok {
		t.Fatal("Registry.Get() returned not found")
	}
	if got.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", got.Name)
	}
}

func TestRegistryList(t *testing.T) {
	reg := skill.NewRegistry()

	s1 := &skill.Skill{Name: "test1"}
	s2 := &skill.Skill{Name: "test2"}
	reg.Register(s1)
	reg.Register(s2)

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(list))
	}
}
