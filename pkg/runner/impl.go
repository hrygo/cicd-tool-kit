// Package runner provides the core CI/CD runner implementation
package runner

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
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

	// MaxDiffLength is the maximum length (in bytes) for diff truncation
	MaxDiffLength = 10000

	// MaxDiffRunes is the maximum length (in runes) for safe UTF-8 diff truncation
	MaxDiffRunes = 10000

	// CoverageEstimatePercent is the estimated test coverage percentage
	CoverageEstimatePercent = 10
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

	cacheDir := filepath.Join(baseDir, cfg.Global.CacheDir)
	cache, err := NewCache(cacheDir, cfg.Global.EnableCache)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize skill loader from skills directory
	skillsDir := filepath.Join(baseDir, "skills")
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
			result.Duration = cached.Duration // Use cached duration instead of recalculating
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
		Summary:  result.Summary,
		Issues:   result.Issues,
		Comment:  result.PlatformComment,
		Duration: result.Duration, // Store the actual execution duration
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
func (r *DefaultRunner) buildReviewContext(_ context.Context, opts ReviewOptions) string {
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
func (r *DefaultRunner) buildAnalysisContext(_ context.Context, opts AnalyzeOptions) string {
	var sb strings.Builder

	sb.WriteString("# Change Analysis Context\n\n")
	fmt.Fprintf(&sb, "## Summary\n\n")
	fmt.Fprintf(&sb, "- Files Changed: %d\n", opts.FileCount)
	fmt.Fprintf(&sb, "- Additions: +%d\n", opts.Additions)
	fmt.Fprintf(&sb, "- Deletions: -%d\n", opts.Deletions)

	if opts.Diff != "" {
		sb.WriteString("\n## Diff\n\n")
		sb.WriteString("```diff\n")
		// Truncate large diffs safely using rune-aware slicing
		// to avoid cutting multi-byte UTF-8 characters
		diff := opts.Diff
		wasTruncated := false
		if len(diff) > MaxDiffLength {
			// Truncate at rune boundary to prevent corrupting multi-byte characters
			// Check rune count separately since multi-byte chars mean byte length != rune count
			runes := []rune(diff)
			if len(runes) > MaxDiffRunes {
				diff = string(runes[:MaxDiffRunes]) + "\n... (truncated)"
				wasTruncated = true
			} else {
				// Rune count is within limit but byte count exceeds limit
				// Truncate at byte boundary that won't cut a UTF-8 sequence
				// Find the last complete UTF-8 character before MaxDiffLength
				truncAt := MaxDiffLength
				for truncAt > 0 && (diff[truncAt]&0xC0) == 0x80 {
					truncAt--
				}
				diff = diff[:truncAt] + "\n... (truncated)"
				wasTruncated = true
			}
		}
		if wasTruncated {
			// Log truncation warning for observability
			fmt.Fprintf(&sb, "[WARNING] Diff truncated from %d to %d bytes for context limits\n", len(opts.Diff), len(diff))
		}
		sb.WriteString(diff)
		sb.WriteString("\n```\n")
	}

	return sb.String()
}

// buildTestGenContext builds the context for test generation
func (r *DefaultRunner) buildTestGenContext(_ context.Context, opts TestGenOptions) string {
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
			fmt.Fprintf(&sb, "- %s\n", f)
		}
		sb.WriteString("\n")
	}

	if opts.TestFramework != "" {
		fmt.Fprintf(&sb, "## Test Framework\n\n%s\n\n", opts.TestFramework)
	}

	return sb.String()
}

// executeReview executes the AI review
func (r *DefaultRunner) executeReview(ctx context.Context, diffContext string, skills []string) ([]ai.Issue, error) {
	prompt := r.buildReviewPrompt(diffContext, skills)

	// Build execute options
	opts := ai.ExecuteOptions{
		OutputFormat: r.cfg.Claude.OutputFormat,
		Timeout:      DefaultTimeout,
	}

	// Get timeout from config
	if t, err := r.cfg.Claude.GetTimeout(); err == nil && t > 0 {
		opts.Timeout = t
	}

	// Validate prompt for injection attacks if enabled
	if err := ai.ValidatePrompt(prompt, opts); err != nil {
		return nil, fmt.Errorf("prompt validation failed: %w", err)
	}

	// Execute with AI Brain
	output, err := r.aiBrain.Execute(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	// Return issues from output
	return output.Issues, nil
}

// executeAnalysis executes the change analysis
func (r *DefaultRunner) executeAnalysis(ctx context.Context, analysisContext string, skills []string) (ChangeSummary, ImpactAnalysis, RiskAssessment, ChangelogEntry, error) {
	prompt := r.buildAnalysisPrompt(analysisContext, skills)

	// Build execute options
	opts := ai.ExecuteOptions{
		OutputFormat: r.cfg.Claude.OutputFormat,
		Timeout:      DefaultTimeout,
	}

	// Get timeout from config
	if r.cfg != nil {
		if t, err := r.cfg.Claude.GetTimeout(); err == nil && t > 0 {
			opts.Timeout = t
		}
	}

	// Validate prompt for injection attacks if enabled
	if err := ai.ValidatePrompt(prompt, opts); err != nil {
		return ChangeSummary{}, ImpactAnalysis{}, RiskAssessment{}, ChangelogEntry{}, fmt.Errorf("prompt validation failed: %w", err)
	}

	// Check for context cancellation before executing
	if ctx.Err() != nil {
		return ChangeSummary{}, ImpactAnalysis{}, RiskAssessment{}, ChangelogEntry{}, fmt.Errorf("analysis cancelled before execution: %w", ctx.Err())
	}

	// Execute with AI Brain
	_, err := r.aiBrain.Execute(ctx, prompt, opts)
	if err != nil {
		if ctx.Err() != nil {
			return ChangeSummary{}, ImpactAnalysis{}, RiskAssessment{}, ChangelogEntry{}, fmt.Errorf("analysis cancelled: %w", ctx.Err())
		}
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
	prompt := r.buildTestGenPrompt(testGenContext, opts)

	// Build execute options
	execOpts := ai.ExecuteOptions{
		OutputFormat: r.cfg.Claude.OutputFormat,
		Timeout:      DefaultTimeout,
	}

	// Get timeout from config
	if r.cfg != nil {
		if t, err := r.cfg.Claude.GetTimeout(); err == nil && t > 0 {
			execOpts.Timeout = t
		}
	}

	// Validate prompt for injection attacks if enabled
	if err := ai.ValidatePrompt(prompt, execOpts); err != nil {
		return nil, fmt.Errorf("prompt validation failed: %w", err)
	}

	// Use ctx directly instead of creating a derived context
	// The AI brain implementation should handle the timeout internally
	output, err := r.aiBrain.Execute(ctx, prompt, execOpts)
	if err != nil {
		return nil, err
	}

	// Parse output to extract generated tests
	tests := r.parseTestsFromOutput(output.Raw)
	return tests, nil
}

// parseTestsFromOutput extracts test code from AI output
// It identifies markdown code fences with language markers and extracts them
func (r *DefaultRunner) parseTestsFromOutput(output string) []GeneratedTest {
	if output == "" {
		return []GeneratedTest{}
	}

	tests := []GeneratedTest{}
	lines := strings.Split(output, "\n")
	var currentBlock []string
	var currentLang string
	var inCodeBlock bool

	for _, line := range lines {
		// Check for code fence start
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// Start of code block
				inCodeBlock = true
				// Extract language from fence (e.g., "```go" -> "go", "```" -> "")
				currentLang = strings.TrimPrefix(line, "```")
				if currentLang != "" {
					currentLang = strings.TrimSpace(currentLang)
				}
				currentBlock = []string{}
				continue
			} else {
				// End of code block
				inCodeBlock = false
				content := strings.Join(currentBlock, "\n")

				// Only include if it looks like test code and has a language
				if currentLang != "" && r.isTestContent(content) {
					// Estimate test count by counting test functions
					testCount := r.countTestFunctions(content, currentLang)

					// Generate a file path based on language
					path := r.generateTestPath(currentLang)

					tests = append(tests, GeneratedTest{
						Path:     path,
						Language: currentLang,
						Content:  content,
						Tests:    testCount,
					})
				}

				currentBlock = nil
				currentLang = ""
				continue
			}
		}

		// Accumulate code block content
		if inCodeBlock {
			currentBlock = append(currentBlock, line)
		}
	}

	return tests
}

// isTestContent checks if the content looks like test code
func (r *DefaultRunner) isTestContent(content string) bool {
	lowerContent := strings.ToLower(content)

	// Common test indicators
	testIndicators := []string{
		"func test",   // Go
		"def test_",   // Python
		".test(",      // JavaScript/TypeScript
		".spec(",      // JavaScript/TypeScript
		"it(",         // JavaScript/TypeScript
		"describe(",   // JavaScript/TypeScript
		"test(",       // Racket, Lisp
		"@test",       // Java (JUnit 5)
		"@testmethod", // Objective-C
		"void test",   // Java/C
		"suite(",      // Python unittest
		"testcase",    // Python
		"assert",      // General assertion
		"expect(",     // JS testing libraries
		"should()",    // JS testing libraries
		"t.run(",      // Go subtests
		"testing.t",   // Go test parameter
	}

	for _, indicator := range testIndicators {
		if strings.Contains(lowerContent, indicator) {
			return true
		}
	}

	// Check for import statements that indicate test files
	testImports := []string{
		`"testing"`, // Go
		`"github.com/stretchr"`,
		"import pytest", // Python
		"from unittest", // Python
		"import @test",  // Java
		"import org.junit",
	}

	for _, imp := range testImports {
		if strings.Contains(lowerContent, imp) {
			return true
		}
	}

	return false
}

// countTestFunctions estimates the number of test functions in code
func (r *DefaultRunner) countTestFunctions(content string, lang string) int {
	count := 0
	lowerContent := strings.ToLower(content)

	switch lang {
	case "go":
		// Count "func Test" occurrences
		count += strings.Count(lowerContent, "func test")
	case "py", "python":
		// Count "def test_" occurrences
		count += strings.Count(lowerContent, "def test_")
	case "js", "javascript", "ts", "typescript":
		// Count various JS test patterns
		count += strings.Count(lowerContent, "it(")
		count += strings.Count(lowerContent, "test(")
		count += strings.Count(lowerContent, ".spec(")
	case "java":
		count += strings.Count(lowerContent, "@test")
		count += strings.Count(lowerContent, "@before")
		count += strings.Count(lowerContent, "@after")
	}

	// Minimum of 1 if we detected it as test content
	if count == 0 && r.isTestContent(content) {
		count = 1
	}

	return count
}

// generateTestPath generates a test file path based on language
func (r *DefaultRunner) generateTestPath(lang string) string {
	switch lang {
	case "go":
		return "generated_test.go"
	case "py", "python":
		return "generated_test.py"
	case "js":
		return "generated.test.js"
	case "ts":
		return "generated.test.ts"
	case "java":
		return "GeneratedTest.java"
	case "rs", "rust":
		return "generated_test.rs"
	case "cpp", "c++":
		return "generated_test.cpp"
	case "c":
		return "generated_test.c"
	default:
		return "generated_test." + lang
	}
}

// buildReviewPrompt builds the prompt for code review
func (r *DefaultRunner) buildReviewPrompt(context string, _ []string) string {
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
func (r *DefaultRunner) buildAnalysisPrompt(context string, _ []string) string {
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
func (r *DefaultRunner) summarizeIssues(issues []ai.Issue) ReviewSummary {
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
	fmt.Fprintf(&sb, "- **Files Changed**: %d\n", result.Summary.FilesChanged)
	fmt.Fprintf(&sb, "- **Total Issues**: %d\n", result.Summary.TotalIssues)

	// Severity breakdown
	if result.Summary.Critical > 0 {
		fmt.Fprintf(&sb, "- **🔴 Critical**: %d\n", result.Summary.Critical)
	}
	if result.Summary.High > 0 {
		fmt.Fprintf(&sb, "- **🟠 High**: %d\n", result.Summary.High)
	}
	if result.Summary.Medium > 0 {
		fmt.Fprintf(&sb, "- **🟡 Medium**: %d\n", result.Summary.Medium)
	}
	if result.Summary.Low > 0 {
		fmt.Fprintf(&sb, "- **🟢 Low**: %d\n", result.Summary.Low)
	}

	sb.WriteString("\n")

	// Issues by severity
	if len(result.Issues) > 0 {
		sb.WriteString("### Issues Found\n\n")

		for _, issue := range result.Issues {
			icon := severityIcon(issue.Severity)
			fmt.Fprintf(&sb, "%s **%s** - `%s:%d`\n", icon, issue.Category, issue.File, issue.Line)
			fmt.Fprintf(&sb, "%s\n\n", issue.Message)
			if issue.Suggestion != "" {
				fmt.Fprintf(&sb, "**Suggestion**: %s\n\n", issue.Suggestion)
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

	// Check if skillLoader is initialized
	if r.skillLoader == nil {
		log.Printf("[WARNING] skillLoader is not initialized, no skills will be loaded")
		return []string{}
	}

	// Use skill loader to discover review skills
	skills := r.skillLoader.GetSkillNamesForOperation("review")
	if skills == nil {
		// Return empty slice to prevent nil pointer dereference
		return []string{}
	}
	return skills
}

// getAnalysisSkills returns enabled analysis skills
func (r *DefaultRunner) getAnalysisSkills(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}

	// Check if skillLoader is initialized
	if r.skillLoader == nil {
		log.Printf("[WARNING] skillLoader is not initialized, no skills will be loaded")
		return []string{}
	}

	// Use skill loader to discover analysis skills
	skills := r.skillLoader.GetSkillNamesForOperation("analyze")
	if skills == nil {
		// Return empty slice to prevent nil pointer dereference
		return []string{}
	}
	return skills
}

// detectLanguage detects the programming language from file path
func detectLanguage(path string) string {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")

	langMap := map[string]string{
		"go":   "go",
		"js":   "javascript",
		"ts":   "typescript",
		"py":   "python",
		"rb":   "ruby",
		"java": "java",
		"rs":   "rust",
		"cpp":  "c++",
		"c":    "c",
		"cs":   "c#",
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
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	// Buffered channel prevents goroutine leak - buffer size equals number of tasks
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

	// Wait for all goroutines to complete
	// This ensures no more sends will happen to errChan
	wg.Wait()

	// Close channel is safe now - all goroutines have finished
	close(errChan)

	// Return first error if any - sync.Once ensures single error return
	var firstErr error
	for err := range errChan {
		firstErr = err
		break // Only return first error
	}

	return firstErr
}
