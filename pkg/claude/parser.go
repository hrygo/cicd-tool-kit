// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package claude

// Parser parses Claude's structured output.
// This will be fully implemented in SPEC-CORE-03.
type Parser struct {
	// TODO: Add parsing configuration
}

// NewParser creates a new output parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses the output from Claude.
func (p *Parser) Parse(output string) (*ParsedResult, error) {
	// TODO: Implement per SPEC-CORE-03
	// Expected format: XML-wrapped JSON
	return nil, nil
}

// ParsedResult represents the parsed output.
type ParsedResult struct {
	Type    string
	Content map[string]any
	Raw     string
}
