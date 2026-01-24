package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLoader(t *testing.T) {
	l := NewLoader()

	if l == nil {
		t.Fatal("NewLoader() returned nil")
	}

	if len(l.skillDirs) != 1 || l.skillDirs[0] != "./skills" {
		t.Errorf("NewLoader() skillDirs = %v, want ['./skills']", l.skillDirs)
	}

	if l.skipInvalid {
		t.Errorf("NewLoader() skipInvalid = true, want false")
	}
}

func TestNewLoaderWithOptions(t *testing.T) {
	l := NewLoader(
		WithSkillDirs("/custom/path", "/another/path"),
		WithSkipInvalid(true),
	)

	if len(l.skillDirs) != 2 {
		t.Errorf("NewLoader() skillDirs length = %d, want 2", len(l.skillDirs))
	}

	if l.skillDirs[0] != "/custom/path" {
		t.Errorf("NewLoader() skillDirs[0] = %v, want '/custom/path'", l.skillDirs[0])
	}

	if !l.skipInvalid {
		t.Errorf("NewLoader() skipInvalid = false, want true")
	}
}

func TestLoader_LoadFromFile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantName   string
		wantPrompt string
		wantErr    error
	}{
		{
			name: "valid skill",
			content: `---
name: test-skill
version: 1.0.0
description: A test skill
---
# Test Skill

This is a test prompt with {{placeholder}}.`,
			wantName:   "test-skill",
			wantPrompt: "# Test Skill\n\nThis is a test prompt with {{placeholder}}.",
		},
		{
			name: "minimal valid skill",
			content: `---
name: minimal
version: 1.0.0
---
Prompt content here.`,
			wantName:   "minimal",
			wantPrompt: "Prompt content here.",
		},
		{
			name: "missing frontmatter",
			content: `No frontmatter here
just content`,
			wantErr: ErrInvalidFrontmatter,
		},
		{
			name: "empty frontmatter",
			content: `---
---
content`,
			wantErr: ErrInvalidFrontmatter,
		},
		{
			name: "missing name",
			content: `---
version: 1.0.0
---
content`,
			wantErr: ErrMissingName,
		},
		{
			name: "missing version",
			content: `---
name: test
---
content`,
			wantErr: ErrMissingVersion,
		},
		{
			name: "invalid name uppercase",
			content: `---
name: TestSkill
version: 1.0.0
---
content`,
			wantErr: ErrInvalidSkillName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "SKILL.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			l := NewLoader()
			skill, err := l.LoadFromFile(tmpFile)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("LoadFromFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadFromFile() unexpected error: %v", err)
			}

			if skill.Name() != tt.wantName {
				t.Errorf("LoadFromFile() Name = %v, want %v", skill.Name(), tt.wantName)
			}

			if skill.Prompt != tt.wantPrompt {
				t.Errorf("LoadFromFile() Prompt = %v, want %v", skill.Prompt, tt.wantPrompt)
			}

			if skill.Metadata.File != tmpFile {
				t.Errorf("LoadFromFile() File = %v, want %v", skill.Metadata.File, tmpFile)
			}
		})
	}
}

func TestLoader_LoadFromFile_NotFound(t *testing.T) {
	l := NewLoader()
	_, err := l.LoadFromFile("/nonexistent/skill.md")

	if err == nil {
		t.Error("LoadFromFile() expected error for non-existent file")
	}
}

func TestLoader_Discover(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	// Valid skill 1
	skill1Dir := filepath.Join(tmpDir, "skill-one")
	if err := os.MkdirAll(skill1Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	skill1Content := `---
name: skill-one
version: 1.0.0
---
Prompt for skill one.`
	if err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(skill1Content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Valid skill 2
	skill2Dir := filepath.Join(tmpDir, "skill-two")
	if err := os.MkdirAll(skill2Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	skill2Content := `---
name: skill-two
version: 1.0.0
---
Prompt for skill two.`
	if err := os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(skill2Content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Directory without SKILL.md (should be skipped)
	emptyDir := filepath.Join(tmpDir, "empty-dir")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// File (not directory, should be skipped)
	if err := os.WriteFile(filepath.Join(tmpDir, "not-a-dir"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(WithSkillDirs(tmpDir))
	skills, errs := l.Discover()

	if len(errs) > 0 {
		t.Errorf("Discover() returned errors: %v", errs)
	}

	if len(skills) != 2 {
		t.Errorf("Discover() found %d skills, want 2", len(skills))
	}

	if _, ok := skills["skill-one"]; !ok {
		t.Error("Discover() did not find skill-one")
	}

	if _, ok := skills["skill-two"]; !ok {
		t.Error("Discover() did not find skill-two")
	}

	if _, ok := skills["empty-dir"]; ok {
		t.Error("Discover() found empty-dir which has no SKILL.md")
	}
}

func TestLoader_Discover_NameMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "wrong-name")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// SKILL.md has different name than directory
	content := `---
name: actual-name
version: 1.0.0
---
Prompt`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(WithSkillDirs(tmpDir))
	skills, errs := l.Discover()

	if len(errs) == 0 {
		t.Error("Discover() expected error for name mismatch")
	}

	if len(skills) != 0 {
		t.Errorf("Discover() found %d skills, want 0", len(skills))
	}
}

func TestLoader_Discover_InvalidSkill(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "invalid-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Missing required field
	content := `---
name: invalid-skill
---
Prompt`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(WithSkillDirs(tmpDir))
	skills, errs := l.Discover()

	if len(errs) == 0 {
		t.Error("Discover() expected error for invalid skill")
	}

	if len(skills) != 0 {
		t.Errorf("Discover() found %d skills, want 0", len(skills))
	}
}

func TestLoader_Discover_SkipInvalid(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid skill
	validDir := filepath.Join(tmpDir, "valid-skill")
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatal(err)
	}
	validContent := `---
name: valid-skill
version: 1.0.0
---
Valid prompt`
	if err := os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte(validContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Invalid skill
	invalidDir := filepath.Join(tmpDir, "invalid-skill")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	invalidContent := `---
name: invalid-skill
---
Invalid prompt`
	if err := os.WriteFile(filepath.Join(invalidDir, "SKILL.md"), []byte(invalidContent), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(WithSkillDirs(tmpDir), WithSkipInvalid(true))
	skills, errs := l.Discover()

	if len(skills) != 1 {
		t.Errorf("Discover() found %d skills, want 1", len(skills))
	}

	if _, ok := skills["valid-skill"]; !ok {
		t.Error("Discover() did not find valid-skill")
	}

	if len(errs) != 1 {
		t.Errorf("Discover() returned %d errors, want 1", len(errs))
	}
}

func TestLoader_Discover_NonExistentDir(t *testing.T) {
	l := NewLoader(WithSkillDirs("/nonexistent/directory"))
	skills, errs := l.Discover()

	if len(errs) > 0 {
		t.Errorf("Discover() returned errors: %v", errs)
	}

	if len(skills) != 0 {
		t.Errorf("Discover() found %d skills, want 0", len(skills))
	}
}

func TestLoader_LoadByName(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: test-skill
version: 1.0.0
---
Test prompt`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(WithSkillDirs(tmpDir))
	skill, err := l.LoadByName("test-skill")

	if err != nil {
		t.Fatalf("LoadByName() unexpected error: %v", err)
	}

	if skill.Name() != "test-skill" {
		t.Errorf("LoadByName() Name = %v, want 'test-skill'", skill.Name())
	}
}

func TestLoader_LoadByName_NotFound(t *testing.T) {
	l := NewLoader(WithSkillDirs(t.TempDir()))
	_, err := l.LoadByName("nonexistent")

	if err == nil {
		t.Error("LoadByName() expected error for non-existent skill")
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantVersion string
		wantPrompt  string
		wantErr     error
	}{
		{
			name: "valid with full metadata",
			content: `---
name: test-skill
version: 1.0.0
description: A test
author: test
license: MIT
options:
  temperature: 0.5
  max_tokens: 1000
tools:
  allow:
    - read
    - grep
inputs:
  - name: input1
    type: string
    required: true
  - name: input2
    type: int
---
# Skill Prompt

Content with {{placeholder}}.`,
			wantName:    "test-skill",
			wantVersion: "1.0.0",
			wantPrompt:  "# Skill Prompt\n\nContent with {{placeholder}}.",
		},
		{
			name: "valid minimal",
			content: `---
name: test
version: 1.0.0
---
Prompt`,
			wantName:    "test",
			wantVersion: "1.0.0",
			wantPrompt:  "Prompt",
		},
		{
			name:    "no frontmatter",
			content: `Just content`,
			wantErr: ErrInvalidFrontmatter,
		},
		{
			name: "empty frontmatter",
			content: `---
---
content`,
			wantErr: ErrInvalidFrontmatter,
		},
		{
			name: "multilines in prompt",
			content: `---
name: test
version: 1.0.0
---
Line 1
Line 2
Line 3`,
			wantName:    "test",
			wantVersion: "1.0.0",
			wantPrompt:  "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, prompt, _, err := parseFrontmatter(tt.content)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("parseFrontmatter() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseFrontmatter() unexpected error: %v", err)
			}

			if metadata.Name != tt.wantName {
				t.Errorf("parseFrontmatter() Name = %v, want %v", metadata.Name, tt.wantName)
			}

			if metadata.Version != tt.wantVersion {
				t.Errorf("parseFrontmatter() Version = %v, want %v", metadata.Version, tt.wantVersion)
			}

			if prompt != tt.wantPrompt {
				t.Errorf("parseFrontmatter() Prompt = %v, want %v", prompt, tt.wantPrompt)
			}
		})
	}
}

func TestParseYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		v       any
		wantErr bool
	}{
		{
			name: "valid simple",
			content: `name: test
version: "1.0.0"`,
			v: &struct {
				Name    string `yaml:"name"`
				Version string `yaml:"version"`
			}{},
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			content: `name: test: :`,
			v:       &struct{ Name string }{},
			wantErr: true,
		},
		{
			name:    "empty",
			content: ``,
			v:       &struct{ Name string }{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseYAML(tt.content, tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodeYAML(t *testing.T) {
	v := struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}{
		Name:    "test",
		Version: "1.0.0",
	}

	result, err := encodeYAML(v)
	if err != nil {
		t.Fatalf("encodeYAML() error = %v", err)
	}

	if !strings.Contains(result, "name: test") {
		t.Errorf("encodeYAML() result = %v, missing 'name: test'", result)
	}

	if !strings.Contains(result, "version: 1.0.0") {
		t.Errorf("encodeYAML() result = %v, missing 'version: 1.0.0'", result)
	}
}

func TestLoader_Integration(t *testing.T) {
	// Integration test: load actual skills from the skills directory
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skip("Could not find project root:", err)
	}

	skillsDir := filepath.Join(projectRoot, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		t.Skip("skills directory not found")
	}

	l := NewLoader(WithSkillDirs(skillsDir))
	skills, errs := l.Discover()

	if len(errs) > 0 {
		t.Logf("Discovery errors (non-fatal): %v", errs)
	}

	// Should find at least the three standard skills
	expectedSkills := []string{"code-reviewer", "test-generator", "committer"}
	for _, name := range expectedSkills {
		if _, ok := skills[name]; !ok {
			t.Errorf("Discover() did not find expected skill: %s", name)
		}
	}

	// Verify each skill has proper metadata
	for name, skill := range skills {
		if skill.Name() != name {
			t.Errorf("Skill %s: Name = %v, want %v", name, skill.Name(), name)
		}
		if skill.Version() == "" {
			t.Errorf("Skill %s: Version is empty", name)
		}
		if skill.Prompt == "" {
			t.Errorf("Skill %s: Prompt is empty", name)
		}
	}
}

func findProjectRoot() (string, error) {
	// Start from current directory and go up until we find go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
