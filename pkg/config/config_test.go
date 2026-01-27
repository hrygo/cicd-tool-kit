// Package config provides configuration unit tests
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
version: "2.0"
claude:
  model: sonnet
  max_budget_usd: 5.0
  max_turns: 50
  timeout: 30m
  output_format: json
  dangerous_skip_permissions: true
skills:
  - name: code-reviewer
    path: ./skills/code-reviewer
    enabled: true
    priority: 100
platform:
  github:
    post_comment: true
    fail_on_error: false
    max_comment_length: 65536
  gitee:
    api_url: https://gitee.com/api/v5
    post_comment: true
  gitlab:
    post_comment: true
    fail_on_error: false
global:
  log_level: info
  cache_dir: .cicd-cache
  enable_cache: true
  parallel_skills: 1
  diff_context: 3
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", cfg.Version)
	}
	if cfg.Claude.Model != "sonnet" {
		t.Errorf("Expected model sonnet, got %s", cfg.Claude.Model)
	}
	if cfg.Claude.MaxBudgetUSD != 5.0 {
		t.Errorf("Expected MaxBudgetUSD 5.0, got %f", cfg.Claude.MaxBudgetUSD)
	}
	if cfg.Claude.MaxTurns != 50 {
		t.Errorf("Expected MaxTurns 50, got %d", cfg.Claude.MaxTurns)
	}
	if cfg.Global.LogLevel != "info" {
		t.Errorf("Expected log_level info, got %s", cfg.Global.LogLevel)
	}
}

func TestLoadWithInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	if err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Version: "2.0",
				Claude: ClaudeConfig{
					Model:        "sonnet",
					MaxBudgetUSD: 5.0,
					MaxTurns:     50,
					Timeout:      "30m",
					OutputFormat: "json",
				},
				Global: GlobalConfig{
					LogLevel:      "info",
					ParallelSkills: 1,
					DiffContext:   3,
				},
				Skills: []SkillConfig{
					{Name: "test", Path: "./skills/test", Enabled: true},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid model",
			cfg: &Config{
				Version: "2.0",
				Claude: ClaudeConfig{
					Model:        "invalid",
					MaxBudgetUSD: 5.0,
					MaxTurns:     50,
					Timeout:      "30m",
				},
				Global: GlobalConfig{
					LogLevel:      "info",
					ParallelSkills: 1,
					DiffContext:   3,
				},
			},
			wantErr: true,
		},
		{
			name: "negative budget",
			cfg: &Config{
				Version: "2.0",
				Claude: ClaudeConfig{
					Model:        "sonnet",
					MaxBudgetUSD: -1.0,
					MaxTurns:     50,
					Timeout:      "30m",
				},
				Global: GlobalConfig{
					LogLevel:      "info",
					ParallelSkills: 1,
					DiffContext:   3,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout format",
			cfg: &Config{
				Version: "2.0",
				Claude: ClaudeConfig{
					Model:        "sonnet",
					MaxBudgetUSD: 5.0,
					MaxTurns:     50,
					Timeout:      "invalid",
				},
				Global: GlobalConfig{
					LogLevel:      "info",
					ParallelSkills: 1,
					DiffContext:   3,
				},
			},
			wantErr: true,
		},
		{
			name: "max turns too high",
			cfg: &Config{
				Version: "2.0",
				Claude: ClaudeConfig{
					Model:        "sonnet",
					MaxBudgetUSD: 5.0,
					MaxTurns:     2000,
					Timeout:      "30m",
				},
				Global: GlobalConfig{
					LogLevel:      "info",
					ParallelSkills: 1,
					DiffContext:   3,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				Version: "2.0",
				Claude: ClaudeConfig{
					Model:        "sonnet",
					MaxBudgetUSD: 5.0,
					MaxTurns:     50,
					Timeout:      "30m",
				},
				Global: GlobalConfig{
					LogLevel:      "invalid",
					ParallelSkills: 1,
					DiffContext:   3,
				},
			},
			wantErr: true,
		},
		{
			name: "parallel skills too low",
			cfg: &Config{
				Version: "2.0",
				Claude: ClaudeConfig{
					Model:        "sonnet",
					MaxBudgetUSD: 5.0,
					MaxTurns:     50,
					Timeout:      "30m",
				},
				Global: GlobalConfig{
					LogLevel:      "info",
					ParallelSkills: 0,
					DiffContext:   3,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTimeout(t *testing.T) {
	cfg := ClaudeConfig{Timeout: "5m30s"}
	duration, err := cfg.GetTimeout()
	if err != nil {
		t.Fatalf("GetTimeout() error = %v", err)
	}
	expected := 5*time.Minute + 30*time.Second
	if duration != expected {
		t.Errorf("GetTimeout() = %v, want %v", duration, expected)
	}
}

func TestIsEnabled(t *testing.T) {
	cfg := &Config{
		Skills: []SkillConfig{
			{Name: "enabled-skill", Path: "./skills/enabled", Enabled: true},
			{Name: "disabled-skill", Path: "./skills/disabled", Enabled: false},
		},
	}

	tests := []struct {
		name     string
		skill    string
		expected bool
	}{
		{"enabled skill", "enabled-skill", true},
		{"disabled skill", "disabled-skill", false},
		{"non-existent skill", "non-existent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.IsEnabled(tt.skill); got != tt.expected {
				t.Errorf("IsEnabled(%s) = %v, want %v", tt.skill, got, tt.expected)
			}
		})
	}
}

func TestGetEnabledSkills(t *testing.T) {
	cfg := &Config{
		Skills: []SkillConfig{
			{Name: "skill1", Path: "./s1", Enabled: true},
			{Name: "skill2", Path: "./s2", Enabled: false},
			{Name: "skill3", Path: "./s3", Enabled: true},
		},
	}

	enabled := cfg.GetEnabledSkills()
	if len(enabled) != 2 {
		t.Errorf("GetEnabledSkills() returned %d skills, want 2", len(enabled))
	}

	// Check order is preserved
	if enabled[0] != "skill1" || enabled[1] != "skill3" {
		t.Errorf("GetEnabledSkills() = %v, want [skill1 skill3]", enabled)
	}
}

func TestGetSkillConfig(t *testing.T) {
	testConfig := map[string]any{
		"threshold": "warning",
		"rules":     []string{"rule1", "rule2"},
	}

	cfg := &Config{
		Skills: []SkillConfig{
			{Name: "with-config", Path: "./s1", Enabled: true, Config: testConfig},
			{Name: "no-config", Path: "./s2", Enabled: true},
		},
	}

	t.Run("skill with config", func(t *testing.T) {
		config, ok := cfg.GetSkillConfig("with-config")
		if !ok {
			t.Error("GetSkillConfig() returned ok=false, expected true")
		}
		if config["threshold"] != "warning" {
			t.Errorf("Config threshold = %v, want 'warning'", config["threshold"])
		}
	})

	t.Run("skill without config", func(t *testing.T) {
		config, ok := cfg.GetSkillConfig("no-config")
		if ok {
			t.Error("GetSkillConfig() returned ok=true, expected false")
		}
		if config != nil {
			t.Error("GetSkillConfig() returned non-nil config for skill without config")
		}
	})

	t.Run("non-existent skill", func(t *testing.T) {
		config, ok := cfg.GetSkillConfig("non-existent")
		if ok {
			t.Error("GetSkillConfig() returned ok=true, expected false")
		}
		if config != nil {
			t.Error("GetSkillConfig() returned non-nil config for non-existent skill")
		}
	})
}

func TestLoadWithOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
version: "2.0"
claude:
  model: sonnet
  max_budget_usd: 5.0
  max_turns: 50
  timeout: 30m
global:
  log_level: info
  cache_dir: .cicd-cache
  parallel_skills: 1
  diff_context: 3
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set environment overrides
	t.Setenv("CICD_MODEL", "opus")
	t.Setenv("CICD_MAX_BUDGET", "15.0")
	t.Setenv("CICD_LOG_LEVEL", "debug")
	t.Setenv("GITHUB_TOKEN", "test-token")

	cfg, err := LoadWithOverrides(configPath)
	if err != nil {
		t.Fatalf("LoadWithOverrides() error = %v", err)
	}

	if cfg.Claude.Model != "opus" {
		t.Errorf("Model = %s, want opus", cfg.Claude.Model)
	}
	if cfg.Claude.MaxBudgetUSD != 15.0 {
		t.Errorf("MaxBudgetUSD = %f, want 15.0", cfg.Claude.MaxBudgetUSD)
	}
	if cfg.Global.LogLevel != "debug" {
		t.Errorf("LogLevel = %s, want debug", cfg.Global.LogLevel)
	}
	if cfg.Platform.GitHub.Token != "test-token" {
		t.Errorf("GitHub.Token = %s, want test-token", cfg.Platform.GitHub.Token)
	}
}
