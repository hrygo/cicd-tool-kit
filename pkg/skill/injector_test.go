package skill

import (
	"strings"
	"testing"
)

func TestNewInjector(t *testing.T) {
	i := NewInjector()

	if i == nil {
		t.Fatal("NewInjector() returned nil")
	}

	if i.placeholderFormat != "{{%s}}" {
		t.Errorf("NewInjector() placeholderFormat = %v, want '{{%%s}}'", i.placeholderFormat)
	}
}

func TestNewInjectorWithOptions(t *testing.T) {
	basePrompt := "You are a helpful assistant."
	i := NewInjector(WithBasePrompt(basePrompt), WithPlaceholderFormat("<<%s>>"))

	if i.basePrompt != basePrompt {
		t.Errorf("NewInjector() basePrompt = %v, want %v", i.basePrompt, basePrompt)
	}

	if i.placeholderFormat != "<<%s>>" {
		t.Errorf("NewInjector() placeholderFormat = %v, want '<<%%s>>'", i.placeholderFormat)
	}
}

func TestInjector_BuildPrompt(t *testing.T) {
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test-skill",
			Version: "1.0.0",
		},
		Prompt: "Process this: {{input}}",
	}

	i := NewInjector()

	tests := []struct {
		name     string
		skill    *Skill
		inputs   map[string]any
		wantErr  bool
		contains string
	}{
		{
			name:     "simple substitution",
			skill:    skill,
			inputs:   map[string]any{"input": "test value"},
			wantErr:  false,
			contains: "Process this: test value",
		},
		{
			name: "multiple placeholders",
			skill: &Skill{
				Metadata: Metadata{
					Name:    "multi",
					Version: "1.0.0",
				},
				Prompt: "Name: {{name}}, Age: {{age}}",
			},
			inputs: map[string]any{
				"name": "John",
				"age":  30,
			},
			wantErr:  false,
			contains: "Name: John, Age: 30",
		},
		{
			name: "missing input uses placeholder",
			skill: &Skill{
				Metadata: Metadata{
					Name:    "test",
					Version: "1.0.0",
				},
				Prompt: "Input: {{missing}}",
			},
			inputs:   map[string]any{},
			wantErr:  false,
			contains: "<missing not provided>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := i

			result, err := injector.BuildPrompt(tt.skill, tt.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !strings.Contains(result, tt.contains) {
				t.Errorf("BuildPrompt() result = %v, does not contain %v", result, tt.contains)
			}
		})
	}
}

func TestInjector_ExtractPlaceholders(t *testing.T) {
	i := NewInjector()

	tests := []struct {
		name        string
		prompt      string
		wantCount   int
		wantPlaceholders []string
	}{
		{
			name:     "no placeholders",
			prompt:   "Just plain text",
			wantCount: 0,
		},
		{
			name:     "single placeholder",
			prompt:   "Hello {{name}}",
			wantCount: 1,
			wantPlaceholders: []string{"name"},
		},
		{
			name:     "multiple placeholders",
			prompt:   "{{greeting}} {{name}}, you are {{age}} years old",
			wantCount: 3,
			wantPlaceholders: []string{"greeting", "name", "age"},
		},
		{
			name:     "duplicate placeholder",
			prompt:   "{{name}} says {{name}}",
			wantCount: 1,
			wantPlaceholders: []string{"name"},
		},
		{
			name:     "nested braces",
			prompt:   "{{outer}} and {{inner}} content",
			wantCount: 2,
		},
		{
			name:     "placeholder with underscores",
			prompt:   "{{user_name}} is valid",
			wantCount: 1,
			wantPlaceholders: []string{"user_name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placeholders := i.ExtractPlaceholders(tt.prompt)

			if len(placeholders) != tt.wantCount {
				t.Errorf("ExtractPlaceholders() count = %d, want %d", len(placeholders), tt.wantCount)
			}

			if tt.wantPlaceholders != nil {
				for _, want := range tt.wantPlaceholders {
					found := false
					for _, got := range placeholders {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ExtractPlaceholders() missing %v", want)
					}
				}
			}
		})
	}
}

func TestInjector_ValidatePrompt(t *testing.T) {
	i := NewInjector()

	tests := []struct {
		name           string
		prompt         string
		availableInputs map[string]bool
		wantErr        bool
	}{
		{
			name:   "all inputs available",
			prompt: "{{name}} is {{age}}",
			availableInputs: map[string]bool{
				"name": true,
				"age":  true,
			},
			wantErr: false,
		},
		{
			name:   "missing input",
			prompt: "{{name}} is {{age}}",
			availableInputs: map[string]bool{
				"name": true,
			},
			wantErr: true,
		},
		{
			name:   "no placeholders",
			prompt: "Just text",
			availableInputs: map[string]bool{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := i.ValidatePrompt(tt.prompt, tt.availableInputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"nil", nil, ""},
		{"Stringer", mockStringer("test"), "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.input)
			if got != tt.want {
				t.Errorf("formatValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockStringer string

func (m mockStringer) String() string {
	return string(m)
}

func TestInjector_BuildPromptWithDefaults(t *testing.T) {
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test",
			Version: "1.0.0",
			Inputs: []InputDef{
				{Name: "required", Type: InputTypeString, Required: true},
				{Name: "optional", Type: InputTypeString, Default: "default-value"},
			},
		},
		Prompt: "{{required}} - {{optional}}",
	}

	i := NewInjector()
	result, err := i.BuildPrompt(skill, map[string]any{"required": "value"})

	if err != nil {
		t.Fatalf("BuildPrompt() error = %v", err)
	}

	if !strings.Contains(result, "value - default-value") {
		t.Errorf("BuildPrompt() result = %v, want 'value - default-value'", result)
	}
}

func TestInjector_CaseInsensitivePlaceholders(t *testing.T) {
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test",
			Version: "1.0.0",
		},
		Prompt: "{{NAME}} in uppercase, {{name}} in lowercase",
	}

	i := NewInjector()
	result, err := i.BuildPrompt(skill, map[string]any{"name": "John"})

	if err != nil {
		t.Fatalf("BuildPrompt() error = %v", err)
	}

	// Both placeholders should be substituted
	if strings.Contains(result, "{{") {
		t.Errorf("BuildPrompt() has unsubstituted placeholders: %v", result)
	}
}

