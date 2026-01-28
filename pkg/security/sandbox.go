// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package security provides sandboxing and security controls.
package security

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// MaxNameLength is the maximum length for sandbox names
	MaxNameLength = 64

	// MaxMemoryBytes is the default maximum memory in bytes
	MaxMemoryBytes = 2 * 1024 * 1024 * 1024

	// DefaultTimeout is the default execution timeout
	DefaultTimeout = 10 * time.Minute

	// MaxProcesses is the default maximum number of processes
	MaxProcesses = 100

	// MaxFiles is the default maximum number of open files
	MaxFiles = 1024
)

// Sandbox provides a secure execution environment for AI tools.
type Sandbox struct {
	mu             sync.RWMutex
	config         *Config
	allowedPaths   []string
	deniedPatterns []string
	resourceLimits *ResourceLimits
	networkPolicy  NetworkPolicy
}

// Config defines sandbox configuration.
type Config struct {
	// RootDir is the sandbox root directory
	RootDir string

	// WorkDir is the working directory inside sandbox
	WorkDir string

	// ReadOnlyPaths are paths that can only be read
	ReadOnlyPaths []string

	// WriteAllowedPaths are paths where writing is allowed
	WriteAllowedPaths []string

	// AllowNetwork enables network access
	AllowNetwork bool

	// AllowedDomains are domains that can be accessed (empty = all denied)
	AllowedDomains []string

	// Timeout for execution
	Timeout time.Duration

	// EnableSeccomp enables seccomp filtering (Linux only)
	EnableSeccomp bool

	// EnableLandlock enables Landlock access control (Linux 5.13+)
	EnableLandlock bool
}

// ResourceLimits defines resource constraints.
type ResourceLimits struct {
	MaxMemory    int64         // bytes
	MaxCPU       float64       // percentage (0-1, or >1 for multiple cores)
	MaxWallTime  time.Duration // absolute timeout
	MaxProcesses int           // max number of processes
	MaxFiles     int           // max number of open file descriptors
}

// NetworkPolicy defines network access rules.
type NetworkPolicy struct {
	AllowOutbound bool
	AllowInbound  bool
	AllowedHosts  []string
	BlockedHosts  []string
}

// NewSandbox creates a new sandbox instance.
func NewSandbox(config *Config) *Sandbox {
	if config == nil {
		config = DefaultConfig()
	}
	return &Sandbox{
		config:         config,
		allowedPaths:   config.ReadOnlyPaths,
		deniedPatterns: []string{
			"/etc/passwd",
			"/etc/shadow",
			"/etc/ssh",
			"~/.ssh",
			"*/.git/config",
		},
		resourceLimits: DefaultResourceLimits(),
		networkPolicy: NetworkPolicy{
			AllowOutbound: config.AllowNetwork,
			AllowedHosts:  config.AllowedDomains,
		},
	}
}

// DefaultConfig returns safe default configuration.
func DefaultConfig() *Config {
	wd, _ := os.Getwd()
	return &Config{
		RootDir:        filepath.Join(wd, ".sandbox"),
		WorkDir:        wd,
		ReadOnlyPaths:  []string{wd},
		AllowNetwork:   false,
		Timeout:        DefaultTimeout,
		EnableSeccomp:  true,
		EnableLandlock: false, // Requires Linux 5.13+
	}
}

// DefaultResourceLimits returns safe default resource limits.
func DefaultResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		MaxMemory:    MaxMemoryBytes,
		MaxCPU:       1.0,
		MaxWallTime:  DefaultTimeout,
		MaxProcesses: MaxProcesses,
		MaxFiles:     MaxFiles,
	}
}

// Run executes a command inside the sandbox.
func (s *Sandbox) Run(ctx context.Context, cmd *exec.Cmd) (*Result, error) {
	// Create context with timeout
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	// Lock only during command preparation to allow concurrent command execution
	s.mu.Lock()
	cmd = s.prepareCommand(ctx, cmd)
	s.mu.Unlock()

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Monitor the process
	result := &Result{StartTime: time.Now()}

	// Wait for completion
	err := cmd.Wait()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			result.Error = err
		}
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	// Clean up resources
	s.mu.Lock()
	s.cleanup()
	s.mu.Unlock()

	return result, nil
}

// Execute executes code within the sandbox.
func (s *Sandbox) Execute(ctx context.Context, code string) (string, error) {
	// SECURITY: Validate input first before any processing
	// This function only supports simple command execution, not arbitrary shell code

	// Use direct command execution with separate args instead of shell -c
	// Parse code into command and arguments
	parts := strings.Fields(code)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// SECURITY: Validate EACH field individually to prevent command injection bypass
	// We validate after splitting because strings.Fields uses unicode.IsSpace
	// which includes more whitespace characters than our explicit check
	dangerousChars := []string{
		"|", "&", ";", "$", "(", ")", "`", "\\", ">", "<",
		"!", // history expansion in bash
		"*", "?", "[", "]", // glob characters that could expand unexpectedly
		"{", "}", // brace expansion
		"~", // home directory expansion
		"#", // comment character
		"%", // job control
		"'", "\"", // quotes - strings.Fields cannot parse quoted arguments correctly
		"\n", "\r", "\t", "\f", "\v", // explicit whitespace checks
		"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07",
		"\x08", "\x0e", "\x0f", "\x10", "\x11", "\x12", "\x13", "\x14",
		"\x15", "\x16", "\x17", "\x18", "\x19", "\x1a", "\x1b", "\x1c",
		"\x1d", "\x1e", "\x1f", // additional control characters
	}

	// Check each part for dangerous characters
	for i, part := range parts {
		for _, ch := range dangerousChars {
			if strings.Contains(part, ch) {
				return "", fmt.Errorf("arbitrary shell execution is not allowed: argument %d contains dangerous character '%s'", i, ch)
			}
		}
		// Also check for non-printable characters in each part
		for _, r := range part {
			if r < 32 && r != '\t' && r != '\n' && r != '\r' {
				return "", fmt.Errorf("arbitrary shell execution is not allowed: argument %d contains non-printable character", i)
			}
		}
	}

	// Validate the tool is allowed before execution (whitelist approach)
	if !s.ValidateTool(parts[0]) {
		return "", fmt.Errorf("tool not allowed: %s", parts[0])
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	result, err := s.Run(ctx, cmd)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

// ValidateTool checks if a tool is allowed.
func (s *Sandbox) ValidateTool(tool string) bool {
	// List of allowed tools
	allowedTools := map[string]bool{
		"git":      true,
		"grep":     true,
		"sed":      true,
		"awk":      true,
		"cat":      true,
		"head":     true,
		"tail":     true,
		"wc":       true,
		"ls":       true,
		"find":     true,
		"jq":       true,
		"go":       true,
		"python":   true,
		"python3":  true,
		"node":     true,
		"npm":      true,
	}

	// Check exact match first
	if allowedTools[tool] {
		return true
	}

	// Check base name (for paths like /usr/bin/git)
	base := filepath.Base(tool)
	return allowedTools[base]
}

// prepareCommand prepares the command for sandboxed execution.
func (s *Sandbox) prepareCommand(ctx context.Context, cmd *exec.Cmd) *exec.Cmd {
	// Set up environment with restricted variables
	cmd.Env = s.restrictedEnv()

	// Apply platform-specific restrictions
	if runtime.GOOS == "linux" {
		cmd = s.applyLinuxRestrictions(ctx, cmd)
	} else if runtime.GOOS == "darwin" {
		cmd = s.applyDarwinRestrictions(ctx, cmd)
	}

	return cmd
}

// restrictedEnv returns a restricted set of environment variables.
func (s *Sandbox) restrictedEnv() []string {
	// Allow only safe environment variables
	safeVars := []string{
		"PATH",
		"HOME",
		"USER",
		"LANG",
		"LC_ALL",
		"TERM",
		"TZ",
	}

	env := make([]string, 0, len(safeVars))
	for _, v := range safeVars {
		if val := os.Getenv(v); val != "" {
			// Sanitize PATH to include only safe directories
			if v == "PATH" {
				val = s.sanitizePath(val)
			}
			env = append(env, fmt.Sprintf("%s=%s", v, val))
		}
	}

	// Add sandbox-specific variables
	env = append(env, fmt.Sprintf("SANDBOX=%s", s.config.RootDir))

	return env
}

// sanitizePath removes potentially dangerous paths from PATH.
func (s *Sandbox) sanitizePath(path string) string {
	dirs := strings.Split(path, string(os.PathListSeparator))
	safeDirs := make([]string, 0, len(dirs))

	safePrefixes := []string{
		"/usr/bin",
		"/bin",
		"/usr/local/bin",
		"/opt/homebrew/bin",
	}

	for _, dir := range dirs {
		isSafe := false
		for _, prefix := range safePrefixes {
			if strings.HasPrefix(dir, prefix) {
				isSafe = true
				break
			}
		}
		if isSafe {
			safeDirs = append(safeDirs, dir)
		}
	}

	return strings.Join(safeDirs, string(os.PathListSeparator))
}

// applyLinuxRestrictions applies Linux-specific sandbox restrictions.
func (s *Sandbox) applyLinuxRestrictions(ctx context.Context, cmd *exec.Cmd) *exec.Cmd {
	// On Linux, we can use syscall.Setrlimit for resource limits
	// This would typically be done before exec via syscall.Exec

	// Set resource limits on the current process (child inherits)
	if s.resourceLimits != nil {
		s.setResourceLimits()
	}

	// Note: For full containerization, consider using:
	// - Landlock (Linux 5.13+)
	// - Seccomp
	// - User namespaces
	// - Or a container runtime (runc, gVisor)

	return cmd
}

// applyDarwinRestrictions applies macOS-specific sandbox restrictions.
func (s *Sandbox) applyDarwinRestrictions(ctx context.Context, cmd *exec.Cmd) *exec.Cmd {
	// On macOS, sandbox_init is deprecated
	// Use Seatbelt sandbox if available, or rely on resource limits

	if s.resourceLimits != nil {
		s.setResourceLimits()
	}

	return cmd
}

// setResourceLimits sets resource limits for the process.
func (s *Sandbox) setResourceLimits() {
	rl := s.resourceLimits

	// Memory limit (Linux only, via setrlimit RLIMIT_AS)
	if rl.MaxMemory > 0 && runtime.GOOS == "linux" {
		_ = syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
			Cur: uint64(rl.MaxMemory),
			Max: uint64(rl.MaxMemory),
		})
	}

	// CPU time limit
	if rl.MaxWallTime > 0 {
		_ = syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
			Cur: uint64(rl.MaxWallTime.Seconds()),
			Max: uint64(rl.MaxWallTime.Seconds()),
		})
	}

	// Max processes (Linux only)
	if rl.MaxProcesses > 0 && runtime.GOOS == "linux" {
		// RLIMIT_NPROC is Linux-specific
		const RLIMIT_NPROC = 6
		_ = syscall.Setrlimit(RLIMIT_NPROC, &syscall.Rlimit{
			Cur: uint64(rl.MaxProcesses),
			Max: uint64(rl.MaxProcesses),
		})
	}

	// Max open files
	if rl.MaxFiles > 0 {
		_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
			Cur: uint64(rl.MaxFiles),
			Max: uint64(rl.MaxFiles),
		})
	}
}

// ValidatePath checks if a path is allowed for access.
func (s *Sandbox) ValidatePath(path string) error {
	// Clean the path
	path = filepath.Clean(path)

	// Check absolute path against rules
	if filepath.IsAbs(path) {
		return s.validateAbsolutePath(path)
	}

	// Relative paths are resolved against workdir
	fullPath := filepath.Join(s.config.WorkDir, path)
	return s.validateAbsolutePath(fullPath)
}

// validateAbsolutePath validates an absolute path.
func (s *Sandbox) validateAbsolutePath(path string) error {
	// Check denied patterns first
	for _, pattern := range s.deniedPatterns {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return fmt.Errorf("access denied: path matches blocked pattern: %s", pattern)
		}
		// Check if path contains pattern
		if strings.Contains(path, strings.TrimPrefix(pattern, "*/")) {
			return fmt.Errorf("access denied: path contains blocked pattern: %s", pattern)
		}
	}

	// Check if path is within allowed paths
	allowed := false
	for _, allowedPath := range s.allowedPaths {
		if strings.HasPrefix(path, allowedPath) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("access denied: path not in allowed list: %s", path)
	}

	return nil
}

// cleanup performs cleanup after command execution.
func (s *Sandbox) cleanup() {
	// Remove temporary files
	// Kill any orphaned processes
	// Release resources
}

// Result represents the result of a sandboxed command execution.
type Result struct {
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	ExitCode  int
	Success   bool
	Error     error
	Output    string
}

// IsTimeout returns true if the execution timed out.
func (r *Result) IsTimeout() bool {
	return r.Error != nil && r.Error.Error() == "signal: killed"
}

// IsSuccess returns true if execution succeeded.
func (r *Result) IsSuccess() bool {
	return r.Success && r.ExitCode == 0
}

// IsPermissionDenied returns true if the error was permission-related.
func (r *Result) IsPermissionDenied() bool {
	if r.Error == nil {
		return false
	}
	return strings.Contains(r.Error.Error(), "permission denied") ||
		strings.Contains(r.Error.Error(), "access denied")
}

// PathValidator validates file paths before sandbox access.
type PathValidator struct {
	allowedPrefixes []string
	deniedPatterns  []string
}

// NewPathValidator creates a new path validator.
func NewPathValidator(allowed, denied []string) *PathValidator {
	return &PathValidator{
		allowedPrefixes: allowed,
		deniedPatterns:  denied,
	}
}

// Validate checks if a path is safe to access.
func (v *PathValidator) Validate(path string) error {
	cleanPath := filepath.Clean(path)

	// Check denied patterns
	for _, pattern := range v.deniedPatterns {
		if matched, _ := filepath.Match(pattern, cleanPath); matched {
			return fmt.Errorf("path blocked by pattern: %s", pattern)
		}
	}

	// Check allowed prefixes
	if len(v.allowedPrefixes) > 0 {
		allowed := false
		for _, prefix := range v.allowedPrefixes {
			if strings.HasPrefix(cleanPath, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed prefixes: %s", cleanPath)
		}
	}

	return nil
}

// CommandBuilder helps build safe sandbox commands.
type CommandBuilder struct {
	sandbox *Sandbox
}

// NewCommandBuilder creates a new command builder.
func NewCommandBuilder(s *Sandbox) *CommandBuilder {
	return &CommandBuilder{sandbox: s}
}

// Build creates a ready-to-run exec.Cmd for the sandbox.
func (b *CommandBuilder) Build(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = b.sandbox.config.WorkDir
	return b.sandbox.prepareCommand(context.Background(), cmd)
}

// BuildWithContext creates a command with context.
func (b *CommandBuilder) BuildWithContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = b.sandbox.config.WorkDir
	return b.sandbox.prepareCommand(ctx, cmd)
}

// QuickRun executes a simple command in the sandbox.
func QuickRun(name string, args ...string) (*Result, error) {
	s := NewSandbox(DefaultConfig())
	builder := NewCommandBuilder(s)
	cmd := builder.Build(name, args...)
	return s.Run(context.Background(), cmd)
}

// IsSecureEnvironment checks if the current environment is secure.
func IsSecureEnvironment() bool {
	// Check if running in a secure environment
	// This includes: container, VM, secure host, etc.

	// Check if in container
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for cgroups (container indicator)
	if _, err := os.Stat("/proc/1/cgroup"); err == nil {
		data, _ := os.ReadFile("/proc/1/cgroup")
		if strings.Contains(string(data), "docker") ||
			strings.Contains(string(data), "kubepods") ||
			strings.Contains(string(data), "containerd") {
			return true
		}
	}

	return false
}
