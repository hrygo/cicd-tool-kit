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
		Metadata: skill.Metadata{
			Name:    "test",
			Version: "1.0.0",
		},
	}
	if err := reg.Register(s); err != nil {
		t.Fatal(err)
	}

	got := reg.Get("test")
	if got == nil {
		t.Fatal("Registry.Get() returned not found")
	}
	if got.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", got.Name)
	}
}

func TestRegistryList(t *testing.T) {
	reg := skill.NewRegistry()

	s1 := &skill.Skill{
		Metadata: skill.Metadata{
			Name:    "test1",
			Version: "1.0.0",
		},
	}
	s2 := &skill.Skill{
		Metadata: skill.Metadata{
			Name:    "test2",
			Version: "1.0.0",
		},
	}
	reg.Register(s1)
	reg.Register(s2)

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(list))
	}
}
