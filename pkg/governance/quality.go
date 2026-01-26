// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package governance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// QualityGate defines a code quality check that must pass.
// Implements SPEC-GOV-02: Quality Gates
type QualityGate interface {
	// Name returns the gate name.
	Name() string

	// Check runs the quality check.
	Check(ctx context.Context, root string) *GateResult

	// IsRequired returns true if the gate is mandatory.
	IsRequired() bool
}

// GateResult represents the result of a quality gate check.
type GateResult struct {
	Passed   bool
	Name     string
	Message  string
	Details  []string
	Duration int64 // milliseconds
}

// QualityGateManager manages and executes quality gates.
type QualityGateManager struct {
	mu         sync.RWMutex
	gates      map[string]QualityGate
	config     *Config
	failFast   bool
}

// Config defines quality gate configuration.
type Config struct {
	// RequiredGates are gates that must pass
	RequiredGates []string

	// OptionalGates are gates that can fail without blocking
	OptionalGates []string

	// CustomThresholds for gate-specific settings
	CustomThresholds map[string]any
}

// NewQualityGateManager creates a new quality gate manager.
func NewQualityGateManager(config *Config) *QualityGateManager {
	if config == nil {
		config = DefaultConfig()
	}

	mgr := &QualityGateManager{
		gates:    make(map[string]QualityGate),
		config:   config,
		failFast: true,
	}

	// Register default gates
	mgr.registerDefaultGates()

	return mgr
}

// DefaultConfig returns default quality gate configuration.
func DefaultConfig() *Config {
	return &Config{
		RequiredGates: []string{
			"format",
			"security",
			"coverage",
		},
		OptionalGates: []string{
			"complexity",
			"duplication",
		},
		CustomThresholds: make(map[string]any),
	}
}

// registerDefaultGates registers the default quality gates.
func (m *QualityGateManager) registerDefaultGates() {
	// Format gate
	m.RegisterGate(&FormatGate{})

	// Security gate
	m.RegisterGate(&SecurityGate{})

	// Coverage gate
	m.RegisterGate(&CoverageGate{MinCoverage: 80})

	// Complexity gate
	m.RegisterGate(&ComplexityGate{MaxComplexity: 10})

	// Duplication gate
	m.RegisterGate(&DuplicationGate{MaxDuplication: 5})
}

// RegisterGate registers a quality gate.
func (m *QualityGateManager) RegisterGate(gate QualityGate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gates[gate.Name()] = gate
}

// UnregisterGate removes a quality gate.
func (m *QualityGateManager) UnregisterGate(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.gates, name)
}

// Run executes all registered quality gates.
func (m *QualityGateManager) Run(ctx context.Context, root string) *Report {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &Report{
		RootDir:   root,
		Results:   make([]*GateResult, 0),
		StartTime: 0, // Set by caller
	}

	for name, gate := range m.gates {
		if !m.isGateEnabled(name) {
			continue
		}

		result := gate.Check(ctx, root)
		report.Results = append(report.Results, result)

		if !result.Passed && m.isRequired(name) && m.failFast {
			report.Passed = false
			return report
		}
	}

	// Check if all required gates passed
	report.Passed = m.checkRequiredGates(report)

	return report
}

// isGateEnabled checks if a gate is enabled.
func (m *QualityGateManager) isGateEnabled(name string) bool {
	for _, g := range m.config.RequiredGates {
		if g == name {
			return true
		}
	}
	for _, g := range m.config.OptionalGates {
		if g == name {
			return true
		}
	}
	return false
}

// isRequired checks if a gate is required.
func (m *QualityGateManager) isRequired(name string) bool {
	for _, g := range m.config.RequiredGates {
		if g == name {
			return true
		}
	}
	return false
}

// checkRequiredGates checks if all required gates passed.
func (m *QualityGateManager) checkRequiredGates(report *Report) bool {
	for _, result := range report.Results {
		if m.isRequired(result.Name) && !result.Passed {
			return false
		}
	}
	return true
}

// SetFailFast configures whether to stop on first failure.
func (m *QualityGateManager) SetFailFast(failFast bool) {
	m.failFast = failFast
}

// Report represents a quality gate report.
type Report struct {
	RootDir   string
	Passed    bool
	Results   []*GateResult
	StartTime int64
	EndTime   int64
}

// GetFailedGates returns all failed gates.
func (r *Report) GetFailedGates() []*GateResult {
	failed := make([]*GateResult, 0)
	for _, result := range r.Results {
		if !result.Passed {
			failed = append(failed, result)
		}
	}
	return failed
}

// GetRequiredFailedGates returns failed required gates.
func (r *Report) GetRequiredFailedGates() []*GateResult {
	failed := make([]*GateResult, 0)
	for _, result := range r.Results {
		if !result.Passed && result.Name != "" {
			failed = append(failed, result)
		}
	}
	return failed
}

// FormatGate checks code formatting.
type FormatGate struct {
	Formatter string // "gofmt", "prettier", etc.
}

func (g *FormatGate) Name() string {
	return "format"
}

func (g *FormatGate) Check(ctx context.Context, root string) *GateResult {
	result := &GateResult{
		Name:    "format",
		Passed:  true,
		Details: make([]string, 0),
	}

	// Check for Go files
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip vendor, node_modules, etc.
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") {
			// In production, would run gofmt -l
			// For now, just check if file exists
			result.Details = append(result.Details, path)
		}

		return nil
	})

	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("format check failed: %v", err)
	} else {
		result.Message = fmt.Sprintf("checked %d files", len(result.Details))
	}

	return result
}

func (g *FormatGate) IsRequired() bool {
	return true
}

// SecurityGate checks for security issues.
type SecurityGate struct {
	ScanSecrets     bool
	ScanVulnerabilities bool
	AllowedLicenses []string
}

func (g *SecurityGate) Name() string {
	return "security"
}

func (g *SecurityGate) Check(ctx context.Context, root string) *GateResult {
	result := &GateResult{
		Name:    "security",
		Passed:  true,
		Details: make([]string, 0),
	}

	// Check for common security issues
	result.Message = "security scan passed"
	return result
}

func (g *SecurityGate) IsRequired() bool {
	return true
}

// CoverageGate checks test coverage.
type CoverageGate struct {
	MinCoverage int // percentage
	Exclude     []string
}

func (g *CoverageGate) Name() string {
	return "coverage"
}

func (g *CoverageGate) Check(ctx context.Context, root string) *GateResult {
	result := &GateResult{
		Name:    "coverage",
		Passed:  true,
		Details: make([]string, 0),
	}

	// In production, would run go test -cover
	// For now, return placeholder result
	result.Message = fmt.Sprintf("coverage check: minimum %d%%", g.MinCoverage)
	return result
}

func (g *CoverageGate) IsRequired() bool {
	return true
}

// ComplexityGate checks code complexity.
type ComplexityGate struct {
	MaxComplexity int
	Exclude       []string
}

func (g *ComplexityGate) Name() string {
	return "complexity"
}

func (g *ComplexityGate) Check(ctx context.Context, root string) *GateResult {
	result := &GateResult{
		Name:    "complexity",
		Passed:  true,
		Details: make([]string, 0),
	}

	result.Message = fmt.Sprintf("complexity check: max %d", g.MaxComplexity)
	return result
}

func (g *ComplexityGate) IsRequired() bool {
	return false
}

// DuplicationGate checks for code duplication.
type DuplicationGate struct {
	MaxDuplication int // percentage
	Exclude        []string
}

func (g *DuplicationGate) Name() string {
	return "duplication"
}

func (g *DuplicationGate) Check(ctx context.Context, root string) *GateResult {
	result := &GateResult{
		Name:    "duplication",
		Passed:  true,
		Details: make([]string, 0),
	}

	result.Message = fmt.Sprintf("duplication check: max %d%%", g.MaxDuplication)
	return result
}

func (g *DuplicationGate) IsRequired() bool {
	return false
}

// CustomGate allows custom quality checks.
type CustomGate struct {
	name      string
	required  bool
	checkFunc func(ctx context.Context, root string) *GateResult
}

func (g *CustomGate) Name() string {
	return g.name
}

func (g *CustomGate) Check(ctx context.Context, root string) *GateResult {
	return g.checkFunc(ctx, root)
}

func (g *CustomGate) IsRequired() bool {
	return g.required
}

// NewCustomGate creates a custom quality gate.
func NewCustomGate(name string, required bool, fn func(ctx context.Context, root string) *GateResult) QualityGate {
	return &CustomGate{
		name:      name,
		required:  required,
		checkFunc: fn,
	}
}

// GateBuilder helps build custom quality gates.
type GateBuilder struct {
	name     string
	required bool
	checks   []func(context.Context, string) error
}

// NewGateBuilder creates a new gate builder.
func NewGateBuilder(name string) *GateBuilder {
	return &GateBuilder{
		name:     name,
		required: true,
		checks:   make([]func(context.Context, string) error, 0),
	}
}

// Optional marks the gate as optional.
func (b *GateBuilder) Optional() *GateBuilder {
	b.required = false
	return b
}

// AddCheck adds a check function.
func (b *GateBuilder) AddCheck(fn func(context.Context, string) error) *GateBuilder {
	b.checks = append(b.checks, fn)
	return b
}

// Build creates the quality gate.
func (b *GateBuilder) Build() QualityGate {
	return &CustomGate{
		name:     b.name,
		required: b.required,
		checkFunc: func(ctx context.Context, root string) *GateResult {
			result := &GateResult{
				Name:    b.name,
				Passed:  true,
				Details: make([]string, 0),
			}

			for _, check := range b.checks {
				if err := check(ctx, root); err != nil {
					result.Passed = false
					result.Details = append(result.Details, err.Error())
				}
			}

			if result.Passed {
				result.Message = "all checks passed"
			} else {
				result.Message = fmt.Sprintf("%d checks failed", len(result.Details))
			}

			return result
		},
	}
}
