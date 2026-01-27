// Package skill provides skill discovery and loading functionality
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a loaded skill definition
type Skill struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Path        string            `json:"path"`
	Options     SkillOptions      `json:"options"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata"`
}

// SkillOptions contains optional skill configuration
type SkillOptions struct {
	Thinking        ThinkingOptions `json:"thinking"`
	AllowedTools    []string        `json:"tools,omitempty"`
	OutputFormat    string          `json:"output_format,omitempty"`
	MaxTurns        int             `json:"max_turns,omitempty"`
	BudgetUSD       float64         `json:"budget_usd,omitempty"`
}

// ThinkingOptions configures thinking behavior
type ThinkingOptions struct {
	BudgetTokens int `json:"budget_tokens,omitempty"`
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
	// Check cache first
	if skill, ok := l.skills[name]; ok {
		return skill, nil
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

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for tools list
		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
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

		// Parse key: value pairs
		if strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
			idx := strings.Index(trimmed, ":")
			key := strings.TrimSpace(trimmed[:idx])
			value := strings.TrimSpace(trimmed[idx+1:])

			switch key {
			case "name":
				// Already set from directory name
			case "description":
				skill.Description = value
			case "budget_tokens":
				var num int
				if _, err := fmt.Sscanf(value, "%d", &num); err == nil {
					skill.Options.Thinking.BudgetTokens = num
				}
			default:
				skill.Metadata[key] = value
			}
		}
	}

	// Set default description if not found
	if skill.Description == "" {
		skill.Description = fmt.Sprintf("Skill: %s", name)
	}

	return skill, nil
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
