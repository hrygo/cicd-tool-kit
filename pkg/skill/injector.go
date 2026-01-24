package skill

import (
	"fmt"
	"strings"
	"text/template"
)

// Injector handles injecting skill prompts into system prompts.
type Injector struct {
	// Base system prompt (e.g., Claude's default instructions)
	basePrompt string

	// Variable placeholder format (default: {{name}})
	placeholderFormat string
}

// InjectorOption configures an Injector.
type InjectorOption func(*Injector)

// WithBasePrompt sets the base system prompt.
func WithBasePrompt(prompt string) InjectorOption {
	return func(i *Injector) {
		i.basePrompt = prompt
	}
}

// WithPlaceholderFormat sets the variable placeholder format.
// Use "%s" as the name placeholder. Default is "{{%s}}".
func WithPlaceholderFormat(format string) InjectorOption {
	return func(i *Injector) {
		i.placeholderFormat = format
	}
}

// NewInjector creates a new prompt injector.
func NewInjector(opts ...InjectorOption) *Injector {
	i := &Injector{
		basePrompt:        "",
		placeholderFormat: "{{%s}}",
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// BuildPrompt constructs the full system prompt by combining:
// 1. Base prompt (if set)
// 2. Skill prompt
// 3. Input values substituted into placeholders
func (i *Injector) BuildPrompt(skill *Skill, inputs map[string]any) (string, error) {
	var builder strings.Builder

	// Add base prompt if set
	if i.basePrompt != "" {
		builder.WriteString(i.basePrompt)
		if !strings.HasSuffix(i.basePrompt, "\n") {
			builder.WriteString("\n")
		}
	}

	// Add skill prompt
	prompt := skill.Prompt
	if prompt == "" {
		prompt = fmt.Sprintf("# Skill: %s\n\nYou are the %s skill.", skill.Name, skill.Name)
	}

	// Merge provided inputs with defaults
	mergedInputs := skill.GetDefaultValues()
	for k, v := range inputs {
		mergedInputs[k] = v
	}

	// Substitute input values
	substituted, err := i.substitutePlaceholders(prompt, mergedInputs)
	if err != nil {
		return "", fmt.Errorf("failed to substitute placeholders: %w", err)
	}

	builder.WriteString(substituted)

	return builder.String(), nil
}

// substitutePlaceholders replaces {{VAR}} style placeholders with actual values.
// Uses strings.Replacer for O(n) single-pass replacement instead of O(n×m).
func (i *Injector) substitutePlaceholders(prompt string, inputs map[string]any) (string, error) {
	// Find all unique placeholder names
	placeholders := i.extractPlaceholders(prompt)

	// Build replacement pairs for strings.Replacer
	// Each placeholder has two entries: lowercase and uppercase variant
	oldNew := make([]string, 0, len(placeholders)*2)

	for name := range placeholders {
		var replacement string
		value, ok := inputs[name]
		if !ok {
			// Try uppercase version too
			if v, ok := inputs[strings.ToUpper(name)]; ok {
				replacement = formatValue(v)
			} else {
				replacement = fmt.Sprintf("<%s not provided>", name)
			}
		} else {
			replacement = formatValue(value)
		}

		// Create both lowercase and uppercase placeholders
		placeholder := fmt.Sprintf(i.placeholderFormat, name)
		placeholderUpper := fmt.Sprintf(i.placeholderFormat, strings.ToUpper(name))

		oldNew = append(oldNew, placeholder, replacement)
		oldNew = append(oldNew, placeholderUpper, replacement)
	}

	// Use strings.Replacer for efficient single-pass replacement
	if len(oldNew) > 0 {
		replacer := strings.NewReplacer(oldNew...)
		return replacer.Replace(prompt), nil
	}
	return prompt, nil
}

// extractPlaceholders finds all unique placeholder names in the prompt.
func (i *Injector) extractPlaceholders(prompt string) map[string]bool {
	placeholders := make(map[string]bool)

	// Simple extraction: find {{name}} patterns
	start := strings.Index(prompt, "{{")
	for start != -1 {
		end := strings.Index(prompt[start:], "}}")
		if end == -1 {
			break
		}
		end += start

		// Extract name between {{ and }}
		name := strings.TrimSpace(prompt[start+2 : end])
		if name != "" {
			placeholders[name] = true
		}

		// Search for next
		prompt = prompt[end+2:]
		start = strings.Index(prompt, "{{")
	}

	return placeholders
}

// formatValue converts any value to a string representation.
func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// BuildPromptWithTemplate uses Go templates for more complex prompt building.
func (i *Injector) BuildPromptWithTemplate(skill *Skill, inputs map[string]any) (string, error) {
	tmpl, err := template.New(skill.Name).Option("missingkey=error").Parse(skill.Prompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var builder strings.Builder
	if i.basePrompt != "" {
		builder.WriteString(i.basePrompt)
		if !strings.HasSuffix(i.basePrompt, "\n") {
			builder.WriteString("\n")
		}
	}

	if err := tmpl.Execute(&builder, inputs); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return builder.String(), nil
}

// ValidatePrompt checks if a prompt has valid placeholder syntax.
func (i *Injector) ValidatePrompt(prompt string, availableInputs map[string]bool) error {
	placeholders := i.extractPlaceholders(prompt)

	for name := range placeholders {
		inputName := strings.ToLower(name)
		if !availableInputs[inputName] {
			return fmt.Errorf("placeholder {{%s}} has no corresponding input", name)
		}
	}

	return nil
}

// ExtractPlaceholders returns all placeholder names found in the prompt.
func (i *Injector) ExtractPlaceholders(prompt string) []string {
	placeholders := i.extractPlaceholders(prompt)
	result := make([]string, 0, len(placeholders))
	for name := range placeholders {
		result = append(result, name)
	}
	return result
}
