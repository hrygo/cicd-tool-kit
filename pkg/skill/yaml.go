package skill

import (
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// parseYAML parses YAML content into the provided value.
func parseYAML(content string, v any) error {
	decoder := yaml.NewDecoder(strings.NewReader(content))
	decoder.KnownFields(true) // Enable strict mode - reject unknown fields
	return decoder.Decode(v)
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

// isYAMLString checks if content looks like YAML (heuristic).
func isYAMLString(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}

	// Check for common YAML patterns
	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		// Look for key: value pattern
		if len(line) > 0 && unicode.IsLetter(rune(line[0])) {
			if idx := strings.Index(line, ":"); idx > 0 && idx < len(line)-1 {
				// Found "key:" followed by something
				return true
			}
		}
	}

	return false
}
