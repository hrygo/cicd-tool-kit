// Package ai provides Crush CLI backend implementation
// Crush is an open-source AI CLI by Charmbracelet (successor to OpenCode)
package ai

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CrushBackend implements the Brain interface using Crush CLI
type CrushBackend struct {
	cfg     *CrushConfig
	cliPath string
}

// CrushConfig contains Crush-specific configuration
type CrushConfig struct {
	// Provider is the AI provider to use
	// Options: anthropic, openai, gemini, groq, ollama, etc.
	Provider string

	// Model is the model identifier
	// For anthropic: claude-sonnet-4-20250514, claude-opus-4-20250514
	// For ollama: llama3:70b, mistral:7b, etc.
	Model string

	// BaseURL for custom endpoints (e.g., local Ollama)
	BaseURL string

	// Timeout for execution
	Timeout string

	// OutputFormat for results
	OutputFormat string
}

// NewCrushBackend creates a new Crush CLI backend
func NewCrushBackend(cfg *CrushConfig) *CrushBackend {
	if cfg == nil {
		cfg = &CrushConfig{
			Provider:     "anthropic",
			Model:        "claude-sonnet-4-20250514",
			OutputFormat: "json",
			Timeout:      "5m",
		}
	}

	return &CrushBackend{
		cfg:     cfg,
		cliPath: "crush",
	}
}

// Execute runs Crush CLI with the given prompt
func (b *CrushBackend) Execute(ctx context.Context, prompt string, opts ExecuteOptions) (*Output, error) {
	start := time.Now()

	// Merge options
	execOpts := b.mergeOptions(opts)

	// Build command
	args := b.buildArgs(prompt, execOpts)

	cmd := exec.CommandContext(ctx, b.cliPath, args...)

	// Setup environment
	env := os.Environ()
	if len(opts.Env) > 0 {
		env = append(env, opts.Env...)
	}
	cmd.Env = env

	// Execute and capture output
	stdout, stderr, err := b.runCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("crush execution failed: %w (stderr: %s)", err, stderr)
	}

	// Parse output
	output := &Output{
		Raw:      stdout,
		Duration: time.Since(start),
		Model:    execOpts.Model,
		Backend:  BackendCrush,
	}

	// Try to extract JSON
	if jsonStr, err := extractJSONBlock(stdout); err == nil {
		output.Result = jsonStr
		// Parse JSON map if possible
		output.JSON = make(map[string]interface{})
		// Full JSON parsing would be done by the caller
	}

	// Extract thinking block
	output.Thinking = extractThinking(stdout)

	return output, nil
}

// ExecuteWithSkill runs Crush CLI with a specific skill loaded
func (b *CrushBackend) ExecuteWithSkill(ctx context.Context, prompt string, skill string, opts ExecuteOptions) (*Output, error) {
	// Crush uses --skill flag similar to Claude
	if opts.Skills == nil {
		opts.Skills = []string{}
	}
	opts.Skills = append(opts.Skills, skill)
	return b.Execute(ctx, prompt, opts)
}

// Validate checks if Crush CLI is available
func (b *CrushBackend) Validate(ctx context.Context) error {
	return validateCommand(ctx, "crush", "--version")
}

// Type returns the backend type
func (b *CrushBackend) Type() BackendType {
	return BackendCrush
}

// Version returns the Crush CLI version
func (b *CrushBackend) Version(ctx context.Context) (string, error) {
	return getCommandVersion(ctx, "crush", "--version")
}

// mergeOptions merges default config with runtime options
func (b *CrushBackend) mergeOptions(opts ExecuteOptions) ExecuteOptions {
	merged := ExecuteOptions{
		Provider:     b.cfg.Provider,
		Model:        b.cfg.Model,
		OutputFormat: b.cfg.OutputFormat,
		BaseURL:      b.cfg.BaseURL,
		Timeout:      opts.Timeout,
		Env:          opts.Env,
	}

	// Override with runtime options
	if opts.Provider != "" {
		merged.Provider = opts.Provider
	}
	if opts.Model != "" {
		merged.Model = opts.Model
	}
	if opts.OutputFormat != "" {
		merged.OutputFormat = opts.OutputFormat
	}
	if opts.BaseURL != "" {
		merged.BaseURL = opts.BaseURL
	}

	// Parse timeout from config if not set
	if merged.Timeout == 0 && b.cfg.Timeout != "" {
		if duration, err := time.ParseDuration(b.cfg.Timeout); err == nil {
			merged.Timeout = duration
		}
	}

	return merged
}

// buildArgs constructs command line arguments for Crush
func (b *CrushBackend) buildArgs(prompt string, opts ExecuteOptions) []string {
	// Validate prompt to prevent empty or extremely long prompts
	if prompt == "" {
		prompt = "(empty)" // Provide a default to avoid empty argument
	}
	// Limit prompt length to prevent command line overflow
	const maxPromptLen = 100000 // 100KB limit
	if len(prompt) > maxPromptLen {
		prompt = prompt[:maxPromptLen] + "... (truncated)"
	}

	args := []string{
		"-p", prompt, // Print/headless mode
	}

	// Provider selection
	if opts.Provider != "" {
		args = append(args, "--provider", opts.Provider)
	}

	// Model selection
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Output format
	if opts.OutputFormat != "" {
		args = append(args, "--output-format", opts.OutputFormat)
	}

	// Base URL for custom endpoints
	if opts.BaseURL != "" {
		args = append(args, "--base-url", opts.BaseURL)
	}

	// Skills - limit number of skills to prevent command overflow
	const maxSkills = 20
	if len(opts.Skills) > maxSkills {
		opts.Skills = opts.Skills[:maxSkills]
	}
	for _, skill := range opts.Skills {
		if skill == "" {
			continue // Skip empty skills
		}
		args = append(args, "--skill", skill)
	}

	return args
}

// runCommand executes the command and returns stdout/stderr
func (b *CrushBackend) runCommand(cmd *exec.Cmd) (stdout string, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()

	return stdoutBuf.String(), stderrBuf.String(), err
}

// extractJSONBlock extracts JSON from markdown code blocks
func extractJSONBlock(output string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	inJSONBlock := false
	var jsonBuf strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check for JSON code block start
		if strings.HasPrefix(line, "```json") || strings.HasPrefix(line, "``` ") {
			inJSONBlock = true
			continue
		}
		if strings.TrimSpace(line) == "```" && !inJSONBlock {
			inJSONBlock = true
			continue
		}

		// Check for code block end
		if inJSONBlock && strings.HasPrefix(line, "```") {
			break
		}

		// Collect JSON content
		if inJSONBlock {
			jsonBuf.WriteString(line)
			jsonBuf.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	jsonStr := strings.TrimSpace(jsonBuf.String())
	if jsonStr == "" {
		return "", fmt.Errorf("no JSON block found")
	}

	return jsonStr, nil
}

// extractThinking extracts the thinking block from output
func extractThinking(output string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	inThinkingBlock := false
	var thinkingBuf strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check for thinking block start
		if strings.Contains(line, "<thinking>") {
			inThinkingBlock = true
			if _, after, ok := strings.Cut(line, "<thinking>"); ok {
				rest := after
				if content, _, ok := strings.Cut(rest, "</thinking>"); ok {
					return strings.TrimSpace(content)
				}
				if rest != "" {
					thinkingBuf.WriteString(rest)
					thinkingBuf.WriteString("\n")
				}
			}
			continue
		}

		// Check for thinking block end
		if inThinkingBlock && strings.Contains(line, "</thinking>") {
			if before, _, ok := strings.Cut(line, "</thinking>"); ok {
				if before != "" {
					thinkingBuf.WriteString(before)
				}
			}
			break
		}

		// Collect thinking content
		if inThinkingBlock {
			thinkingBuf.WriteString(line)
			thinkingBuf.WriteString("\n")
		}
	}

	return strings.TrimSpace(thinkingBuf.String())
}

// GetDefaultCrushConfig returns default Crush configuration
func GetDefaultCrushConfig() CrushConfig {
	return CrushConfig{
		Provider:     "anthropic",
		Model:        "claude-sonnet-4-20250514",
		OutputFormat: "json",
		Timeout:      "5m",
	}
}

// ValidateCrushProvider validates a Crush provider string
func ValidateCrushProvider(provider string) error {
	// Crush supports many providers - this is a non-exhaustive list
	// In practice, Crush will validate the provider
	validProviders := []string{
		"anthropic", "openai", "gemini", "groq",
		"ollama", "azure", "bedrock", "vertex",
	}
	for _, p := range validProviders {
		if strings.EqualFold(provider, p) {
			return nil
		}
	}
	// Allow unknown providers as Crush may add more
	return nil
}
