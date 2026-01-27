// Package runner provides the core CI/CD runner orchestration
package runner

import (
	"context"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/claude"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
)

// Runner orchestrates the CI/CD review process
type Runner interface {
	// Review runs code review on a pull/merge request
	Review(ctx context.Context, opts ReviewOptions) (*ReviewResult, error)

	// Analyze runs change analysis on a pull/merge request
	Analyze(ctx context.Context, opts AnalyzeOptions) (*AnalyzeResult, error)

	// GenerateTests generates tests based on code changes
	GenerateTests(ctx context.Context, opts TestGenOptions) (*TestGenResult, error)

	// Health checks the runner's health
	Health(ctx context.Context) error
}

// ReviewOptions contains options for code review
type ReviewOptions struct {
	PRID        int
	Diff        string
	BaseSHA     string
	HeadSHA     string
	Skills      []string
	Force       bool // Skip cache
}

// AnalyzeOptions contains options for change analysis
type AnalyzeOptions struct {
	PRID       int
	Diff       string
	FileCount  int
	Additions  int
	Deletions  int
	Skills     []string
}

// TestGenOptions contains options for test generation
type TestGenOptions struct {
	Diff          string
	TargetFiles   []string
	TestFramework string
	CreateFiles   bool // Actually write test files
}

// ReviewResult contains the result of a code review
type ReviewResult struct {
	// Summary contains aggregated statistics
	Summary ReviewSummary

	// Issues contains all found issues
	Issues []claude.Issue

	// PlatformComment is the formatted comment for PR
	PlatformComment string

	// Cached indicates if result was from cache
	Cached bool

	// Duration is how long the review took
	Duration time.Duration
}

// ReviewSummary contains review statistics
type ReviewSummary struct {
	FilesChanged int
	TotalIssues  int
	Critical     int
	High         int
	Medium       int
	Low          int
}

// AnalyzeResult contains the result of change analysis
type AnalyzeResult struct {
	Summary      ChangeSummary
	Impact       ImpactAnalysis
	Risk         RiskAssessment
	Changelog    ChangelogEntry
	Suggestions  []string
	Duration     time.Duration
}

// ChangeSummary describes the changes
type ChangeSummary struct {
	Title        string
	Description  string
	FilesChanged int
	LinesAdded   int
	LinesRemoved int
}

// ImpactAnalysis analyzes the change impact
type ImpactAnalysis struct {
	BreakingChanges    []string
	APIChanges         []string
	DatabaseMigrations bool
	ConfigChanges      []string
	AffectedModules    []string
}

// RiskAssessment provides risk scoring
type RiskAssessment struct {
	Score           int  // 1-10
	Factors         []string
	TestingLevel    string
	RollbackComplexity string
}

// ChangelogEntry represents a changelog entry
type ChangelogEntry struct {
	Added      []string
	Changed    []string
	Deprecated []string
	Removed    []string
	Fixed      []string
}

// TestGenResult contains the result of test generation
type TestGenResult struct {
	TestFiles  []GeneratedTest
	Summary    TestGenSummary
	Duration   time.Duration
}

// GeneratedTest represents a generated test file
type GeneratedTest struct {
	Path     string
	Language string
	Content  string
	Tests    int
}

// TestGenSummary contains test generation statistics
type TestGenSummary struct {
	FilesCreated  int
	TotalTests    int
	CoverageEst   string
}

// Builder builds context for Claude execution
type Builder interface {
	// BuildDiffContext builds the diff context for review
	BuildDiffContext(ctx context.Context, diff string) (string, error)

	// BuildLogContext builds context from log files
	BuildLogContext(ctx context.Context, logs []string) (string, error)

	// BuildFileContext builds context from specific files
	BuildFileContext(ctx context.Context, files []string) (string, error)

	// Chunk splits large diffs into manageable chunks
	Chunk(ctx context.Context, diff string, maxTokens int) ([]string, error)
}

// Reporter reports results back to platforms
type Reporter interface {
	// PostReview posts review results to PR
	PostReview(ctx context.Context, prID int, result *ReviewResult) error

	// PostAnalysis posts analysis results to PR
	PostAnalysis(ctx context.Context, prID int, result *AnalyzeResult) error

	// FormatReview formats review result as markdown/comment
	FormatReview(result *ReviewResult) string

	// FormatAnalysis formats analysis result as markdown/comment
	FormatAnalysis(result *AnalyzeResult) string
}

// Config represents the runner configuration
type Config struct {
	Version   string
	Claude    config.ClaudeConfig
	Skills    []config.SkillConfig
	Platform  config.PlatformConfig
	Global    config.GlobalConfig
	Advanced  config.AdvancedConfig
}
