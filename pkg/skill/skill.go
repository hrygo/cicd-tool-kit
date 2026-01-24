// Package skill provides the core types and interfaces for skill definition,
// loading, validation, and execution.
package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Common errors
var (
	ErrSkillNotFound     = errors.New("skill not found")
	ErrInvalidSkillName  = errors.New("invalid skill name: must be lowercase alphanumeric with hyphens")
	ErrMissingName       = errors.New("missing required field: name")
	ErrMissingVersion    = errors.New("missing required field: version")
	ErrInvalidInputType  = errors.New("invalid input type")
	ErrDuplicateInput    = errors.New("duplicate input name")
	ErrInvalidToolPerm   = errors.New("invalid tool permission")
	ErrFileNotFound      = errors.New("skill file not found")
	ErrInvalidFrontmatter = errors.New("invalid YAML frontmatter")
)

// InputType represents the valid data types for skill inputs.
type InputType string

const (
	InputTypeString InputType = "string"
	InputTypeInt    InputType = "int"
	InputTypeFloat  InputType = "float"
	InputTypeBool   InputType = "bool"
	InputTypeArray  InputType = "array"
	InputTypeObject InputType = "object"
)

// IsValid checks if the InputType is valid.
func (t InputType) IsValid() bool {
	switch t {
	case InputTypeString, InputTypeInt, InputTypeFloat, InputTypeBool, InputTypeArray, InputTypeObject:
		return true
	}
	return false
}

// InputDef defines a single input parameter for a skill.
type InputDef struct {
	Name        string    `yaml:"name"`
	Type        InputType `yaml:"type"`
	Description string    `yaml:"description"`
	Default     any       `yaml:"default,omitempty"`
	Required    bool      `yaml:"required,omitempty"`
}

// ToolPermission defines tool access permissions.
type ToolPermission struct {
	Allow []string `yaml:"allow,omitempty"`
	Deny  []string `yaml:"deny,omitempty"`
}

// RuntimeOptions defines execution-time options for the skill.
type RuntimeOptions struct {
	Temperature     float64 `yaml:"temperature,omitempty"`
	MaxTokens       int     `yaml:"max_tokens,omitempty"`
	BudgetTokens    int     `yaml:"budget_tokens,omitempty"` // For "Thinking" block
	TopP            float64 `yaml:"top_p,omitempty"`
	TopK            int     `yaml:"top_k,omitempty"`
	TimeoutSeconds  int     `yaml:"timeout_seconds,omitempty"`
}

// Metadata contains the skill's frontmatter metadata.
type Metadata struct {
	Name        string          `yaml:"name"`
	Version     string          `yaml:"version"`
	Description string          `yaml:"description,omitempty"`
	Author      string          `yaml:"author,omitempty"`
	License     string          `yaml:"license,omitempty"`
	Options     RuntimeOptions  `yaml:"options,omitempty"`
	Tools       ToolPermission  `yaml:"tools,omitempty"`
	Inputs      []InputDef      `yaml:"inputs,omitempty"`
}

// Skill represents a loaded skill with its metadata and prompt content.
type Skill struct {
	// Metadata from frontmatter
	Metadata

	// Path to the skill directory
	Dir string

	// Path to the SKILL.md file
	File string

	// The markdown body content (System Prompt + Task Instruction + Output Contract)
	Prompt string

	// Raw frontmatter YAML (for debugging)
	RawFrontmatter string
}

// ValidateName checks if the skill name follows naming conventions.
func ValidateName(name string) error {
	if name == "" {
		return ErrInvalidSkillName
	}
	if strings.ToLower(name) != name {
		return fmt.Errorf("%w: must be lowercase", ErrInvalidSkillName)
	}
	// Check for leading or trailing hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return ErrInvalidSkillName
	}
	// Check for consecutive hyphens
	if strings.Contains(name, "--") {
		return ErrInvalidSkillName
	}
	for _, r := range name {
		if !isAlphaNumericHyphen(r) {
			return fmt.Errorf("%w: invalid character '%c'", ErrInvalidSkillName, r)
		}
	}
	return nil
}

func isAlphaNumericHyphen(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
}

// GetInput returns the input definition by name, or nil if not found.
func (s *Skill) GetInput(name string) *InputDef {
	for i := range s.Inputs {
		if s.Inputs[i].Name == name {
			return &s.Inputs[i]
		}
	}
	return nil
}

// GetDefaultValues returns a map of default values for all inputs.
func (s *Skill) GetDefaultValues() map[string]any {
	result := make(map[string]any)
	for _, input := range s.Inputs {
		if input.Default != nil {
			result[input.Name] = input.Default
		}
	}
	return result
}

// Validate checks if the skill metadata is valid.
func (m *Metadata) Validate() error {
	if m.Name == "" {
		return ErrMissingName
	}
	if err := ValidateName(m.Name); err != nil {
		return err
	}
	if m.Version == "" {
		return ErrMissingVersion
	}

	// Validate inputs
	seenInputs := make(map[string]bool)
	for i, input := range m.Inputs {
		if input.Name == "" {
			return fmt.Errorf("input at index %d: missing name", i)
		}
		if !input.Type.IsValid() {
			return fmt.Errorf("input %s: %w", input.Name, ErrInvalidInputType)
		}
		if seenInputs[input.Name] {
			return fmt.Errorf("%w: %s", ErrDuplicateInput, input.Name)
		}
		seenInputs[input.Name] = true
	}

	// Validate runtime options
	if m.Options.Temperature < 0 || m.Options.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2, got %f", m.Options.Temperature)
	}
	if m.Options.TopP < 0 || m.Options.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1, got %f", m.Options.TopP)
	}
	if m.Options.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be non-negative, got %d", m.Options.MaxTokens)
	}
	if m.Options.BudgetTokens < 0 {
		return fmt.Errorf("budget_tokens must be non-negative, got %d", m.Options.BudgetTokens)
	}

	return nil
}

// ValidatePath checks if the skill file path exists.
func (s *Skill) ValidatePath() error {
	if s.File == "" {
		return nil // Path not set yet
	}
	info, err := os.Stat(s.File)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected a file, got directory: %s", s.File)
	}
	return nil
}

// FullID returns a unique identifier for the skill (name@version).
func (s *Skill) FullID() string {
	return fmt.Sprintf("%s@%s", s.Name, s.Version)
}

// String returns a string representation of the skill.
func (s *Skill) String() string {
	return fmt.Sprintf("Skill{name=%s, version=%s}", s.Name, s.Version)
}

// ResolveInputValues merges provided values with defaults.
func (s *Skill) ResolveInputValues(provided map[string]any) (map[string]any, error) {
	result := s.GetDefaultValues()

	// Set provided values
	for k, v := range provided {
		input := s.GetInput(k)
		if input == nil {
			return nil, fmt.Errorf("unknown input: %s", k)
		}
		result[k] = v
	}

	// Check required inputs
	for _, input := range s.Inputs {
		if input.Required {
			if _, ok := result[input.Name]; !ok {
				return nil, fmt.Errorf("required input missing: %s", input.Name)
			}
		}
	}

	return result, nil
}

// SkillDir returns the expected directory path for a skill name.
func SkillDir(baseDir, name string) string {
	return filepath.Join(baseDir, name)
}

// SkillFile returns the expected SKILL.md path for a skill name.
func SkillFile(baseDir, name string) string {
	return filepath.Join(SkillDir(baseDir, name), "SKILL.md")
}
