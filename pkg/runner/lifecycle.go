// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package runner provides the core execution engine for CICD AI Toolkit.
package runner

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cicd-ai-toolkit/pkg/config"
	"github.com/cicd-ai-toolkit/pkg/skill"
)

// State represents the runner lifecycle state.
type State int

const (
	StateUninitialized State = iota
	StateInitializing
	StateReady
	StateRunning
	StateShuttingDown
	StateStopped
)

func (s State) String() string {
	switch s {
	case StateUninitialized:
		return "uninitialized"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateShuttingDown:
		return "shutting_down"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// Runner represents the main execution engine.
type Runner struct {
	mu sync.RWMutex

	// Core components
	config       *config.Config
	configLoader *config.Loader
	skillLoader  *skill.Loader

	// Process management
	process     *ClaudeProcess
	processPool *ProcessPool

	// Watchdog and retry
	watchdog      *Watchdog
	retryExecutor *RetryExecutor
	fallback      *FallbackHandler

	// State
	state         State
	bootstrapTime time.Duration

	// Metrics
	metrics *BootstrapMetrics

	// Options
	opts *Options

	// Signal handling
	signalChan chan os.Signal
	shutdownCh chan struct{}
}

// Options holds runner configuration options.
type Options struct {
	// ConfigPath is the path to the config file.
	ConfigPath string
	// WorkDir is the working directory.
	WorkDir string
	// SkillDirs are the directories to scan for skills.
	SkillDirs []string
	// PreWarmClaude enables Claude process pre-warming.
	PreWarmClaude bool
	// GracefulTimeout is the timeout for graceful shutdown.
	GracefulTimeout time.Duration
	// Verbose enables verbose logging.
	Verbose bool
	// DryRun runs without posting results.
	DryRun bool
}

// DefaultOptions returns the default runner options.
func DefaultOptions() *Options {
	return &Options{
		ConfigPath:      ".cicd-ai-toolkit.yaml",
		WorkDir:         ".",
		SkillDirs:       []string{".skills", "skills"},
		PreWarmClaude:   false,
		GracefulTimeout: 5 * time.Second,
		Verbose:         false,
		DryRun:          false,
	}
}

// BootstrapMetrics holds bootstrap timing metrics.
type BootstrapMetrics struct {
	StartTime    time.Time
	ConfigLoad   time.Duration
	PlatformInit time.Duration
	SkillScan    time.Duration
	ClaudeWarmup time.Duration
	TotalTime    time.Duration
}

// RunRequest defines the input for a run.
type RunRequest struct {
	// SkillName is the name of the skill to execute.
	SkillName string
	// Inputs are the input values for the skill.
	Inputs map[string]any
	// Timeout is the execution timeout.
	Timeout time.Duration
	// DryRun runs without posting results.
	DryRun bool
}

// RunResult defines the output of a run.
type RunResult struct {
	// ExitCode is the exit code.
	ExitCode int
	// Output is the captured output.
	Output string
	// Error is any error that occurred.
	Error error
	// Duration is the execution duration.
	Duration time.Duration
	// Skipped indicates if execution was skipped (fallback).
	Skipped bool
	// SkipReason is the reason for skipping.
	SkipReason string
	// Retries is the number of retries performed.
	Retries int
}

// New creates a new Runner instance with default options.
func New() *Runner {
	return NewWithOptions(DefaultOptions())
}

// NewWithOptions creates a new Runner instance with the given options.
func NewWithOptions(opts *Options) *Runner {
	if opts == nil {
		opts = DefaultOptions()
	}

	r := &Runner{
		opts:         opts,
		configLoader: config.NewLoader(),
		skillLoader:  skill.NewLoader(),
		state:        StateUninitialized,
		shutdownCh:   make(chan struct{}),
		signalChan:   make(chan os.Signal, 1),
	}

	// Initialize watchdog with default config
	r.watchdog = NewWatchdog()
	r.retryExecutor = NewRetryExecutor(DefaultRetryPolicy())
	r.fallback = NewFallbackHandler()
	r.processPool = NewProcessPool()

	return r
}

// Bootstrap initializes the runner.
// This performs parallel initialization of config, platform, and skills.
func (r *Runner) Bootstrap(ctx context.Context) error {
	r.mu.Lock()
	if r.state != StateUninitialized && r.state != StateStopped {
		r.mu.Unlock()
		return fmt.Errorf("cannot bootstrap: runner is in state %s", r.state)
	}
	r.state = StateInitializing
	r.mu.Unlock()

	start := time.Now()
	r.metrics = &BootstrapMetrics{StartTime: start}

	// Phase 1: Parallel quick init
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// 1.1 Load config
	configStart := time.Now()
	wg.Add(1)
	go func() {
		defer wg.Done()
		cfg, err := r.loadConfig()
		if err != nil {
			errChan <- fmt.Errorf("config load failed: %w", err)
			return
		}
		r.config = cfg
		r.metrics.ConfigLoad = time.Since(configStart)
	}()

	// 1.2 Validate workspace
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := r.validateWorkspace(); err != nil {
			// Non-fatal, just warn
			if r.opts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}()

	// 1.3 Scan skills
	skillStart := time.Now()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := r.scanSkills(); err != nil {
			// Non-fatal for bootstrap
			if r.opts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: skill scan failed: %v\n", err)
			}
		}
		r.metrics.SkillScan = time.Since(skillStart)
	}()

	wg.Wait()
	close(errChan)

	// Check for critical errors
	for err := range errChan {
		if err != nil {
			r.mu.Lock()
			r.state = StateUninitialized
			r.mu.Unlock()
			return err
		}
	}

	// Phase 2: Optional pre-warm Claude
	if r.opts.PreWarmClaude {
		warmStart := time.Now()
		go func() {
			if err := r.processPool.Warmup(ctx); err != nil {
				if r.opts.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: Claude warmup failed: %v\n", err)
				}
			}
			r.metrics.ClaudeWarmup = time.Since(warmStart)
		}()
	}

	// Setup signal handling
	r.setupSignalHandler()

	r.bootstrapTime = time.Since(start)
	r.metrics.TotalTime = r.bootstrapTime

	// Log bootstrap time
	if r.bootstrapTime > 3*time.Second && r.opts.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Bootstrap took %v (target: <3s)\n", r.bootstrapTime)
	}

	r.mu.Lock()
	r.state = StateReady
	r.mu.Unlock()

	return nil
}

// loadConfig loads the configuration with precedence.
func (r *Runner) loadConfig() (*config.Config, error) {
	if r.opts.WorkDir != "" {
		r.configLoader = r.configLoader.WithProjectRoot(r.opts.WorkDir)
	}
	return r.configLoader.Load()
}

// validateWorkspace validates the current workspace.
func (r *Runner) validateWorkspace() error {
	// Check if .git exists
	workDir := r.opts.WorkDir
	if workDir == "" {
		workDir = "."
	}

	gitDir := workDir + "/.git"
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return ErrWorkspaceNotGit
	}

	// Check for CLAUDE.md (warning only)
	claudeMD := workDir + "/CLAUDE.md"
	if _, err := os.Stat(claudeMD); os.IsNotExist(err) {
		if r.opts.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: CLAUDE.md not found\n")
		}
	}

	return nil
}

// scanSkills scans for available skills.
func (r *Runner) scanSkills() error {
	for _, dir := range r.opts.SkillDirs {
		if err := r.skillLoader.ScanDirectory(dir); err != nil {
			// Continue scanning other directories
			continue
		}
	}
	return nil
}

// setupSignalHandler sets up signal handling for graceful shutdown.
func (r *Runner) setupSignalHandler() {
	signal.Notify(r.signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-r.signalChan:
			if r.opts.Verbose {
				fmt.Fprintf(os.Stderr, "\nReceived signal: %v, initiating graceful shutdown...\n", sig)
			}
			ctx, cancel := context.WithTimeout(context.Background(), r.opts.GracefulTimeout)
			defer cancel()
			_ = r.Shutdown(ctx)
		case <-r.shutdownCh:
			return
		}
	}()
}

// Run executes the AI analysis with the given request.
func (r *Runner) Run(ctx context.Context, req *RunRequest) (*RunResult, error) {
	r.mu.RLock()
	if r.state != StateReady {
		r.mu.RUnlock()
		return nil, fmt.Errorf("%w: runner is in state %s", ErrNotInitialized, r.state)
	}
	r.mu.RUnlock()

	r.mu.Lock()
	r.state = StateRunning
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		if r.state == StateRunning {
			r.state = StateReady
		}
		r.mu.Unlock()
	}()

	start := time.Now()
	result := &RunResult{}

	// Load the skill
	sk, err := r.skillLoader.Load(req.SkillName)
	if err != nil {
		result.Error = fmt.Errorf("%w: %s", ErrSkillNotFound, req.SkillName)
		result.ExitCode = ExitInfraError
		result.Duration = time.Since(start)
		return result, result.Error
	}

	// Resolve input values
	inputs, err := sk.ResolveInputValues(req.Inputs)
	if err != nil {
		result.Error = err
		result.ExitCode = ExitInfraError
		result.Duration = time.Since(start)
		return result, err
	}

	// Create execution context with timeout
	timeout := req.Timeout
	if timeout == 0 && r.config != nil {
		timeout = r.config.Claude.Timeout
	}
	if timeout == 0 {
		timeout = 5 * time.Minute // Default timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute with retry
	var lastErr error
	for attempt := 0; attempt <= r.retryExecutor.policy.MaxRetries; attempt++ {
		result.Retries = attempt

		if attempt > 0 {
			delay := r.retryExecutor.CalculateDelay(attempt)
			select {
			case <-time.After(delay):
			case <-execCtx.Done():
				result.Error = ErrTimeout
				result.ExitCode = ExitTimeout
				result.Duration = time.Since(start)
				return result, result.Error
			}
		}

		output, execErr := r.executeSkill(execCtx, sk, inputs, req.DryRun)
		if execErr == nil {
			result.Output = output
			result.ExitCode = ExitSuccess
			result.Duration = time.Since(start)
			return result, nil
		}

		lastErr = execErr

		// Check if error is retryable
		classified := ClassifyError(execErr)
		if !classified.Retryable {
			// Handle fallback
			fallbackResult := r.fallback.Handle(execCtx, classified, req)
			if fallbackResult != nil {
				result.Skipped = fallbackResult.Skipped
				result.SkipReason = fallbackResult.SkipReason
				result.Output = fallbackResult.Output
				result.ExitCode = ExitSuccess
				result.Duration = time.Since(start)
				return result, nil
			}
			break
		}
	}

	result.Error = fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
	result.ExitCode = ExitClaudeError
	result.Duration = time.Since(start)
	return result, result.Error
}

// executeSkill executes a single skill.
func (r *Runner) executeSkill(ctx context.Context, sk *skill.Skill, inputs map[string]any, dryRun bool) (string, error) {
	// Inject inputs into prompt
	injector := skill.NewInjector()
	prompt, err := injector.Inject(sk.Prompt, inputs)
	if err != nil {
		return "", err
	}

	// Create and start Claude process
	process := NewClaudeProcess(r.buildClaudeArgs(sk))
	if err := process.Start(ctx); err != nil {
		return "", err
	}

	// Register process for shutdown management
	r.mu.Lock()
	r.process = process
	r.mu.Unlock()

	defer func() {
		_ = process.Stop()
		r.mu.Lock()
		r.process = nil
		r.mu.Unlock()
	}()

	// Write prompt to stdin
	if err := process.WritePrompt(prompt); err != nil {
		return "", err
	}

	// Wait for completion and read output
	output, err := process.Wait(ctx)
	if err != nil {
		return "", err
	}

	return output, nil
}

// buildClaudeArgs builds Claude CLI arguments.
func (r *Runner) buildClaudeArgs(sk *skill.Skill) []string {
	args := []string{"-p"} // Print/headless mode

	// Add dangerous permissions flag only if explicitly configured
	// WARNING: This bypasses Claude's permission prompts - only enable in trusted CI environments
	if r.config != nil && r.config.Claude.SkipPermissions {
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

// Shutdown gracefully stops the runner.
func (r *Runner) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	if r.state == StateStopped || r.state == StateShuttingDown {
		r.mu.Unlock()
		return nil
	}
	r.state = StateShuttingDown
	r.mu.Unlock()

	// Signal shutdown to signal handler goroutine
	close(r.shutdownCh)

	// Stop signal handling
	signal.Stop(r.signalChan)

	// Stop any running process
	if r.process != nil && r.process.IsRunning() {
		if err := r.process.Stop(); err != nil {
			if r.opts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: error stopping process: %v\n", err)
			}
		}

		// Wait for process to exit with timeout
		done := make(chan struct{})
		go func() {
			_, _ = r.process.Wait(context.Background())
			close(done)
		}()

		select {
		case <-done:
			// Process exited cleanly
		case <-ctx.Done():
			// Force kill if timeout
			_ = r.process.Kill()
			return ErrShutdownTimeout
		}
	}

	r.mu.Lock()
	r.state = StateStopped
	r.mu.Unlock()

	return nil
}

// State returns the current runner state.
func (r *Runner) State() State {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// Config returns the loaded configuration.
func (r *Runner) Config() *config.Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// BootstrapMetrics returns the bootstrap timing metrics.
func (r *Runner) BootstrapMetrics() *BootstrapMetrics {
	return r.metrics
}
