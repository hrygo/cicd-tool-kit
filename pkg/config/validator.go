// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package config

import (
	"fmt"
	"strings"
)

// Validator validates configuration.
type Validator struct {
	allowSecrets bool // For testing only
}

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{
		allowSecrets: false,
	}
}

// Validate validates a configuration.
func (v *Validator) Validate(cfg *Config) error {
	if err := v.ValidateClaude(&cfg.Claude); err != nil {
		return err
	}
	if err := v.ValidatePlatform(&cfg.Platform); err != nil {
		return err
	}
	if err := v.ValidateGlobal(&cfg.Global); err != nil {
		return err
	}
	return nil
}

// ValidateClaude validates Claude configuration.
func (v *Validator) ValidateClaude(cfg *ClaudeConfig) error {
	// Model validation
	validModels := []string{"sonnet", "opus", "haiku"}
	if cfg.Model != "" {
		valid := false
		for _, m := range validModels {
			if strings.EqualFold(cfg.Model, m) {
				valid = true
				break
			}
		}
		if !valid {
			return &ValidationError{
				Field:   "claude.model",
				Value:   cfg.Model,
				Message: fmt.Sprintf("must be one of: %s", strings.Join(validModels, ", ")),
			}
		}
	}

	// Timeout validation
	if cfg.Timeout < 0 {
		return &ValidationError{
			Field:   "claude.timeout",
			Value:   cfg.Timeout,
			Message: "must be positive",
		}
	}

	// Budget validation
	if cfg.MaxBudget < 0 {
		return &ValidationError{
			Field:   "claude.max_budget_usd",
			Value:   cfg.MaxBudget,
			Message: "must be non-negative",
		}
	}

	return nil
}

// ValidatePlatform validates platform configuration.
func (v *Validator) ValidatePlatform(cfg *PlatformConfig) error {
	// Check for plaintext secrets (security validation)
	if !v.allowSecrets {
		if err := v.checkForSecrets(cfg); err != nil {
			return err
		}
	}
	return nil
}

// checkForSecrets checks that no plaintext secrets are in config.
func (v *Validator) checkForSecrets(cfg *PlatformConfig) error {
	if cfg.GitHub != nil {
		if cfg.GitHub.TokenEnv == "" {
			return &ValidationError{
				Field:   "platform.github.token_env",
				Message: "must be set (token field is not allowed)",
			}
		}
	}
	return nil
}

// ValidateGlobal validates global configuration.
func (v *Validator) ValidateGlobal(cfg *GlobalConfig) error {
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if cfg.LogLevel != "" {
		valid := false
		for _, level := range validLogLevels {
			if strings.EqualFold(cfg.LogLevel, level) {
				valid = true
				break
			}
		}
		if !valid {
			return &ValidationError{
				Field:   "global.log_level",
				Value:   cfg.LogLevel,
				Message: fmt.Sprintf("must be one of: %s", strings.Join(validLogLevels, ", ")),
			}
		}
	}

	if cfg.MaxCacheMB < 0 {
		return &ValidationError{
			Field:   "global.max_cache_mb",
			Value:   cfg.MaxCacheMB,
			Message: "must be non-negative",
		}
	}

	return nil
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation error for %s: %s (got: %v)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}
