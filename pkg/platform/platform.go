// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package platform provides CI/CD platform abstractions.
package platform

import (
	"context"
	"io"
	"os"
	"strings"
)

// Platform represents a CI/CD platform (GitHub, GitLab, Gitee, Jenkins, etc.).
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

	// GetFileContent retrieves a file from the repository.
	GetFileContent(ctx context.Context, path, ref string) (string, error)

	// ListFiles lists files in a directory.
	ListFiles(ctx context.Context, path, ref string) ([]string, error)

	// CreateStatus creates a status check for a commit.
	CreateStatus(ctx context.Context, sha, state, description, context string) error
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
	BaseSHA   string
	HeadSHA   string
}

// Event represents a CI/CD event (push, PR, etc.).
type Event struct {
	Type      EventType
	Actor     string
	Action    string
	PRNumber  int
	CommitSHA string
	Ref       string
	RefType   string // branch, tag
}

// EventType represents the type of CI/CD event.
type EventType string

const (
	EventPush     EventType = "push"
	EventPR       EventType = "pull_request"
	EventIssue    EventType = "issue"
	EventComment  EventType = "comment"
	EventWorkflow EventType = "workflow_run"
	EventUnknown  EventType = "unknown"
)

// StatusState represents the state of a status check.
type StatusState string

const (
	StatusPending    StatusState = "pending"
	StatusSuccess    StatusState = "success"
	StatusError      StatusState = "error"
	StatusFailure    StatusState = "failure"
)

// Adapter is a helper base for platform implementations.
type Adapter struct {
	name   string
	detect func() bool
	client Client
}

// Client represents the HTTP client for platform APIs.
type Client interface {
	Get(ctx context.Context, path string) ([]byte, error)
	Post(ctx context.Context, path string, body []byte) ([]byte, error)
}

// NewAdapter creates a new platform adapter.
func NewAdapter(name string, detect func() bool, client Client) *Adapter {
	return &Adapter{
		name:   name,
		detect: detect,
		client: client,
	}
}

// Name returns the platform name.
func (a *Adapter) Name() string {
	return a.name
}

// Detect checks if this adapter is the current platform.
func (a *Adapter) Detect() bool {
	if a.detect != nil {
		return a.detect()
	}
	return false
}

// DetectFromEnvironment detects the platform from environment variables.
func DetectFromEnvironment() string {
	// Check GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return "github"
	}

	// Check GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		return "gitlab"
	}

	// Check Gitee
	if os.Getenv("GITEE_CI") == "true" {
		return "gitee"
	}

	// Check Jenkins
	if os.Getenv("JENKINS_HOME") != "" {
		return "jenkins"
	}

	// Check for git repository
	if _, err := os.Stat(".git"); err == nil {
		// Try to detect from git remote
		return detectFromGitRemote()
	}

	return "unknown"
}

// detectFromGitRemote detects platform from git remote URL.
func detectFromGitRemote() string {
	// This would read .git/config and parse remote URLs
	// For now, return unknown
	return "unknown"
}

// GetEventFromEnvironment parses event from environment variables.
func GetEventFromEnvironment() *Event {
	ev := &Event{Type: EventUnknown}

	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		ev.Type = parseEventType(os.Getenv("GITHUB_EVENT_NAME"))
		ev.Actor = os.Getenv("GITHUB_ACTOR")
		ev.Ref = os.Getenv("GITHUB_REF")
		ev.CommitSHA = os.Getenv("GITHUB_SHA")
		if prNum := os.Getenv("PR_NUMBER"); prNum != "" {
			// Handle pull_request event
		}
		return ev
	}

	// GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		// Parse GitLab event
		return ev
	}

	return ev
}

// parseEventType converts event name string to EventType.
func parseEventType(name string) EventType {
	switch strings.ToLower(name) {
	case "push":
		return EventPush
	case "pull_request", "merge_request":
		return EventPR
	case "issue":
		return EventIssue
	case "comment":
		return EventComment
	case "workflow_run":
		return EventWorkflow
	default:
		return EventUnknown
	}
}

// FileContentHelper helps retrieve file content across platforms.
type FileContentHelper struct {
	platform Platform
}

// NewFileContentHelper creates a new helper.
func NewFileContentHelper(p Platform) *FileContentHelper {
	return &FileContentHelper{platform: p}
}

// ReadFile reads a file from the repository.
func (h *FileContentHelper) ReadFile(ctx context.Context, path string) (string, error) {
	return h.platform.GetFileContent(ctx, path, "HEAD")
}

// ReadFileAtRef reads a file at a specific ref.
func (h *FileContentHelper) ReadFileAtRef(ctx context.Context, path, ref string) (string, error) {
	return h.platform.GetFileContent(ctx, path, ref)
}

// Exists checks if a file exists.
func (h *FileContentHelper) Exists(ctx context.Context, path string) bool {
	_, err := h.ReadFile(ctx, path)
	return err == nil
}

// DiffParser parses and provides access to diff content.
type DiffParser struct {
	content string
}

// NewDiffParser creates a new diff parser.
func NewDiffParser(content string) *DiffParser {
	return &DiffParser{content: content}
}

// Parse parses the diff and returns affected files.
func (d *DiffParser) Parse() ([]*FileDiff, error) {
	// Parse unified diff format
	// Return list of affected files with their changes
	return nil, nil
}

// GetFileChanges returns changes for a specific file.
func (d *DiffParser) GetFileChanges(filename string) (*FileDiff, error) {
	// Find and return changes for the specific file
	return nil, nil
}

// FileDiff represents changes to a file.
type FileDiff struct {
	Path         string
	OldPath      string
	IsNew        bool
	IsDeleted    bool
	IsRenamed    bool
	Additions    int
	Deletions    int
	Chunks       []*DiffChunk
}

// DiffChunk represents a section of changes in a file.
type DiffChunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []string
}

// StreamReader reads streaming content.
type StreamReader struct {
	reader io.Reader
}

// NewStreamReader creates a new stream reader.
func NewStreamReader(r io.Reader) *StreamReader {
	return &StreamReader{reader: r}
}

// ReadAll reads all content.
func (s *StreamReader) ReadAll() ([]byte, error) {
	return io.ReadAll(s.reader)
}

// ReadString reads content as string.
func (s *StreamReader) ReadString() (string, error) {
	data, err := io.ReadAll(s.reader)
	return string(data), err
}
