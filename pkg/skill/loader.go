// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoaderOption configures a Loader.
type LoaderOption func(*Loader)

// WithSkillDirs sets the directories to search for skills.
func WithSkillDirs(dirs ...string) LoaderOption {
	return func(l *Loader) {
		l.skillDirs = dirs
	}
}

// WithSkipInvalid configures whether to skip invalid skills during discovery.
func WithSkipInvalid(skip bool) LoaderOption {
	return func(l *Loader) {
		l.skipInvalid = skip
	}
}

// Loader loads skills from SKILL.md files.
type Loader struct {
	skillDirs   []string
	skipInvalid bool
}

// NewLoader creates a new skill loader.
func NewLoader(opts ...LoaderOption) *Loader {
	l := &Loader{
		skillDirs:   []string{"./skills"}, // Default skill directory
		skipInvalid: false,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LoadFromFile loads a skill from a SKILL.md file.
func (l *Loader) LoadFromFile(path string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	// Use parseFrontmatter from yaml.go which returns (Metadata, string, map[string]any, error)
	metadata, prompt, _, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	// Validate metadata
	if err := metadata.Validate(); err != nil {
		return nil, err
	}

	skill := &Skill{
		Metadata: metadata,
		Prompt:   prompt,
		File:     path,
		Dir:      filepath.Dir(path),
	}

	return skill, nil
}

// Discover finds and loads all skills from the configured directories.
// Returns a map of skill name to skill, and a slice of any errors encountered.
func (l *Loader) Discover() (map[string]*Skill, []error) {
	skills := make(map[string]*Skill)
	var errs []error

	for _, dir := range l.skillDirs {
		dirSkills, dirErrs := l.discoverInDir(dir)
		for name, skill := range dirSkills {
			skills[name] = skill
		}
		errs = append(errs, dirErrs...)
	}

	return skills, errs
}

// discoverInDir discovers skills in a single directory.
func (l *Loader) discoverInDir(dir string) (map[string]*Skill, []error) {
	skills := make(map[string]*Skill)
	var errs []error

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return empty results
			return skills, nil
		}
		errs = append(errs, fmt.Errorf("failed to read directory %s: %w", dir, err))
		return skills, errs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillFile := filepath.Join(dir, skillName, "SKILL.md")

		// Check if SKILL.md exists before attempting to load
		if _, err := os.Stat(skillFile); os.IsNotExist(err) {
			// Directory doesn't contain a SKILL.md file, skip it silently
			continue
		}

		skill, err := l.LoadFromFile(skillFile)
		if err != nil {
			if l.skipInvalid {
				errs = append(errs, fmt.Errorf("skipping %s: %w", skillName, err))
				continue
			}
			errs = append(errs, fmt.Errorf("failed to load skill %s: %w", skillName, err))
			if len(errs) > 0 && !l.skipInvalid {
				// Return immediately if not skipping invalid
				return skills, errs
			}
			continue
		}

		// Verify skill name matches directory name
		if skill.Name() != skillName {
			err := fmt.Errorf("skill name '%s' does not match directory name '%s'", skill.Name(), skillName)
			if l.skipInvalid {
				errs = append(errs, fmt.Errorf("skipping %s: %w", skillName, err))
				continue
			}
			errs = append(errs, err)
			return skills, errs
		}

		skills[skillName] = skill
	}

	return skills, errs
}

// LoadByName loads a skill by name from the configured directories.
func (l *Loader) LoadByName(name string) (*Skill, error) {
	for _, dir := range l.skillDirs {
		skillFile := filepath.Join(dir, name, "SKILL.md")
		skill, err := l.LoadFromFile(skillFile)
		if err == nil {
			// Verify skill name matches
			if skill.Name() != name {
				return nil, fmt.Errorf("skill name '%s' does not match requested name '%s'", skill.Name(), name)
			}
			return skill, nil
		}
		if !os.IsNotExist(err) {
			// Non-existence error, try next directory
			return nil, err
		}
	}
	return nil, fmt.Errorf("%w: skill '%s' not found", ErrFileNotFound, name)
}

// LoadFromPath loads a skill from a directory (deprecated, use LoadFromFile).
func (l *Loader) LoadFromPath(_ interface{}) (*Skill, error) {
	// TODO: Implement per SPEC-SKILL-01
	// This method is kept for backward compatibility
	return nil, fmt.Errorf("not implemented: use LoadFromFile instead")
}

// LoadFromDir loads all skills from a directory (deprecated, use Discover).
func (l *Loader) LoadFromDir(_ string) ([]*Skill, error) {
	// TODO: Implement per SPEC-SKILL-01
	// This method is kept for backward compatibility
	return nil, fmt.Errorf("not implemented: use Discover instead")
}

// loadSkillMD parses a SKILL.md file.
// Deprecated: Use LoadFromFile instead.
func (l *Loader) loadSkillMD(path string) (*Skill, error) {
	return l.LoadFromFile(path)
}
