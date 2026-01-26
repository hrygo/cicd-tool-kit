// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Registry manages integration hooks.
// Implements SPEC-HOOKS-01: Integration Hooks
type Registry struct {
	mu       sync.RWMutex
	hooks    map[EventType][]*Hook
	ctx      context.Context
	cancel   context.CancelFunc
	stateDir string
}

// Hook represents an integration hook.
type Hook struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Event    EventType   `json:"event"`
	Command  string      `json:"command,omitempty"`
	Handler  HandlerFunc `json:"-"`
	Enabled  bool        `json:"enabled"`
	Timeout  int         `json:"timeout"` // seconds
	Metadata Metadata     `json:"metadata"`
}

// Metadata for hooks.
type Metadata map[string]string

// EventType represents the event that triggers a hook.
type EventType string

const (
	EventPreBuild      EventType = "pre_build"
	EventPostBuild     EventType = "post_build"
	EventPreTest       EventType = "pre_test"
	EventPostTest      EventType = "post_test"
	EventPreDeploy     EventType = "pre_deploy"
	EventPostDeploy    EventType = "post_deploy"
	EventPreCommit     EventType = "pre_commit"
	EventPostCommit    EventType = "post_commit"
	EventPrePush       EventType = "pre_push"
	EventPostPush      EventType = "post_push"
	EventOnFailure     EventType = "on_failure"
	EventOnSuccess     EventType = "on_success"
	EventOnAlert       EventType = "on_alert"
	EventCustom        EventType = "custom"
)

// HandlerFunc is the function called when a hook is triggered.
type HandlerFunc func(ctx context.Context, event *Event) error

// Event represents a hook event.
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]any         `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
	Context   context.Context        `json:"-"`
}

// Result represents the result of a hook execution.
type Result struct {
	HookID    string        `json:"hook_id"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  int64         `json:"duration_ms"`
	Output    string        `json:"output,omitempty"`
}

// NewRegistry creates a new hook registry.
func NewRegistry(stateDir string) (*Registry, error) {
	if stateDir == "" {
		homeDir, _ := os.UserHomeDir()
		stateDir = filepath.Join(homeDir, ".cicd-ai-toolkit", "hooks")
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	reg := &Registry{
		hooks:    make(map[EventType][]*Hook),
		ctx:      ctx,
		cancel:   cancel,
		stateDir: stateDir,
	}

	// Load hooks from disk
	if err := reg.Load(); err != nil {
		// Continue without loading existing hooks
	}

	return reg, nil
}

// Register registers a new hook.
func (r *Registry) Register(hook *Hook) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if hook.ID == "" {
		hook.ID = generateHookID()
	}

	if hook.Timeout == 0 {
		hook.Timeout = 30 // default 30 seconds
	}

	r.hooks[hook.Event] = append(r.hooks[hook.Event], hook)

	return r.Save()
}

// Unregister removes a hook.
func (r *Registry) Unregister(hookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for eventType, hooks := range r.hooks {
		filtered := make([]*Hook, 0)
		for _, h := range hooks {
			if h.ID != hookID {
				filtered = append(filtered, h)
			}
		}
		r.hooks[eventType] = filtered
	}

	return r.Save()
}

// Trigger triggers all hooks for an event type.
func (r *Registry) Trigger(eventType EventType, data map[string]any) []*Result {
	r.mu.RLock()
	hooks := r.hooks[eventType]
	r.mu.RUnlock()

	results := make([]*Result, 0)

	for _, hook := range hooks {
		if !hook.Enabled {
			continue
		}

		result := r.executeHook(hook, data)
		results = append(results, result)
	}

	return results
}

// TriggerAsync triggers hooks asynchronously.
func (r *Registry) TriggerAsync(eventType EventType, data map[string]any) <-chan *Result {
	resultChan := make(chan *Result, 10)

	go func() {
		defer close(resultChan)

		r.mu.RLock()
		hooks := r.hooks[eventType]
		r.mu.RUnlock()

		for _, hook := range hooks {
			if !hook.Enabled {
				continue
			}

			result := r.executeHook(hook, data)
			resultChan <- result
		}
	}()

	return resultChan
}

// executeHook executes a single hook.
func (r *Registry) executeHook(hook *Hook, data map[string]any) *Result {
	startTime := currentTimeMillis()

	result := &Result{
		HookID:  hook.ID,
		Success: false,
	}

	event := &Event{
		Type:      hook.Event,
		Timestamp: startTime,
		Data:      data,
		Metadata:  make(map[string]string),
		Context:   r.ctx,
	}

	var err error

	// Use handler function if provided
	if hook.Handler != nil {
		err = hook.Handler(r.ctx, event)
		result.Output = fmt.Sprintf("handler executed")
	} else if hook.Command != "" {
		// Execute command
		result.Output, err = r.executeCommand(hook.Command)
	} else {
		err = fmt.Errorf("no handler or command specified")
	}

	result.Duration = currentTimeMillis() - startTime
	result.Success = (err == nil)

	if err != nil {
		result.Error = err.Error()
	}

	return result
}

// executeCommand executes a hook command.
func (r *Registry) executeCommand(command string) (string, error) {
	// In production, this would execute the command safely
	// For now, return a placeholder
	return fmt.Sprintf("executed: %s", command), nil
}

// GetHooks returns all hooks for an event type.
func (r *Registry) GetHooks(eventType EventType) []*Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks := r.hooks[eventType]
	result := make([]*Hook, len(hooks))
	copy(result, hooks)
	return result
}

// GetAllHooks returns all registered hooks.
func (r *Registry) GetAllHooks() []*Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make([]*Hook, 0)
	for _, hooks := range r.hooks {
		all = append(all, hooks...)
	}
	return all
}

// GetHook retrieves a hook by ID.
func (r *Registry) GetHook(id string) (*Hook, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, hooks := range r.hooks {
		for _, hook := range hooks {
			if hook.ID == id {
				return hook, true
			}
		}
	}

	return nil, false
}

// Enable enables a hook.
func (r *Registry) Enable(id string) error {
	hook, ok := r.GetHook(id)
	if !ok {
		return fmt.Errorf("hook not found: %s", id)
	}

	hook.Enabled = true
	return r.Save()
}

// Disable disables a hook.
func (r *Registry) Disable(id string) error {
	hook, ok := r.GetHook(id)
	if !ok {
		return fmt.Errorf("hook not found: %s", id)
	}

	hook.Enabled = false
	return r.Save()
}

// Save saves hooks to disk.
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.hooks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(r.stateDir, "hooks.json"), data, 0644)
}

// Load loads hooks from disk.
func (r *Registry) Load() error {
	path := filepath.Join(r.stateDir, "hooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &r.hooks)
}

// Close closes the registry.
func (r *Registry) Close() error {
	r.cancel()
	return r.Save()
}

// currentTimeMillis returns current time in milliseconds.
func currentTimeMillis() int64 {
	return 0 // Placeholder
}

// generateHookID generates a unique hook ID.
func generateHookID() string {
	return fmt.Sprintf("hook-%d", currentTimeMillis())
}

// Builder helps build hooks.
type Builder struct {
	hook *Hook
}

// NewBuilder creates a new hook builder.
func NewBuilder(name string, eventType EventType) *Builder {
	return &Builder{
		hook: &Hook{
			Name:    name,
			Event:   eventType,
			Enabled: true,
			Timeout: 30,
			Metadata: make(Metadata),
		},
	}
}

// WithCommand sets the command to execute.
func (b *Builder) WithCommand(command string) *Builder {
	b.hook.Command = command
	return b
}

// WithHandler sets the handler function.
func (b *Builder) WithHandler(handler HandlerFunc) *Builder {
	b.hook.Handler = handler
	return b
}

// WithTimeout sets the timeout.
func (b *Builder) WithTimeout(timeout int) *Builder {
	b.hook.Timeout = timeout
	return b
}

// WithMetadata adds metadata.
func (b *Builder) WithMetadata(key, value string) *Builder {
	if b.hook.Metadata == nil {
		b.hook.Metadata = make(Metadata)
	}
	b.hook.Metadata[key] = value
	return b
}

// Disabled creates the hook as disabled.
func (b *Builder) Disabled() *Builder {
	b.hook.Enabled = false
	return b
}

// Build creates the hook.
func (b *Builder) Build() *Hook {
	return b.hook
}

// Common hook builders

// PreBuild creates a pre-build hook.
func PreBuild(name string) *Builder {
	return NewBuilder(name, EventPreBuild)
}

// PostBuild creates a post-build hook.
func PostBuild(name string) *Builder {
	return NewBuilder(name, EventPostBuild)
}

// PreTest creates a pre-test hook.
func PreTest(name string) *Builder {
	return NewBuilder(name, EventPreTest)
}

// PostTest creates a post-test hook.
func PostTest(name string) *Builder {
	return NewBuilder(name, EventPostTest)
}

// PreDeploy creates a pre-deploy hook.
func PreDeploy(name string) *Builder {
	return NewBuilder(name, EventPreDeploy)
}

// PostDeploy creates a post-deploy hook.
func PostDeploy(name string) *Builder {
	return NewBuilder(name, EventPostDeploy)
}

// OnFailure creates a failure hook.
func OnFailure(name string) *Builder {
	return NewBuilder(name, EventOnFailure)
}

// OnSuccess creates a success hook.
func OnSuccess(name string) *Builder {
	return NewBuilder(name, EventOnSuccess)
}

// Chain creates a chain of hooks.
type Chain struct {
	hooks []*Hook
}

// NewChain creates a new hook chain.
func NewChain() *Chain {
	return &Chain{
		hooks: make([]*Hook, 0),
	}
}

// Add adds a hook to the chain.
func (c *Chain) Add(hook *Hook) *Chain {
	c.hooks = append(c.hooks, hook)
	return c
}

// Execute executes all hooks in the chain.
func (c *Chain) Execute(ctx context.Context, data map[string]any) []*Result {
	results := make([]*Result, 0)

	for _, hook := range c.hooks {
		if !hook.Enabled {
			continue
		}

		// Create a simple registry-like execution
		event := &Event{
			Type: hook.Event,
			Data: data,
			Context: ctx,
		}

		startTime := currentTimeMillis()
		result := &Result{
			HookID:  hook.ID,
			Success: false,
		}

		var err error
		if hook.Handler != nil {
			err = hook.Handler(ctx, event)
		}

		result.Duration = currentTimeMillis() - startTime
		result.Success = (err == nil)

		if err != nil {
			result.Error = err.Error()
		}

		results = append(results, result)

		// Stop on failure
		if !result.Success {
			break
		}
	}

	return results
}
