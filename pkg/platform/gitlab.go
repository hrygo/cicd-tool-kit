// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package platform

import (
	"context"
)

// GitLab is the GitLab platform adapter.
// This will be fully implemented in SPEC-PLAT-03.
type GitLab struct {
	// TODO: Add client and configuration fields
}

// NewGitLab creates a new GitLab adapter.
func NewGitLab() *GitLab {
	return &GitLab{}
}

// Name returns the platform name.
func (g *GitLab) Name() string {
	return "gitlab"
}

// GetPullRequest retrieves an MR from GitLab.
func (g *GitLab) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	// TODO: Implement per SPEC-PLAT-03
	return nil, nil
}

// PostComment posts a comment on a GitLab MR.
func (g *GitLab) PostComment(ctx context.Context, number int, body string) error {
	// TODO: Implement per SPEC-PLAT-03
	return nil
}

// GetDiff retrieves the diff for a GitLab MR.
func (g *GitLab) GetDiff(ctx context.Context, number int) (string, error) {
	// TODO: Implement per SPEC-PLAT-03
	return "", nil
}

// GetEvent returns the current GitLab event.
func (g *GitLab) GetEvent(ctx context.Context) (*Event, error) {
	// TODO: Implement per SPEC-PLAT-03
	return nil, nil
}
