// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package claude

import (
	"context"
)

// StreamHandler handles streaming output from Claude.
// This will be fully implemented in SPEC-CORE-01.
type StreamHandler struct {
	// TODO: Add streaming fields
}

// NewStreamHandler creates a new stream handler.
func NewStreamHandler() *StreamHandler {
	return &StreamHandler{}
}

// Handle processes streaming chunks.
func (s *StreamHandler) Handle(ctx context.Context, chunk []byte) error {
	// TODO: Implement per SPEC-CORE-01
	return nil
}

// Collect collects all chunks into a single response.
func (s *StreamHandler) Collect(ctx context.Context) (*Response, error) {
	// TODO: Implement per SPEC-CORE-01
	return nil, nil
}
