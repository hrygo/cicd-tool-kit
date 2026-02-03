// Package runner provides the core CI/CD runner implementation
package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/ai"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/buildcontext"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/platform"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/skill"
)

const (
	// DefaultTimeout is the default timeout for Claude operations
	DefaultTimeout = 5 * time.Minute
)

// DefaultRunner implements the Runner interface
type DefaultRunner struct {
	cfg         *config.Config
	platform    platform.Platform
	builder     *buildcontext.Builder
	aiBrain     ai.Brain
	cache       *Cache
	skillLoader *skill.Loader
}

// NewRunner creates a new runner instance
func NewRunner(cfg *config.Config, platform platform.Platform, baseDir string) (*DefaultRunner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if platform == nil {
		return nil, fmt.Errorf("platform cannot be nil")
	}
	if baseDir == "" {
		return nil, fmt.Errorf("baseDir cannot be empty")
	}

	builder := buildcontext.NewBuilder(
		baseDir,
		cfg.Global.DiffContext,
		cfg.Global.Exclude,
	)

	cacheDir := baseDir + "/" + cfg.Global.CacheDir
	cache, err := NewCache(cacheDir, cfg.Global.EnableCache)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize skill loader from skills directory
	skillsDir := baseDir + "/skills"
	skillLoader := skill.NewLoader(skillsDir)

	// Create AI Brain using factory
	factory := ai.NewFactory(baseDir)
	aiBrain, err := factory.CreateFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI brain: %w", err)
	}

	return &DefaultRunner{
		cfg:         cfg,
		platform:    platform,
		builder:     builder,
		aiBrain:     aiBrain,
		cache:       cache,
		skillLoader: skillLoader,
	}, nil
}

// Review runs code review on a pull/merge request
func (r *DefaultRunner) Review(ctx context.Context, opts ReviewOptions) (*ReviewResult, error) {
	start := time.Now()
	result := &ReviewResult{}

	// Check cache first
	if !opts.Force {
		if cached, ok := r.cache.GetReview(opts.PRID); ok {
			result.Cached = true
			result.Summary = cached.Summary
			result.Issues = cached.Issues
			result.PlatformComment = cached.Comment
			result.Duration = cached.Duration
			return result, nil
		}
	}

	// Get enabled review skills
	skills := r.getReviewSkills(opts.Skills)

	// Build context and execute
	diffContext := r.buildDiffContext(opts.Diff, opts.PRID)
	issues, err := r.executeWithSkill(ctx, diffContext, skills, "review")
	if err != nil {
		return nil, fmt.Errorf("review execution failed: %w", err)
	}

	result.Issues = issues
	result.Summary = r.summarizeIssues(issues)
	result.PlatformComment = r.formatReviewComment(result)
	result.Duration = time.Since(start)

	// Cache the result
	r.cache.SetReview(opts.PRID, CachedReview{
		Summary:  result.Summary,
		Issues:   result.Issues,
		Comment:  result.PlatformComment,
		Duration: result.Duration,
	})

	return result, nil
}

// Analyze runs change analysis on a pull/merge request
func (r *DefaultRunner) Analyze(ctx context.Context, opts AnalyzeOptions) (*AnalyzeResult, error) {
	start := time.Now()

	skills := r.getAnalysisSkills(opts.Skills)

	// Build context with summary stats
	context := fmt.Sprintf("# Change Analysis\n\nFiles: %d, Additions: +%d, Deletions: -%d\n\n%s",
		opts.FileCount, opts.Additions, opts.Deletions, opts.Diff)

	// Execute with skill - returns structured analysis
	_, err := r.executeWithSkill(ctx, context, skills, "analyze")
	if err != nil {
		return nil, fmt.Errorf("analysis execution failed: %w", err)
	}

	// Return placeholder - full parsing would be done by Claude Code via MCP
	result := &AnalyzeResult{
		Summary: ChangeSummary{
			FilesChanged: opts.FileCount,
			LinesAdded:   opts.Additions,
			LinesRemoved: opts.Deletions,
		},
		Risk: RiskAssessment{
			Score: 5, // Default mid-range score
		},
		Duration: time.Since(start),
	}

	return result, nil
}

// GenerateTests generates tests based on code changes
func (r *DefaultRunner) GenerateTests(ctx context.Context, opts TestGenOptions) (*TestGenResult, error) {
	start := time.Now()

	skills := []string{"test-generator"}

	// Build context
	context := fmt.Sprintf("# Test Generation\n\n%s\n\nFramework: %s",
		opts.Diff, opts.TestFramework)

	// Execute with skill
	output, err := r.executeRawWithSkill(ctx, context, skills, "test-gen")
	if err != nil {
		return nil, fmt.Errorf("test generation failed: %w", err)
	}

	// In Claude-First architecture, Claude Code writes files directly via Edit tool
	// This is a simplified version that returns metadata
	result := &TestGenResult{
		TestFiles: []GeneratedTest{
			{
				Path:     "generated_tests",
				Language: detectTestLanguage(opts.TargetFiles),
				Content:  output,
				Tests:    estimateTestCount(output),
			},
		},
		Summary: TestGenSummary{
			FilesCreated: 1,
			TotalTests:   estimateTestCount(output),
			CoverageEst:  "~70-80%",
		},
		Duration: time.Since(start),
	}

	return result, nil
}

// Health checks the runner's health
func (r *DefaultRunner) Health(ctx context.Context) error {
	// Check platform
	if err := r.platform.Health(ctx); err != nil {
		return fmt.Errorf("platform unhealthy: %w", err)
	}

	// Check if we're in a git repo
	if !r.builder.IsGitRepo() {
		return fmt.Errorf("not in a git repository")
	}

	return nil
}

// buildDiffContext builds diff context for review
func (r *DefaultRunner) buildDiffContext(diff string, prID int) string {
	context := "# Code Review\n\n"
	if prID > 0 {
		context += fmt.Sprintf("PR #%d\n\n", prID)
	}
	context += "```diff\n" + diff + "\n```\n"
	return context
}

// executeWithSkill executes AI with skill and returns issues
func (r *DefaultRunner) executeWithSkill(ctx context.Context, context string, skills []string, operation string) ([]ai.Issue, error) {
	opts := ai.ExecuteOptions{
		OutputFormat: r.cfg.Claude.OutputFormat,
		Timeout:      DefaultTimeout,
		Skills:       skills,
	}

	// Get timeout from config
	if t, err := r.cfg.Claude.GetTimeout(); err == nil && t > 0 {
		opts.Timeout = t
	}

	// Validate prompt
	if err := ai.ValidatePrompt(context, opts); err != nil {
		return nil, fmt.Errorf("prompt validation failed: %w", err)
	}

	// Execute
	output, err := r.aiBrain.Execute(ctx, context, opts)
	if err != nil {
		return nil, err
	}

	return output.Issues, nil
}

// executeRawWithSkill executes AI and returns raw output
func (r *DefaultRunner) executeRawWithSkill(ctx context.Context, context string, skills []string, operation string) (string, error) {
	opts := ai.ExecuteOptions{
		OutputFormat: r.cfg.Claude.OutputFormat,
		Timeout:      DefaultTimeout,
		Skills:       skills,
	}

	if t, err := r.cfg.Claude.GetTimeout(); err == nil && t > 0 {
		opts.Timeout = t
	}

	if err := ai.ValidatePrompt(context, opts); err != nil {
		return "", fmt.Errorf("prompt validation failed: %w", err)
	}

	output, err := r.aiBrain.Execute(ctx, context, opts)
	if err != nil {
		return "", err
	}

	return output.Raw, nil
}

// summarizeIssues aggregates issues into a summary
func (r *DefaultRunner) summarizeIssues(issues []ai.Issue) ReviewSummary {
	summary := ReviewSummary{}

	files := make(map[string]bool)
	for _, issue := range issues {
		if issue.File != "" {
			files[issue.File] = true
		}

		switch issue.Severity {
		case "critical":
			summary.Critical++
		case "high":
			summary.High++
		case "medium":
			summary.Medium++
		case "low":
			summary.Low++
		}
	}

	summary.FilesChanged = len(files)
	summary.TotalIssues = len(issues)

	return summary
}

// formatReviewComment formats the review result as a markdown comment
func (r *DefaultRunner) formatReviewComment(result *ReviewResult) string {
	comment := "## 🔍 Code Review Results\n\n### Summary\n\n"
	comment += fmt.Sprintf("- **Files Changed**: %d\n", result.Summary.FilesChanged)
	comment += fmt.Sprintf("- **Total Issues**: %d\n", result.Summary.TotalIssues)

	if result.Summary.Critical > 0 {
		comment += fmt.Sprintf("- **🔴 Critical**: %d\n", result.Summary.Critical)
	}
	if result.Summary.High > 0 {
		comment += fmt.Sprintf("- **🟠 High**: %d\n", result.Summary.High)
	}
	if result.Summary.Medium > 0 {
		comment += fmt.Sprintf("- **🟡 Medium**: %d\n", result.Summary.Medium)
	}
	if result.Summary.Low > 0 {
		comment += fmt.Sprintf("- **🟢 Low**: %d\n", result.Summary.Low)
	}

	comment += "\n"

	if len(result.Issues) > 0 {
		comment += "### Issues Found\n\n"
		for _, issue := range result.Issues {
			icon := severityIcon(issue.Severity)
			comment += fmt.Sprintf("%s **%s** - `%s:%d`\n", icon, issue.Category, issue.File, issue.Line)
			comment += fmt.Sprintf("%s\n\n", issue.Message)
			if issue.Suggestion != "" {
				comment += fmt.Sprintf("**Suggestion**: %s\n\n", issue.Suggestion)
			}
		}
	} else {
		comment += "### ✅ No Issues Found\n\nGreat job! No issues were detected.\n\n"
	}

	if result.Cached {
		comment += "*_Results served from cache_*\n"
	}

	return comment
}

// getReviewSkills returns enabled review skills
func (r *DefaultRunner) getReviewSkills(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}

	if r.skillLoader == nil {
		log.Printf("[WARNING] skillLoader is not initialized")
		return []string{"code-reviewer"}
	}

	skills := r.skillLoader.GetSkillNamesForOperation("review")
	if skills == nil {
		return []string{"code-reviewer"}
	}
	return skills
}

// getAnalysisSkills returns enabled analysis skills
func (r *DefaultRunner) getAnalysisSkills(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}

	if r.skillLoader == nil {
		log.Printf("[WARNING] skillLoader is not initialized")
		return []string{"change-analyzer"}
	}

	skills := r.skillLoader.GetSkillNamesForOperation("analyze")
	if skills == nil {
		return []string{"change-analyzer"}
	}
	return skills
}

// detectTestLanguage detects test language from target files
func detectTestLanguage(files []string) string {
	if len(files) == 0 {
		return "go"
	}

	// Simple detection based on first file extension
	for _, f := range files {
		if len(f) > 4 {
			switch f[len(f)-3:] {
			case ".go":
				return "go"
			case ".py":
				return "python"
			case ".js":
				return "javascript"
			case ".ts":
				return "typescript"
			case "ava":
				return "java"
			}
		}
	}
	return "go"
}

// estimateTestCount estimates test count from output
func estimateTestCount(output string) int {
	// Simple heuristic: count "test" occurrences
	count := 0
	for i := 0; i < len(output)-4; i++ {
		if output[i:i+4] == "test" {
			count++
		}
	}
	if count == 0 && len(output) > 100 {
		return 3 // Default estimate
	}
	return min(count/2, 20) // Rough estimate, max 20
}

// severityIcon returns an emoji for a severity level
func severityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

// RunParallel runs multiple skills in parallel
func (r *DefaultRunner) RunParallel(ctx context.Context, tasks []func(context.Context) error) error {
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, task := range tasks {
		wg.Add(1)
		go func(t func(context.Context) error) {
			defer wg.Done()
			if err := t(cancelCtx); err != nil {
				select {
				case errChan <- err:
					cancel()
				case <-cancelCtx.Done():
					return
				}
			}
		}(task)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}
