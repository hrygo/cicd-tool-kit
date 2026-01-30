// Package ai provides Claude Code CLI backend implementation
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/claude"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
)

// ClaudeBackend implements the Brain interface using Claude Code CLI
type ClaudeBackend struct {
	cfg       *config.ClaudeConfig
	cliPath   string
	validator func(ctx context.Context) error
}

// NewClaudeBackend creates a new Claude Code CLI backend
func NewClaudeBackend(cfg *config.ClaudeConfig) *ClaudeBackend {
	if cfg == nil {
		cfg = &config.ClaudeConfig{
			Model:        "sonnet",
			OutputFormat: "json",
		}
	}

	return &ClaudeBackend{
		cfg:     cfg,
		cliPath: "claude",
		validator: func(ctx context.Context) error {
			// Check if claude command exists
			return validateCommand(ctx, "claude", "--version")
		},
	}
}

// Execute runs Claude Code CLI with the given prompt
func (b *ClaudeBackend) Execute(ctx context.Context, prompt string, opts ExecuteOptions) (*Output, error) {
	// Merge default config with options
	execOpts := b.mergeOptions(opts)

	start := time.Now()

	// Create Claude session
	session, err := claude.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create claude session: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Build execute options for Claude
	claudeOpts := claude.ExecuteOptions{
		Prompt:          prompt,
		Model:           execOpts.Model,
		MaxTurns:        execOpts.MaxTurns,
		MaxBudgetUSD:    execOpts.MaxBudgetUSD,
		OutputFormat:    execOpts.OutputFormat,
		SkipPermissions: true,
		Env:             execOpts.Env,
	}

	// Add skills
	claudeOpts.Skills = append(claudeOpts.Skills, execOpts.Skills...)

	// Add timeout if specified
	if execOpts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, execOpts.Timeout)
		defer cancel()
	}

	// Execute
	result, err := session.Execute(ctx, claudeOpts)
	if err != nil {
		return nil, fmt.Errorf("claude execution failed: %w", err)
	}

	// Convert claude.Issue to ai.Issue
	issues := make([]Issue, len(result.Issues))
	for i, issue := range result.Issues {
		issues[i] = Issue{
			Severity:    issue.Severity,
			Category:    issue.Category,
			File:        issue.File,
			Line:        issue.Line,
			Rule:        issue.Rule,
			Message:     issue.Message,
			Suggestion:  issue.Suggestion,
			CodeSnippet: issue.CodeSnippet,
			Note:        issue.Note,
		}
	}

	// Convert TokenUsage
	var tokensUsed *TokenUsage
	if result.TokensUsed != nil {
		tokensUsed = &TokenUsage{
			InputTokens:  result.TokensUsed.InputTokens,
			OutputTokens: result.TokensUsed.OutputTokens,
			TotalTokens:  result.TokensUsed.TotalTokens,
			CostUSD:      result.TokensUsed.CostUSD,
		}
	}

	// Convert to our Output format
	output := &Output{
		Raw:        result.Raw,
		JSON:       result.JSON,
		Thinking:   result.Thinking,
		Result:     result.Result,
		Issues:     issues,
		Duration:   time.Since(start),
		TokensUsed: tokensUsed,
		Model:      execOpts.Model,
		Backend:    BackendClaude,
	}

	return output, nil
}

// ExecuteWithSkill runs Claude Code CLI with a specific skill loaded
func (b *ClaudeBackend) ExecuteWithSkill(ctx context.Context, prompt string, skill string, opts ExecuteOptions) (*Output, error) {
	// Claude uses --skill flag - add it to the options
	if opts.Skills == nil {
		opts.Skills = []string{}
	}
	opts.Skills = append(opts.Skills, skill)
	return b.Execute(ctx, prompt, opts)
}

// Validate checks if Claude Code CLI is available
func (b *ClaudeBackend) Validate(ctx context.Context) error {
	return b.validator(ctx)
}

// Type returns the backend type
func (b *ClaudeBackend) Type() BackendType {
	return BackendClaude
}

// Version returns the Claude Code CLI version
func (b *ClaudeBackend) Version(ctx context.Context) (string, error) {
	return getCommandVersion(ctx, "claude", "--version")
}

// mergeOptions merges default config with runtime options
func (b *ClaudeBackend) mergeOptions(opts ExecuteOptions) ExecuteOptions {
	merged := ExecuteOptions{
		Model:        b.cfg.Model,
		OutputFormat: b.cfg.OutputFormat,
		MaxTurns:     b.cfg.MaxTurns,
		MaxBudgetUSD: b.cfg.MaxBudgetUSD,
		Timeout:      opts.Timeout,
		Env:          opts.Env,
	}

	// Override with runtime options
	if opts.Model != "" {
		merged.Model = opts.Model
	}
	if opts.OutputFormat != "" {
		merged.OutputFormat = opts.OutputFormat
	}
	if opts.MaxTurns > 0 {
		merged.MaxTurns = opts.MaxTurns
	}
	if opts.MaxBudgetUSD > 0 {
		merged.MaxBudgetUSD = opts.MaxBudgetUSD
	}

	// Parse timeout from config if not set
	if merged.Timeout == 0 && b.cfg.Timeout != "" {
		if duration, err := time.ParseDuration(b.cfg.Timeout); err == nil {
			merged.Timeout = duration
		}
	}

	return merged
}

// GetDefaultConfig returns default Claude configuration
func GetDefaultClaudeConfig() config.ClaudeConfig {
	return config.ClaudeConfig{
		Model:        "sonnet",
		MaxTurns:     10,
		MaxBudgetUSD: 1.0,
		Timeout:      "5m",
		OutputFormat: "json",
	}
}

// ParseModel validates a Claude model string
func ValidateClaudeModel(model string) error {
	validModels := []string{"sonnet", "opus", "haiku"}
	for _, m := range validModels {
		if strings.EqualFold(model, m) {
			return nil
		}
	}
	return fmt.Errorf("invalid claude model: %s (valid: sonnet, opus, haiku)", model)
}
