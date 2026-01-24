// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package skill

import (
	"context"
	"os"
)

// Loader loads skills from SKILL.md files.
// This will be fully implemented in SPEC-SKILL-01.
type Loader struct {
	skillPaths []string //nolint:unused // TODO: Add loader configuration (SPEC-SKILL-01)
}

// NewLoader creates a new skill loader.
func NewLoader() *Loader {
	return &Loader{}
}

// LoadFromPath loads a skill from a directory.
func (l *Loader) LoadFromPath(ctx context.Context, path string) (*Skill, error) {
	// TODO: Implement per SPEC-SKILL-01
	// Parse SKILL.md and extract metadata
	return nil, nil
}

// LoadFromDir loads all skills from a directory.
func (l *Loader) LoadFromDir(ctx context.Context, dir string) ([]*Skill, error) {
	// TODO: Implement per SPEC-SKILL-01
	return nil, nil
}

// loadSkillMD parses a SKILL.md file.
//nolint:unused // TODO: Implement in SPEC-SKILL-01
func (l *Loader) loadSkillMD(path string) (*Skill, error) {
	// TODO: Implement per SPEC-SKILL-01
	// Parse YAML frontmatter and prompt content
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_ = data
	return nil, nil
}
