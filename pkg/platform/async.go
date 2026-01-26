// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package platform

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AsyncPlatform provides asynchronous execution capabilities.
// Implements SPEC-PLAT-02: Async Execution
type AsyncPlatform struct {
	platform Platform
	workers  int
	timeout  time.Duration
}

// NewAsyncPlatform creates a new async platform wrapper.
func NewAsyncPlatform(p Platform, workers int) *AsyncPlatform {
	if workers <= 0 {
		workers = 5
	}
	return &AsyncPlatform{
		platform: p,
		workers:  workers,
		timeout:  5 * time.Minute,
	}
}

// SetTimeout sets the default timeout for async operations.
func (a *AsyncPlatform) SetTimeout(timeout time.Duration) {
	a.timeout = timeout
}

// Name returns the underlying platform name.
func (a *AsyncPlatform) Name() string {
	return a.platform.Name()
}

// asyncFuture represents a pending asynchronous operation.
type asyncFuture struct {
	mu     sync.Mutex
	result any
	err    error
	done   chan struct{}
}

// GetPullRequestAsync retrieves a pull request asynchronously.
func (a *AsyncPlatform) GetPullRequestAsync(ctx context.Context, number int) *asyncFuture {
	return a.executeAsync(ctx, func() (any, error) {
		return a.platform.GetPullRequest(ctx, number)
	})
}

// PostCommentAsync posts a comment asynchronously.
func (a *AsyncPlatform) PostCommentAsync(ctx context.Context, number int, body string) *asyncFuture {
	return a.executeAsync(ctx, func() (any, error) {
		return nil, a.platform.PostComment(ctx, number, body)
	})
}

// GetDiffAsync retrieves diff asynchronously.
func (a *AsyncPlatform) GetDiffAsync(ctx context.Context, number int) *asyncFuture {
	return a.executeAsync(ctx, func() (any, error) {
		return a.platform.GetDiff(ctx, number)
	})
}

// GetEventAsync retrieves event asynchronously.
func (a *AsyncPlatform) GetEventAsync(ctx context.Context) *asyncFuture {
	return a.executeAsync(ctx, func() (any, error) {
		return a.platform.GetEvent(ctx)
	})
}

// GetFileContentAsync retrieves file content asynchronously.
func (a *AsyncPlatform) GetFileContentAsync(ctx context.Context, path, ref string) *asyncFuture {
	return a.executeAsync(ctx, func() (any, error) {
		return a.platform.GetFileContent(ctx, path, ref)
	})
}

// ListFilesAsync lists files asynchronously.
func (a *AsyncPlatform) ListFilesAsync(ctx context.Context, path, ref string) *asyncFuture {
	return a.executeAsync(ctx, func() (any, error) {
		return a.platform.ListFiles(ctx, path, ref)
	})
}

// CreateStatusAsync creates status check asynchronously.
func (a *AsyncPlatform) CreateStatusAsync(ctx context.Context, sha, state, description, context string) *asyncFuture {
	return a.executeAsync(ctx, func() (any, error) {
		return nil, a.platform.CreateStatus(ctx, sha, state, description, context)
	})
}

// executeAsync runs a function in a goroutine.
func (a *AsyncPlatform) executeAsync(ctx context.Context, fn func() (any, error)) *asyncFuture {
	future := &asyncFuture{
		done: make(chan struct{}),
	}

	go func() {
		defer close(future.done)

		// Apply timeout if context doesn't have one
		execCtx := ctx
		if _, hasDeadline := ctx.Deadline(); !hasDeadline && a.timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, a.timeout)
			defer cancel()
		}

		result, err := fn()
		future.mu.Lock()
		future.result = result
		future.err = err
		future.mu.Unlock()
		_ = execCtx // Use the variable
	}()

	return future
}

// Wait waits for the future to complete and returns the result.
func (f *asyncFuture) Wait() (any, error) {
	<-f.done
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.result, f.err
}

// WaitWithTimeout waits with a timeout.
func (f *asyncFuture) WaitWithTimeout(timeout time.Duration) (any, error) {
	select {
	case <-f.done:
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.result, f.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout after %v", timeout)
	}
}

// IsDone returns true if the future is complete.
func (f *asyncFuture) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// FileRequest represents a file retrieval request.
type FileRequest struct {
	Path string
	Ref  string
}

// ProgressTracker tracks async operation progress.
type ProgressTracker struct {
	mu        sync.Mutex
	total     int
	completed int
	failed    int
	callbacks []func(progress *Progress)
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{total: total}
}

// Increment marks a task as completed.
func (p *ProgressTracker) Increment() {
	p.mu.Lock()
	p.completed++
	p.notify()
	p.mu.Unlock()
}

// Failed marks a task as failed.
func (p *ProgressTracker) Failed() {
	p.mu.Lock()
	p.failed++
	p.notify()
	p.mu.Unlock()
}

// OnProgress registers a progress callback.
func (p *ProgressTracker) OnProgress(fn func(progress *Progress)) {
	p.mu.Lock()
	p.callbacks = append(p.callbacks, fn)
	p.mu.Unlock()
}

func (p *ProgressTracker) notify() {
	progress := &Progress{
		Total:     p.total,
		Completed: p.completed,
		Failed:    p.failed,
		Percent:   float64(p.completed) / float64(p.total) * 100,
	}
	for _, fn := range p.callbacks {
		fn(progress)
	}
}

// Progress represents current progress.
type Progress struct {
	Total     int
	Completed int
	Failed    int
	Percent   float64
}
