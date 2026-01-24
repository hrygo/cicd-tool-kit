// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package skill

import (
	"context"
)

// Executor executes skills.
// This will be fully implemented in SPEC-SKILL-01.
type Executor struct {
	// TODO: Add executor fields
	registry *Registry
}

// NewExecutor creates a new skill executor.
func NewExecutor(reg *Registry) *Executor {
	return &Executor{
		registry: reg,
	}
}

// Execute executes a skill by name.
func (e *Executor) Execute(ctx context.Context, name string, input string) (string, error) {
	// TODO: Implement per SPEC-SKILL-01
	skill, ok := e.registry.Get(name)
	if !ok {
		return "", ErrSkillNotFound
	}
	return skill.Execute(ctx, input)
}

// ExecuteBatch executes multiple skills.
func (e *Executor) ExecuteBatch(ctx context.Context, names []string, input string) (map[string]string, error) {
	// TODO: Implement per SPEC-SKILL-01
	return nil, nil
}

var (
	// ErrSkillNotFound is returned when a skill is not found.
	ErrSkillNotFound = &SkillError{Code: "NOT_FOUND"}
)

// SkillError represents a skill execution error.
type SkillError struct {
	Code string
	Err  error
}

func (e *SkillError) Error() string {
	return e.Code + ": " + e.Err.Error()
}
