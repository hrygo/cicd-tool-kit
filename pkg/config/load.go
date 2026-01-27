// Package config handles configuration loading and validation
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Default config file names to search for
var defaultConfigFiles = []string{
	".cicd-ai-toolkit.yaml",
	".cicd-ai-toolkit.yml",
	"cicd-ai-toolkit.yaml",
	"cicd-ai-toolkit.yml",
}

// Load loads configuration from a specific file path
func Load(path string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.ConfigError(fmt.Sprintf("failed to read config file: %s", path), err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.ConfigError(fmt.Sprintf("failed to parse config file: %s", path), err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, errors.ConfigError("config validation failed", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

// LoadDefault searches for and loads configuration from default locations
// Search order:
// 1. Current directory
// 2. Parent directories (up to root)
// 3. User home directory (.config/cicd-ai-toolkit/)
func LoadDefault() (*Config, error) {
	// Check current directory and parents
	if cfg, err := findInParents("."); err == nil {
		return cfg, nil
	}

	// Check user config directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".config", "cicd-ai-toolkit", "config.yaml")
		if cfg, err := Load(userConfigPath); err == nil {
			return cfg, nil
		}
	}

	// No config found - return minimal default config
	return defaultConfig(), nil
}

// LoadFromEnv loads config from environment variable path
// CICD_AI_TOOLKIT_CONFIG can override the config file path
func LoadFromEnv() (*Config, error) {
	if path := os.Getenv("CICD_AI_TOOLKIT_CONFIG"); path != "" {
		return Load(path)
	}
	return LoadDefault()
}

// findInParents searches for config file in current directory and parent directories
func findInParents(startDir string) (*Config, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	for {
		for _, filename := range defaultConfigFiles {
			configPath := filepath.Join(dir, filename)
			if _, err := os.Stat(configPath); err == nil {
				return Load(configPath)
			}
		}

		// Move to parent directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached root
			break
		}
		dir = parentDir
	}

	return nil, errors.ConfigError("no config file found", nil)
}

// defaultConfig returns a minimal default configuration
func defaultConfig() *Config {
	return &Config{
		Version: "2.0",
		Claude: ClaudeConfig{
			Model:           "sonnet",
			MaxBudgetUSD:    10.0,
			MaxTurns:        50,
			Timeout:         "30m",
			OutputFormat:    "json",
			SkipPermissions: true,
		},
		Skills: []SkillConfig{
			{
				Name:   "code-reviewer",
				Path:   "./skills/code-reviewer",
				Enabled: true,
				Priority: 100,
			},
		},
		Platform: PlatformConfig{
			GitHub: GitHubConfig{
				PostComment:      true,
				FailOnError:      false,
				MaxCommentLength: 65536,
			},
		},
		Global: GlobalConfig{
			LogLevel:      "info",
			CacheDir:      ".cicd-cache",
			EnableCache:   true,
			ParallelSkills: 1,
			DiffContext:   3,
		},
	}
}

// applyDefaults sets default values for optional fields
func applyDefaults(cfg *Config) {
	// Set default version if not specified
	if cfg.Version == "" {
		cfg.Version = "2.0"
	}

	// Claude defaults
	if cfg.Claude.Model == "" {
		cfg.Claude.Model = "sonnet"
	}
	if cfg.Claude.MaxBudgetUSD == 0 {
		cfg.Claude.MaxBudgetUSD = 10.0
	}
	if cfg.Claude.MaxTurns == 0 {
		cfg.Claude.MaxTurns = 50
	}
	if cfg.Claude.Timeout == "" {
		cfg.Claude.Timeout = "30m"
	}
	if cfg.Claude.OutputFormat == "" {
		cfg.Claude.OutputFormat = "json"
	}

	// GitHub defaults
	if cfg.Platform.GitHub.MaxCommentLength == 0 {
		cfg.Platform.GitHub.MaxCommentLength = 65536
	}

	// Global defaults
	if cfg.Global.LogLevel == "" {
		cfg.Global.LogLevel = "info"
	}
	if cfg.Global.CacheDir == "" {
		cfg.Global.CacheDir = ".cicd-cache"
	}
	if cfg.Global.ParallelSkills == 0 {
		cfg.Global.ParallelSkills = 1
	}
	if cfg.Global.DiffContext == 0 {
		cfg.Global.DiffContext = 3
	}
}

// LoadWithOverrides loads config and applies environment variable overrides
func LoadWithOverrides(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	// Apply environment overrides
	if val := os.Getenv("CICD_MODEL"); val != "" {
		cfg.Claude.Model = val
	}
	if val := os.Getenv("CICD_MAX_BUDGET"); val != "" {
		var budget float64
		if _, err := fmt.Sscanf(val, "%f", &budget); err == nil {
			cfg.Claude.MaxBudgetUSD = budget
		}
	}
	if val := os.Getenv("CICD_TIMEOUT"); val != "" {
		cfg.Claude.Timeout = val
	}
	if val := os.Getenv("CICD_LOG_LEVEL"); val != "" {
		cfg.Global.LogLevel = val
	}
	if val := os.Getenv("CICD_CACHE_DIR"); val != "" {
		cfg.Global.CacheDir = val
	}
	if val := os.Getenv("GITHUB_TOKEN"); val != "" && cfg.Platform.GitHub.Token == "" {
		cfg.Platform.GitHub.Token = val
	}

	return cfg, nil
}
