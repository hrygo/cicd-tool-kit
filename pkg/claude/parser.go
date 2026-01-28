// Package claude handles Claude Code subprocess management and output parsing
package claude

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// parser implements OutputParser
type parser struct{}

// NewParser creates a new output parser
func NewParser() OutputParser {
	return &parser{}
}

// ParseJSON extracts and parses JSON from Claude output
func (p *parser) ParseJSON(output string, target interface{}) error {
	jsonStr, err := p.ExtractJSONBlock(output)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

// ExtractJSONBlock extracts JSON code blocks from markdown
func (p *parser) ExtractJSONBlock(output string) (string, error) {
	return extractJSONBlock(output)
}

// ExtractThinking extracts the thinking block
func (p *parser) ExtractThinking(output string) string {
	return extractThinking(output)
}

// ExtractIssues extracts issue arrays from review output
func (p *parser) ExtractIssues(output string) ([]Issue, error) {
	// Try to extract JSON block
	jsonStr, err := p.ExtractJSONBlock(output)
	if err != nil {
		// No JSON found, try to parse text format
		return p.parseTextIssues(output)
	}

	// Parse JSON
	var result struct {
		Issues []Issue `json:"issues"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse issues JSON: %w", err)
	}

	return result.Issues, nil
}

// parseTextIssues parses issues from plain text output
func (p *parser) parseTextIssues(output string) ([]Issue, error) {
	var issues []Issue

	// Common patterns for issue reporting in text format
	patterns := []struct {
		regex   string
		severity string
		category string
	}{
		{`(?i)critical|security|vulnerability`, "critical", "security"},
		{`(?i)high.*risk|major.*issue`, "high", "quality"},
		{`(?i)warning|medium.*priority`, "medium", "quality"},
		{`(?i)low.*priority|minor.*issue`, "low", "style"},
	}

	lines := strings.Split(output, "\n")
	currentFile := ""
	currentLine := 0

	// Try to extract file:line patterns
	fileLineRe := regexp.MustCompile(`^([^\s:]+):(\d+):`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for file:line pattern
		if matches := fileLineRe.FindStringSubmatch(line); matches != nil && len(matches) >= 3 {
			currentFile = matches[1]
			if _, err := fmt.Sscanf(matches[2], "%d", &currentLine); err != nil {
				currentLine = 0 // Reset to default on parse error
			}
		}

		// Check for severity/category patterns
		for _, pattern := range patterns {
			matched, _ := regexp.MatchString(pattern.regex, line)
			if matched && !strings.HasPrefix(line, "#") {
				issues = append(issues, Issue{
					Severity:   pattern.severity,
					Category:   pattern.category,
					File:       currentFile,
					Line:       currentLine,
					Message:    line,
					Suggestion: "",
				})
				break
			}
		}
	}

	return issues, nil
}

// ExtractReviewSummary extracts a summary from review output
func (p *parser) ExtractReviewSummary(output string) string {
	// Look for summary section
	lines := strings.Split(output, "\n")
	inSummary := false
	var summary strings.Builder

	summaryMarkers := []string{
		"## Summary",
		"### Summary",
		"# Summary",
		"**Summary**",
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we're at a summary marker
		for _, marker := range summaryMarkers {
			if strings.HasPrefix(trimmed, marker) {
				inSummary = true
				continue
			}
		}

		// If we hit another ## section, stop
		if inSummary && strings.HasPrefix(trimmed, "##") && !strings.Contains(strings.ToLower(trimmed), "summary") {
			break
		}

		// Collect summary lines
		if inSummary && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if summary.Len() > 0 {
				summary.WriteString("\n")
			}
			summary.WriteString(trimmed)
		}

		// Fallback: if no summary found, use first non-empty paragraph
		if !inSummary && i < 20 && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if summary.Len() > 0 {
				summary.WriteString(" ")
			}
			summary.WriteString(trimmed)
		}
	}

	// If we found a proper summary section, return it
	// Otherwise return the fallback text
	result := strings.TrimSpace(summary.String())
	return result
}

// ExtractCodeChanges extracts code change suggestions from output
func (p *parser) ExtractCodeChanges(output string) []CodeChange {
	var changes []CodeChange

	// Look for code blocks with file paths
	lines := strings.Split(output, "\n")
	inCodeBlock := false
	currentFile := ""
	var codeBuf strings.Builder
	lastNonCodeLine := ""

	fileHeaderRe := regexp.MustCompile(`^([\w/._-]+\.[a-z]+)`)

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End of code block
				if currentFile != "" && codeBuf.Len() > 0 {
					changes = append(changes, CodeChange{
						File:    currentFile,
						Content: codeBuf.String(),
					})
				}
				inCodeBlock = false
				currentFile = ""
				codeBuf.Reset()
			} else {
				// Start of code block - try to extract file from context
				inCodeBlock = true
				// Check if the last non-code line contains a file reference
				if matches := fileHeaderRe.FindStringSubmatch(lastNonCodeLine); matches != nil && len(matches) >= 2 {
					currentFile = matches[1]
				}
			}
			continue
		}

		if inCodeBlock {
			// First line in code block might be the file reference
			if currentFile == "" {
				if matches := fileHeaderRe.FindStringSubmatch(line); matches != nil && len(matches) >= 2 {
					currentFile = matches[1]
				}
			}
			codeBuf.WriteString(line)
			codeBuf.WriteString("\n")
		} else {
			if strings.TrimSpace(line) != "" {
				lastNonCodeLine = line
			}
		}
	}

	return changes
}

// CodeChange represents a suggested code change
type CodeChange struct {
	File    string
	Content string
	Reason  string
}

// ValidateJSONSchema validates output against a JSON schema
func (p *parser) ValidateJSONSchema(output string, schema map[string]interface{}) error {
	jsonStr, err := p.ExtractJSONBlock(output)
	if err != nil {
		return fmt.Errorf("no JSON block to validate: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Basic required field validation
	for key := range schema {
		if _, exists := data[key]; !exists {
			return fmt.Errorf("missing required field: %s", key)
		}
	}

	return nil
}

// ExtractStructuredOutput extracts structured output using thinking + JSON pattern
func (p *parser) ExtractStructuredOutput(output string, target interface{}) error {
	// Extract thinking for context
	_ = p.ExtractThinking(output)

	// Parse JSON into target
	return p.ParseJSON(output, target)
}

// ExtractTokenUsage extracts token usage information from output
func (p *parser) ExtractTokenUsage(output string) *TokenUsage {
	// Look for token usage patterns in output
	// Claude may report this in stderr or special comments
	lines := strings.Split(output, "\n")

	usageRe := regexp.MustCompile(`(?i)(tokens?|cost):\s*(\d+)`)
	costRe := regexp.MustCompile(`(?i)cost:\s*\$?([\d.]+)`)

	usage := &TokenUsage{}

	for _, line := range lines {
	 if matches := usageRe.FindStringSubmatch(line); matches != nil {
	 // This is a simplified extraction
	 // Real implementation would parse actual Claude output format
	 }

	 if matches := costRe.FindStringSubmatch(line); matches != nil && len(matches) >= 2 {
	 fmt.Sscanf(matches[1], "%f", &usage.CostUSD)
	 }
	}

	if usage.TotalTokens == 0 && usage.CostUSD == 0 {
	 return nil
	}

	return usage
}
