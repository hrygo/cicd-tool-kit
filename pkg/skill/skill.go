// Package skill provides skill discovery and loading functionality
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// validSkillNamePattern matches safe skill names (alphanumeric, hyphens, underscores)
var validSkillNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// isValidSkillName validates that a skill name is safe to use in file paths
func isValidSkillName(name string) bool {
	if name == "" {
		return false
	}
	// Check for path traversal attempts
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	// Check against safe pattern
	return validSkillNamePattern.MatchString(name)
}

// Skill represents a loaded skill definition
type Skill struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Description string            `json:"description"`
	Author      string            `json:"author,omitempty"`
	License     string            `json:"license,omitempty"`
	Path        string            `json:"path"`
	Options     SkillOptions      `json:"options"`
	Inputs      []SkillInput      `json:"inputs,omitempty"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata"`
}

// SkillInput represents an input parameter for a skill
type SkillInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// SkillOptions contains optional skill configuration
type SkillOptions struct {
	Thinking     ThinkingOptions `json:"thinking"`
	AllowedTools []string        `json:"tools,omitempty"`
	OutputFormat string          `json:"output_format,omitempty"`
	MaxTurns     int             `json:"max_turns,omitempty"`
	BudgetUSD    float64         `json:"budget_usd,omitempty"`
}

// ThinkingOptions configures thinking behavior
type ThinkingOptions struct {
	BudgetTokens int  `json:"budget_tokens,omitempty"`
	Enabled      bool `json:"enabled,omitempty"`
}

// Loader handles skill discovery and loading
type Loader struct {
	skillsDir string
	skills    map[string]*Skill
}

// NewLoader creates a new skill loader
func NewLoader(skillsDir string) *Loader {
	return &Loader{
		skillsDir: skillsDir,
		skills:    make(map[string]*Skill),
	}
}

// Discover finds all skills in the skills directory
func (l *Loader) Discover() ([]string, error) {
	var skillNames []string

	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return skillNames, nil
		}
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(l.skillsDir, entry.Name())
		skillFile := filepath.Join(skillPath, "SKILL.md")

		if _, err := os.Stat(skillFile); err == nil {
			skillNames = append(skillNames, entry.Name())
		}
	}

	return skillNames, nil
}

// Load loads a specific skill by name
func (l *Loader) Load(name string) (*Skill, error) {
	if name == "" {
		return nil, fmt.Errorf("skill name cannot be empty")
	}
	if l.skillsDir == "" {
		return nil, fmt.Errorf("skills directory not configured")
	}

	// Check cache first
	if skill, ok := l.skills[name]; ok {
		return skill, nil
	}

	// Validate skill name to prevent path traversal
	if !isValidSkillName(name) {
		return nil, fmt.Errorf("invalid skill name: %s", name)
	}

	skillPath := filepath.Join(l.skillsDir, name, "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	skill, err := l.parseSkill(name, skillPath, string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}

	// Cache the skill
	l.skills[name] = skill

	return skill, nil
}

// LoadAll loads all discovered skills
func (l *Loader) LoadAll() ([]*Skill, error) {
	names, err := l.Discover()
	if err != nil {
		return nil, err
	}

	var skills []*Skill
	for _, name := range names {
		skill, err := l.Load(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load skill %s: %w", name, err)
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// Get returns a loaded skill by name
func (l *Loader) Get(name string) (*Skill, bool) {
	skill, ok := l.skills[name]
	return skill, ok
}

// isInputProperty checks if a key is a known input property name
func isInputProperty(key string) bool {
	switch strings.ToLower(key) {
	case "name", "type", "description", "required", "default":
		return true
	}
	return false
}

// parseSkill parses a skill definition from SKILL.md content
func (l *Loader) parseSkill(name, path, content string) (*Skill, error) {
	skill := &Skill{
		Name:     name,
		Path:     path,
		Metadata: make(map[string]string),
	}

	// Split by --- to extract frontmatter and content
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		// No frontmatter, use entire content
		skill.Content = strings.TrimSpace(content)
		skill.Description = fmt.Sprintf("Skill: %s", name)
		return skill, nil
	}

	// Parse frontmatter (second part after first ---)
	frontmatter := parts[1]
	skill.Content = strings.TrimSpace(parts[2])

	// Simple key-value parsing
	lines := strings.Split(frontmatter, "\n")
	inTools := false
	inInputs := false
	var currentInput *SkillInput

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for tools list
		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			inInputs = false
			currentInput = nil
			continue
		}

		// Check for inputs list
		if strings.HasPrefix(trimmed, "inputs:") {
			inInputs = true
			inTools = false
			currentInput = nil
			continue
		}

		// Parse tool list item
		if inTools {
			if strings.HasPrefix(trimmed, "- ") {
				tool := strings.TrimPrefix(trimmed, "- ")
				skill.Options.AllowedTools = append(skill.Options.AllowedTools, strings.TrimSpace(tool))
				continue
			}
			// Exit tools section on non-list item
			if !strings.HasPrefix(trimmed, "-") {
				inTools = false
			}
		}

		// Parse input list item
		if inInputs {
			// Check if this is a new list item (starts with - after trimming leading whitespace)
			if strings.HasPrefix(trimmed, "- ") {
				// Save previous input if exists
				if currentInput != nil && currentInput.Name != "" {
					skill.Inputs = append(skill.Inputs, *currentInput)
				}
				// Start new input
				inputContent := strings.TrimPrefix(trimmed, "- ")
				currentInput = &SkillInput{}

				// Check if this is inline format "- name: type (required): description"
				// or nested format "- name:" or "- key: value"
				if strings.Contains(inputContent, ":") {
					idx := strings.Index(inputContent, ":")
					firstKey := strings.TrimSpace(inputContent[:idx])
					firstValue := strings.TrimSpace(inputContent[idx+1:])

					// If first key is a known property name, it's nested format
					if isInputProperty(firstKey) {
						l.parseInputProperty(currentInput, firstKey, firstValue)
					} else {
						// It's inline format: "name: type (required): description"
						l.parseInputInline(inputContent, currentInput)
					}
				}
				continue
			}
			// Parse nested input properties (indented lines that aren't list items)
			// If we're in inputs section, not starting a new item, and has a colon, it's a nested property
			if currentInput != nil && !strings.HasPrefix(trimmed, "-") && strings.Contains(trimmed, ":") {
				idx := strings.Index(trimmed, ":")
				key := strings.TrimSpace(trimmed[:idx])
				value := strings.TrimSpace(trimmed[idx+1:])
				// Only process if it's a known property (not a new section)
				if isInputProperty(key) {
					l.parseInputProperty(currentInput, key, value)
				} else {
					// Not a property, might be a new section - exit inputs
					if currentInput.Name != "" {
						skill.Inputs = append(skill.Inputs, *currentInput)
						currentInput = nil
					}
					inInputs = false
				}
				continue
			}
			// Exit inputs section on non-list item at base indentation (no leading whitespace)
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(trimmed, "-") {
				if currentInput != nil && currentInput.Name != "" {
					skill.Inputs = append(skill.Inputs, *currentInput)
					currentInput = nil
				}
				inInputs = false
			}
		}

		// Parse key: value pairs (only if not in a list section)
		if !inTools && !inInputs && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
			idx := strings.Index(trimmed, ":")
			key := strings.TrimSpace(trimmed[:idx])
			value := strings.TrimSpace(trimmed[idx+1:])

			switch key {
			case "name":
				// Already set from directory name
			case "description":
				skill.Description = value
			case "version":
				skill.Version = value
			case "author":
				skill.Author = value
			case "license":
				skill.License = value
			case "budget_tokens":
				var num int
				if _, err := fmt.Sscanf(value, "%d", &num); err == nil {
					skill.Options.Thinking.BudgetTokens = num
				}
			case "thinking_enabled":
				skill.Options.Thinking.Enabled = strings.ToLower(value) == "true"
			case "max_turns":
				var num int
				if _, err := fmt.Sscanf(value, "%d", &num); err == nil {
					skill.Options.MaxTurns = num
				}
			case "output_format":
				skill.Options.OutputFormat = value
			case "budget_usd":
				var num float64
				if _, err := fmt.Sscanf(value, "%f", &num); err == nil {
					skill.Options.BudgetUSD = num
				}
			default:
				skill.Metadata[key] = value
			}
		}
	}

	// Save last input if exists
	if currentInput != nil && currentInput.Name != "" {
		skill.Inputs = append(skill.Inputs, *currentInput)
	}

	// Set default description if not found
	if skill.Description == "" {
		skill.Description = fmt.Sprintf("Skill: %s", name)
	}

	return skill, nil
}

// parseInputInline parses inline input format like "param: string (required?): description"
func (l *Loader) parseInputInline(content string, input *SkillInput) {
	// Format: "name: type (required?): description"
	// First, try to split by ":"
	parts := strings.SplitN(content, ":", 2)
	if len(parts) < 2 {
		return
	}

	input.Name = strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

	// Extract metadata from parentheses - support multiple paren groups
	// Process all parentheses to extract required/default flags
	for strings.Contains(rest, "(") && strings.Contains(rest, ")") {
		openIdx := strings.Index(rest, "(")
		closeIdx := strings.Index(rest[openIdx:], ")") + openIdx
		if closeIdx <= openIdx {
			break
		}
		parenContent := strings.ToLower(strings.TrimSpace(rest[openIdx+1 : closeIdx]))

		// Check if this paren contains "required"
		if strings.Contains(parenContent, "required") {
			input.Required = true
		}

		// Check if this paren contains "default:"
		if strings.Contains(parenContent, "default:") {
			defaultVal := strings.TrimSpace(rest[openIdx+9 : closeIdx])
			input.Default = defaultVal
		}

		// Remove the paren and its content
		rest = rest[:openIdx] + strings.TrimSpace(rest[closeIdx+1:])
		rest = strings.TrimSpace(rest)
	}

	// Extract type - it should be the first word
	words := strings.Fields(rest)
	if len(words) > 0 {
		// Clean type of trailing characters
		typeStr := strings.TrimSuffix(words[0], ":")
		typeStr = strings.TrimSuffix(typeStr, ",")
		input.Type = strings.TrimSpace(typeStr)
	}

	// Extract description (rest after type)
	if len(words) > 1 {
		// Skip first word (type), join rest as description
		description := strings.Join(words[1:], " ")
		description = strings.TrimPrefix(description, ":")
		description = strings.TrimPrefix(description, "-")
		input.Description = strings.TrimSpace(description)
	}
}

// parseInputProperty parses a single input property
func (l *Loader) parseInputProperty(input *SkillInput, key, value string) {
	switch strings.ToLower(key) {
	case "name":
		input.Name = value
	case "type":
		input.Type = value
	case "description":
		input.Description = value
	case "required":
		input.Required = strings.ToLower(value) == "true" || value == "1"
	case "default":
		input.Default = value
	}
}

// GetPromptForSkill returns the full prompt content for a skill
func (l *Loader) GetPromptForSkill(name string) (string, error) {
	skill, err := l.Load(name)
	if err != nil {
		return "", err
	}
	return skill.Content, nil
}

// GetSkillsByCategory returns skills matching a category pattern
func (l *Loader) GetSkillsByCategory(pattern string) ([]*Skill, error) {
	all, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	var filtered []*Skill
	for _, skill := range all {
		if strings.Contains(strings.ToLower(skill.Name), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(skill.Description), strings.ToLower(pattern)) {
			filtered = append(filtered, skill)
		}
	}

	return filtered, nil
}

// Validate checks if a skill definition is valid
func (l *Loader) Validate(skill *Skill) error {
	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	if skill.Content == "" {
		return fmt.Errorf("skill content is empty")
	}

	return nil
}

// ListNames returns all discovered skill names
func (l *Loader) ListNames() []string {
	names, _ := l.Discover()
	return names
}

// GetSkillNamesForOperation returns skill names for a given operation type
func (l *Loader) GetSkillNamesForOperation(op string) []string {
	all, _ := l.LoadAll()
	var names []string

	opLower := strings.ToLower(op)
	for _, skill := range all {
		skillName := strings.ToLower(skill.Name)
		switch opLower {
		case "review":
			if strings.Contains(skillName, "review") {
				names = append(names, skill.Name)
			}
		case "analyze", "change":
			// Match change-analyzer specifically for change analysis
			if strings.Contains(skillName, "change-analyzer") || strings.Contains(skillName, "change") {
				names = append(names, skill.Name)
			}
		case "test", "test-gen":
			if strings.Contains(skillName, "test") {
				names = append(names, skill.Name)
			}
		case "log":
			if strings.Contains(skillName, "log") {
				names = append(names, skill.Name)
			}
		}
	}

	return names
}
