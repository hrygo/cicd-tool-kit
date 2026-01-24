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
