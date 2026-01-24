package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader handles discovering and parsing skill definitions.
type Loader struct {
	// Base directories to search for skills
	skillDirs []string

	// Whether to skip invalid skills during discovery
	skipInvalid bool
}

// LoaderOption configures a Loader.
type LoaderOption func(*Loader)

// WithSkillDirs adds directories to search for skills.
// Replaces the default "./skills" directory.
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

// LoadFromFile loads a single skill from a SKILL.md file.
func (l *Loader) LoadFromFile(path string) (*Skill, error) {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	// Parse frontmatter and body
	metadata, prompt, rawFrontmatter, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	// Create skill
	skill := &Skill{
		Metadata:      *metadata,
		File:          path,
		Dir:           filepath.Dir(path),
		Prompt:        prompt,
		RawFrontmatter: rawFrontmatter,
	}

	// Validate metadata
	if err := skill.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return skill, nil
}

// LoadByName loads a skill by name from the configured directories.
func (l *Loader) LoadByName(name string) (*Skill, error) {
	for _, dir := range l.skillDirs {
		path := SkillFile(dir, name)
		if _, err := os.Stat(path); err == nil {
			return l.LoadFromFile(path)
		}
	}
	return nil, fmt.Errorf("%w: %s (searched: %v)", ErrSkillNotFound, name, l.skillDirs)
}

// Discover scans all configured directories for skills.
// Returns a map of skill name -> Skill.
func (l *Loader) Discover() (map[string]*Skill, []error) {
	skills := make(map[string]*Skill)
	var errs []error

	for _, dir := range l.skillDirs {
		found, dirErrs := l.discoverInDir(dir)

		// Collect errors
		if len(dirErrs) > 0 {
			for _, e := range dirErrs {
				errs = append(errs, fmt.Errorf("failed to discover in %s: %w", dir, e))
			}
			if !l.skipInvalid {
				return skills, errs
			}
		}

		// Merge found skills even if there were errors (when skipInvalid is true)
		for name, skill := range found {
			if existing, ok := skills[name]; ok {
				// Duplicate skill name - prefer first found
				errs = append(errs, fmt.Errorf("duplicate skill %s: using %s, ignoring %s",
					name, existing.File, skill.File))
			} else {
				skills[name] = skill
			}
		}
	}

	return skills, errs
}

// discoverInDir scans a single directory for skill definitions.
func (l *Loader) discoverInDir(dir string) (map[string]*Skill, []error) {
	skills := make(map[string]*Skill)
	var errs []error

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return skills, nil // Directory doesn't exist, not an error
		}
		return nil, []error{err}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		skillPath := SkillFile(dir, name)

		// Check if SKILL.md exists
		if _, err := os.Stat(skillPath); err != nil {
			if os.IsNotExist(err) {
				continue // No SKILL.md in this directory
			}
			errs = append(errs, fmt.Errorf("failed to stat %s: %w", skillPath, err))
			continue
		}

		// Load the skill
		skill, err := l.LoadFromFile(skillPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to load %s: %w", name, err))
			if !l.skipInvalid {
				return skills, errs
			}
			continue
		}

		// Verify skill name matches directory name
		if skill.Name != name {
			errs = append(errs, fmt.Errorf("skill name mismatch: directory is %s but metadata specifies %s",
				name, skill.Name))
			if !l.skipInvalid {
				return skills, errs
			}
			continue
		}

		skills[name] = skill
	}

	return skills, errs
}

// parseFrontmatter extracts YAML frontmatter and markdown body.
// Expected format:
// ---yaml
// key: value
// ---
// markdown content
func parseFrontmatter(content string) (*Metadata, string, string, error) {
	lines := strings.Split(content, "\n")

	// Must start with delimiter
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "---") {
		return nil, "", "", fmt.Errorf("%w: file must start with YAML frontmatter delimited by ---", ErrInvalidFrontmatter)
	}

	// Find end delimiter
	var frontmatterLines []string
	var bodyLines []string
	var inFrontmatter = true

	for i, line := range lines {
		if i == 0 {
			continue // Skip first delimiter
		}
		if strings.HasPrefix(line, "---") {
			inFrontmatter = false
			continue
		}
		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	if len(frontmatterLines) == 0 {
		return nil, "", "", fmt.Errorf("%w: empty frontmatter", ErrInvalidFrontmatter)
	}

	frontmatterStr := strings.Join(frontmatterLines, "\n")
	prompt := strings.Join(bodyLines, "\n")

	// Parse YAML
	var metadata Metadata
	if err := parseYAMLStrict(frontmatterStr, &metadata); err != nil {
		return nil, "", "", fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	return &metadata, prompt, frontmatterStr, nil
}

// parseYAMLStrict parses YAML and returns an error for unknown fields.
func parseYAMLStrict(content string, v any) error {
	// Note: yaml.v3 doesn't have strict mode built-in.
	// For now, use standard unmarshaling. Can add custom validation later if needed.
	return parseYAML(content, v)
}
