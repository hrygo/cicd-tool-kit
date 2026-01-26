// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package platform

import (
	"context"
)

// Jenkins is the Jenkins platform adapter.
// This will be fully implemented in SPEC-PLAT-04.
type Jenkins struct {
	// TODO: Add client and configuration fields
}

// NewJenkins creates a new Jenkins adapter.
func NewJenkins() *Jenkins {
	return &Jenkins{}
}

// Name returns the platform name.
func (j *Jenkins) Name() string {
	return "jenkins"
}

// GetPullRequest retrieves PR info for Jenkins.
func (j *Jenkins) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	// TODO: Implement per SPEC-PLAT-04
	return nil, nil
}

// PostComment posts a comment (Jenkins uses build log/comments).
func (j *Jenkins) PostComment(ctx context.Context, number int, body string) error {
	// TODO: Implement per SPEC-PLAT-04
	return nil
}

// GetDiff retrieves the diff for Jenkins builds.
func (j *Jenkins) GetDiff(ctx context.Context, number int) (string, error) {
	// TODO: Implement per SPEC-PLAT-04
	return "", nil
}

// GetEvent returns the current Jenkins event.
func (j *Jenkins) GetEvent(ctx context.Context) (*Event, error) {
	// TODO: Implement per SPEC-PLAT-04
	return nil, nil
}

// GetFileContent retrieves a file from the Jenkins workspace.
func (j *Jenkins) GetFileContent(ctx context.Context, path, ref string) (string, error) {
	// TODO: Implement per SPEC-PLAT-04
	return "", nil
}

// ListFiles lists files in a directory in the Jenkins workspace.
func (j *Jenkins) ListFiles(ctx context.Context, path, ref string) ([]string, error) {
	// TODO: Implement per SPEC-PLAT-04
	return nil, nil
}

// CreateStatus creates a status check for a Jenkins build.
func (j *Jenkins) CreateStatus(ctx context.Context, sha, state, description, context string) error {
	// TODO: Implement per SPEC-PLAT-04
	return nil
}
