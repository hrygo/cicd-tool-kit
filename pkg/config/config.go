// Package config handles configuration loading and validation
package config

import (
	"time"
)

// Config represents the complete configuration
type Config struct {
	Version  string              `yaml:"version"`
	Claude   ClaudeConfig        `yaml:"claude"`
	Skills   []SkillConfig       `yaml:"skills"`
	Platform PlatformConfig      `yaml:"platform"`
	Global   GlobalConfig        `yaml:"global"`
	Advanced AdvancedConfig      `yaml:"advanced,omitempty"`
}

// ClaudeConfig contains Claude-specific settings
type ClaudeConfig struct {
	Model             string  `yaml:"model"`                        // sonnet, opus, haiku
	MaxBudgetUSD      float64 `yaml:"max_budget_usd"`
	MaxTurns          int     `yaml:"max_turns"`
	Timeout           string  `yaml:"timeout"`                      // Go duration format
	OutputFormat      string  `yaml:"output_format"`                // json, stream-json, text
	SkipPermissions   bool    `yaml:"dangerous_skip_permissions"`
	AllowedTools      []string `yaml:"allowed_tools,omitempty"`
}

// SkillConfig defines a skill configuration
type SkillConfig struct {
	Name   string                 `yaml:"name"`
	Path   string                 `yaml:"path"`
	Enabled bool                  `yaml:"enabled"`
	Priority int                  `yaml:"priority,omitempty"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// PlatformConfig contains platform-specific settings
type PlatformConfig struct {
	GitHub GitHubConfig `yaml:"github"`
	Gitee  GiteeConfig  `yaml:"gitee"`
	GitLab GitLabConfig `yaml:"gitlab"`
}

// GitHubConfig contains GitHub-specific settings
type GitHubConfig struct {
	Token            string `yaml:"token,omitempty"`           // GitHub token (usually from env)
	PostComment      bool   `yaml:"post_comment"`
	FailOnError      bool   `yaml:"fail_on_error"`
	MaxCommentLength int    `yaml:"max_comment_length"`
	PostAsReview     bool   `yaml:"post_as_review"`
	APIURL           string `yaml:"api_url,omitempty"` // For GitHub Enterprise
}

// GiteeConfig contains Gitee-specific settings
type GiteeConfig struct {
	APIURL       string `yaml:"api_url"`
	PostComment  bool   `yaml:"post_comment"`
	EnterpriseID string `yaml:"enterprise_id,omitempty"`
}

// GitLabConfig contains GitLab-specific settings
type GitLabConfig struct {
	PostComment           bool   `yaml:"post_comment"`
	FailOnError           bool   `yaml:"fail_on_error"`
	MergeRequestDiscussion bool  `yaml:"merge_request_discussion"`
	APIURL                string `yaml:"api_url,omitempty"` // For GitLab self-hosted
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	LogLevel      string            `yaml:"log_level"`       // debug, info, warn, error
	CacheDir      string            `yaml:"cache_dir"`
	EnableCache   bool              `yaml:"enable_cache"`
	ParallelSkills int              `yaml:"parallel_skills"`
	DiffContext   int               `yaml:"diff_context"`
	Exclude       []string          `yaml:"exclude"`
	Env           map[string]string `yaml:"env,omitempty"`
}

// AdvancedConfig contains advanced/experimental settings
type AdvancedConfig struct {
	MCPServers    []MCPServer `yaml:"mcp_servers,omitempty"`
	Memory        MemoryConfig `yaml:"memory,omitempty"`
	Reflective    ReflectiveConfig `yaml:"reflective,omitempty"`
}

// MCPServer defines an MCP server connection
type MCPServer struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	Env     []string `yaml:"env,omitempty"`
}

// MemoryConfig configures the memory system
type MemoryConfig struct {
	Enabled bool   `yaml:"enabled"`
	Backend string `yaml:"backend"` // file, redis, postgres
	TTL     string `yaml:"ttl"`      // Go duration format
}

// ReflectiveConfig configures the reflective runtime (VIGIL pattern)
type ReflectiveConfig struct {
	Enabled         bool `yaml:"enabled"`
	ObserverEnabled bool `yaml:"observer_enabled"`
	CorrectorEnabled bool `yaml:"corrector_enabled"`
	MaxCorrections   int  `yaml:"max_corrections"`
}



// GetTimeout returns the timeout as a time.Duration
func (c *ClaudeConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}

// IsEnabled returns true if a skill is enabled
func (c *Config) IsEnabled(skillName string) bool {
	for _, s := range c.Skills {
		if s.Name == skillName {
			return s.Enabled
		}
	}
	return false
}

// GetEnabledSkills returns names of all enabled skills
func (c *Config) GetEnabledSkills() []string {
	var enabled []string
	for _, s := range c.Skills {
		if s.Enabled {
			enabled = append(enabled, s.Name)
		}
	}
	return enabled
}

// GetSkillConfig returns configuration for a specific skill
func (c *Config) GetSkillConfig(skillName string) (map[string]interface{}, bool) {
	for _, s := range c.Skills {
		if s.Name == skillName {
			return s.Config, len(s.Config) > 0
		}
	}
	return nil, false
}
