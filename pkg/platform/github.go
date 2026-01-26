// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package platform

import (
	"context"
)

// GitHub is the GitHub platform adapter.
// This will be fully implemented in SPEC-PLAT-01.
type GitHub struct {
	// TODO: Add client and configuration fields
}

// NewGitHub creates a new GitHub adapter.
func NewGitHub() *GitHub {
	return &GitHub{}
}

// Name returns the platform name.
func (g *GitHub) Name() string {
	return "github"
}

// GetPullRequest retrieves a PR from GitHub.
func (g *GitHub) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	// TODO: Implement per SPEC-PLAT-01
	return nil, nil
}

// PostComment posts a comment on a GitHub PR.
func (g *GitHub) PostComment(ctx context.Context, number int, body string) error {
	// TODO: Implement per SPEC-PLAT-01
	return nil
}

// GetDiff retrieves the diff for a GitHub PR.
func (g *GitHub) GetDiff(ctx context.Context, number int) (string, error) {
	// TODO: Implement per SPEC-PLAT-01
	return "", nil
}

// GetEvent returns the current GitHub event.
func (g *GitHub) GetEvent(ctx context.Context) (*Event, error) {
	// TODO: Implement per SPEC-PLAT-01
	return nil, nil
}

// GetFileContent retrieves a file from the GitHub repository.
func (g *GitHub) GetFileContent(ctx context.Context, path, ref string) (string, error) {
	// TODO: Implement per SPEC-PLAT-01
	return "", nil
}

// ListFiles lists files in a directory on GitHub.
func (g *GitHub) ListFiles(ctx context.Context, path, ref string) ([]string, error) {
	// TODO: Implement per SPEC-PLAT-01
	return nil, nil
}

// CreateStatus creates a status check for a commit on GitHub.
func (g *GitHub) CreateStatus(ctx context.Context, sha, state, description, context string) error {
	// TODO: Implement per SPEC-PLAT-01
	return nil
}
