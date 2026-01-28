// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.

package security

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// PromptInjectionDetector detects and protects against malicious prompts.
type PromptInjectionDetector struct {
	patterns        []*injectionPattern
	strictMode      bool
	maxPromptLength int
	allowedPrefixes []string
}

// injectionPattern represents a pattern that may indicate prompt injection.
type injectionPattern struct {
	pattern  *regexp.Regexp
	severity Severity
	category string
}

// Severity represents the severity level of a detected pattern.
type Severity int

const (
	SeverityLow    Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// NewPromptInjectionDetector creates a new detector.
func NewPromptInjectionDetector() *PromptInjectionDetector {
	d := &PromptInjectionDetector{
		strictMode:      true,
		maxPromptLength: 50000, // 50k characters
		allowedPrefixes: []string{
			"Please analyze",
			"Please review",
			"Please explain",
			"Help me",
			"I need",
			"Can you",
		},
	}
	d.initPatterns()
	return d
}

// NewLenientDetector creates a detector with relaxed rules.
func NewLenientDetector() *PromptInjectionDetector {
	d := NewPromptInjectionDetector()
	d.strictMode = false
	return d
}

// initPatterns initializes the injection detection patterns.
func (d *PromptInjectionDetector) initPatterns() {
	patterns := []struct {
		pattern  string
		severity Severity
		category string
	}{
		// Critical: Direct override attempts
		{`(?i)ignore\s+(all\s+)?(previous|above|the)\s+(instructions?|prompts?|commands?)(\s+and|$)`, SeverityCritical, "override"},
		{`(?i)disregard\s+(all\s+)?(previous|above|the)\s+(instructions?|prompts?|commands?)`, SeverityCritical, "override"},
		{`(?i)forget\s+(all\s+)?(previous|above|the)\s+(instructions?|prompts?|commands?)`, SeverityCritical, "override"},

		// Critical: Role confusion
		{`(?i)you\s+are\s+now\s+(a\s+)?new\s+(AI|assistant|persona|chatbot|model)`, SeverityCritical, "role_confusion"},
		{`(?i)from\s+now\s+on\s+you\s+are`, SeverityCritical, "role_confusion"},
		{`(?i)act\s+as\s+(if\s+you\s+are\s+(a\s+))?(different|another|new)`, SeverityHigh, "role_confusion"},

		// Critical: System prompt extraction
		{`(?i)show\s+me\s+your\s+(instructions?|prompts?|system\s+prompt|initial\s+prompt)`, SeverityCritical, "extraction"},
		{`(?i)print\s+(your|the)\s+(instructions?|prompts?|system\s+prompt)`, SeverityCritical, "extraction"},
		{`(?i)repeat\s+(everything|all\s+text)\s+(above|before)`, SeverityCritical, "extraction"},
		{`(?i)tell\s+me\s+what\s+you\s+were\s+told\s+to\s+do`, SeverityHigh, "extraction"},

		// High: Jailbreak attempts
		{`(?i)(jailbreak|jail\s*break)\s*(mode|technique|method|protocol)`, SeverityHigh, "jailbreak"},
		{`(?i)developer\s+mode`, SeverityHigh, "jailbreak"},
		{`(?i)(unrestricted|uncensored|filterless)\s+mode`, SeverityHigh, "jailbreak"},
		{`(?i)DAN\s+(mode|protocol)`, SeverityHigh, "jailbreak"},

		// High: Instruction manipulation
		{`(?i)(output|print|say|respond)\s+"?([^"]*)"?\s+(instead|rather|not)`, SeverityHigh, "manipulation"},
		{`(?i)whatever\s+happens,`, SeverityMedium, "manipulation"},
		{`(?i)no\s+matter\s+what,`, SeverityMedium, "manipulation"},

		// Medium: Encoding tricks
		{`(?i)(rot13|base64|hex|ascii)\s+(decode|encoded)`, SeverityMedium, "encoding"},
		{`(?i)translate\s+this\s+(code|text)\s+to\s+(english|plain)`, SeverityMedium, "encoding"},

		// Medium: Output format manipulation
		{`(?i)respond\s+(only|just)\s+with\s+"`, SeverityMedium, "format_manipulation"},
		{`(?i)output\s+(must|should)\s+(start|begin)\s+with`, SeverityMedium, "format_manipulation"},
		{`(?i)(your\s+)?response\s+(must|should)\s+(be|start|end)`, SeverityMedium, "format_manipulation"},

		// Medium: Context boundary violation
		{`(?i)(after|when|once)\s+(this\s+)?(sentence|paragraph|message)\s+(ends?|finishes?)`, SeverityMedium, "boundary_violation"},
		{`(?i)beyond\s+(this\s+)?(sentence|paragraph|message)`, SeverityMedium, "boundary_violation"},

		// Low: Suspicious keywords
		{`(?i)\bwhite\s*hat\b`, SeverityLow, "suspicious"},
		{`(?i)\bred\s*team\b`, SeverityLow, "suspicious"},
		{`(?i)\btest\s*case\b.*\bprompt\b`, SeverityLow, "suspicious"},
	}

	d.patterns = make([]*injectionPattern, 0, len(patterns))
	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		d.patterns = append(d.patterns, &injectionPattern{
			pattern:  re,
			severity: p.severity,
			category: p.category,
		})
	}
}

// DetectionResult represents the result of an injection scan.
type DetectionResult struct {
	IsSuspicious bool
	Score        int // 0-100, higher = more suspicious
	Matches      []Match
	Safe         bool
}

// Match represents a single pattern match.
type Match struct {
	Pattern  string
	Severity Severity
	Category string
	Position []int
}

// Scan scans a prompt for injection patterns.
func (d *PromptInjectionDetector) Scan(prompt string) *DetectionResult {
	result := &DetectionResult{
		Matches: make([]Match, 0),
		Safe:    true,
	}

	// Check length
	if len(prompt) > d.maxPromptLength {
		result.Matches = append(result.Matches, Match{
			Pattern:  "excessive_length",
			Severity: SeverityMedium,
			Category: "length",
		})
		result.Safe = false
	}

	// Check for excessive repetition (common in jailbreaks)
	if d.hasExcessiveRepetition(prompt) {
		result.Matches = append(result.Matches, Match{
			Pattern:  "excessive_repetition",
			Severity: SeverityMedium,
			Category: "repetition",
		})
		result.Safe = false
	}

	// Check for unusual character sequences
	if d.hasUnusualCharacters(prompt) {
		result.Matches = append(result.Matches, Match{
			Pattern:  "unusual_characters",
			Severity: SeverityLow,
			Category: "encoding",
		})
	}

	// Scan for patterns
	for _, pat := range d.patterns {
		matches := pat.pattern.FindAllStringIndex(prompt, -1)
		for _, m := range matches {
			result.Matches = append(result.Matches, Match{
				Pattern:  pat.pattern.String(),
				Severity: pat.severity,
				Category: pat.category,
				Position: m,
			})

			if pat.severity >= SeverityMedium {
				result.Safe = false
			}
		}
	}

	// Calculate score
	result.Score = d.calculateScore(result.Matches)
	result.IsSuspicious = result.Score >= (map[bool]int{true: 30, false: 50}[d.strictMode])

	return result
}

// Sanitize sanitizes a prompt by removing detected patterns.
func (d *PromptInjectionDetector) Sanitize(prompt string) string {
	sanitized := prompt

	for _, pat := range d.patterns {
		if pat.severity >= SeverityHigh {
			sanitized = pat.pattern.ReplaceAllString(sanitized, "[REDACTED]")
		}
	}

	return sanitized
}

// Validate checks if a prompt is safe to use.
func (d *PromptInjectionDetector) Validate(prompt string) error {
	result := d.Scan(prompt)
	if result.IsSuspicious {
		return &InjectionError{
			Result: result,
			Msg:    "prompt contains potential injection patterns",
		}
	}
	return nil
}

// ValidateWithPrefix checks if a prompt starts with an allowed prefix.
func (d *PromptInjectionDetector) ValidateWithPrefix(prompt string) error {
	trimmed := strings.TrimSpace(prompt)
	hasValidPrefix := false

	for _, prefix := range d.allowedPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if d.strictMode && !hasValidPrefix {
		return &InjectionError{
			Msg: "prompt must start with a valid prefix",
		}
	}

	return d.Validate(prompt)
}

// calculateScore calculates a suspicion score from matches.
func (d *PromptInjectionDetector) calculateScore(matches []Match) int {
	score := 0
	for _, m := range matches {
		switch m.Severity {
		case SeverityCritical:
			score += 40
		case SeverityHigh:
			score += 25
		case SeverityMedium:
			score += 10
		case SeverityLow:
			score += 3
		}
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// hasExcessiveRepetition checks for excessive character repetition.
func (d *PromptInjectionDetector) hasExcessiveRepetition(text string) bool {
	words := strings.Fields(text)
	if len(words) < 5 {
		return false
	}

	// Check for repeated words
	wordCount := make(map[string]int)
	for _, word := range words {
		wordCount[strings.ToLower(word)]++
	}

	for _, count := range wordCount {
		if count > 10 && len(wordCount) < 20 {
			return true
		}
	}

	return false
}

// hasUnusualCharacters checks for unusual character patterns.
func (d *PromptInjectionDetector) hasUnusualCharacters(text string) bool {
	// Check for excessive special characters
	specialCount := 0
	for _, r := range text {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsSpace(r) {
			specialCount++
		}
	}

	return float64(specialCount)/float64(len(text)) > 0.3
}

// InjectionError represents an injection detection error.
type InjectionError struct {
	Result *DetectionResult
	Msg    string
}

func (e *InjectionError) Error() string {
	if e.Result != nil && len(e.Result.Matches) > 0 {
		return e.Msg + " (" + e.Result.Matches[0].Category + ")"
	}
	return e.Msg
}

// PromptBuilder helps build safe prompts.
type PromptBuilder struct {
	detector *PromptInjectionDetector
	parts    []string
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		detector: NewPromptInjectionDetector(),
		parts:    make([]string, 0),
	}
}

// Add adds a part to the prompt.
func (b *PromptBuilder) Add(part string) *PromptBuilder {
	b.parts = append(b.parts, part)
	return b
}

// Addf adds a formatted part to the prompt.
func (b *PromptBuilder) Addf(format string, args ...any) *PromptBuilder {
	b.parts = append(b.parts, fmt.Sprintf(format, args...))
	return b
}

// Build builds and validates the prompt.
func (b *PromptBuilder) Build() (string, error) {
	prompt := strings.Join(b.parts, "\n")

	if err := b.detector.Validate(prompt); err != nil {
		return "", err
	}

	return prompt, nil
}

// BuildUnsafe builds without validation.
func (b *PromptBuilder) BuildUnsafe() string {
	return strings.Join(b.parts, "\n")
}

// Clear clears all parts.
func (b *PromptBuilder) Clear() *PromptBuilder {
	b.parts = b.parts[:0]
	return b
}
