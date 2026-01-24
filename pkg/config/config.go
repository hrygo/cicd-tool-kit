// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

// Package config provides configuration management for CICD AI Toolkit.
//
// Configuration Loading Order (later overrides earlier):
// 1. Defaults (hardcoded)
// 2. Global Config: $HOME/.cicd-ai-toolkit/config.yaml
// 3. Project Config: ./.cicd-ai-toolkit.yaml
// 4. Environment Variables: CICD_TOOLKIT_*
package config

import (
	"time"
)

// Config represents the complete application configuration.
type Config struct {
	Claude   ClaudeConfig   `yaml:"claude"`
	Skills   []SkillConfig  `yaml:"skills"`
	Platform PlatformConfig `yaml:"platform"`
	Global   GlobalConfig   `yaml:"global"`
}

// ClaudeConfig contains Claude AI settings.
type ClaudeConfig struct {
	Model        string        `yaml:"model"`
	MaxBudget    float64       `yaml:"max_budget_usd"`
	Timeout      time.Duration `yaml:"timeout"`
	AllowedTools []string      `yaml:"allowed_tools"`
}

// SkillConfig represents a single skill configuration.
type SkillConfig struct {
	Name     string         `yaml:"name"`
	Enabled  bool           `yaml:"enabled"`
	Options  map[string]any `yaml:"options,omitempty"`
	Priority int            `yaml:"priority,omitempty"`
}

// PlatformConfig contains platform-specific settings.
type PlatformConfig struct {
	GitHub  *GitHubPlatformConfig  `yaml:"github,omitempty"`
	GitLab  *GitLabPlatformConfig  `yaml:"gitlab,omitempty"`
	Gitee   *GiteePlatformConfig   `yaml:"gitee,omitempty"`
	Jenkins *JenkinsPlatformConfig `yaml:"jenkins,omitempty"`
}

// GitHubPlatformConfig contains GitHub-specific settings.
type GitHubPlatformConfig struct {
	TokenEnv string `yaml:"token_env"` // e.g., "GITHUB_TOKEN"
	// token field is NOT allowed - must use token_env
}

// GitLabPlatformConfig contains GitLab-specific settings.
type GitLabPlatformConfig struct {
	TokenEnv string `yaml:"token_env"` // e.g., "GITLAB_TOKEN"
	URL      string `yaml:"url"`       // Custom GitLab URL
}

// GiteePlatformConfig contains Gitee-specific settings.
type GiteePlatformConfig struct {
	TokenEnv string `yaml:"token_env"` // e.g., "GITEE_TOKEN"`
}

// JenkinsPlatformConfig contains Jenkins-specific settings.
type JenkinsPlatformConfig struct {
	TokenEnv string `yaml:"token_env"` // e.g., "JENKINS_TOKEN"`
	URL      string `yaml:"url"`       // Jenkins URL
}

// GlobalConfig contains global application settings.
type GlobalConfig struct {
	LogLevel   string `yaml:"log_level"`    // debug, info, warn, error
	CachePath  string `yaml:"cache_path"`   // Path to cache directory
	MaxCacheMB int    `yaml:"max_cache_mb"` // Max cache size in MB
}

// EnvConfig represents environment variable based configuration.
// These take highest priority and override file-based config.
type EnvConfig struct {
	// CICD_TOOLKIT_CLAUDE__MODEL
	ClaudeModel string
	// CICD_TOOLKIT_CLAUDE__TIMEOUT
	ClaudeTimeout time.Duration
	// CICD_TOOLKIT_GLOBAL__LOG_LEVEL
	LogLevel string
	// CICD_TOOLKIT_PLATFORM__GITHUB__TOKEN
	GitHubToken string
	// CICD_TOOLKIT_PLATFORM__GITLAB__TOKEN
	GitLabToken string
	// CICD_TOOLKIT_PLATFORM__GITEE__TOKEN
	GiteeToken string
	// CICD_TOOLKIT_PLATFORM__JENKINS__TOKEN
	JenkinsToken string
}
