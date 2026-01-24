// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package skill provides skill management and execution.
package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrMissingName is returned when a skill metadata is missing the name field.
	ErrMissingName = errors.New("missing required field: name")
	// ErrMissingVersion is returned when a skill metadata is missing the version field.
	ErrMissingVersion = errors.New("missing required field: version")
	// ErrInvalidSkillName is returned when a skill name is invalid.
	ErrInvalidSkillName = errors.New("invalid skill name")
	// ErrSkillNotFound is returned when a skill is not found in the registry.
	ErrSkillNotFound = errors.New("skill not found")
	// ErrInvalidInputType is returned when an input type is invalid.
	ErrInvalidInputType = errors.New("invalid input type")
	// ErrDuplicateInput is returned when duplicate input names are found.
	ErrDuplicateInput = errors.New("duplicate input name")
	// ErrInvalidFrontmatter is returned when frontmatter parsing fails.
	ErrInvalidFrontmatter = errors.New("invalid frontmatter")
	// ErrFileNotFound is returned when a skill file is not found.
	ErrFileNotFound = errors.New("skill file not found")
	// ErrInvalidInputValue is returned when an input value is invalid.
	ErrInvalidInputValue = errors.New("invalid input value")
)

// InputType represents the type of a skill input.
type InputType string

const (
	// InputTypeString represents a string input.
	InputTypeString InputType = "string"
	// InputTypeInt represents an integer input.
	InputTypeInt = "int"
	// InputTypeFloat represents a float input.
	InputTypeFloat = "float"
	// InputTypeBool represents a boolean input.
	InputTypeBool = "bool"
	// InputTypeArray represents an array input.
	InputTypeArray = "array"
	// InputTypeObject represents an object input.
	InputTypeObject = "object"
)

// InputDef defines a skill input parameter.
type InputDef struct {
	Name     string    `yaml:"name"`
	Type     InputType `yaml:"type"`
	Required bool      `yaml:"required,omitempty"`
	Default  any       `yaml:"default,omitempty"`
	Desc     string    `yaml:"description,omitempty"`
}

// RuntimeOptions contains skill execution options.
type RuntimeOptions struct {
	Temperature    float64 `yaml:"temperature,omitempty"`
	MaxTokens      int     `yaml:"max_tokens,omitempty"`
	BudgetTokens   int     `yaml:"budget_tokens,omitempty"`
	TopP           float64 `yaml:"top_p,omitempty"`
	TimeoutSeconds int     `yaml:"timeout,omitempty"`

	// Thinking contains thinking-related options.
	Thinking map[string]any `yaml:"thinking,omitempty"`

	// Extra holds any additional options not explicitly defined.
	// This allows for flexible configuration without breaking changes.
	Extra map[string]any `yaml:",inline,omitempty"`
}

// ToolsConfig contains tool access configuration.
type ToolsConfig struct {
	Allow []string `yaml:"allow,omitempty"`
	Deny  []string `yaml:"deny,omitempty"`
}

// Metadata contains the skill metadata from frontmatter.
type Metadata struct {
	Name        string         `yaml:"name"`
	Version     string         `yaml:"version"`
	Description string         `yaml:"description,omitempty"`
	Author      string         `yaml:"author,omitempty"`
	License     string         `yaml:"license,omitempty"`
	Options     RuntimeOptions `yaml:"options,omitempty"`
	Tools       *ToolsConfig   `yaml:"tools,omitempty"`
	Inputs      []InputDef     `yaml:"inputs,omitempty"`
}

// Skill represents an AI skill.
type Skill struct {
	Metadata `yaml:",inline"`

	// Prompt is the markdown content from SKILL.md (after frontmatter)
	Prompt string `yaml:"-"`

	// File is the path to the SKILL.md file
	File string `yaml:"-"`

	// Dir is the directory containing the skill
	Dir string `yaml:"-"`
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

		// Validate type (supports array and object types)
		validTypes := map[InputType]bool{
			InputTypeString: true,
			InputTypeInt:    true,
			InputTypeFloat:  true,
			InputTypeBool:   true,
			InputTypeArray:  true,
			InputTypeObject: true,
		}
		if input.Type != "" && !validTypes[input.Type] {
			return ErrInvalidInputType
		}
	}

	// Validate options
	if m.Options.Temperature < 0 || m.Options.Temperature > 2 {
		return fmt.Errorf("%w: temperature must be between 0 and 2", ErrInvalidInputValue)
	}
	if m.Options.TopP < 0 || m.Options.TopP > 1 {
		return fmt.Errorf("%w: top_p must be between 0 and 1", ErrInvalidInputValue)
	}
	if m.Options.MaxTokens < 0 {
		return fmt.Errorf("%w: max_tokens must be non-negative", ErrInvalidInputValue)
	}

	return nil
}

// NewSkill creates a new skill with the given metadata.
func NewSkill(metadata Metadata) *Skill {
	return &Skill{
		Metadata: metadata,
	}
}

// GetDefaultValues returns a map of default values for all inputs that have defaults.
func (s *Skill) GetDefaultValues() map[string]any {
	result := make(map[string]any)
	for _, input := range s.Inputs {
		if input.Default != nil {
			result[input.Name] = input.Default
		}
	}
	return result
}

// GetInput returns the InputDef for the given input name, or nil if not found.
func (s *Skill) GetInput(name string) *InputDef {
	for i := range s.Inputs {
		if s.Inputs[i].Name == name {
			return &s.Inputs[i]
		}
	}
	return nil
}

// ResolveInputValues resolves all input values, applying defaults and validating.
func (s *Skill) ResolveInputValues(provided map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// First, apply all defaults
	for _, input := range s.Inputs {
		if input.Default != nil {
			result[input.Name] = input.Default
		}
	}

	// Then override with provided values
	for name, value := range provided {
		inputDef := s.GetInput(name)
		if inputDef == nil {
			return nil, fmt.Errorf("%w: unknown input '%s'", ErrInvalidInputValue, name)
		}
		result[name] = value
	}

	// Validate required inputs
	for _, input := range s.Inputs {
		if input.Required {
			if _, ok := result[input.Name]; !ok {
				return nil, fmt.Errorf("%w: missing required input '%s'", ErrInvalidInputValue, input.Name)
			}
		}
	}

	return result, nil
}

// FullID returns the full skill identifier in the format "name@version".
func (s *Skill) FullID() string {
	return fmt.Sprintf("%s@%s", s.Name(), s.Version())
}

// String returns a string representation of the skill.
func (s *Skill) String() string {
	return fmt.Sprintf("Skill{name=%s, version=%s}", s.Name(), s.Version())
}

// ValidatePath checks if the skill file exists.
func (s *Skill) ValidatePath() error {
	if s.File == "" {
		return nil // Empty path is valid (skill may not be file-based)
	}
	if _, err := os.Stat(s.File); errors.Is(err, os.ErrNotExist) {
		return ErrFileNotFound
	}
	return nil
}

// ValidateName validates a skill name.
// Skill names must be lowercase, contain only alphanumeric characters and hyphens,
// not start or end with a hyphen, and not have consecutive hyphens.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty name", ErrInvalidSkillName)
	}

	// Must be all lowercase
	if strings.ToLower(name) != name {
		return fmt.Errorf("%w: must be lowercase", ErrInvalidSkillName)
	}

	// Check for invalid characters
	for i, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return fmt.Errorf("%w: invalid character '%c'", ErrInvalidSkillName, ch)
		}
		// Check for underscore, dot, or space specifically
		if ch == '_' || ch == '.' || ch == ' ' {
			return fmt.Errorf("%w: invalid character '%c'", ErrInvalidSkillName, ch)
		}
		// Check for hyphen at start or end
		if ch == '-' && (i == 0 || i == len(name)-1) {
			return fmt.Errorf("%w: cannot start or end with hyphen", ErrInvalidSkillName)
		}
		// Check for consecutive hyphens
		if ch == '-' && i > 0 && name[i-1] == '-' {
			return fmt.Errorf("%w: cannot have consecutive hyphens", ErrInvalidSkillName)
		}
	}

	// Must match pattern: [a-z0-9]+(-[a-z0-9]+)*
	validName := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("%w: invalid character or format", ErrInvalidSkillName)
	}

	return nil
}

// SkillDir returns the directory path for a skill in the given base directory.
func SkillDir(baseDir, name string) string {
	return filepath.Join(baseDir, name)
}

// SkillFile returns the file path for a skill's SKILL.md in the given base directory.
func SkillFile(baseDir, name string) string {
	return filepath.Join(baseDir, name, "SKILL.md")
}

// Name returns the skill name.
func (s *Skill) Name() string {
	return s.Metadata.Name
}

// Version returns the skill version.
func (s *Skill) Version() string {
	return s.Metadata.Version
}
