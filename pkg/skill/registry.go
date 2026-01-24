// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package skill

import (
	"errors"
	"sync"
)

var (
	// ErrSkillNotFound is returned when a skill is not found in the registry.
	ErrSkillNotFound = errors.New("skill not found")
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

// Register registers a skill.
func (r *Registry) Register(s *Skill) error {
	if s == nil {
		return ErrInvalidSkill
	}
	if s.Metadata.Name == "" {
		return ErrInvalidSkill
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[s.Metadata.Name] = s
	return nil
}

// RegisterAll registers multiple skills at once.
// Returns an error if any skill fails to register.
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
// Returns the skill or nil if not found.
func (r *Registry) Get(name string) *Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.skills[name]
}

// MustGet retrieves a skill by name and panics if not found.
func (r *Registry) MustGet(name string) *Skill {
	s := r.Get(name)
	if s == nil {
		panic(ErrSkillNotFound)
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

// LoadFrom loads skills from a loader.
func (r *Registry) LoadFrom(l *Loader) ([]error, error) {
	skills, errs := l.Discover()

	for _, skill := range skills {
		if err := r.Register(skill); err != nil {
			return errs, err
		}
	}

	return errs, nil
}
