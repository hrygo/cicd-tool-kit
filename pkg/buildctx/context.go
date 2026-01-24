// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package buildctx provides context building for CI/CD analysis.
package buildctx

import (
	"context"
)

// Context represents the CI/CD execution context.
// This will be fully implemented in SPEC-CORE-02.
type Context struct {
	// Platform info
	Platform string
	Event    *Event

	// Git info
	Branch    string
	CommitSHA string
	CommitMsg string
	Author    string

	// PR/MR info (if applicable)
	PRNumber int
	PRTitle  string
	PRBody   string
}

// Event represents a CI/CD event.
type Event struct {
	Type    string
	Actor   string
	Action  string
	Payload map[string]any
}

// NewContext creates a new build context.
func NewContext() *Context {
	return &Context{}
}

// Build populates the context from the environment.
func (c *Context) Build(ctx context.Context) error {
	// TODO: Implement per SPEC-CORE-02
	return nil
}
