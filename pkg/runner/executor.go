// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/cicd-ai-toolkit/pkg/skill"
)

// Executor executes individual skills.
type Executor struct {
	// Configuration
	defaultTimeout  time.Duration
	maxRetries      int
	skipPermissions bool // WARNING: Only enable in trusted CI environments

	// Components
	processManager *ProcessManager
	retryExecutor  *RetryExecutor
	fallback       *FallbackHandler

	// Options
	verbose bool
	dryRun  bool
}

// ExecutorOption configures an Executor.
type ExecutorOption func(*Executor)

// WithDefaultTimeout sets the default execution timeout.
func WithDefaultTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.defaultTimeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(retries int) ExecutorOption {
	return func(e *Executor) {
		e.maxRetries = retries
	}
}

// WithVerbose enables verbose logging.
func WithVerbose(verbose bool) ExecutorOption {
	return func(e *Executor) {
		e.verbose = verbose
	}
}

// WithDryRun enables dry run mode.
func WithDryRun(dryRun bool) ExecutorOption {
	return func(e *Executor) {
		e.dryRun = dryRun
	}
}

// WithSkipPermissions enables dangerous permission skip mode.
// WARNING: Only enable in trusted CI environments.
func WithSkipPermissions(skip bool) ExecutorOption {
	return func(e *Executor) {
		e.skipPermissions = skip
	}
}

// NewExecutor creates a new executor with the given options.
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		defaultTimeout: 5 * time.Minute,
		maxRetries:     3,
		processManager: NewProcessManager(),
		retryExecutor:  NewRetryExecutor(DefaultRetryPolicy()),
		fallback:       NewFallbackHandler(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ExecuteRequest defines the input for skill execution.
type ExecuteRequest struct {
	Skill   *skill.Skill
	Inputs  map[string]any
	Timeout time.Duration
	DryRun  bool
}

// ExecuteResult defines the output of skill execution.
type ExecuteResult struct {
	Output   string
	ExitCode int
	Duration time.Duration
	Retries  int
	Error    error
}

// Execute executes a skill with the given request.
func (e *Executor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
	if req == nil || req.Skill == nil {
		return nil, fmt.Errorf("invalid request: skill is required")
	}

	start := time.Now()
	result := &ExecuteResult{}

	// Resolve input values
	inputs, err := req.Skill.ResolveInputValues(req.Inputs)
	if err != nil {
		result.Error = err
		result.ExitCode = ExitInfraError
		result.Duration = time.Since(start)
		return result, err
	}

	// Inject inputs into prompt
	injector := skill.NewInjector()
	prompt, err := injector.Inject(req.Skill.Prompt, inputs)
	if err != nil {
		result.Error = err
		result.ExitCode = ExitInfraError
		result.Duration = time.Since(start)
		return result, err
	}

	// Set timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = e.defaultTimeout
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build Claude arguments
	args := e.buildArgs(req.Skill)

	// Execute with retry
	var output string
	var lastErr error

	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		result.Retries = attempt

		if attempt > 0 {
			delay := e.retryExecutor.CalculateDelay(attempt)
			select {
			case <-time.After(delay):
			case <-execCtx.Done():
				result.Error = ErrTimeout
				result.ExitCode = ExitTimeout
				result.Duration = time.Since(start)
				return result, result.Error
			}
		}

		output, lastErr = e.executeOnce(execCtx, args, prompt)
		if lastErr == nil {
			result.Output = output
			result.ExitCode = ExitSuccess
			result.Duration = time.Since(start)
			return result, nil
		}

		// Check if error is retryable
		classified := ClassifyError(lastErr)
		if !classified.Retryable {
			break
		}
	}

	result.Error = fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
	result.ExitCode = ExitClaudeError
	result.Duration = time.Since(start)
	return result, result.Error
}

// executeOnce executes a single attempt.
func (e *Executor) executeOnce(ctx context.Context, args []string, prompt string) (string, error) {
	process := NewClaudeProcess(args)

	if err := process.Start(ctx); err != nil {
		return "", err
	}
	defer func() { _ = process.Stop() }()

	if err := process.WritePrompt(prompt); err != nil {
		return "", err
	}

	return process.Wait(ctx)
}

// buildArgs builds Claude CLI arguments from skill configuration.
func (e *Executor) buildArgs(sk *skill.Skill) []string {
	args := []string{"-p"} // Print/headless mode

	// Add dangerous permissions flag only if explicitly configured
	// WARNING: This bypasses Claude's permission prompts - only enable in trusted CI environments
	if e.skipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Add timeout if specified
	if sk.Options.TimeoutSeconds > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", sk.Options.TimeoutSeconds))
	}

	// Add allowed tools
	if sk.Tools != nil && len(sk.Tools.Allow) > 0 {
		for _, tool := range sk.Tools.Allow {
			args = append(args, "--allowedTools", tool)
		}
	}

	// Add max tokens if specified
	if sk.Options.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", sk.Options.MaxTokens))
	}

	return args
}

// ExecuteSkillByName executes a skill by name.
func (e *Executor) ExecuteSkillByName(ctx context.Context, skillName string, inputs map[string]any) (*ExecuteResult, error) {
	loader := skill.NewLoader()
	sk, err := loader.Load(skillName)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, skillName)
	}

	return e.Execute(ctx, &ExecuteRequest{
		Skill:  sk,
		Inputs: inputs,
	})
}
