// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package skill

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrInvalidFrontmatter is returned when frontmatter parsing fails.
	ErrInvalidFrontmatter = errors.New("invalid frontmatter")
	// ErrMissingName is returned when skill name is missing.
	ErrMissingName = errors.New("missing required field: name")
	// ErrMissingVersion is returned when skill version is missing.
	ErrMissingVersion = errors.New("missing required field: version")
	// ErrInvalidSkillName is returned when skill name is invalid.
	ErrInvalidSkillName = errors.New("invalid skill name: must be lowercase kebab-case")
)

// LoaderOption configures a Loader.
type LoaderOption func(*Loader)

// WithSkillDirs sets the directories to search for skills.
func WithSkillDirs(dirs ...string) LoaderOption {
	return func(l *Loader) {
		l.skillDirs = dirs
	}
}

// WithSkipInvalid sets whether to skip invalid skills during discovery.
func WithSkipInvalid(skip bool) LoaderOption {
	return func(l *Loader) {
		l.skipInvalid = skip
	}
}

// Loader loads skills from SKILL.md files.
// This will be fully implemented in SPEC-SKILL-01.
type Loader struct {
	skillDirs   []string
	skipInvalid bool
}

// NewLoader creates a new skill loader.
func NewLoader(opts ...LoaderOption) *Loader {
	l := &Loader{
		skillDirs:   []string{"./skills"},
		skipInvalid: false,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LoadFromPath loads a skill from a directory.
func (l *Loader) LoadFromPath(ctx context.Context, path string) (*Skill, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	skillFile := filepath.Join(path, "SKILL.md")
	return l.LoadFromFile(skillFile)
}

// LoadFromDir loads all skills from a directory.
func (l *Loader) LoadFromDir(ctx context.Context, dir string) ([]*Skill, error) {
	// TODO: Implement per SPEC-SKILL-01
	return nil, nil
}

// LoadFromFile loads a skill from a SKILL.md file.
func (l *Loader) LoadFromFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	metadata, prompt, raw, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, err
	}

	// Validate skill name
	if err := validateSkillName(metadata.Name); err != nil {
		return nil, err
	}

	metadata.File = path

	// Parse inputs from raw frontmatter if present
	if inputsRaw, ok := raw["inputs"]; ok {
		if inputsList, ok := inputsRaw.([]any); ok {
			metadata.Inputs = parseInputs(inputsList)
		}
	}

	return &Skill{
		Metadata: metadata,
		Prompt:   prompt,
	}, nil
}

// Discover finds and loads all skills in the configured directories.
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
			return skills, nil
		}
		return skills, []error{err}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(dir, entry.Name())
		skillFile := filepath.Join(skillDir, "SKILL.md")

		if _, err := os.Stat(skillFile); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, err)
			continue
		}

		skill, err := l.LoadFromFile(skillFile)
		if err != nil {
			if l.skipInvalid {
				errs = append(errs, fmt.Errorf("skip %s: %w", entry.Name(), err))
				continue
			}
			errs = append(errs, fmt.Errorf("%s: %w", entry.Name(), err))
			continue
		}

		// Verify directory name matches skill name
		if skill.Metadata.Name != entry.Name() {
			err := fmt.Errorf("skill name '%s' does not match directory name '%s'", skill.Metadata.Name, entry.Name())
			if l.skipInvalid {
				errs = append(errs, err)
				continue
			}
			delete(skills, skill.Metadata.Name)
			errs = append(errs, err)
			continue
		}

		skills[skill.Metadata.Name] = skill
	}

	return skills, errs
}

// LoadByName loads a skill by name from the configured directories.
func (l *Loader) LoadByName(name string) (*Skill, error) {
	for _, dir := range l.skillDirs {
		skillDir := filepath.Join(dir, name)
		skillFile := filepath.Join(skillDir, "SKILL.md")

		if _, err := os.Stat(skillFile); err == nil {
			return l.LoadFromFile(skillFile)
		}
	}
	return nil, fmt.Errorf("skill not found: %s", name)
}

// parseInputs parses input definitions from raw YAML.
func parseInputs(raw []any) []InputDef {
	inputs := make([]InputDef, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			input := InputDef{
				Name:     getString(m, "name"),
				Required: getBool(m, "required"),
			}

			if typeName := getString(m, "type"); typeName != "" {
				input.Type = InputType(typeName)
			} else {
				input.Type = InputTypeString
			}

			if val, ok := m["default"]; ok {
				input.Default = val
			}

			if desc := getString(m, "description"); desc != "" {
				input.Description = desc
			}

			inputs = append(inputs, input)
		}
	}
	return inputs
}

// getString extracts a string value from map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getBool extracts a bool value from map.
func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// validateSkillName validates that the skill name follows kebab-case convention.
func validateSkillName(name string) error {
	return ValidateName(name)
}
