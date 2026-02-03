// Package ai provides a pluggable abstraction layer for AI CLI backends
// Supported backend: Claude Code CLI
package ai

import (
	"context"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/security"
)

// BackendType represents the type of AI backend
type BackendType string

const (
	// BackendClaude uses Claude Code CLI
	BackendClaude BackendType = "claude"
)

// Brain is the abstraction interface for AI CLI backends
// All backends must implement this interface to ensure compatibility
type Brain interface {
	// Execute runs the AI CLI with the given prompt and options
	// Returns the raw output and any error encountered
	Execute(ctx context.Context, prompt string, opts ExecuteOptions) (*Output, error)

	// ExecuteWithSkill runs the AI CLI with a specific skill loaded
	// Skills are loaded from the skills/ directory in SKILL.md format
	ExecuteWithSkill(ctx context.Context, prompt string, skill string, opts ExecuteOptions) (*Output, error)

	// Validate checks if the backend CLI is available and properly configured
	// Returns nil if the backend is ready to use
	Validate(ctx context.Context) error

	// Type returns the backend type identifier
	Type() BackendType

	// Version returns the CLI version if available
	Version(ctx context.Context) (string, error)
}

// ExecuteOptions contains options for AI execution
type ExecuteOptions struct {
	// Model specifies which model to use
	// For Claude: sonnet, opus, haiku
	Model string

	// MaxTurns limits the number of reasoning iterations (Claude-specific)
	MaxTurns int

	// MaxBudgetUSD limits API spending in USD (Claude-specific)
	MaxBudgetUSD float64

	// Timeout for the execution
	Timeout time.Duration

	// OutputFormat specifies desired output format (json, text, stream-json)
	OutputFormat string

	// Env contains additional environment variables to pass to the CLI
	Env []string

	// StdinContent is optional content to pipe to stdin
	StdinContent string

	// Skills is a list of skill paths to load
	Skills []string

	// EnablePromptInjectionValidation enables prompt injection detection
	// When true, prompts are validated before being sent to the AI backend
	EnablePromptInjectionValidation bool

	// InjectionDetector is the custom detector to use for validation
	// If nil and validation is enabled, a default detector is created
	InjectionDetector *security.PromptInjectionDetector
}

// Output represents the parsed output from an AI backend
type Output struct {
	// Raw is the complete raw output from the CLI
	Raw string

	// JSON contains the parsed JSON if output was JSON format
	JSON map[string]any

	// Thinking contains the thinking/reasoning block if present
	Thinking string

	// Result contains the main structured result
	Result any

	// Issues contains any issues found (for review skills)
	Issues []Issue

	// Duration is how long the execution took
	Duration time.Duration

	// TokensUsed contains token usage if available
	TokensUsed *TokenUsage

	// Model used for this execution
	Model string

	// Backend used for this execution
	Backend BackendType
}

// Issue represents a code review issue or finding
// This struct is shared across all backends for consistency
type Issue struct {
	Severity    string `json:"severity"` // critical, high, medium, low
	Category    string `json:"category"` // security, performance, logic, architecture
	File        string `json:"file"`
	Line        int    `json:"line"`
	Rule        string `json:"rule,omitempty"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"`
	CodeSnippet string `json:"code_snippet,omitempty"`
	Note        string `json:"note,omitempty"`
}

// TokenUsage contains token usage statistics
type TokenUsage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}

// ReviewSummary contains aggregated review statistics
type ReviewSummary struct {
	FilesChanged int `json:"files_changed"`
	TotalIssues  int `json:"total_issues"`
	Critical     int `json:"critical"`
	High         int `json:"high"`
	Medium       int `json:"medium"`
	Low          int `json:"low"`
}

// DefaultOptions returns sensible default execution options
func DefaultOptions() ExecuteOptions {
	return ExecuteOptions{
		OutputFormat: "json",
		Timeout:      5 * time.Minute,
	}
}

// String returns the string representation of a BackendType
func (b BackendType) String() string {
	return string(b)
}

// IsValid checks if the backend type is valid
func (b BackendType) IsValid() bool {
	return b == BackendClaude
}
