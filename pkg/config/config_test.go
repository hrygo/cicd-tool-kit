// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cicd-ai-toolkit/pkg/config"
)

// TestDefaultConfig tests the default configuration.
func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Claude.Model != "sonnet" {
		t.Errorf("Expected default model 'sonnet', got '%s'", cfg.Claude.Model)
	}

	if cfg.Claude.Timeout != 300*time.Second {
		t.Errorf("Expected default timeout 300s, got %v", cfg.Claude.Timeout)
	}

	if len(cfg.Skills) != 1 {
		t.Errorf("Expected 1 default skill, got %d", len(cfg.Skills))
	}

	if cfg.Skills[0].Name != "code-reviewer" {
		t.Errorf("Expected default skill 'code-reviewer', got '%s'", cfg.Skills[0].Name)
	}

	if cfg.Global.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", cfg.Global.LogLevel)
	}
}

// TestLoadFromPath tests loading config from a file.
func TestLoadFromPath(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
claude:
  model: opus
  timeout: 600s
  max_budget_usd: 20.0

global:
  log_level: debug
  max_cache_mb: 200

skills:
  - name: test-generator
    enabled: true
    priority: 2

platform:
  github:
    token_env: "GITHUB_TOKEN"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := config.NewLoader()
	cfg, err := loader.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Claude.Model != "opus" {
		t.Errorf("Expected model 'opus', got '%s'", cfg.Claude.Model)
	}

	if cfg.Claude.Timeout != 600*time.Second {
		t.Errorf("Expected timeout 600s, got %v", cfg.Claude.Timeout)
	}

	if cfg.Claude.MaxBudget != 20.0 {
		t.Errorf("Expected max_budget 20.0, got %f", cfg.Claude.MaxBudget)
	}

	if cfg.Global.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.Global.LogLevel)
	}

	if cfg.Global.MaxCacheMB != 200 {
		t.Errorf("Expected max_cache_mb 200, got %d", cfg.Global.MaxCacheMB)
	}
}

// TestLoadFromPathInvalid tests loading an invalid config file.
func TestLoadFromPathInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
claude:
  model: invalid_model
  timeout: not_a_duration
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := config.NewLoader()
	_, err := loader.LoadFromPath(configPath)
	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}
}

// TestLoadWithEnvOverrides tests environment variable overrides.
func TestLoadWithEnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("CICD_TOOLKIT_CLAUDE__MODEL", "haiku")
	os.Setenv("CICD_TOOLKIT_CLAUDE__TIMEOUT", "120s")
	os.Setenv("CICD_TOOLKIT_GLOBAL__LOG_LEVEL", "warn")
	defer func() {
		os.Unsetenv("CICD_TOOLKIT_CLAUDE__MODEL")
		os.Unsetenv("CICD_TOOLKIT_CLAUDE__TIMEOUT")
		os.Unsetenv("CICD_TOOLKIT_GLOBAL__LOG_LEVEL")
	}()

	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify env overrides
	if cfg.Claude.Model != "haiku" {
		t.Errorf("Expected model 'haiku' from env, got '%s'", cfg.Claude.Model)
	}

	if cfg.Claude.Timeout != 120*time.Second {
		t.Errorf("Expected timeout 120s from env, got %v", cfg.Claude.Timeout)
	}

	if cfg.Global.LogLevel != "warn" {
		t.Errorf("Expected log level 'warn' from env, got '%s'", cfg.Global.LogLevel)
	}
}

// TestLoadWithEnvInvalidTimeout tests invalid timeout from env.
func TestLoadWithEnvInvalidTimeout(t *testing.T) {
	os.Setenv("CICD_TOOLKIT_CLAUDE__TIMEOUT", "invalid")
	defer os.Unsetenv("CICD_TOOLKIT_CLAUDE__TIMEOUT")

	loader := config.NewLoader()
	_, err := loader.Load()
	if err == nil {
		t.Error("Expected error for invalid timeout in env, got nil")
	}
}

// TestValidator tests the configuration validator.
func TestValidator(t *testing.T) {
	v := config.NewValidator()

	// Test valid config
	cfg := config.DefaultConfig()
	if err := v.Validate(cfg); err != nil {
		t.Errorf("Valid config should pass validation, got error: %v", err)
	}

	// Test invalid model
	invalidModel := config.DefaultConfig()
	invalidModel.Claude.Model = "invalid"
	if err := v.Validate(invalidModel); err == nil {
		t.Error("Invalid model should fail validation")
	}

	// Test invalid log level
	invalidLogLevel := config.DefaultConfig()
	invalidLogLevel.Global.LogLevel = "trace"
	if err := v.Validate(invalidLogLevel); err == nil {
		t.Error("Invalid log level should fail validation")
	}

	// Test negative timeout
	negativeTimeout := config.DefaultConfig()
	negativeTimeout.Claude.Timeout = -1
	if err := v.Validate(negativeTimeout); err == nil {
		t.Error("Negative timeout should fail validation")
	}
}

// TestValidatorPlatformSecrets tests secret validation.
func TestValidatorPlatformSecrets(t *testing.T) {
	v := config.NewValidator()

	// Valid config with token_env
	validCfg := config.DefaultConfig()
	validCfg.Platform.GitHub = &config.GitHubPlatformConfig{
		TokenEnv: "GITHUB_TOKEN",
	}
	if err := v.Validate(validCfg); err != nil {
		t.Errorf("Config with token_env should be valid, got error: %v", err)
	}

	// Invalid config without token_env
	invalidCfg := config.DefaultConfig()
	invalidCfg.Platform.GitHub = &config.GitHubPlatformConfig{}
	if err := v.Validate(invalidCfg); err == nil {
		t.Error("Config without token_env should fail validation")
	}
}

// TestMergeConfig tests config merging.
func TestMergeConfig(t *testing.T) {
	// This tests internal mergeConfig functionality via loader
	tmpDir := t.TempDir()
	projectConfigPath := filepath.Join(tmpDir, ".cicd-ai-toolkit.yaml")
	projectConfigContent := `
claude:
  model: opus

skills:
  - name: test-skill
    enabled: true
`
	if err := os.WriteFile(projectConfigPath, []byte(projectConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := config.NewLoader().WithProjectRoot(tmpDir).SkipGlobal()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults are preserved where not overridden
	if cfg.Claude.Timeout != 300*time.Second {
		t.Errorf("Default timeout should be preserved, got %v", cfg.Claude.Timeout)
	}

	// Verify override
	if cfg.Claude.Model != "opus" {
		t.Errorf("Model should be overridden to 'opus', got '%s'", cfg.Claude.Model)
	}

	if len(cfg.Skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(cfg.Skills))
	}

	if cfg.Skills[0].Name != "test-skill" {
		t.Errorf("Expected skill 'test-skill', got '%s'", cfg.Skills[0].Name)
	}
}

// TestFindConfigPaths tests finding config files.
func TestFindConfigPaths(t *testing.T) {
	paths := config.FindConfigPaths()
	// Should return empty array if no configs exist
	if paths == nil {
		t.Error("Expected non-nil paths array")
	}
}

// TestGetEnvConfig tests getting environment config.
func TestGetEnvConfig(t *testing.T) {
	os.Setenv("CICD_TOOLKIT_CLAUDE__MODEL", "opus")
	defer os.Unsetenv("CICD_TOOLKIT_CLAUDE__MODEL")

	envCfg := config.GetEnvConfig()
	if envCfg == nil {
		t.Error("Expected non-nil env config map")
	}

	if _, ok := envCfg["CICD_TOOLKIT_CLAUDE__MODEL"]; !ok {
		t.Error("Expected CICD_TOOLKIT_CLAUDE__MODEL in env config")
	}
}

// TestValidationError tests ValidationError formatting.
func TestValidationError(t *testing.T) {
	err := &config.ValidationError{
		Field:   "test.field",
		Value:   "invalid",
		Message: "is not valid",
	}
	expected := "validation error for test.field: is not valid (got: invalid)"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

// TestConfigError tests ConfigError formatting.
func TestConfigError(t *testing.T) {
	err := &config.ConfigError{
		Path: "/path/to/config.yaml",
		Err:   &config.ValidationError{Field: "test", Message: "failed"},
	}
	expected := "config error in /path/to/config.yaml: validation error for test: failed"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

// TestConfigErrorField tests ConfigError with field.
func TestConfigErrorField(t *testing.T) {
	err := &config.ConfigError{
		Field: "claude.timeout",
		Err:   &config.ValidationError{Field: "test", Message: "failed"},
	}
	expected := "config error for claude.timeout: validation error for test: failed"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

// TestPrecedenceOrder tests config precedence: defaults < global < project < env.
func TestPrecedenceOrder(t *testing.T) {
	// Setup: create global and project configs
	tmpDir := t.TempDir()

	// Global config: model = opus
	globalDir := filepath.Join(tmpDir, "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatalf("Failed to create global dir: %v", err)
	}
	globalConfigPath := filepath.Join(globalDir, "config.yaml")
	globalConfigContent := `
claude:
  model: opus
  timeout: 200s
`
	if err := os.WriteFile(globalConfigPath, []byte(globalConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}

	// Project config: model = haiku, timeout = 400s
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	projectConfigPath := filepath.Join(projectDir, ".cicd-ai-toolkit.yaml")
	projectConfigContent := `
claude:
  model: haiku
  timeout: 400s
`
	if err := os.WriteFile(projectConfigPath, []byte(projectConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write project config: %v", err)
	}

	// Set HOME to tmpDir for global config lookup
	home := os.Getenv("HOME")
	defer os.Setenv("HOME", home)
	os.Setenv("HOME", tmpDir)

	loader := config.NewLoader().WithProjectRoot(projectDir)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify precedence:
	// - model from project (haiku) overrides global (opus)
	// - timeout from project (400s) overrides global (200s)
	if cfg.Claude.Model != "haiku" {
		t.Errorf("Expected model 'haiku' from project config, got '%s'", cfg.Claude.Model)
	}

	if cfg.Claude.Timeout != 400*time.Second {
		t.Errorf("Expected timeout 400s from project config, got %v", cfg.Claude.Timeout)
	}

	// Now set env var to override project config
	os.Setenv("CICD_TOOLKIT_CLAUDE__MODEL", "sonnet")
	defer os.Unsetenv("CICD_TOOLKIT_CLAUDE__MODEL")

	cfg2, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config with env: %v", err)
	}

	if cfg2.Claude.Model != "sonnet" {
		t.Errorf("Expected model 'sonnet' from env, got '%s'", cfg2.Claude.Model)
	}

	// Timeout should still come from project config
	if cfg2.Claude.Timeout != 400*time.Second {
		t.Errorf("Expected timeout 400s from project config, got %v", cfg2.Claude.Timeout)
	}
}

// TestDefaultConfigPaths tests default path functions.
func TestDefaultConfigPaths(t *testing.T) {
	cachePath := config.GetDefaultCachePath()
	if cachePath == "" {
		t.Error("Expected non-empty cache path")
	}

	configPath := config.GetDefaultConfigPath()
	if configPath == "" {
		t.Error("Expected non-empty config path")
	}

	projectPath := config.GetProjectConfigPath("")
	if projectPath == "" {
		t.Error("Expected non-empty project path")
	}

	if projectPath != ".cicd-ai-toolkit.yaml" {
		t.Errorf("Expected '.cicd-ai-toolkit.yaml', got '%s'", projectPath)
	}
}
