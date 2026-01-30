// Package skill tests
package skill

import (
	"os"
	"path/filepath"
	"strings"
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

	if err := os.MkdirAll(filepath.Join(tmpDir, "skill1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skill1", "SKILL.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "skill2"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skill2", "SKILL.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "not-a-skill"), 0755); err != nil {
		t.Fatal(err)
	}

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
		if err := os.MkdirAll(skillPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# "+name+" Skill"), 0644); err != nil {
			t.Fatal(err)
		}
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
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# Test Content"), 0644); err != nil {
		t.Fatal(err)
	}

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
		if err := os.MkdirAll(skillPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
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

func TestParseSkill_EnhancedMetadata(t *testing.T) {
	content := `---
name: enhanced-skill
description: A skill with enhanced metadata
version: 1.2.3
author: CICD AI Toolkit
license: Apache-2.0
thinking_enabled: true
max_turns: 10
output_format: json
budget_usd: 0.50
tools:
  - git
  - grep
inputs:
  - path: string (required): The file path to analyze
  - depth: int (default: 3): Search depth
---

# Enhanced Skill

This is an enhanced skill with full metadata.
`

	loader := NewLoader("./skills")
	skill, err := loader.parseSkill("enhanced-skill", "/path/to/SKILL.md", content)

	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}

	// Basic metadata
	if skill.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", skill.Version)
	}
	if skill.Author != "CICD AI Toolkit" {
		t.Errorf("Expected author 'CICD AI Toolkit', got '%s'", skill.Author)
	}
	if skill.License != "Apache-2.0" {
		t.Errorf("Expected license 'Apache-2.0', got '%s'", skill.License)
	}

	// Options
	if !skill.Options.Thinking.Enabled {
		t.Error("Expected thinking_enabled to be true")
	}
	if skill.Options.MaxTurns != 10 {
		t.Errorf("Expected max_turns 10, got %d", skill.Options.MaxTurns)
	}
	if skill.Options.OutputFormat != "json" {
		t.Errorf("Expected output_format 'json', got '%s'", skill.Options.OutputFormat)
	}
	if skill.Options.BudgetUSD != 0.50 {
		t.Errorf("Expected budget_usd 0.50, got %f", skill.Options.BudgetUSD)
	}

	// Tools
	if len(skill.Options.AllowedTools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(skill.Options.AllowedTools))
	}

	// Inputs
	if len(skill.Inputs) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(skill.Inputs))
	}

	// Check first input
	pathInput := skill.Inputs[0]
	if pathInput.Name != "path" {
		t.Errorf("Expected input name 'path', got '%s'", pathInput.Name)
	}
	if pathInput.Type != "string" {
		t.Errorf("Expected input type 'string', got '%s'", pathInput.Type)
	}
	if !pathInput.Required {
		t.Error("Expected 'path' input to be required")
	}

	// Check second input
	depthInput := skill.Inputs[1]
	if depthInput.Name != "depth" {
		t.Errorf("Expected input name 'depth', got '%s'", depthInput.Name)
	}
	if depthInput.Type != "int" {
		t.Errorf("Expected input type 'int', got '%s'", depthInput.Type)
	}
	if depthInput.Required {
		t.Error("Expected 'depth' input to be optional")
	}
	if depthInput.Default != "3" {
		t.Errorf("Expected default '3', got '%s'", depthInput.Default)
	}
}

func TestParseSkill_InputsNestedFormat(t *testing.T) {
	content := `---
name: nested-inputs-skill
description: Skill with nested input format
inputs:
  - name: repository
    type: string
    description: The repository URL
    required: true
  - name: branch
    type: string
    description: Branch name
    required: false
    default: main
---

# Nested Inputs Skill
`

	loader := NewLoader("./skills")
	skill, err := loader.parseSkill("nested-inputs-skill", "/path/to/SKILL.md", content)

	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}

	if len(skill.Inputs) != 2 {
		t.Logf("Inputs: %+v", skill.Inputs)
		t.Errorf("Expected 2 inputs, got %d", len(skill.Inputs))
		return
	}

	// Check first input
	repoInput := skill.Inputs[0]
	if repoInput.Name != "repository" {
		t.Errorf("Expected input name 'repository', got '%s'", repoInput.Name)
	}
	if repoInput.Type != "string" {
		t.Errorf("Expected input type 'string', got '%s'", repoInput.Type)
	}
	if !repoInput.Required {
		t.Error("Expected 'repository' input to be required")
	}

	// Check second input
	branchInput := skill.Inputs[1]
	if branchInput.Name != "branch" {
		t.Errorf("Expected input name 'branch', got '%s'", branchInput.Name)
	}
	if branchInput.Default != "main" {
		t.Errorf("Expected default 'main', got '%s'", branchInput.Default)
	}
}

func TestParseSkill_NoFrontmatter(t *testing.T) {
	content := `# Simple Skill

This skill has no frontmatter.`

	loader := NewLoader("./skills")
	skill, err := loader.parseSkill("simple-skill", "/path/to/SKILL.md", content)

	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}

	expectedContent := strings.TrimSpace(content)
	if skill.Content != expectedContent {
		t.Errorf("Content should match trimmed content.\nExpected: '%s'\nGot: '%s'", expectedContent, skill.Content)
	}
	if skill.Description != "Skill: simple-skill" {
		t.Errorf("Expected default description, got '%s'", skill.Description)
	}
}
