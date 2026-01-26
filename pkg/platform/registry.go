// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package platform

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// Registry manages platform adapters.
type Registry struct {
	mu        sync.RWMutex
	platforms map[string]Platform
	detectors []Platform
}

// NewRegistry creates a new platform registry.
func NewRegistry() *Registry {
	r := &Registry{
		platforms: make(map[string]Platform),
	}
	return r
}

// Register registers a platform adapter.
func (r *Registry) Register(name string, p Platform) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.platforms[name] = p
}

// RegisterDetector registers a platform for auto-detection.
func (r *Registry) RegisterDetector(p Platform) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detectors = append(r.detectors, p)
}

// Get retrieves a platform by name.
func (r *Registry) Get(name string) (Platform, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.platforms[name]
	return p, ok
}

// Detect auto-detects the current platform.
func (r *Registry) Detect(ctx context.Context) (Platform, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try registered detectors first
	for _, p := range r.detectors {
		if adapter, ok := p.(*DetectorAdapter); ok {
			if adapter.DetectFunc != nil && adapter.DetectFunc() {
				return adapter, nil
			}
		}
	}

	// Fallback to environment detection
	platformName := DetectFromEnvironment()
	if p, ok := r.platforms[platformName]; ok {
		return p, nil
	}

	return nil, fmt.Errorf("no platform adapter found for: %s", platformName)
}

// MustGet returns a platform or panics.
func (r *Registry) MustGet(name string) Platform {
	p, ok := r.Get(name)
	if !ok {
		panic(fmt.Sprintf("platform not found: %s", name))
	}
	return p
}

// List returns all registered platforms.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.platforms))
	for name := range r.platforms {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry is the global default registry.
var DefaultRegistry = NewRegistry()

// Register registers a platform in the default registry.
func Register(name string, p Platform) {
	DefaultRegistry.Register(name, p)
}

// Get retrieves a platform from the default registry.
func Get(name string) (Platform, bool) {
	return DefaultRegistry.Get(name)
}

// Detect detects the current platform using the default registry.
func Detect(ctx context.Context) (Platform, error) {
	return DefaultRegistry.Detect(ctx)
}

// DetectorAdapter is a platform adapter with custom detection.
type DetectorAdapter struct {
	*Adapter
	DetectFunc func() bool
}

// NewDetectorAdapter creates a new detector adapter.
func NewDetectorAdapter(name string, detectFunc func() bool, client Client) *DetectorAdapter {
	return &DetectorAdapter{
		Adapter:    NewAdapter(name, nil, client),
		DetectFunc: detectFunc,
	}
}

// Detect runs the custom detection function.
func (d *DetectorAdapter) Detect() bool {
	if d.DetectFunc != nil {
		return d.DetectFunc()
	}
	return false
}

// GetPullRequest retrieves a pull request by number.
func (d *DetectorAdapter) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

// PostComment posts a comment on a pull request.
func (d *DetectorAdapter) PostComment(ctx context.Context, number int, body string) error {
	return fmt.Errorf("not implemented")
}

// GetDiff retrieves the diff for a pull request.
func (d *DetectorAdapter) GetDiff(ctx context.Context, number int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// GetEvent returns the current CI/CD event.
func (d *DetectorAdapter) GetEvent(ctx context.Context) (*Event, error) {
	return GetEventFromEnvironment(), nil
}

// GetFileContent retrieves a file from the repository.
func (d *DetectorAdapter) GetFileContent(ctx context.Context, path, ref string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// ListFiles lists files in a directory.
func (d *DetectorAdapter) ListFiles(ctx context.Context, path, ref string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// CreateStatus creates a status check for a commit.
func (d *DetectorAdapter) CreateStatus(ctx context.Context, sha, state, description, context string) error {
	return fmt.Errorf("not implemented")
}

// GitHub detector
func detectGitHub() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// GitLab detector
func detectGitLab() bool {
	return os.Getenv("GITLAB_CI") == "true"
}

// Gitee detector
func detectGitee() bool {
	return os.Getenv("GITEE_CI") == "true" || os.Getenv("GITEE_SERVER_URL") != ""
}

// Jenkins detector
func detectJenkins() bool {
	return os.Getenv("JENKINS_HOME") != "" || os.Getenv("JENKINS_URL") != ""
}

// InitializeDefaultRegistry initializes the default registry with common platforms.
func InitializeDefaultRegistry() {
	// Register platform detectors
	DefaultRegistry.RegisterDetector(&DetectorAdapter{
		Adapter:    NewAdapter("github", detectGitHub, nil),
		DetectFunc: detectGitHub,
	})
	DefaultRegistry.RegisterDetector(&DetectorAdapter{
		Adapter:    NewAdapter("gitlab", detectGitLab, nil),
		DetectFunc: detectGitLab,
	})
	DefaultRegistry.RegisterDetector(&DetectorAdapter{
		Adapter:    NewAdapter("gitee", detectGitee, nil),
		DetectFunc: detectGitee,
	})
	DefaultRegistry.RegisterDetector(&DetectorAdapter{
		Adapter:    NewAdapter("jenkins", detectJenkins, nil),
		DetectFunc: detectJenkins,
	})
}

func init() {
	InitializeDefaultRegistry()
}
