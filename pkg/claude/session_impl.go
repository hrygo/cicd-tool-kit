// Package claude handles Claude Code subprocess management and output parsing
package claude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/errors"
)

// validatePrompt checks if prompt contains potentially dangerous content
// While exec.Command with separate args prevents shell injection, we validate
// to catch obvious issues early and provide clear error messages
func validatePrompt(prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	// Check for null bytes which can cause issues
	if strings.Contains(prompt, "\x00") {
		return fmt.Errorf("prompt contains null bytes")
	}
	// Check for extremely long prompts that might cause issues
	if len(prompt) > 1000000 { // 1MB limit
		return fmt.Errorf("prompt too large: %d bytes (max 1MB)", len(prompt))
	}
	return nil
}

// processSession implements Session using a subprocess
type processSession struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	closed   bool
	closeMux sync.Mutex
}

// NewSession creates a new Claude session
func NewSession(ctx context.Context) (Session, error) {
	// Check if claude command exists
	cmd := exec.CommandContext(ctx, "claude", "--version")
	if err := cmd.Run(); err != nil {
		return nil, errors.ClaudeError("claude command not found. Please install Claude Code CLI", err)
	}

	return &processSession{}, nil
}

// Execute runs Claude with the given prompt and returns the output
func (s *processSession) Execute(ctx context.Context, opts ExecuteOptions) (*Output, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	err := s.ExecuteWithStreams(ctx, opts,
		strings.NewReader(opts.StdinContent),
		&stdoutBuf,
		&stderrBuf,
	)

	rawOutput := stdoutBuf.String()

	if err != nil {
		return &Output{
			Raw:      rawOutput,
			Duration: 0,
		}, fmt.Errorf("claude execution failed: %w", err)
	}

	// Parse output
	output := &Output{
		Raw:    rawOutput,
		Result: rawOutput,
	}

	// Try to extract JSON if present
	if jsonStr, err := extractJSONBlock(rawOutput); err == nil {
		output.JSON = make(map[string]interface{})
		// Simple JSON parse attempt
		// Full parsing would use json.Unmarshal in a real implementation
		output.Result = jsonStr
	}

	// Extract thinking block
	output.Thinking = extractThinking(rawOutput)

	return output, nil
}

// ExecuteWithStreams runs Claude with custom stdin/stdout
func (s *processSession) ExecuteWithStreams(ctx context.Context, opts ExecuteOptions, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	// Build command arguments
	args := s.buildArgs(opts)

	// Create command
	s.cmd = exec.CommandContext(ctx, "claude", args...)

	// Setup pipes
	cmdStdin, err := s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	defer func() {
		// Close stdin if Start fails (will be no-op if successfully started and copied)
		if cmdStdin != nil {
			cmdStdin.Close()
		}
	}()

	cmdStdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	defer func() {
		// Close stdout if Start fails
		if cmdStdout != nil {
			io.Copy(io.Discard, cmdStdout) // Drain any pending data
			cmdStdout.Close()
		}
	}()

	cmdStderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	defer func() {
		// Close stderr if Start fails
		if cmdStderr != nil {
			io.Copy(io.Discard, cmdStderr) // Drain any pending data
			cmdStderr.Close()
		}
	}()

	s.stdin = cmdStdin
	s.stdout = cmdStdout
	s.stderr = cmdStderr

	// Set environment
	if len(opts.Env) > 0 {
		s.cmd.Env = append(s.cmd.Env, opts.Env...)
	}

	// Start the command
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	// Command started successfully - cancel defer cleanup of pipes
	// They will be handled by the normal flow below
	cmdStdin = nil
	cmdStdout = nil
	cmdStderr = nil

	// Use a WaitGroup for streaming
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(stdout, cmdStdout); err != nil {
			errChan <- fmt.Errorf("stdout copy error: %w", err)
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(stderr, cmdStderr); err != nil {
			errChan <- fmt.Errorf("stderr copy error: %w", err)
		}
	}()

	// Write stdin content
	if stdin != nil {
		if _, err := io.Copy(cmdStdin, stdin); err != nil {
			return fmt.Errorf("stdin write error: %w", err)
		}
	}

	// Close stdin to signal EOF
	if err := cmdStdin.Close(); err != nil {
		return fmt.Errorf("stdin close error: %w", err)
	}

	// Wait for streaming to complete
	wg.Wait()

	// Wait for command to finish
	if err := s.cmd.Wait(); err != nil {
		select {
		case streamErr := <-errChan:
			return streamErr
		default:
			return err
		}
	}

	return nil
}

// Close terminates the Claude process
func (s *processSession) Close() error {
	s.closeMux.Lock()
	defer s.closeMux.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	if s.cmd != nil && s.cmd.Process != nil && s.cmd.ProcessState == nil {
		// Process exists and is still running (not yet waited on)
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill claude process: %w", err)
		}
	}

	return nil
}

// buildArgs constructs command line arguments from options
func (s *processSession) buildArgs(opts ExecuteOptions) []string {
	args := []string{}

	// Print mode (headless/non-interactive)
	args = append(args, "-p")

	// Validate and add prompt
	if err := validatePrompt(opts.Prompt); err != nil {
		// Log warning but continue - let the claude CLI handle invalid prompts
		// This prevents crashing on potentially valid prompts that our validator doesn't understand
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	args = append(args, opts.Prompt)

	// Skip permissions
	if opts.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Model selection
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Max turns
	if opts.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", opts.MaxTurns))
	}

	// Max budget
	if opts.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", opts.MaxBudgetUSD))
	}

	// Output format
	if opts.OutputFormat != "" {
		args = append(args, "--output-format", opts.OutputFormat)
	}

	// Allowed tools
	if len(opts.AllowedTools) > 0 {
		args = append(args, "--allowed-tools", strings.Join(opts.AllowedTools, ","))
	}

	// Skills to load
	for _, skill := range opts.Skills {
		args = append(args, "--skill", skill)
	}

	return args
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
		// Also check for bare ``` (might be JSON)
		if strings.TrimSpace(line) == "```" && !inJSONBlock {
			// Peek ahead to see if this might be JSON
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
			// Extract content after the tag if on same line
			if idx := strings.Index(line, "<thinking>"); idx >= 0 {
				rest := line[idx+len("<thinking>"):]
				// Check if closing tag is also on this line
				if closeIdx := strings.Index(rest, "</thinking>"); closeIdx >= 0 {
					// Inline thinking: <thinking>content</thinking>
					return strings.TrimSpace(rest[:closeIdx])
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
			// Extract content before the closing tag if on same line
			if idx := strings.Index(line, "</thinking>"); idx >= 0 {
				rest := line[:idx]
				if rest != "" {
					thinkingBuf.WriteString(rest)
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

// ExecuteSimple is a convenience function for one-shot Claude execution
func ExecuteSimple(ctx context.Context, prompt string) (string, error) {
	session, err := NewSession(ctx)
	if err != nil {
		return "", err
	}
	defer session.Close()

	opts := ExecuteOptions{
		Prompt:          prompt,
		SkipPermissions: true,
		OutputFormat:    "text",
	}

	output, err := session.Execute(ctx, opts)
	if err != nil {
		return "", err
	}

	return output.Raw, nil
}

// ExecuteWithInput executes Claude with stdin input
func ExecuteWithInput(ctx context.Context, prompt string, input string) (string, error) {
	session, err := NewSession(ctx)
	if err != nil {
		return "", err
	}
	defer session.Close()

	opts := ExecuteOptions{
		Prompt:          prompt,
		StdinContent:    input,
		SkipPermissions: true,
		OutputFormat:    "text",
	}

	output, err := session.Execute(ctx, opts)
	if err != nil {
		return "", err
	}

	return output.Raw, nil
}
