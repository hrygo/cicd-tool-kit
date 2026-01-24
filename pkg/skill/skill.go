// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package skill provides skill management and execution.
package skill

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrInvalidInputType is returned when an input type is invalid.
	ErrInvalidInputType = errors.New("invalid input type")
	// ErrDuplicateInput is returned when duplicate input names are found.
	ErrDuplicateInput = errors.New("duplicate input name")
	// ErrFileNotFound is returned when a skill file is not found.
	ErrFileNotFound = errors.New("skill file not found")
)

// InputType defines the type of an input parameter.
type InputType string

const (
	// InputTypeString represents a string input.
	InputTypeString InputType = "string"
	// InputTypeInt represents an integer input.
	InputTypeInt InputType = "int"
	// InputTypeFloat represents a float input.
	InputTypeFloat InputType = "float"
	// InputTypeBool represents a boolean input.
	InputTypeBool InputType = "bool"
)

// InputDef defines an input parameter for a skill.
type InputDef struct {
	Name        string
	Type        InputType
	Required    bool
	Default     any
	Description string
}

// Metadata contains skill metadata from SKILL.md frontmatter.
type Metadata struct {
	Name        string
	Version     string
	Description string
	Author      string
	License     string
	File        string // Source file path
	Options     *RuntimeOptions
	Tools       *ToolsConfig
	Inputs      []InputDef
}

// Validate validates the metadata.
func (m *Metadata) Validate() error {
	if m.Name == "" {
		return ErrMissingName
	}
	if m.Version == "" {
		return ErrMissingVersion
	}
	if err := ValidateName(m.Name); err != nil {
		return err
	}

	// Validate input types and check for duplicates
	inputNames := make(map[string]bool)
	for _, input := range m.Inputs {
		if input.Name == "" {
			return ErrInvalidInputType
		}
		if inputNames[input.Name] {
			return ErrDuplicateInput
		}
		inputNames[input.Name] = true

		// Validate type
		validTypes := map[InputType]bool{
			InputTypeString: true,
			InputTypeInt:    true,
			InputTypeFloat:  true,
			InputTypeBool:   true,
		}
		if input.Type != "" && !validTypes[input.Type] {
			return ErrInvalidInputType
		}
	}

	// Validate options
	if m.Options != nil {
		if m.Options.Temperature < 0 || m.Options.Temperature > 1 {
			return fmt.Errorf("temperature must be between 0 and 1")
		}
		if m.Options.MaxTokens < 0 {
			return fmt.Errorf("max_tokens must be non-negative")
		}
	}

	return nil
}

// RuntimeOptions contains runtime execution options for a skill.
type RuntimeOptions struct {
	Temperature    float64
	MaxTokens      int
	BudgetTokens   int
	TopP           float64
	TimeoutSeconds int
}

// SkillOptions contains skill execution options (legacy alias).
type SkillOptions = RuntimeOptions

// ToolsConfig contains tool access configuration.
type ToolsConfig struct {
	Allow []string
	Deny  []string
}

// Skill represents an AI skill.
// This will be fully implemented in SPEC-SKILL-01.
type Skill struct {
	Metadata Metadata
	Prompt   string
}

// Name returns the skill name.
func (s *Skill) Name() string {
	return s.Metadata.Name
}

// Version returns the skill version.
func (s *Skill) Version() string {
	return s.Metadata.Version
}

// FullID returns the full skill identifier in format "name@version".
func (s *Skill) FullID() string {
	return fmt.Sprintf("%s@%s", s.Metadata.Name, s.Metadata.Version)
}

// String returns a string representation of the skill.
func (s *Skill) String() string {
	return fmt.Sprintf("Skill{name=%s, version=%s}", s.Metadata.Name, s.Metadata.Version)
}

// GetInput returns the input definition by name, or nil if not found.
func (s *Skill) GetInput(name string) *InputDef {
	for i := range s.Metadata.Inputs {
		if s.Metadata.Inputs[i].Name == name {
			return &s.Metadata.Inputs[i]
		}
	}
	return nil
}

// ResolveInputValues merges provided values with defaults and validates required inputs.
func (s *Skill) ResolveInputValues(provided map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// Start with defaults
	for _, input := range s.Metadata.Inputs {
		if !input.Required && input.Default != nil {
			result[input.Name] = input.Default
		}
	}

	// Override with provided values
	for k, v := range provided {
		// Check if input is defined
		inputDef := s.GetInput(k)
		if inputDef == nil {
			return nil, fmt.Errorf("unknown input: %s", k)
		}
		result[k] = v
	}

	// Check required inputs
	for _, input := range s.Metadata.Inputs {
		if input.Required {
			if _, ok := result[input.Name]; !ok {
				return nil, fmt.Errorf("missing required input: %s", input.Name)
			}
		}
	}

	return result, nil
}

// ValidatePath checks if the skill file exists.
func (s *Skill) ValidatePath() error {
	if s.Metadata.File == "" {
		return nil // Empty path is valid (skill may not have a file)
	}
	if _, err := os.Stat(s.Metadata.File); err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	return nil
}

// GetDefaultValues returns a map of default values for optional inputs.
func (s *Skill) GetDefaultValues() map[string]any {
	result := make(map[string]any)
	for _, input := range s.Metadata.Inputs {
		if !input.Required && input.Default != nil {
			result[input.Name] = input.Default
		}
	}
	return result
}

// Execute runs the skill.
func (s *Skill) Execute(ctx context.Context, input string) (string, error) {
	// TODO: Implement per SPEC-SKILL-01
	return "", nil
}

// ValidateName validates that a skill name follows kebab-case convention.
func ValidateName(name string) error {
	if name == "" {
		return ErrInvalidSkillName
	}

	// Must be lowercase
	if strings.ToLower(name) != name {
		return fmt.Errorf("invalid skill name: must be lowercase")
	}

	// Must contain only lowercase letters, numbers, and hyphens (no dots, underscores, etc.)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("invalid character '%c' in skill name", r)
		}
	}

	// Must not start or end with hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return ErrInvalidSkillName
	}

	// Must not have consecutive hyphens
	if strings.Contains(name, "--") {
		return ErrInvalidSkillName
	}

	return nil
}

// SkillDir returns the directory path for a skill.
func SkillDir(baseDir, name string) string {
	return filepath.Join(baseDir, name)
}

// SkillFile returns the SKILL.md file path for a skill.
func SkillFile(baseDir, name string) string {
	return filepath.Join(SkillDir(baseDir, name), "SKILL.md")
}
