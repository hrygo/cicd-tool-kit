// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package platform provides CI/CD platform abstractions.
package platform

import (
	"context"
)

// Platform represents a CI/CD platform (GitHub, GitLab, Gitee, Jenkins, etc.).
// This will be fully implemented in SPEC-PLAT-01.
type Platform interface {
	// Name returns the platform name.
	Name() string

	// GetPullRequest retrieves a pull request by number.
	GetPullRequest(ctx context.Context, number int) (*PullRequest, error)

	// PostComment posts a comment on a pull request.
	PostComment(ctx context.Context, number int, body string) error

	// GetDiff retrieves the diff for a pull request.
	GetDiff(ctx context.Context, number int) (string, error)

	// GetEvent returns the current CI/CD event.
	GetEvent(ctx context.Context) (*Event, error)
}

// PullRequest represents a pull/merge request.
type PullRequest struct {
	Number    int
	Title     string
	Body      string
	Author    string
	Source    string
	Target    string
	Labels    []string
	Milestone string
}

// Event represents a CI/CD event (push, PR, etc.).
type Event struct {
	Type      string
	Actor     string
	Action    string
	PRNumber  int
	CommitSHA string
	Ref       string
}
