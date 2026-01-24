package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErr       bool
		errorContains string
	}{
		{"valid simple", "myskill", false, ""},
		{"valid with hyphen", "my-skill", false, ""},
		{"valid with numbers", "skill-123", false, ""},
		{"empty", "", true, "invalid skill name"},
		{"uppercase", "MySkill", true, "must be lowercase"},
		{"with underscore", "my_skill", true, "invalid character '_'"},
		{"with dot", "my.skill", true, "invalid character '.'"},
		{"with space", "my skill", true, "invalid character ' '"},
		{"starts with hyphen", "-skill", true, "invalid skill name"},
		{"ends with hyphen", "skill-", true, "invalid skill name"},
		{"double hyphen", "skill--name", true, "invalid skill name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateName() expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ValidateName() = %v, want error containing %q", err, tt.errorContains)
				}
			} else if err != nil {
				t.Errorf("ValidateName() unexpected error: %v", err)
			}
		})
	}
}

func TestMetadata_Validate(t *testing.T) {
	tests := []struct {
		name    string
		m       Metadata
		wantErr error
	}{
		{
			name: "valid minimal",
			m: Metadata{
				Name:    "test-skill",
				Version: "1.0.0",
			},
			wantErr: nil,
		},
		{
			name: "valid complete",
			m: Metadata{
				Name:        "test-skill",
				Version:     "1.0.0",
				Description: "A test skill",
				Author:      "test",
				License:     "MIT",
				Inputs: []InputDef{
					{Name: "input1", Type: InputTypeString, Required: true},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			m: Metadata{
				Version: "1.0.0",
			},
			wantErr: ErrMissingName,
		},
		{
			name: "missing version",
			m: Metadata{
				Name: "test-skill",
			},
			wantErr: ErrMissingVersion,
		},
		{
			name: "invalid name uppercase",
			m: Metadata{
				Name:    "TestSkill",
				Version: "1.0.0",
			},
			wantErr: ErrInvalidSkillName,
		},
		{
			name: "invalid input type",
			m: Metadata{
				Name:    "test-skill",
				Version: "1.0.0",
				Inputs: []InputDef{
					{Name: "input1", Type: "invalid"},
				},
			},
			wantErr: ErrInvalidInputType,
		},
		{
			name: "duplicate input",
			m: Metadata{
				Name:    "test-skill",
				Version: "1.0.0",
				Inputs: []InputDef{
					{Name: "input1", Type: InputTypeString},
					{Name: "input1", Type: InputTypeInt},
				},
			},
			wantErr: ErrDuplicateInput,
		},
		{
			name: "missing input name",
			m: Metadata{
				Name:    "test-skill",
				Version: "1.0.0",
				Inputs: []InputDef{
					{Type: InputTypeString},
				},
			},
			wantErr: ErrInvalidInputType, // Placeholder - any error expected
		},
		{
			name: "temperature too high",
			m: Metadata{
				Name:    "test-skill",
				Version: "1.0.0",
				Options: RuntimeOptions{
					Temperature: 3.0,
				},
			},
			wantErr: ErrInvalidInputType, // Placeholder - any error expected
		},
		{
			name: "valid options",
			m: Metadata{
				Name:    "test-skill",
				Version: "1.0.0",
				Options: RuntimeOptions{
					Temperature:    0.5,
					MaxTokens:      4096,
					BudgetTokens:   1000,
					TopP:           0.9,
					TimeoutSeconds: 30,
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestSkill_GetInput(t *testing.T) {
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test",
			Version: "1.0.0",
			Inputs: []InputDef{
				{Name: "input1", Type: InputTypeString},
				{Name: "input2", Type: InputTypeInt},
			},
		},
	}

	tests := []struct {
		name    string
		input   string
		wantNil bool
	}{
		{"existing input", "input1", false},
		{"existing input 2", "input2", false},
		{"non-existing input", "input3", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skill.GetInput(tt.input)
			if (got == nil) != tt.wantNil {
				t.Errorf("GetInput() = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestSkill_ResolveInputValues(t *testing.T) {
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test",
			Version: "1.0.0",
			Inputs: []InputDef{
				{Name: "required1", Type: InputTypeString, Required: true},
				{Name: "optional1", Type: InputTypeString, Default: "default-value"},
				{Name: "optional2", Type: InputTypeInt, Default: 42},
			},
		},
	}

	tests := []struct {
		name        string
		provided    map[string]any
		wantErr     bool
		wantKeys    int
		checkValues map[string]any
	}{
		{
			name:     "all required provided",
			provided: map[string]any{"required1": "value"},
			wantErr:  false,
			wantKeys: 3, // all inputs
			checkValues: map[string]any{
				"required1": "value",
				"optional1": "default-value",
				"optional2": 42,
			},
		},
		{
			name:     "override default",
			provided: map[string]any{"required1": "value", "optional1": "custom"},
			wantErr:  false,
			wantKeys: 3,
			checkValues: map[string]any{
				"optional1": "custom",
			},
		},
		{
			name:     "missing required",
			provided: map[string]any{},
			wantErr:  true,
		},
		{
			name:     "unknown input",
			provided: map[string]any{"required1": "value", "unknown": "value"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := skill.ResolveInputValues(tt.provided)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveInputValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != tt.wantKeys {
					t.Errorf("ResolveInputValues() got %d keys, want %d", len(got), tt.wantKeys)
				}
				for k, v := range tt.checkValues {
					if got[k] != v {
						t.Errorf("ResolveInputValues() key %s = %v, want %v", k, got[k], v)
					}
				}
			}
		})
	}
}

func TestSkill_FullID(t *testing.T) {
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test-skill",
			Version: "1.2.3",
		},
	}

	want := "test-skill@1.2.3"
	if got := skill.FullID(); got != want {
		t.Errorf("FullID() = %v, want %v", got, want)
	}
}

func TestSkill_String(t *testing.T) {
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test",
			Version: "1.0.0",
		},
	}

	want := "Skill{name=test, version=1.0.0}"
	if got := skill.String(); got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}

func TestSkillDirAndFile(t *testing.T) {
	baseDir := "/test/skills"
	name := "my-skill"

	wantDir := "/test/skills/my-skill"
	if got := SkillDir(baseDir, name); got != wantDir {
		t.Errorf("SkillDir() = %v, want %v", got, wantDir)
	}

	wantFile := "/test/skills/my-skill/SKILL.md"
	if got := SkillFile(baseDir, name); got != wantFile {
		t.Errorf("SkillFile() = %v, want %v", got, wantFile)
	}
}

func TestSkill_ValidatePath(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "SKILL.md")
		if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}

		skill := &Skill{File: tmpFile}
		if err := skill.ValidatePath(); err != nil {
			t.Errorf("ValidatePath() unexpected error: %v", err)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		skill := &Skill{File: "/nonexistent/file.md"}
		if err := skill.ValidatePath(); err != ErrFileNotFound {
			t.Errorf("ValidatePath() = %v, want %v", err, ErrFileNotFound)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		skill := &Skill{File: ""}
		if err := skill.ValidatePath(); err != nil {
			t.Errorf("ValidatePath() unexpected error: %v", err)
		}
	})
}
