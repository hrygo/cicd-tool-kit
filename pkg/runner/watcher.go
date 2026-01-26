// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package runner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RetryPolicy defines the retry strategy.
type RetryPolicy struct {
	MaxRetries   int           // Maximum number of retries (default: 3)
	InitialDelay time.Duration // Initial delay between retries (default: 1s)
	MaxDelay     time.Duration // Maximum delay between retries (default: 10s)
	Multiplier   float64       // Delay multiplier for exponential backoff (default: 2.0)
}

// DefaultRetryPolicy returns the default retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryExecutor executes functions with retry logic.
type RetryExecutor struct {
	policy *RetryPolicy
}

// NewRetryExecutor creates a new retry executor with the given policy.
func NewRetryExecutor(policy *RetryPolicy) *RetryExecutor {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}
	return &RetryExecutor{policy: policy}
}

// Execute executes the given function with retry logic.
func (re *RetryExecutor) Execute(ctx context.Context, fn func() error) error {
	var lastErr error
	delay := re.policy.InitialDelay

	for attempt := 0; attempt <= re.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}

			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * re.policy.Multiplier)
			if delay > re.policy.MaxDelay {
				delay = re.policy.MaxDelay
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		classified := ClassifyError(err)
		if !classified.Retryable {
			return err
		}

		lastErr = err
	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// CalculateDelay calculates the delay for a given attempt number.
func (re *RetryExecutor) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	delay := re.policy.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * re.policy.Multiplier)
		if delay > re.policy.MaxDelay {
			return re.policy.MaxDelay
		}
	}
	return delay
}

// Watchdog monitors the runner for health and timeouts.
type Watchdog struct {
	mu sync.RWMutex

	// Configuration
	checkInterval time.Duration
	timeout       time.Duration

	// State
	running   bool
	lastCheck time.Time
	stopCh    chan struct{}

	// Callbacks
	onTimeout   func()
	onUnhealthy func(error)
}

// NewWatchdog creates a new watchdog.
func NewWatchdog() *Watchdog {
	return &Watchdog{
		checkInterval: 5 * time.Second,
		timeout:       5 * time.Minute,
		stopCh:        make(chan struct{}),
	}
}

// WithCheckInterval sets the health check interval.
func (w *Watchdog) WithCheckInterval(interval time.Duration) *Watchdog {
	w.checkInterval = interval
	return w
}

// WithTimeout sets the execution timeout.
func (w *Watchdog) WithTimeout(timeout time.Duration) *Watchdog {
	w.timeout = timeout
	return w
}

// OnTimeout sets the callback for timeout events.
func (w *Watchdog) OnTimeout(fn func()) *Watchdog {
	w.onTimeout = fn
	return w
}

// OnUnhealthy sets the callback for unhealthy events.
func (w *Watchdog) OnUnhealthy(fn func(error)) *Watchdog {
	w.onUnhealthy = fn
	return w
}

// Watch starts monitoring the runner.
func (w *Watchdog) Watch(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watchdog already running")
	}
	w.running = true
	w.lastCheck = time.Now()
	w.mu.Unlock()

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			w.stop()
			return ctx.Err()

		case <-w.stopCh:
			w.stop()
			return nil

		case <-ticker.C:
			// Check timeout
			elapsed := time.Since(startTime)
			if elapsed > w.timeout {
				if w.onTimeout != nil {
					w.onTimeout()
				}
				w.stop()
				return ErrTimeout
			}

			w.mu.Lock()
			w.lastCheck = time.Now()
			w.mu.Unlock()
		}
	}
}

// Stop stops the watchdog.
func (w *Watchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}
	w.running = false
	close(w.stopCh)
	// Recreate stopCh for potential reuse
	w.stopCh = make(chan struct{})
}

func (w *Watchdog) stop() {
	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
}

// IsRunning returns true if the watchdog is running.
func (w *Watchdog) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// WatchProcess monitors a specific process with timeout.
func WatchProcess(ctx context.Context, process *ClaudeProcess, timeout time.Duration, onTimeout func()) error {
	if process == nil {
		return fmt.Errorf("process is nil")
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			// Timeout reached
			if process.IsRunning() {
				if onTimeout != nil {
					onTimeout()
				}
				// Send SIGKILL
				if err := process.Kill(); err != nil {
					return fmt.Errorf("failed to kill process on timeout: %w", err)
				}
				return ErrTimeout
			}
			return nil

		default:
			// Check if process has exited
			if !process.IsRunning() {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
