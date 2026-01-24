// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// EnvPrefix is the prefix for all environment variables.
	EnvPrefix = "CICD_TOOLKIT"
	// ProjectConfigFile is the project-level config file name.
	ProjectConfigFile = ".cicd-ai-toolkit.yaml"
	// GlobalConfigDir is the global config directory name.
	GlobalConfigDir = ".cicd-ai-toolkit"
	// GlobalConfigFile is the global config file name.
	GlobalConfigFile = "config.yaml"
)

// Loader loads configuration from files and environment.
type Loader struct {
	projectRoot string
	skipGlobal  bool
}

// NewLoader creates a new config loader.
func NewLoader() *Loader {
	return &Loader{}
}

// WithProjectRoot sets the project root directory.
func (l *Loader) WithProjectRoot(root string) *Loader {
	l.projectRoot = root
	return l
}

// SkipGlobal skips loading global config.
func (l *Loader) SkipGlobal() *Loader {
	l.skipGlobal = true
	return l
}

// Load loads configuration with full precedence order:
// 1. Defaults
// 2. Global Config ($HOME/.cicd-ai-toolkit/config.yaml)
// 3. Project Config (./.cicd-ai-toolkit.yaml)
// 4. Environment Variables (CICD_TOOLKIT_*)
func (l *Loader) Load() (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Load global config if not skipped
	if !l.skipGlobal {
		globalCfg, err := l.loadGlobalConfig()
		if err == nil {
			mergeConfig(cfg, globalCfg)
		}
		// Ignore errors for global config (it's optional)
	}

	// Load project config
	projectCfg, err := l.loadProjectConfig()
	if err == nil {
		mergeConfig(cfg, projectCfg)
	}
	// Ignore errors for project config (it's optional)

	// Apply environment overrides
	if err := l.applyEnvOverrides(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromPath loads configuration from a specific path.
func (l *Loader) LoadFromPath(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, &ConfigError{Path: path, Err: err}
	}

	return cfg, nil
}

// loadGlobalConfig loads global config from $HOME/.cicd-ai-toolkit/config.yaml.
func (l *Loader) loadGlobalConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	globalPath := filepath.Join(homeDir, GlobalConfigDir, GlobalConfigFile)
	return l.LoadFromPath(globalPath)
}

// loadProjectConfig loads project config from ./.cicd-ai-toolkit.yaml.
func (l *Loader) loadProjectConfig() (*Config, error) {
	root := l.projectRoot
	if root == "" {
		root = "."
	}

	projectPath := filepath.Join(root, ProjectConfigFile)
	return l.LoadFromPath(projectPath)
}

// applyEnvOverrides applies environment variable overrides.
// Format: CICD_TOOLKIT__SECTION__KEY=value
func (l *Loader) applyEnvOverrides(cfg *Config) error {
	// Claude settings
	if v := os.Getenv("CICD_TOOLKIT_CLAUDE__MODEL"); v != "" {
		cfg.Claude.Model = v
	}
	if v := os.Getenv("CICD_TOOLKIT_CLAUDE__TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return &ConfigError{
				Field: "claude.timeout",
				Err:   err,
			}
		}
		cfg.Claude.Timeout = d
	}

	// Global settings
	if v := os.Getenv("CICD_TOOLKIT_GLOBAL__LOG_LEVEL"); v != "" {
		cfg.Global.LogLevel = v
	}
	if v := os.Getenv("CICD_TOOLKIT_GLOBAL__CACHE_PATH"); v != "" {
		cfg.Global.CachePath = v
	}

	// Platform tokens
	if v := os.Getenv("CICD_TOOLKIT_PLATFORM__GITHUB__TOKEN"); v != "" {
		if cfg.Platform.GitHub == nil {
			cfg.Platform.GitHub = &GitHubPlatformConfig{}
		}
		// Directly set the token (not recommended, but allowed for env var)
		cfg.Platform.GitHub.TokenEnv = "GITHUB_TOKEN"
		os.Setenv("GITHUB_TOKEN", v)
	}
	if v := os.Getenv("CICD_TOOLKIT_PLATFORM__GITLAB__TOKEN"); v != "" {
		if cfg.Platform.GitLab == nil {
			cfg.Platform.GitLab = &GitLabPlatformConfig{}
		}
		cfg.Platform.GitLab.TokenEnv = "GITLAB_TOKEN"
		os.Setenv("GITLAB_TOKEN", v)
	}
	if v := os.Getenv("CICD_TOOLKIT_PLATFORM__GITEE__TOKEN"); v != "" {
		if cfg.Platform.Gitee == nil {
			cfg.Platform.Gitee = &GiteePlatformConfig{}
		}
		cfg.Platform.Gitee.TokenEnv = "GITEE_TOKEN"
		os.Setenv("GITEE_TOKEN", v)
	}

	return nil
}

// mergeConfig merges src into dst (src overrides dst).
func mergeConfig(dst, src *Config) {
	if src.Claude.Model != "" {
		dst.Claude.Model = src.Claude.Model
	}
	if src.Claude.MaxBudget > 0 {
		dst.Claude.MaxBudget = src.Claude.MaxBudget
	}
	if src.Claude.Timeout > 0 {
		dst.Claude.Timeout = src.Claude.Timeout
	}
	if len(src.Claude.AllowedTools) > 0 {
		dst.Claude.AllowedTools = src.Claude.AllowedTools
	}

	if len(src.Skills) > 0 {
		dst.Skills = src.Skills
	}

	if src.Platform.GitHub != nil {
		dst.Platform.GitHub = src.Platform.GitHub
	}
	if src.Platform.GitLab != nil {
		dst.Platform.GitLab = src.Platform.GitLab
	}
	if src.Platform.Gitee != nil {
		dst.Platform.Gitee = src.Platform.Gitee
	}
	if src.Platform.Jenkins != nil {
		dst.Platform.Jenkins = src.Platform.Jenkins
	}

	if src.Global.LogLevel != "" {
		dst.Global.LogLevel = src.Global.LogLevel
	}
	if src.Global.CachePath != "" {
		dst.Global.CachePath = src.Global.CachePath
	}
	if src.Global.MaxCacheMB > 0 {
		dst.Global.MaxCacheMB = src.Global.MaxCacheMB
	}
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Path  string
	Field string
	Err   error
}

func (e *ConfigError) Error() string {
	if e.Path != "" {
		return "config error in " + e.Path + ": " + e.Err.Error()
	}
	if e.Field != "" {
		return "config error for " + e.Field + ": " + e.Err.Error()
	}
	return "config error: " + e.Err.Error()
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// DetectProjectRoot finds the project root by looking for the config file.
func DetectProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, ProjectConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return ".", nil
		}
		dir = parent
	}
}

// FindConfigPaths returns all config file paths in precedence order.
func FindConfigPaths() []string {
	paths := []string{}

	// Global config
	if homeDir, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(homeDir, GlobalConfigDir, GlobalConfigFile)
		if _, err := os.Stat(globalPath); err == nil {
			paths = append(paths, globalPath)
		}
	}

	// Project config
	if root, err := DetectProjectRoot(); err == nil {
		projectPath := filepath.Join(root, ProjectConfigFile)
		if _, err := os.Stat(projectPath); err == nil {
			paths = append(paths, projectPath)
		}
	}

	return paths
}

// GetEnvConfig returns all environment variables that start with CICD_TOOLKIT_.
func GetEnvConfig() map[string]string {
	result := make(map[string]string)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, EnvPrefix+"_") {
			kv := strings.SplitN(env, "=", 2)
			if len(kv) == 2 {
				result[kv[0]] = kv[1]
			}
		}
	}

	return result
}
