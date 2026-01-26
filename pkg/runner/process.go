// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ClaudeProcess manages a Claude subprocess.
type ClaudeProcess struct {
	mu sync.RWMutex

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	args    []string
	binary  string
	started bool
	exited  bool

	// Output buffers
	stdoutBuf bytes.Buffer
	stderrBuf bytes.Buffer

	// Wait channel
	waitCh   chan error
	exitCode int
}

// NewClaudeProcess creates a new Claude process with the given arguments.
func NewClaudeProcess(args []string) *ClaudeProcess {
	return &ClaudeProcess{
		args:   args,
		binary: "claude",
		waitCh: make(chan error, 1),
	}
}

// WithBinary sets a custom binary path for Claude.
func (p *ClaudeProcess) WithBinary(binary string) *ClaudeProcess {
	p.binary = binary
	return p
}

// Start starts the Claude process.
func (p *ClaudeProcess) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return ErrProcessAlreadyRun
	}

	// Check if claude binary exists
	if _, err := exec.LookPath(p.binary); err != nil {
		return ErrClaudeNotFound
	}

	// Create the command
	p.cmd = exec.CommandContext(ctx, p.binary, p.args...)

	// Set up pipes
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Claude process: %w", err)
	}

	p.started = true

	// Start goroutines to capture output
	go p.captureOutput(p.stdout, &p.stdoutBuf)
	go p.captureOutput(p.stderr, &p.stderrBuf)

	// Start goroutine to wait for process
	go func() {
		err := p.cmd.Wait()
		p.mu.Lock()
		p.exited = true
		if p.cmd.ProcessState != nil {
			p.exitCode = p.cmd.ProcessState.ExitCode()
		}
		p.mu.Unlock()
		p.waitCh <- err
	}()

	return nil
}

// captureOutput captures output from a reader into a buffer.
// Uses io.Copy instead of bufio.Scanner to avoid line length limitations.
func (p *ClaudeProcess) captureOutput(r io.Reader, buf *bytes.Buffer) {
	// Use a small buffer to copy incrementally with lock protection
	copyBuf := make([]byte, 32*1024) // 32KB chunks
	for {
		n, err := r.Read(copyBuf)
		if n > 0 {
			p.mu.Lock()
			buf.Write(copyBuf[:n])
			p.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
}

// WritePrompt writes the prompt to Claude's stdin and closes stdin.
func (p *ClaudeProcess) WritePrompt(prompt string) error {
	p.mu.RLock()
	if !p.started {
		p.mu.RUnlock()
		return ErrProcessNotRunning
	}
	stdin := p.stdin
	p.mu.RUnlock()

	if _, err := io.WriteString(stdin, prompt); err != nil {
		return fmt.Errorf("failed to write prompt: %w", err)
	}

	// Close stdin to signal end of input
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	return nil
}

// Wait waits for the process to complete and returns the output.
func (p *ClaudeProcess) Wait(ctx context.Context) (string, error) {
	select {
	case err := <-p.waitCh:
		p.mu.RLock()
		output := p.stdoutBuf.String()
		stderr := p.stderrBuf.String()
		exitCode := p.exitCode
		p.mu.RUnlock()

		if err != nil {
			// Check if it's a context timeout/cancellation
			if ctx.Err() != nil {
				return "", ErrTimeout
			}
			// Include stderr in error
			if stderr != "" {
				return output, fmt.Errorf("process failed (exit code %d): %s", exitCode, stderr)
			}
			return output, fmt.Errorf("process failed (exit code %d): %w", exitCode, err)
		}
		return output, nil

	case <-ctx.Done():
		// Context cancelled or timed out
		_ = p.Kill()
		return "", ErrTimeout
	}
}

// Stop gracefully stops the Claude process.
func (p *ClaudeProcess) Stop() error {
	p.mu.RLock()
	if !p.started || p.exited {
		p.mu.RUnlock()
		return nil
	}
	cmd := p.cmd
	p.mu.RUnlock()

	if cmd.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process might have already exited
		if !strings.Contains(err.Error(), "process already finished") {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}
	}

	return nil
}

// Kill forcefully kills the Claude process.
func (p *ClaudeProcess) Kill() error {
	p.mu.RLock()
	if !p.started || p.exited {
		p.mu.RUnlock()
		return nil
	}
	cmd := p.cmd
	p.mu.RUnlock()

	if cmd.Process == nil {
		return nil
	}

	// Send SIGKILL
	if err := cmd.Process.Kill(); err != nil {
		if !strings.Contains(err.Error(), "process already finished") {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	return nil
}

// IsRunning checks if the process is running.
func (p *ClaudeProcess) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started && !p.exited
}

// ExitCode returns the process exit code.
func (p *ClaudeProcess) ExitCode() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitCode
}

// Stdout returns the captured stdout.
func (p *ClaudeProcess) Stdout() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stdoutBuf.String()
}

// Stderr returns the captured stderr.
func (p *ClaudeProcess) Stderr() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stderrBuf.String()
}

// ProcessManager manages Claude subprocess lifecycle.
type ProcessManager struct {
	mu sync.RWMutex

	processes map[string]*ClaudeProcess
	binary    string
}

// NewProcessManager creates a new process manager.
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[string]*ClaudeProcess),
		binary:    "claude",
	}
}

// WithBinary sets a custom binary path.
func (pm *ProcessManager) WithBinary(binary string) *ProcessManager {
	pm.binary = binary
	return pm
}

// Start starts a new Claude process with the given ID and arguments.
func (pm *ProcessManager) Start(ctx context.Context, id string, args []string) (*ClaudeProcess, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if process already exists
	if _, exists := pm.processes[id]; exists {
		return nil, fmt.Errorf("process with ID %s already exists", id)
	}

	process := NewClaudeProcess(args).WithBinary(pm.binary)
	if err := process.Start(ctx); err != nil {
		return nil, err
	}

	pm.processes[id] = process
	return process, nil
}

// Get returns a process by ID.
func (pm *ProcessManager) Get(id string) (*ClaudeProcess, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.processes[id]
	return p, ok
}

// Stop stops a process by ID.
func (pm *ProcessManager) Stop(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	p, ok := pm.processes[id]
	if !ok {
		return fmt.Errorf("process with ID %s not found", id)
	}

	err := p.Stop()
	delete(pm.processes, id)
	return err
}

// StopAll stops all managed processes.
func (pm *ProcessManager) StopAll() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var lastErr error
	for id, p := range pm.processes {
		if err := p.Stop(); err != nil {
			lastErr = err
		}
		delete(pm.processes, id)
	}
	return lastErr
}

// IsRunning checks if a process with the given ID is running.
func (pm *ProcessManager) IsRunning(id string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, ok := pm.processes[id]
	if !ok {
		return false
	}
	return p.IsRunning()
}

// ProcessPool manages Claude process pre-warming.
type ProcessPool struct {
	mu sync.RWMutex

	warm    bool
	binary  string
	timeout time.Duration
}

// NewProcessPool creates a new process pool.
func NewProcessPool() *ProcessPool {
	return &ProcessPool{
		binary:  "claude",
		timeout: 30 * time.Second,
	}
}

// Warmup warms up the Claude process.
func (pp *ProcessPool) Warmup(ctx context.Context) error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.warm {
		return nil
	}

	// Check if claude binary exists
	if _, err := exec.LookPath(pp.binary); err != nil {
		return ErrClaudeNotFound
	}

	// Run a minimal command to warm up
	cmd := exec.CommandContext(ctx, pp.binary, "-p", "--help")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Claude warmup failed: %w", err)
	}

	duration := time.Since(start)
	if duration > 2*time.Second {
		fmt.Fprintf(os.Stderr, "Warning: Claude warmup took %v (target: <2s)\n", duration)
	}

	pp.warm = true
	return nil
}

// IsWarm returns true if the pool is warmed up.
func (pp *ProcessPool) IsWarm() bool {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	return pp.warm
}
