// Package ai provides utility functions for AI backend management
package ai

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/security"
)

// validateCommand checks if a command exists and is executable
func validateCommand(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s command not found or not working: %w", command, err)
	}
	return nil
}

// getCommandVersion executes a command with a version flag and returns the output
func getCommandVersion(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get %s version: %w", command, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ConvertIssues safely converts various issue types to ai.Issue
// This function handles type assertions safely with proper checking
func ConvertIssues(fromIssues []interface{}) []Issue {
	issues := make([]Issue, 0, len(fromIssues))
	for _, iface := range fromIssues {
		// Safe type assertion with proper checking
		switch v := iface.(type) {
		case Issue:
			issues = append(issues, v)
		case map[string]interface{}:
			// Handle map-based issues (from JSON parsing)
			issue := Issue{}
			if sev, ok := v["severity"].(string); ok {
				issue.Severity = sev
			}
			if cat, ok := v["category"].(string); ok {
				issue.Category = cat
			}
			if file, ok := v["file"].(string); ok {
				issue.File = file
			}
			if line, ok := v["line"].(float64); ok {
				issue.Line = int(line)
			}
			if msg, ok := v["message"].(string); ok {
				issue.Message = msg
			}
			if sug, ok := v["suggestion"].(string); ok {
				issue.Suggestion = sug
			}
			issues = append(issues, issue)
		default:
			// Skip unknown types - log warning in production
			continue
		}
	}
	return issues
}

// MergeExecuteOptions merges multiple ExecuteOptions with later options taking precedence
func MergeExecuteOptions(opts ...ExecuteOptions) ExecuteOptions {
	merged := DefaultOptions()
	for _, opt := range opts {
		if opt.Model != "" {
			merged.Model = opt.Model
		}
		if opt.MaxTurns > 0 {
			merged.MaxTurns = opt.MaxTurns
		}
		if opt.MaxBudgetUSD > 0 {
			merged.MaxBudgetUSD = opt.MaxBudgetUSD
		}
		if opt.Timeout > 0 {
			merged.Timeout = opt.Timeout
		}
		if opt.OutputFormat != "" {
			merged.OutputFormat = opt.OutputFormat
		}
		if opt.Provider != "" {
			merged.Provider = opt.Provider
		}
		if opt.BaseURL != "" {
			merged.BaseURL = opt.BaseURL
		}
		if len(opt.Env) > 0 {
			merged.Env = append(merged.Env, opt.Env...)
		}
	}
	return merged
}

// ParseTimeout parses a timeout string into a time.Duration
// Returns default timeout if parsing fails
func ParseTimeout(timeout string, defaultTimeout time.Duration) time.Duration {
	if timeout == "" {
		return defaultTimeout
	}
	if duration, err := time.ParseDuration(timeout); err == nil {
		return duration
	}
	return defaultTimeout
}

// ValidatePrompt validates a prompt for injection attacks
// If EnablePromptInjectionValidation is set in opts, it validates the prompt
// Returns an error if the prompt contains potential injection patterns
func ValidatePrompt(prompt string, opts ExecuteOptions) error {
	if !opts.EnablePromptInjectionValidation {
		return nil
	}

	detector := opts.InjectionDetector
	if detector == nil {
		// Use default detector if none provided
		detector = security.NewPromptInjectionDetector()
	}

	return detector.Validate(prompt)
}
