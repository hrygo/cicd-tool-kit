// Package config handles configuration loading and validation
package config

import (
	"time"
)

// Config represents the complete configuration
type Config struct {
	Version   string         `yaml:"version"`
	AIBackend string         `yaml:"ai_backend"` // "claude" or "crush"
	Claude    ClaudeConfig   `yaml:"claude"`
	Crush     CrushConfig    `yaml:"crush"`
	Skills    []SkillConfig  `yaml:"skills"`
	Platform  PlatformConfig `yaml:"platform"`
	Global    GlobalConfig   `yaml:"global"`
	Advanced  AdvancedConfig `yaml:"advanced,omitempty"`
}

// ClaudeConfig contains Claude-specific settings
// Based on best practices from docs/BEST_PRACTICE_CLI_AGENT.md
type ClaudeConfig struct {
	Model           string   `yaml:"model"` // sonnet, opus, haiku
	MaxBudgetUSD    float64  `yaml:"max_budget_usd"`
	MaxTurns        int      `yaml:"max_turns"`
	Timeout         string   `yaml:"timeout"`       // Go duration format
	OutputFormat    string   `yaml:"output_format"` // json, stream-json, text (stream-json recommended)
	SkipPermissions bool     `yaml:"dangerous_skip_permissions"`
	AllowedTools    []string `yaml:"allowed_tools,omitempty"`

	// Session Management - Based on docs/BEST_PRACTICE_CLI_AGENT.md section 7.2
	SessionDir    string `yaml:"session_dir,omitempty"`     // Directory for session data
	SessionTTL    string `yaml:"session_ttl,omitempty"`     // Session time-to-live (default: 24h)
	MaxRetries    int    `yaml:"max_retries,omitempty"`     // Max retry attempts (default: 3)
	UseExplicitID bool   `yaml:"use_explicit_id,omitempty"` // Use explicit session ID strategy (recommended)
}

// CrushConfig contains Crush-specific settings
type CrushConfig struct {
	Provider     string `yaml:"provider"`      // anthropic, openai, ollama, etc.
	Model        string `yaml:"model"`         // e.g., claude-sonnet-4-20250514
	BaseURL      string `yaml:"base_url"`      // For custom endpoints (e.g., Ollama)
	Timeout      string `yaml:"timeout"`       // Go duration format
	OutputFormat string `yaml:"output_format"` // json, text
}

// SkillConfig defines a skill configuration
type SkillConfig struct {
	Name     string         `yaml:"name"`
	Path     string         `yaml:"path"`
	Enabled  bool           `yaml:"enabled"`
	Priority int            `yaml:"priority,omitempty"`
	Config   map[string]any `yaml:"config,omitempty"`
}

// PlatformConfig contains platform-specific settings
type PlatformConfig struct {
	GitHub GitHubConfig `yaml:"github"`
	Gitee  GiteeConfig  `yaml:"gitee"`
	GitLab GitLabConfig `yaml:"gitlab"`
}

// GitHubConfig contains GitHub-specific settings
type GitHubConfig struct {
	Token            string `yaml:"token,omitempty"` // GitHub token (usually from env)
	TokenFromEnv     bool   `yaml:"-"`               // Internal: whether token was loaded from env
	PostComment      bool   `yaml:"post_comment"`
	FailOnError      bool   `yaml:"fail_on_error"`
	MaxCommentLength int    `yaml:"max_comment_length"`
	PostAsReview     bool   `yaml:"post_as_review"`
	APIURL           string `yaml:"api_url,omitempty"` // For GitHub Enterprise
}

// GiteeConfig contains Gitee-specific settings
type GiteeConfig struct {
	Token        string `yaml:"token,omitempty"` // Gitee token (usually from env)
	APIURL       string `yaml:"api_url"`
	PostComment  bool   `yaml:"post_comment"`
	EnterpriseID string `yaml:"enterprise_id,omitempty"`
}

// GitLabConfig contains GitLab-specific settings
type GitLabConfig struct {
	Token                  string `yaml:"token,omitempty"` // GitLab token (usually from env)
	PostComment            bool   `yaml:"post_comment"`
	FailOnError            bool   `yaml:"fail_on_error"`
	MergeRequestDiscussion bool   `yaml:"merge_request_discussion"`
	APIURL                 string `yaml:"api_url,omitempty"` // For GitLab self-hosted
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	LogLevel       string            `yaml:"log_level"` // debug, info, warn, error
	CacheDir       string            `yaml:"cache_dir"`
	EnableCache    bool              `yaml:"enable_cache"`
	ParallelSkills int               `yaml:"parallel_skills"`
	DiffContext    int               `yaml:"diff_context"`
	Exclude        []string          `yaml:"exclude"`
	Env            map[string]string `yaml:"env,omitempty"`
}

// AdvancedConfig contains advanced/experimental settings
type AdvancedConfig struct {
	MCPServers []MCPServer      `yaml:"mcp_servers,omitempty"`
	Memory     MemoryConfig     `yaml:"memory,omitempty"`
	Reflective ReflectiveConfig `yaml:"reflective,omitempty"`
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
	TTL     string `yaml:"ttl"`     // Go duration format
}

// ReflectiveConfig configures the reflective runtime (VIGIL pattern)
type ReflectiveConfig struct {
	Enabled          bool `yaml:"enabled"`
	ObserverEnabled  bool `yaml:"observer_enabled"`
	CorrectorEnabled bool `yaml:"corrector_enabled"`
	MaxCorrections   int  `yaml:"max_corrections"`
}

// GetTimeout returns the timeout as a time.Duration
func (c *ClaudeConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}

// GetSessionTTL returns the session TTL as a time.Duration
// Default: 24 hours
func (c *ClaudeConfig) GetSessionTTL() time.Duration {
	if c.SessionTTL == "" {
		return 24 * time.Hour
	}
	if ttl, err := time.ParseDuration(c.SessionTTL); err == nil {
		return ttl
	}
	return 24 * time.Hour
}

// GetMaxRetries returns the maximum retry attempts
// Default: 3
func (c *ClaudeConfig) GetMaxRetries() int {
	if c.MaxRetries > 0 {
		return c.MaxRetries
	}
	return 3
}

// GetTimeout returns the timeout as a time.Duration for Crush
func (c *CrushConfig) GetTimeout() (time.Duration, error) {
	if c.Timeout == "" {
		return 5 * time.Minute, nil // Default timeout
	}
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
func (c *Config) GetSkillConfig(skillName string) (map[string]any, bool) {
	for _, s := range c.Skills {
		if s.Name == skillName {
			return s.Config, len(s.Config) > 0
		}
	}
	return nil, false
}
