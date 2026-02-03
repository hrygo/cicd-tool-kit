// Package ai provides factory functions for creating AI backend instances
package ai

import (
	"context"
	"fmt"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
)

// Factory creates AI Brain instances based on configuration
type Factory struct {
	baseDir string
}

// NewFactory creates a new AI Brain factory
func NewFactory(baseDir string) *Factory {
	return &Factory{baseDir: baseDir}
}

// Create creates an AI Brain instance based on the backend type
// If backendType is empty, it defaults to BackendClaude
func (f *Factory) Create(backendType BackendType, cfg *config.Config) (Brain, error) {
	if backendType == "" {
		backendType = BackendClaude
	}

	if !backendType.IsValid() {
		return nil, fmt.Errorf("invalid backend type: %s", backendType)
	}

	switch backendType {
	case BackendClaude:
		return f.createClaudeBackend(cfg)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}

// CreateFromConfig creates an AI Brain from the configuration
// It reads the ai_backend field from the config
func (f *Factory) CreateFromConfig(cfg *config.Config) (Brain, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	backendType := BackendType(cfg.AIBackend)
	if backendType == "" {
		backendType = BackendClaude
	}

	return f.Create(backendType, cfg)
}

// createClaudeBackend creates a Claude Code CLI backend
func (f *Factory) createClaudeBackend(cfg *config.Config) (Brain, error) {
	backend := NewClaudeBackend(&cfg.Claude)

	// Validate the backend is available
	ctx := context.Background()
	if err := backend.Validate(ctx); err != nil {
		return nil, fmt.Errorf("claude backend validation failed: %w", err)
	}

	return backend, nil
}

// DetectBackend attempts to auto-detect the best available backend
// It checks for Claude Code CLI
func (f *Factory) DetectBackend() (BackendType, error) {
	ctx := context.Background()

	// Try Claude
	if err := validateCommand(ctx, "claude", "--version"); err == nil {
		return BackendClaude, nil
	}

	return "", fmt.Errorf("no supported AI backend found (need claude)")
}

// ListAvailableBackends returns a list of available backends
func (f *Factory) ListAvailableBackends() []BackendType {
	ctx := context.Background()
	var available []BackendType

	if err := validateCommand(ctx, "claude", "--version"); err == nil {
		available = append(available, BackendClaude)
	}

	return available
}

// IsBackendAvailable checks if a specific backend is available
func (f *Factory) IsBackendAvailable(backendType BackendType) bool {
	ctx := context.Background()

	switch backendType {
	case BackendClaude:
		return validateCommand(ctx, "claude", "--version") == nil
	default:
		return false
	}
}
