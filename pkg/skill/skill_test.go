// Package skill tests
package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader("./skills")
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if loader.skillsDir != "./skills" {
		t.Errorf("Expected skillsDir './skills', got '%s'", loader.skillsDir)
	}
	if loader.skills == nil {
		t.Error("Skills map not initialized")
	}
}

func TestDiscover(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "skill1"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "skill1", "SKILL.md"), []byte("# Test"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "skill2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "skill2", "SKILL.md"), []byte("# Test"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "not-a-skill"), 0755)

	loader := NewLoader(tmpDir)
	names, err := loader.Discover()

	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(names) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(names))
	}
}

func TestParseSkill(t *testing.T) {
	content := `---
name: test-skill
description: A test skill
options:
  thinking:
    budget_tokens: 4096
  tools:
    - grep
    - read
---

# Test Skill

This is a test skill content.
`

	loader := NewLoader("./skills")
	skill, err := loader.parseSkill("test-skill", "/path/to/SKILL.md", content)

	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", skill.Name)
	}

	if skill.Description != "A test skill" {
		t.Errorf("Expected description 'A test skill', got '%s'", skill.Description)
	}

	if len(skill.Options.AllowedTools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(skill.Options.AllowedTools))
	}

	if skill.Content == "" {
		t.Error("Content should not be empty")
	}
}

func TestValidate(t *testing.T) {
	loader := NewLoader("./skills")

	tests := []struct {
		name    string
		skill   *Skill
		wantErr bool
	}{
		{
			name:    "valid skill",
			skill:   &Skill{Name: "test", Content: "# Test"},
			wantErr: false,
		},
		{
			name:    "empty name",
			skill:   &Skill{Name: "", Content: "# Test"},
			wantErr: true,
		},
		{
			name:    "empty content",
			skill:   &Skill{Name: "test", Content: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Validate(tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadAll(t *testing.T) {
	tmpDir := t.TempDir()

	skillNames := []string{"reviewer", "analyzer", "tester"}
	for _, name := range skillNames {
		skillPath := filepath.Join(tmpDir, name)
		os.MkdirAll(skillPath, 0755)
		os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# "+name+" Skill"), 0644)
	}

	loader := NewLoader(tmpDir)
	skills, err := loader.LoadAll()

	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("Expected 3 skills, got %d", len(skills))
	}
}

func TestGetPromptForSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillPath, 0755)
	os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# Test Content"), 0644)

	loader := NewLoader(tmpDir)
	prompt, err := loader.GetPromptForSkill("test-skill")

	if err != nil {
		t.Fatalf("GetPromptForSkill failed: %v", err)
	}

	if prompt != "# Test Content" {
		t.Errorf("Expected prompt '# Test Content', got '%s'", prompt)
	}
}

func TestGetSkillNamesForOperation(t *testing.T) {
	tmpDir := t.TempDir()

	skills := []string{"code-reviewer", "test-generator", "change-analyzer", "log-analyzer"}
	for _, name := range skills {
		skillPath := filepath.Join(tmpDir, name)
		os.MkdirAll(skillPath, 0755)
		os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# "+name), 0644)
	}

	loader := NewLoader(tmpDir)

	reviewSkills := loader.GetSkillNamesForOperation("review")
	if len(reviewSkills) != 1 || reviewSkills[0] != "code-reviewer" {
		t.Errorf("Expected [code-reviewer], got %v", reviewSkills)
	}

	testSkills := loader.GetSkillNamesForOperation("test-gen")
	if len(testSkills) != 1 || testSkills[0] != "test-generator" {
		t.Errorf("Expected [test-generator], got %v", testSkills)
	}

	analyzeSkills := loader.GetSkillNamesForOperation("analyze")
	// change-analyzer matches "analyze" in the name
	if len(analyzeSkills) != 1 {
		t.Errorf("Expected 1 analyze skill, got %d: %v", len(analyzeSkills), analyzeSkills)
	}
}
