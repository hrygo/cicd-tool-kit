// Package context provides context utilities with proper resource cleanup
package context

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"
)

// signalContext provides a context that cancels on OS signals.
// It properly manages goroutine lifecycle to prevent leaks.
type signalContext struct {
	context.Context

	cancel   context.CancelFunc
	stopOnce sync.Once
	stopCh   chan struct{}
}

// Done returns the channel that closes when the context is cancelled
// or when a signal is received.
func (sc *signalContext) Done() <-chan struct{} {
	return sc.Context.Done()
}

// stop stops the signalContext and releases all resources.
// It can be called multiple times safely.
func (sc *signalContext) stop() {
	sc.stopOnce.Do(func() {
		// Cancel the parent context first
		sc.cancel()
		// Close the stop channel to signal the goroutine to exit
		close(sc.stopCh)
	})
}

// WithSignal creates a new context that cancels when any of the specified
// signals are received. The returned cancel function must be called to
// release resources and prevent goroutine leaks.
//
// Example:
//
//	ctx, cancel := WithSignal(context.Background(), os.Interrupt)
//	defer cancel() // Important: always call cancel to prevent goroutine leaks
func WithSignal(parent context.Context, sigs ...os.Signal) (context.Context, context.CancelFunc) {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(parent)

	sc := &signalContext{
		Context: ctx,
		cancel:  cancel,
		stopCh:  make(chan struct{}),
	}

	// Create a buffered channel for signals to prevent blocking
	// if signals arrive before we start listening
	ch := make(chan os.Signal, len(sigs))
	signal.Notify(ch, sigs...)

	// Start a goroutine to watch for signals
	go func() {
		select {
		case <-ch:
			// Signal received - cancel the context
			cancel()
		case <-sc.stopCh:
			// Context stopped - exit goroutine
			return
		case <-ctx.Done():
			// Parent context cancelled - exit goroutine
			return
		}
	}()

	// Return a cancel function that properly cleans up
	return sc, sc.stop
}

// WithSignalTimeout creates a context that cancels on signal or timeout.
// This is useful when you want to ensure the context doesn't block forever
// waiting for signals that may never arrive.
//
// The timeout ensures that even if no signals are received, the context
// will eventually expire and resources will be released.
func WithSignalTimeout(parent context.Context, timeout time.Duration, sigs ...os.Signal) (context.Context, context.CancelFunc) {
	// Create a context with timeout first
	ctx, cancel := context.WithTimeout(parent, timeout)

	sc := &signalContext{
		Context: ctx,
		cancel:  cancel,
		stopCh:  make(chan struct{}),
	}

	// Create a buffered channel for signals
	ch := make(chan os.Signal, len(sigs))
	signal.Notify(ch, sigs...)

	// Start a goroutine to watch for signals
	go func() {
		select {
		case <-ch:
			// Signal received - cancel the context
			cancel()
		case <-sc.stopCh:
			// Context stopped - exit goroutine
			return
		case <-ctx.Done():
			// Context cancelled or timed out - exit goroutine
			return
		}
	}()

	return sc, sc.stop
}
