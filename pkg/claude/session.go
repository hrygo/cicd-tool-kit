// Package claude handles Claude Code subprocess management and output parsing
package claude

import (
	"context"
	"io"
	"time"
)

// Session manages a Claude Code subprocess
type Session interface {
	// Execute runs Claude with the given prompt and returns the output
	Execute(ctx context.Context, opts ExecuteOptions) (*Output, error)

	// ExecuteWithStreams runs Claude with custom stdin/stdout
	ExecuteWithStreams(ctx context.Context, opts ExecuteOptions, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// Close terminates the Claude process
	Close() error
}

// ExecuteOptions contains options for Claude execution
type ExecuteOptions struct {
	// Prompt is the main query/prompt for Claude
	Prompt string

	// StdinContent is optional content to pipe to stdin
	StdinContent string

	// Model specifies which Claude model to use (sonnet, opus, haiku)
	Model string

	// Skills is a list of skill paths to load
	Skills []string

	// MaxTurns limits the number of reasoning iterations
	MaxTurns int

	// MaxBudgetUSD limits API spending
	MaxBudgetUSD float64

	// Timeout for the execution
	Timeout time.Duration

	// AllowedTools restricts which tools Claude can use
	AllowedTools []string

	// OutputFormat specifies desired output format (json, stream-json, text)
	OutputFormat string

	// SkipPermissions skips interactive permission prompts
	SkipPermissions bool

	// Environment variables to pass to Claude
	Env []string
}

// Output represents parsed Claude output
type Output struct {
	// Raw is the complete raw output
	Raw string

	// JSON contains the parsed JSON if output was JSON format
	JSON map[string]interface{}

	// Thinking contains the thinking block if present
	Thinking string

	// Result contains the main structured result
	Result interface{}

	// Issues contains any issues found (for review skills)
	Issues []Issue

	// Duration is how long the execution took
	Duration time.Duration

	// TokensUsed contains token usage if available
	TokensUsed *TokenUsage
}

// Issue represents a code review issue or finding
type Issue struct {
	Severity   string `json:"severity"`   // critical, high, medium, low
	Category   string `json:"category"`   // security, performance, logic, architecture
	File       string `json:"file"`
	Line       int    `json:"line"`
	Rule       string `json:"rule,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	CodeSnippet string `json:"code_snippet,omitempty"`
	Note       string `json:"note,omitempty"`
}

// TokenUsage contains token usage statistics
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
}

// OutputParser parses Claude's output format
type OutputParser interface {
	// ParseJSON extracts and parses JSON from Claude output
	ParseJSON(output string, target interface{}) error

	// ExtractJSONBlock extracts JSON code blocks from markdown
	ExtractJSONBlock(output string) (string, error)

	// ExtractThinking extracts the thinking block
	ExtractThinking(output string) string

	// ExtractIssues extracts issue arrays from review output
	ExtractIssues(output string) ([]Issue, error)

	// ExtractReviewSummary extracts a summary from review output
	ExtractReviewSummary(output string) string

	// ExtractCodeChanges extracts code change suggestions from output
	ExtractCodeChanges(output string) []CodeChange
}
