// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package buildctx

import (
	"context"
)

// DiffGetter retrieves git diffs.
// This will be fully implemented in SPEC-CORE-02.
type DiffGetter struct {
	// TODO: Add git client fields
}

// NewDiffGetter creates a new diff getter.
func NewDiffGetter() *DiffGetter {
	return &DiffGetter{}
}

// GetDiff retrieves the diff for a commit or PR.
func (dg *DiffGetter) GetDiff(ctx context.Context, ref string) (string, error) {
	// TODO: Implement per SPEC-CORE-02
	return "", nil
}

// GetFileDiff retrieves the diff for a specific file.
func (dg *DiffGetter) GetFileDiff(ctx context.Context, ref, path string) (string, error) {
	// TODO: Implement per SPEC-CORE-02
	return "", nil
}

// GetChangedFiles returns a list of changed files.
func (dg *DiffGetter) GetChangedFiles(ctx context.Context, ref string) ([]string, error) {
	// TODO: Implement per SPEC-CORE-02
	return nil, nil
}
