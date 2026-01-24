// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package config

import (
	"os"
	"path/filepath"
	"time"
)

// DefaultConfig returns the default configuration.
// These values are used when no config file is present.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cicd-ai-toolkit", "cache")

	return &Config{
		Claude: DefaultClaudeConfig(),
		Skills: DefaultSkills(),
		Platform: PlatformConfig{
			GitHub: DefaultGitHubPlatform(),
		},
		Global: DefaultGlobalConfig(cacheDir),
	}
}

// DefaultClaudeConfig returns default Claude configuration.
func DefaultClaudeConfig() ClaudeConfig {
	return ClaudeConfig{
		Model:        "sonnet",
		MaxBudget:    10.0,                     // $10 USD default budget
		Timeout:      300 * time.Second,        // 5 minutes
		AllowedTools: []string{"read", "grep"}, // Basic tools
	}
}

// DefaultSkills returns default skills configuration.
func DefaultSkills() []SkillConfig {
	return []SkillConfig{
		{
			Name:     "code-reviewer",
			Enabled:  true,
			Priority: 1,
		},
	}
}

// DefaultGitHubPlatform returns default GitHub platform config.
func DefaultGitHubPlatform() *GitHubPlatformConfig {
	return &GitHubPlatformConfig{
		TokenEnv: "GITHUB_TOKEN",
	}
}

// DefaultGitLabPlatform returns default GitLab platform config.
func DefaultGitLabPlatform() *GitLabPlatformConfig {
	return &GitLabPlatformConfig{
		TokenEnv: "GITLAB_TOKEN",
		URL:      "https://gitlab.com",
	}
}

// DefaultGiteePlatform returns default Gitee platform config.
func DefaultGiteePlatform() *GiteePlatformConfig {
	return &GiteePlatformConfig{
		TokenEnv: "GITEE_TOKEN",
	}
}

// DefaultJenkinsPlatform returns default Jenkins platform config.
func DefaultJenkinsPlatform() *JenkinsPlatformConfig {
	return &JenkinsPlatformConfig{
		TokenEnv: "JENKINS_TOKEN",
	}
}

// DefaultGlobalConfig returns default global configuration.
func DefaultGlobalConfig(cachePath string) GlobalConfig {
	return GlobalConfig{
		LogLevel:   "info",
		CachePath:  cachePath,
		MaxCacheMB: 100, // 100MB default cache
	}
}

// GetDefaultCachePath returns the default cache directory path.
func GetDefaultCachePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cicd-ai-toolkit", "cache")
}

// GetDefaultConfigPath returns the default global config file path.
func GetDefaultConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cicd-ai-toolkit", "config.yaml")
}

// GetProjectConfigPath returns the project config file path.
func GetProjectConfigPath(projectRoot string) string {
	if projectRoot == "" {
		projectRoot = "."
	}
	return filepath.Join(projectRoot, ".cicd-ai-toolkit.yaml")
}
