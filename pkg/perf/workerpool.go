// Package perf provides performance optimization utilities
package perf

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	// defaultQueueMultiplier is the multiplier for task queue size relative to maxWorkers
	defaultQueueMultiplier = 2
)

// WorkerPool manages a pool of goroutines for concurrent task execution
type WorkerPool struct {
	maxWorkers int
	taskQueue  chan func()
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	cancelOnce sync.Once
	stopped    atomic.Bool
	activeJobs atomic.Int32
}

// NewWorkerPool creates a new worker pool with the specified maximum number of workers
func NewWorkerPool(maxWorkers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan func(), maxWorkers*defaultQueueMultiplier),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker processes tasks from the queue
func (p *WorkerPool) worker(_ int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			p.activeJobs.Add(1)
			func() {
				// Ensure counter decrements even if task panics
				defer p.activeJobs.Add(-1)
				defer func() {
					if r := recover(); r != nil {
						// Log panic but don't crash the worker
					}
				}()
				task()
			}()
		}
	}
}

// Submit submits a task to the worker pool
// Returns false if the pool is closed, the task is nil, or the task queue is full
func (p *WorkerPool) Submit(task func()) bool {
	if task == nil {
		return false
	}
	// Check stopped flag first to avoid submitting to closed channel
	if p.stopped.Load() {
		return false
	}
	select {
	case <-p.ctx.Done():
		return false
	case p.taskQueue <- task:
		return true
	default:
		return false // Queue is full
	}
}

// SubmitWait submits a task and waits for it to complete
func (p *WorkerPool) SubmitWait(task func()) error {
	done := make(chan struct{})

	wrappedTask := func() {
		// Ensure done is closed even if task panics
		defer close(done)
		// Recover from panic to prevent deadlock
		defer func() {
			if r := recover(); r != nil {
				// Panic will be propagated by the worker's own recover
			}
		}()
		task()
	}

	if !p.Submit(wrappedTask) {
		return fmt.Errorf("worker pool queue is full")
	}

	select {
	case <-done:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool was stopped")
	}
}

// Stop stops the worker pool gracefully
// Safe to call multiple times - subsequent calls are no-ops
func (p *WorkerPool) Stop() {
	if !p.stopped.CompareAndSwap(false, true) {
		return // Already stopped
	}

	// Step 1: Cancel context to signal all workers to stop
	p.cancelOnce.Do(func() {
		p.cancel()
	})

	// Step 2: Wait for all workers to finish their current tasks
	p.wg.Wait()

	// Step 3: Close the task queue (no longer needed after workers are done)
	// This must be after wg.Wait() to avoid sending to closed channel
	close(p.taskQueue)
}

// ActiveJobs returns the number of currently active jobs
func (p *WorkerPool) ActiveJobs() int {
	return int(p.activeJobs.Load())
}

// QueueSize returns the current size of the task queue
func (p *WorkerPool) QueueSize() int {
	return len(p.taskQueue)
}

// Batch processes a batch of tasks concurrently
func (p *WorkerPool) Batch(tasks []func()) error {
	if len(tasks) == 0 {
		return nil
	}

	errCh := make(chan error, len(tasks))
	var wg sync.WaitGroup

	submittedCount := 0
	for _, task := range tasks {
		wg.Add(1)
		t := task
		if !p.Submit(func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					// Use select to avoid blocking if errCh is closed
					select {
					case errCh <- fmt.Errorf("task panic: %v", r):
					default:
					}
				}
			}()
			t()
		}) {
			wg.Done()
			// Wait for already submitted tasks to complete
			go func() {
				wg.Wait()
				close(errCh)
			}()
			return fmt.Errorf("failed to submit task to worker pool")
		}
		submittedCount++
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// Map applies a function to each element of a slice concurrently
func Map[T, R any](ctx context.Context, items []T, fn func(T) (R, error), concurrency int) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}

	if concurrency <= 0 {
		concurrency = 1
	}

	// Create a cancellable context to cancel remaining work on error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]R, len(items))
	errCh := make(chan error, len(items))
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()
			select {
			case sem <- struct{}{}: // Acquire
				defer func() { <-sem }() // Release
			case <-ctx.Done():
				return // Context cancelled, exit early
			}

			result, err := fn(it)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("error at index %d: %w", idx, err):
					cancel() // Cancel remaining work
				case <-ctx.Done():
				}
				return
			}

			results[idx] = result
		}(i, item)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return results, nil
}

// Filter filters a slice concurrently based on a predicate function
func Filter[T any](ctx context.Context, items []T, predicate func(T) (bool, error), concurrency int) ([]T, error) {
	if len(items) == 0 {
		return nil, nil
	}

	if concurrency <= 0 {
		concurrency = 1
	}

	// Create a cancellable context to cancel remaining work on error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		idx  int
		keep bool
		err  error
	}

	resultCh := make(chan result, len(items))
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return // Context cancelled, exit early
			}

			keep, err := predicate(it)
			select {
			case resultCh <- result{idx: idx, keep: keep, err: err}:
				if err != nil {
					cancel() // Cancel remaining work
				}
			case <-ctx.Done():
			}
		}(i, item)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	filtered := make([]T, 0)
	for res := range resultCh {
		if res.err != nil {
			return nil, fmt.Errorf("error at index %d: %w", res.idx, res.err)
		}
		if res.keep {
			filtered = append(filtered, items[res.idx])
		}
	}

	return filtered, nil
}

// Each applies a function to each element of a slice concurrently
func Each[T any](ctx context.Context, items []T, fn func(T) error, concurrency int) error {
	if len(items) == 0 {
		return nil
	}

	if concurrency <= 0 {
		concurrency = 1
	}

	// Create a cancellable context to cancel remaining work on error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(items))
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for _, item := range items {
		wg.Add(1)
		go func(it T) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return // Context cancelled, exit early
			}

			if err := fn(it); err != nil {
				select {
				case errCh <- err:
					cancel() // Cancel remaining work
				case <-ctx.Done():
				}
			}
		}(item)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Parallel executes multiple functions in parallel and returns their results
func Parallel[R any](ctx context.Context, fns ...func() (R, error)) ([]R, error) {
	if len(fns) == 0 {
		return nil, nil
	}

	// Create a cancellable context to cancel remaining work on error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		idx  int
		val  R
		err  error
	}

	resultCh := make(chan result, len(fns))

	for i, fn := range fns {
		go func(idx int, f func() (R, error)) {
			val, err := f()
			select {
			case resultCh <- result{idx: idx, val: val, err: err}:
				if err != nil {
					cancel() // Cancel remaining work
				}
			case <-ctx.Done():
			}
		}(i, fn)
	}

	results := make([]R, len(fns))
	for i := 0; i < len(fns); i++ {
		select {
		case res := <-resultCh:
			if res.err != nil {
				return nil, res.err
			}
			results[res.idx] = res.val
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return results, nil
}

// RateLimiter limits the rate of operations
type RateLimiter struct {
	sem   chan struct{}
	close chan struct{}
	once  sync.Once
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxConcurrent int) *RateLimiter {
	return &RateLimiter{
		sem:   make(chan struct{}, maxConcurrent),
		close: make(chan struct{}),
	}
}

// Do executes a function with rate limiting
func (r *RateLimiter) Do(fn func() error) error {
	select {
	case r.sem <- struct{}{}:
		defer func() { <-r.sem }()
		return fn()
	case <-r.close:
		return fmt.Errorf("rate limiter is closed")
	}
}

// Close closes the rate limiter
func (r *RateLimiter) Close() error {
	r.once.Do(func() {
		close(r.close)
	})
	return nil
}
