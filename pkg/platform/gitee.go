// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package platform

import (
	"context"
)

// Gitee is the Gitee platform adapter.
// This will be fully implemented in SPEC-PLAT-06.
type Gitee struct {
	// TODO: Add client and configuration fields
}

// NewGitee creates a new Gitee adapter.
func NewGitee() *Gitee {
	return &Gitee{}
}

// Name returns the platform name.
func (g *Gitee) Name() string {
	return "gitee"
}

// GetPullRequest retrieves a PR from Gitee.
func (g *Gitee) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	// TODO: Implement per SPEC-PLAT-06
	return nil, nil
}

// PostComment posts a comment on a Gitee PR.
func (g *Gitee) PostComment(ctx context.Context, number int, body string) error {
	// TODO: Implement per SPEC-PLAT-06
	return nil
}

// GetDiff retrieves the diff for a Gitee PR.
func (g *Gitee) GetDiff(ctx context.Context, number int) (string, error) {
	// TODO: Implement per SPEC-PLAT-06
	return "", nil
}

// GetEvent returns the current Gitee event.
func (g *Gitee) GetEvent(ctx context.Context) (*Event, error) {
	// TODO: Implement per SPEC-PLAT-06
	return nil, nil
}
