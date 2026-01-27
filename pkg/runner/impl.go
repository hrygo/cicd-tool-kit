// Package runner provides the core CI/CD runner implementation
package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/buildcontext"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/claude"
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
	cfg       *config.Config
	platform  platform.Platform
	builder   *buildcontext.Builder
	parser    claude.OutputParser
	cache     *Cache
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

	cacheDir := filepath.Join(baseDir, cfg.Global.CacheDir)
	cache, err := NewCache(cacheDir, cfg.Global.EnableCache)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize skill loader from skills directory
	skillsDir := filepath.Join(baseDir, "skills")
	skillLoader := skill.NewLoader(skillsDir)

	return &DefaultRunner{
		cfg:         cfg,
		platform:    platform,
		builder:     builder,
		parser:      claude.NewParser(),
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
			result.Duration = time.Since(start)
			return result, nil
		}
	}

	// Get enabled review skills
	skills := r.getReviewSkills(opts.Skills)

	// Build context
	diffContext := r.buildReviewContext(ctx, opts)

	// Execute Claude review
	issues, err := r.executeReview(ctx, diffContext, skills)
	if err != nil {
		return nil, fmt.Errorf("review execution failed: %w", err)
	}

	result.Issues = issues
	result.Summary = r.summarizeIssues(issues)
	result.PlatformComment = r.formatReviewComment(result)
	result.Duration = time.Since(start)

	// Cache the result
	r.cache.SetReview(opts.PRID, CachedReview{
		Summary: result.Summary,
		Issues:  result.Issues,
		Comment: result.PlatformComment,
	})

	return result, nil
}

// Analyze runs change analysis on a pull/merge request
func (r *DefaultRunner) Analyze(ctx context.Context, opts AnalyzeOptions) (*AnalyzeResult, error) {
	start := time.Now()

	skills := r.getAnalysisSkills(opts.Skills)

	// Build analysis context
	context := r.buildAnalysisContext(ctx, opts)

	// Execute analysis
	summary, impact, risk, changelog, err := r.executeAnalysis(ctx, context, skills)
	if err != nil {
		return nil, fmt.Errorf("analysis execution failed: %w", err)
	}

	result := &AnalyzeResult{
		Summary:   summary,
		Impact:    impact,
		Risk:      risk,
		Changelog: changelog,
		Duration:  time.Since(start),
	}

	return result, nil
}

// GenerateTests generates tests based on code changes
func (r *DefaultRunner) GenerateTests(ctx context.Context, opts TestGenOptions) (*TestGenResult, error) {
	start := time.Now()

	// Build test generation context
	context := r.buildTestGenContext(ctx, opts)

	// Execute test generation
	tests, err := r.executeTestGen(ctx, context, opts)
	if err != nil {
		return nil, fmt.Errorf("test generation failed: %w", err)
	}

	result := &TestGenResult{
		TestFiles: tests,
		Summary:   r.summarizeTests(tests),
		Duration:  time.Since(start),
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

// buildReviewContext builds the context for code review
func (r *DefaultRunner) buildReviewContext(ctx context.Context, opts ReviewOptions) string {
	var sb strings.Builder

	sb.WriteString("# Code Review Context\n\n")

	// Add diff
	if opts.Diff != "" {
		sb.WriteString("## Changes to Review\n\n")
		sb.WriteString("```diff\n")
		sb.WriteString(opts.Diff)
		sb.WriteString("\n```\n\n")
	}

	// Add PR info if available
	if opts.PRID > 0 {
		sb.WriteString(fmt.Sprintf("## Pull Request #%d\n\n", opts.PRID))
	}

	return sb.String()
}

// buildAnalysisContext builds the context for change analysis
func (r *DefaultRunner) buildAnalysisContext(ctx context.Context, opts AnalyzeOptions) string {
	var sb strings.Builder

	sb.WriteString("# Change Analysis Context\n\n")
	sb.WriteString(fmt.Sprintf("## Summary\n\n"))
	sb.WriteString(fmt.Sprintf("- Files Changed: %d\n", opts.FileCount))
	sb.WriteString(fmt.Sprintf("- Additions: +%d\n", opts.Additions))
	sb.WriteString(fmt.Sprintf("- Deletions: -%d\n", opts.Deletions))

	if opts.Diff != "" {
		sb.WriteString("\n## Diff\n\n")
		sb.WriteString("```diff\n")
		// Truncate large diffs
		diff := opts.Diff
		if len(diff) > 10000 {
			diff = diff[:10000] + "\n... (truncated)"
		}
		sb.WriteString(diff)
		sb.WriteString("\n```\n")
	}

	return sb.String()
}

// buildTestGenContext builds the context for test generation
func (r *DefaultRunner) buildTestGenContext(ctx context.Context, opts TestGenOptions) string {
	var sb strings.Builder

	sb.WriteString("# Test Generation Context\n\n")

	if opts.Diff != "" {
		sb.WriteString("## Code Changes\n\n")
		sb.WriteString("```diff\n")
		sb.WriteString(opts.Diff)
		sb.WriteString("\n```\n\n")
	}

	if len(opts.TargetFiles) > 0 {
		sb.WriteString("## Target Files\n\n")
		for _, f := range opts.TargetFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if opts.TestFramework != "" {
		sb.WriteString(fmt.Sprintf("## Test Framework\n\n%s\n\n", opts.TestFramework))
	}

	return sb.String()
}

// executeReview executes the Claude review
func (r *DefaultRunner) executeReview(ctx context.Context, diffContext string, skills []string) ([]claude.Issue, error) {
	session, err := claude.NewSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	prompt := r.buildReviewPrompt(diffContext, skills)

	timeout, err := r.cfg.Claude.GetTimeout()
	if err != nil || timeout == 0 {
		timeout = DefaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := session.Execute(execCtx, claude.ExecuteOptions{
		Prompt:          prompt,
		StdinContent:    "",
		Model:           r.cfg.Claude.Model,
		MaxTurns:        r.cfg.Claude.MaxTurns,
		MaxBudgetUSD:    r.cfg.Claude.MaxBudgetUSD,
		OutputFormat:    r.cfg.Claude.OutputFormat,
		SkipPermissions: r.cfg.Claude.SkipPermissions,
	})

	if err != nil {
		return nil, err
	}

	// Parse issues from output
	return r.parser.ExtractIssues(output.Raw)
}

// executeAnalysis executes the change analysis
func (r *DefaultRunner) executeAnalysis(ctx context.Context, analysisContext string, skills []string) (ChangeSummary, ImpactAnalysis, RiskAssessment, ChangelogEntry, error) {
	session, err := claude.NewSession(ctx)
	if err != nil {
		return ChangeSummary{}, ImpactAnalysis{}, RiskAssessment{}, ChangelogEntry{}, err
	}
	defer session.Close()

	prompt := r.buildAnalysisPrompt(analysisContext, skills)

	timeout, err := r.cfg.Claude.GetTimeout()
	if err != nil || timeout == 0 {
		timeout = DefaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err = session.Execute(execCtx, claude.ExecuteOptions{
		Prompt:          prompt,
		Model:           r.cfg.Claude.Model,
		MaxTurns:        r.cfg.Claude.MaxTurns,
		SkipPermissions: r.cfg.Claude.SkipPermissions,
	})

	if err != nil {
		return ChangeSummary{}, ImpactAnalysis{}, RiskAssessment{}, ChangelogEntry{}, err
	}

	// Parse the output - for now return defaults
	// A full implementation would parse JSON output
	return ChangeSummary{
		Title:        "Analysis Complete",
		FilesChanged: 0,
	}, ImpactAnalysis{}, RiskAssessment{
		Score: 5,
	}, ChangelogEntry{}, nil
}

// executeTestGen executes test generation
func (r *DefaultRunner) executeTestGen(ctx context.Context, testGenContext string, opts TestGenOptions) ([]GeneratedTest, error) {
	session, err := claude.NewSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	prompt := r.buildTestGenPrompt(testGenContext, opts)

	timeout, _ := r.cfg.Claude.GetTimeout()
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := session.Execute(execCtx, claude.ExecuteOptions{
		Prompt:          prompt,
		Model:           r.cfg.Claude.Model,
		SkipPermissions: r.cfg.Claude.SkipPermissions,
	})

	if err != nil {
		return nil, err
	}

	// Extract code changes as test files
	changes := r.parser.ExtractCodeChanges(output.Raw)

	var tests []GeneratedTest
	for _, change := range changes {
		tests = append(tests, GeneratedTest{
			Path:     change.File,
			Content:  change.Content,
			Language: detectLanguage(change.File),
		})
	}

	return tests, nil
}

// buildReviewPrompt builds the prompt for code review
func (r *DefaultRunner) buildReviewPrompt(context string, skills []string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert code reviewer. Analyze the provided code changes.\n\n")
	sb.WriteString(context)

	sb.WriteString("\n## Instructions\n\n")
	sb.WriteString("Provide a thorough code review focusing on:\n")
	sb.WriteString("1. Security vulnerabilities\n")
	sb.WriteString("2. Performance issues\n")
	sb.WriteString("3. Logic errors\n")
	sb.WriteString("4. Code quality and maintainability\n")
	sb.WriteString("5. Architectural concerns\n\n")

	sb.WriteString("## Output Format\n\n")
	sb.WriteString("Respond with a JSON object in the following format:\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"issues": [{"severity": "critical|high|medium|low", "category": "security|performance|logic|style", "file": "path/to/file", "line": 123, "message": "description", "suggestion": "fix suggestion"}]}`)
	sb.WriteString("\n```\n")

	return sb.String()
}

// buildAnalysisPrompt builds the prompt for change analysis
func (r *DefaultRunner) buildAnalysisPrompt(context string, skills []string) string {
	return fmt.Sprintf("Analyze the following changes and provide impact assessment, risk analysis, and changelog entry.\n\n%s\n\nProvide a summary with risk score (1-10), impact analysis, and structured changelog.", context)
}

// buildTestGenPrompt builds the prompt for test generation
func (r *DefaultRunner) buildTestGenPrompt(context string, opts TestGenOptions) string {
	prompt := fmt.Sprintf("Generate comprehensive tests for the following code changes.\n\n%s\n\n", context)

	if opts.TestFramework != "" {
		prompt += fmt.Sprintf("Use %s as the test framework.\n\n", opts.TestFramework)
	}

	prompt += "Provide complete test files with proper setup, teardown, and assertions."

	return prompt
}

// summarizeIssues aggregates issues into a summary
func (r *DefaultRunner) summarizeIssues(issues []claude.Issue) ReviewSummary {
	summary := ReviewSummary{}

	// Count files
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

// summarizeTests aggregates test generation results
func (r *DefaultRunner) summarizeTests(tests []GeneratedTest) TestGenSummary {
	summary := TestGenSummary{
		FilesCreated: len(tests),
	}

	for _, test := range tests {
		summary.TotalTests += test.Tests
	}

	// Estimate coverage
	if summary.TotalTests > 0 {
		summary.CoverageEst = "~70-80%"
	}

	return summary
}

// formatReviewComment formats the review result as a markdown comment
func (r *DefaultRunner) formatReviewComment(result *ReviewResult) string {
	var sb strings.Builder

	sb.WriteString("## 🔍 Code Review Results\n\n")

	// Summary section
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Files Changed**: %d\n", result.Summary.FilesChanged))
	sb.WriteString(fmt.Sprintf("- **Total Issues**: %d\n", result.Summary.TotalIssues))

	// Severity breakdown
	if result.Summary.Critical > 0 {
		sb.WriteString(fmt.Sprintf("- **🔴 Critical**: %d\n", result.Summary.Critical))
	}
	if result.Summary.High > 0 {
		sb.WriteString(fmt.Sprintf("- **🟠 High**: %d\n", result.Summary.High))
	}
	if result.Summary.Medium > 0 {
		sb.WriteString(fmt.Sprintf("- **🟡 Medium**: %d\n", result.Summary.Medium))
	}
	if result.Summary.Low > 0 {
		sb.WriteString(fmt.Sprintf("- **🟢 Low**: %d\n", result.Summary.Low))
	}

	sb.WriteString("\n")

	// Issues by severity
	if len(result.Issues) > 0 {
		sb.WriteString("### Issues Found\n\n")

		for _, issue := range result.Issues {
			icon := severityIcon(issue.Severity)
			sb.WriteString(fmt.Sprintf("%s **%s** - `%s:%d`\n", icon, issue.Category, issue.File, issue.Line))
			sb.WriteString(fmt.Sprintf("%s\n\n", issue.Message))
			if issue.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("**Suggestion**: %s\n\n", issue.Suggestion))
			}
		}
	} else {
		sb.WriteString("### ✅ No Issues Found\n\n")
		sb.WriteString("Great job! No issues were detected in this review.\n\n")
	}

	if result.Cached {
		sb.WriteString("*_Results served from cache_*\n")
	}

	return sb.String()
}

// getReviewSkills returns enabled review skills
func (r *DefaultRunner) getReviewSkills(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}

	// Use skill loader to discover review skills
	return r.skillLoader.GetSkillNamesForOperation("review")
}

// getAnalysisSkills returns enabled analysis skills
func (r *DefaultRunner) getAnalysisSkills(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}

	// Use skill loader to discover analysis skills
	return r.skillLoader.GetSkillNamesForOperation("analyze")
}

// detectLanguage detects the programming language from file path
func detectLanguage(path string) string {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")

	langMap := map[string]string{
		"go":  "go",
		"js":  "javascript",
		"ts":  "typescript",
		"py":  "python",
		"rb":  "ruby",
		"java": "java",
		"rs":  "rust",
		"cpp": "c++",
		"c":   "c",
		"cs":  "c#",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "unknown"
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
// If any task fails, all other tasks are cancelled via context
func (r *DefaultRunner) RunParallel(ctx context.Context, tasks []func(context.Context) error) error {
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
					cancel() // Cancel other tasks on first error
				case <-cancelCtx.Done():
					return // Another task already failed
				}
			}
		}(task)
	}

	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
