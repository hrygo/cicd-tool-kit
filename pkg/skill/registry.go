// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package skill

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrInvalidSkill is returned when trying to register an invalid skill.
	ErrInvalidSkill = errors.New("invalid skill: name is required")
)

// Registry manages available skills.
// This will be fully implemented in SPEC-SKILL-01.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]*Skill
}

// NewRegistry creates a new skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*Skill),
	}
}

// Register registers a skill, returning an error if the skill is invalid.
// If a skill with the same name already exists, it will be replaced.
func (r *Registry) Register(s *Skill) error {
	if s == nil {
		return fmt.Errorf("cannot register nil skill")
	}
	if err := s.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[s.Name()] = s
	return nil
}

// RegisterAll registers multiple skills at once.
// Returns an error if any skill fails to validate, but may have partially registered skills.
func (r *Registry) RegisterAll(skills []*Skill) error {
	for _, s := range skills {
		if err := r.Register(s); err != nil {
			return err
		}
	}
	return nil
}

// Unregister removes a skill from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.skills, name)
}

// Get retrieves a skill by name.
// Returns nil if the skill is not found.
func (r *Registry) Get(name string) *Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.skills[name]
}

// MustGet retrieves a skill by name, panicking if not found.
// This method is intended for use during initialization and testing only.
// In production code, use Get() and handle the nil case appropriately.
func (r *Registry) MustGet(name string) *Skill {
	s := r.Get(name)
	if s == nil {
		panic(fmt.Sprintf("skill not found: %s", name))
	}
	return s
}

// Exists checks if a skill is registered.
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.skills[name]
	return ok
}

// Count returns the number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// List returns all registered skills.
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	return result
}

// ListSkills returns all registered skill names.
func (r *Registry) ListSkills() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.skills))
	for name := range r.skills {
		result = append(result, name)
	}
	return result
}

// Clear removes all skills from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills = make(map[string]*Skill)
}

// LoadFrom loads skills from a Loader into the registry.
// Returns a slice of any errors encountered during loading.
func (r *Registry) LoadFrom(l *Loader) ([]error, error) {
	skills, errs := l.Discover()

	// Register all successfully loaded skills
	for _, skill := range skills {
		if err := r.Register(skill); err != nil {
			return nil, err
		}
	}

	return errs, nil
}
