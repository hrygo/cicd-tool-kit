// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package claude

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Parser parses Claude's structured output.
// Implements SPEC-CORE-03: Output Parsing
type Parser struct {
	strictMode       bool
	allowedTypes     map[string]bool
	maxContentLength int
}

// NewParser creates a new output parser.
func NewParser() *Parser {
	return &Parser{
		strictMode: false,
		allowedTypes: map[string]bool{
			"code":        true,
			"explanation": true,
			"error":       true,
			"suggestion":  true,
			"review":      true,
			"plan":        true,
			"result":      true,
		},
		maxContentLength: 10 * 1024 * 1024, // 10MB
	}
}

// NewStrictParser creates a parser that enforces strict type checking.
func NewStrictParser() *Parser {
	p := NewParser()
	p.strictMode = true
	return p
}

// Parse parses the output from Claude.
// Supports multiple formats:
// - XML-wrapped JSON: <claude_output type="code">{...}</claude_output>
// - Markdown code blocks: ```json ... ```
// - Direct JSON: {...}
func (p *Parser) Parse(output string) (*ParsedResult, error) {
	if len(output) > p.maxContentLength {
		return nil, fmt.Errorf("output exceeds maximum length of %d bytes", p.maxContentLength)
	}

	output = strings.TrimSpace(output)

	// Try XML format first
	if xmlResult, err := p.parseXML(output); err == nil && xmlResult != nil {
		return xmlResult, nil
	}

	// Try markdown code blocks
	if mdResult, err := p.parseMarkdown(output); err == nil && mdResult != nil {
		return mdResult, nil
	}

	// Try direct JSON
	if jsonResult, err := p.parseJSON(output); err == nil && jsonResult != nil {
		return jsonResult, nil
	}

	// Return raw output as fallback
	return &ParsedResult{
		Type:    "raw",
		Content: map[string]any{"text": output},
		Raw:     output,
	}, nil
}

// ParseWithExpectedType parses output expecting a specific type.
func (p *Parser) ParseWithExpectedType(output, expectedType string) (*ParsedResult, error) {
	result, err := p.Parse(output)
	if err != nil {
		return nil, err
	}

	if p.strictMode && result.Type != expectedType && result.Type != "raw" {
		return nil, fmt.Errorf("expected type %q, got %q", expectedType, result.Type)
	}

	return result, nil
}

// parseXML parses XML-wrapped JSON output.
func (p *Parser) parseXML(output string) (*ParsedResult, error) {
	// Match <claude_output type="...">...</claude_output>
	xmlRegex := regexp.MustCompile(`<(?:claude_output|result)\s+type=["']([^"']+)["']>\s*(.*?)\s*</(?:claude_output|result)>`)
	matches := xmlRegex.FindStringSubmatch(output)

	if len(matches) < 3 {
		return nil, fmt.Errorf("no XML wrapper found")
	}

	outputType := matches[1]
	content := strings.TrimSpace(matches[2])

	// Validate type
	if p.strictMode && !p.allowedTypes[outputType] {
		return nil, fmt.Errorf("unknown output type: %s", outputType)
	}

	// Parse JSON content
	var parsedContent map[string]any
	if err := json.Unmarshal([]byte(content), &parsedContent); err != nil {
		// If JSON parsing fails, treat as text
		parsedContent = map[string]any{"text": content}
	}

	return &ParsedResult{
		Type:    outputType,
		Content: parsedContent,
		Raw:     output,
	}, nil
}

// parseMarkdown parses markdown code block output.
func (p *Parser) parseMarkdown(output string) (*ParsedResult, error) {
	// Match ```json ... ``` or ```<type> ... ```
	mdRegex := regexp.MustCompile("```(?:json)?(?:\\s+(\\w+))?\\s*\\n([\\s\\S]*?)\\n```")
	matches := mdRegex.FindStringSubmatch(output)

	if len(matches) < 2 {
		return nil, fmt.Errorf("no markdown code block found")
	}

	outputType := "code"
	if matches[1] != "" {
		outputType = matches[1]
	}

	content := strings.TrimSpace(matches[2])

	var parsedContent map[string]any
	if err := json.Unmarshal([]byte(content), &parsedContent); err != nil {
		parsedContent = map[string]any{"text": content}
	}

	return &ParsedResult{
		Type:    outputType,
		Content: parsedContent,
		Raw:     output,
	}, nil
}

// parseJSON parses direct JSON output.
func (p *Parser) parseJSON(output string) (*ParsedResult, error) {
	if !strings.HasPrefix(output, "{") && !strings.HasPrefix(output, "[") {
		return nil, fmt.Errorf("not a JSON output")
	}

	var parsedContent map[string]any
	if err := json.Unmarshal([]byte(output), &parsedContent); err != nil {
		return nil, err
	}

	// Check for type field
	outputType := "result"
	if t, ok := parsedContent["type"].(string); ok {
		outputType = t
	}

	return &ParsedResult{
		Type:    outputType,
		Content: parsedContent,
		Raw:     output,
	}, nil
}

// ExtractCodeBlocks extracts code blocks from the output.
func (p *Parser) ExtractCodeBlocks(output string) []*CodeBlock {
	var blocks []*CodeBlock

	// Match ```language ... ``` blocks
	regex := regexp.MustCompile("```(\\w+)?\\s*\\n([\\s\\S]*?)\\n```")
	matches := regex.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		language := "text"
		if len(match) > 1 && match[1] != "" {
			language = match[1]
		}

		code := ""
		if len(match) > 2 {
			code = strings.TrimSpace(match[2])
		}

		blocks = append(blocks, &CodeBlock{
			Language: language,
			Code:     code,
		})
	}

	return blocks
}

// ExtractJSON extracts and parses JSON from output.
func (p *Parser) ExtractJSON(output string) (map[string]any, error) {
	// Find JSON objects in the output
	jsonRegex := regexp.MustCompile(`\\{[\\s\\S]*?\\}`)
	matches := jsonRegex.FindAllString(output, -1)

	for _, match := range matches {
		var result map[string]any
		if err := json.Unmarshal([]byte(match), &result); err == nil {
			return result, nil
		}
	}

	return nil, fmt.Errorf("no valid JSON found")
}

// SetAllowedTypes sets the allowed output types.
func (p *Parser) SetAllowedTypes(types ...string) {
	p.allowedTypes = make(map[string]bool)
	for _, t := range types {
		p.allowedTypes[t] = true
	}
}

// ParsedResult represents the parsed output.
type ParsedResult struct {
	Type    string
	Content map[string]any
	Raw     string
}

// GetString returns a string value from content.
func (r *ParsedResult) GetString(key string) string {
	if val, ok := r.Content[key].(string); ok {
		return val
	}
	return ""
}

// GetInt returns an int value from content.
func (r *ParsedResult) GetInt(key string) int {
	if val, ok := r.Content[key].(float64); ok {
		return int(val)
	}
	return 0
}

// GetBool returns a bool value from content.
func (r *ParsedResult) GetBool(key string) bool {
	if val, ok := r.Content[key].(bool); ok {
		return val
	}
	return false
}

// GetSlice returns a slice from content.
func (r *ParsedResult) GetSlice(key string) []any {
	if val, ok := r.Content[key].([]any); ok {
		return val
	}
	return nil
}

// IsType checks if the result is of a specific type.
func (r *ParsedResult) IsType(t string) bool {
	return r.Type == t
}

// CodeBlock represents a code block in the output.
type CodeBlock struct {
	Language string
	Code     string
}

// StreamParser parses streaming output from Claude.
type StreamParser struct {
	buffer strings.Builder
	parser *Parser
}

// NewStreamParser creates a new stream parser.
func NewStreamParser() *StreamParser {
	return &StreamParser{
		parser: NewParser(),
	}
}

// Write writes a chunk to the buffer.
func (s *StreamParser) Write(chunk string) {
	s.buffer.WriteString(chunk)
}

// ParseCurrent parses the current buffer content.
func (s *StreamParser) ParseCurrent() (*ParsedResult, error) {
	return s.parser.Parse(s.buffer.String())
}

// Reset clears the buffer.
func (s *StreamParser) Reset() {
	s.buffer.Reset()
}
