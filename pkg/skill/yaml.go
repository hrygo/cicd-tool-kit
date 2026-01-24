package skill

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseFrontmatter parses YAML frontmatter from content.
// Returns metadata, prompt content, and raw YAML map.
func parseFrontmatter(content string) (Metadata, string, map[string]any, error) {
	var metadata Metadata
	raw := make(map[string]any)

	// Check for frontmatter delimiter
	if !strings.HasPrefix(content, "---") {
		return metadata, "", nil, ErrInvalidFrontmatter
	}

	// Find the end of frontmatter
	endIndex := strings.Index(content[3:], "---")
	if endIndex == -1 {
		return metadata, "", nil, ErrInvalidFrontmatter
	}
	endIndex += 3 // Account for the opening "---"

	// Extract YAML frontmatter
	yamlContent := content[3:endIndex]
	if strings.TrimSpace(yamlContent) == "" {
		return metadata, "", nil, ErrInvalidFrontmatter
	}

	// Parse YAML
	if err := parseYAML(yamlContent, &raw); err != nil {
		return metadata, "", nil, ErrInvalidFrontmatter
	}

	// Extract known fields
	metadata.Name = getString(raw, "name")
	metadata.Version = getString(raw, "version")
	metadata.Description = getString(raw, "description")
	metadata.Author = getString(raw, "author")
	metadata.License = getString(raw, "license")

	// Parse options if present
	if optsRaw, ok := raw["options"]; ok {
		if optsMap, ok := optsRaw.(map[string]any); ok {
			metadata.Options = RuntimeOptions{
				BudgetTokens:   getInt(optsMap, "budget_tokens"),
				TimeoutSeconds: getInt(optsMap, "timeout") + getInt(optsMap, "timeout_seconds"),
				Temperature:    getFloat(optsMap, "temperature"),
				MaxTokens:      getInt(optsMap, "max_tokens"),
				TopP:           getFloat(optsMap, "top_p"),
			}
		}
	}

	// Parse tools if present
	if toolsRaw, ok := raw["tools"]; ok {
		if toolsMap, ok := toolsRaw.(map[string]any); ok {
			tools := &ToolsConfig{}
			if allowRaw, ok := toolsMap["allow"]; ok {
				if allowList, ok := allowRaw.([]any); ok {
					for _, a := range allowList {
						if s, ok := a.(string); ok {
							tools.Allow = append(tools.Allow, s)
						}
					}
				}
			}
			if denyRaw, ok := toolsMap["deny"]; ok {
				if denyList, ok := denyRaw.([]any); ok {
					for _, d := range denyList {
						if s, ok := d.(string); ok {
							tools.Deny = append(tools.Deny, s)
						}
					}
				}
			}
			metadata.Tools = tools
		}
	}

	// Extract prompt content (everything after the closing ---)
	prompt := strings.TrimSpace(content[endIndex+3:])

	// Validate required fields
	if metadata.Name == "" {
		return metadata, "", nil, ErrMissingName
	}
	if metadata.Version == "" {
		return metadata, "", nil, ErrMissingVersion
	}

	return metadata, prompt, raw, nil
}

// parseYAML parses YAML content into the provided value.
func parseYAML(content string, v any) error {
	decoder := yaml.NewDecoder(strings.NewReader(content))
	decoder.KnownFields(true) // Enable strict mode - reject unknown fields
	if err := decoder.Decode(v); err != nil {
		return errors.New("invalid YAML")
	}
	return nil
}

// encodeYAML encodes a value to YAML.
func encodeYAML(v any) (string, error) {
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2) // Use 2 spaces for indentation
	if err := encoder.Encode(v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// getInt extracts an int value from map.
func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return 0
}

// getFloat extracts a float64 value from map.
func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// getString extracts a string value from map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
