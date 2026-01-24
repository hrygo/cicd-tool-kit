// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package output provides result formatting and reporting.
package output

import (
	"context"
)

// Formatter formats analysis results.
// This will be fully implemented in SPEC-CORE-03.
type Formatter struct {
	// TODO: Add formatting configuration
	format string
}

// NewFormatter creates a new formatter.
func NewFormatter() *Formatter {
	return &Formatter{
		format: "markdown",
	}
}

// Format formats a result.
func (f *Formatter) Format(ctx context.Context, result *Result) (string, error) {
	// TODO: Implement per SPEC-CORE-03
	return "", nil
}

// Result represents an analysis result.
type Result struct {
	Skill    string
	Success  bool
	Content  string
	Comments []Comment
}

// Comment represents a single comment.
type Comment struct {
	File     string
	Line     int
	Severity string
	Message  string
}
