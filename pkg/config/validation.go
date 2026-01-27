// Package config handles configuration loading and validation
package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate version
	if c.Version == "" {
		return fmt.Errorf("config version is required")
	}

	// Validate Claude config
	if err := c.Claude.Validate(); err != nil {
		return fmt.Errorf("claude config: %w", err)
	}

	// Validate skills
	for i, skill := range c.Skills {
		if err := skill.Validate(); err != nil {
			return fmt.Errorf("skills[%d]: %w", i, err)
		}
	}

	// Validate global config
	if err := c.Global.Validate(); err != nil {
		return fmt.Errorf("global config: %w", err)
	}

	// Validate advanced config if present
	if c.Advanced.Memory.Enabled {
		if err := c.Advanced.Memory.Validate(); err != nil {
			return fmt.Errorf("memory config: %w", err)
		}
	}

	return nil
}

// Validate validates the Claude configuration
func (c *ClaudeConfig) Validate() error {
	// Validate model
	validModels := map[string]bool{
		"haiku":  true,
		"sonnet": true,
		"opus":   true,
	}
	if !validModels[strings.ToLower(c.Model)] {
		return fmt.Errorf("invalid model: %s (must be haiku, sonnet, or opus)", c.Model)
	}

	// Validate budget
	if c.MaxBudgetUSD < 0 {
		return fmt.Errorf("max_budget_usd must be non-negative")
	}

	// Validate max turns
	if c.MaxTurns < 1 {
		return fmt.Errorf("max_turns must be at least 1")
	}
	if c.MaxTurns > 1000 {
		return fmt.Errorf("max_turns must not exceed 1000")
	}

	// Validate timeout format
	if _, err := time.ParseDuration(c.Timeout); err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	// Validate output format
	validFormats := map[string]bool{
		"text":        true,
		"json":        true,
		"stream-json": true,
	}
	if c.OutputFormat != "" && !validFormats[c.OutputFormat] {
		return fmt.Errorf("invalid output_format: %s (must be text, json, or stream-json)", c.OutputFormat)
	}

	return nil
}

// Validate validates the skill configuration
func (s *SkillConfig) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if s.Path == "" {
		return fmt.Errorf("skill path is required")
	}
	if s.Priority < 0 {
		return fmt.Errorf("skill priority must be non-negative")
	}
	return nil
}

// Validate validates the global configuration
func (g *GlobalConfig) Validate() error {
	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if g.LogLevel != "" && !validLevels[strings.ToLower(g.LogLevel)] {
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", g.LogLevel)
	}

	// Validate parallel skills
	if g.ParallelSkills < 1 {
		return fmt.Errorf("parallel_skills must be at least 1")
	}
	if g.ParallelSkills > 10 {
		return fmt.Errorf("parallel_skills must not exceed 10")
	}

	// Validate diff context
	if g.DiffContext < 0 {
		return fmt.Errorf("diff_context must be non-negative")
	}
	if g.DiffContext > 20 {
		return fmt.Errorf("diff_context must not exceed 20")
	}

	return nil
}

// Validate validates the memory configuration
func (m *MemoryConfig) Validate() error {
	if !m.Enabled {
		return nil
	}

	validBackends := map[string]bool{
		"file":     true,
		"redis":    true,
		"postgres": true,
		"memory":   true,
	}
	if !validBackends[m.Backend] {
		return fmt.Errorf("invalid memory backend: %s (must be file, redis, postgres, or memory)", m.Backend)
	}

	// Validate TTL format
	if _, err := time.ParseDuration(m.TTL); err != nil {
		return fmt.Errorf("invalid memory TTL format: %w", err)
	}

	return nil
}
