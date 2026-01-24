package skill

import (
	"strings"

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
