// Package runner provides runner implementation tests
package runner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/ai"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/platform"
)

// MockPlatform for testing
type mockPlatform struct{}

func (m *mockPlatform) Name() string { return "mock" }
func (m *mockPlatform) PostComment(ctx context.Context, opts platform.CommentOptions) error {
	return nil
}
func (m *mockPlatform) GetDiff(ctx context.Context, prID int) (string, error) {
	return "diff content", nil
}
func (m *mockPlatform) GetFile(ctx context.Context, path, ref string) (string, error) {
	return "file content", nil
}
func (m *mockPlatform) GetPRInfo(ctx context.Context, prID int) (*platform.PRInfo, error) {
	return &platform.PRInfo{Number: prID}, nil
}
func (m *mockPlatform) Health(ctx context.Context) error { return nil }

func TestNewRunner(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{
			CacheDir:    ".cache",
			EnableCache: false,
			DiffContext: 3,
		},
		Claude: config.ClaudeConfig{
			Model:        "sonnet",
			MaxTurns:     50,
			Timeout:      "30m",
			MaxBudgetUSD: 10.0,
		},
		Skills: []config.SkillConfig{
			{Name: "test-skill", Path: "./skills/test", Enabled: true},
		},
	}

	runner, err := NewRunner(cfg, &mockPlatform{}, ".")
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if runner == nil {
		t.Error("NewRunner() returned nil runner")
	}
}

func TestSummarizeIssues(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	issues := []ai.Issue{
		{Severity: "critical", Category: "security", File: "auth.go", Line: 10},
		{Severity: "high", Category: "performance", File: "cache.go", Line: 25},
		{Severity: "medium", Category: "style", File: "utils.go", Line: 5},
		{Severity: "low", Category: "style", File: "utils.go", Line: 6},
		{Severity: "low", Category: "style", File: "main.go", Line: 1},
	}

	summary := runner.summarizeIssues(issues)

	if summary.TotalIssues != 5 {
		t.Errorf("TotalIssues = %d, want 5", summary.TotalIssues)
	}

	if summary.Critical != 1 {
		t.Errorf("Critical = %d, want 1", summary.Critical)
	}

	if summary.High != 1 {
		t.Errorf("High = %d, want 1", summary.High)
	}

	if summary.Medium != 1 {
		t.Errorf("Medium = %d, want 1", summary.Medium)
	}

	if summary.Low != 2 {
		t.Errorf("Low = %d, want 2", summary.Low)
	}

	if summary.FilesChanged != 4 {
		t.Errorf("FilesChanged = %d, want 4", summary.FilesChanged)
	}
}

func TestFormatReviewComment(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	result := &ReviewResult{
		Summary: ReviewSummary{
			FilesChanged: 2,
			TotalIssues:  3,
			Critical:     1,
			High:         1,
			Medium:       1,
			Low:          0,
		},
		Issues: []ai.Issue{
			{
				Severity: "critical",
				Category: "security",
				File:     "auth.go",
				Line:     10,
				Message:  "SQL injection vulnerability",
			},
		},
	}

	comment := runner.formatReviewComment(result)

	if comment == "" {
		t.Error("formatReviewComment() returned empty string")
	}

	checks := []struct {
		substring string
		negative  bool
	}{
		{"Code Review Results", false},
		{"**Files Changed**: 2", false},
		{"**Total Issues**: 3", false},
		{"SQL injection vulnerability", false},
	}

	for _, check := range checks {
		contains := strings.Contains(comment, check.substring)
		if check.negative {
			if contains {
				t.Errorf("Comment should not contain: %s", check.substring)
			}
		} else {
			if !contains {
				t.Errorf("Comment should contain: %s", check.substring)
			}
		}
	}
}

func TestSeverityIcon(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "🔴"},
		{"high", "🟠"},
		{"medium", "🟡"},
		{"low", "🟢"},
		{"unknown", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			if got := severityIcon(tt.severity); got != tt.expected {
				t.Errorf("severityIcon(%s) = %s, want %s", tt.severity, got, tt.expected)
			}
		})
	}
}

func TestDetectTestLanguage(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{"go files", []string{"main.go", "utils.go"}, "go"},
		{"python files", []string{"app.py", "utils.py"}, "python"},
		{"js files", []string{"app.js"}, "javascript"},
		{"ts files", []string{"app.ts"}, "typescript"},
		{"java files", []string{"App.java"}, "java"},
		{"empty", []string{}, "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectTestLanguage(tt.files); got != tt.expected {
				t.Errorf("detectTestLanguage() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestEstimateTestCount(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{"empty", "", 0},
		{"short", "test", 0}, // Too short for estimation
		{"multiple", "test test test test test", 2},
		{"long output", strings.Repeat("test ", 100), 20},
		{"long enough", strings.Repeat("some content ", 10), 3}, // > 100 chars defaults to 3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := estimateTestCount(tt.output); got != tt.expected {
				t.Errorf("estimateTestCount() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestCache(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewCache(tmpDir, true)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	review := CachedReview{
		Summary: ReviewSummary{TotalIssues: 5},
		Issues: []ai.Issue{
			{Severity: "critical", Message: "Test issue"},
		},
		Comment: "Test comment",
	}

	cache.SetReview(123, review)

	got, ok := cache.GetReview(123)
	if !ok {
		t.Fatal("GetReview() returned ok=false")
	}

	if got.Summary.TotalIssues != 5 {
		t.Errorf("TotalIssues = %d, want 5", got.Summary.TotalIssues)
	}

	cache.Invalidate(123)

	_, ok = cache.GetReview(123)
	if ok {
		t.Error("GetReview() should return ok=false after Invalidate")
	}
}

func TestCacheDisabled(t *testing.T) {
	cache, err := NewCache("", false)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	review := CachedReview{Comment: "test"}
	cache.SetReview(123, review)

	_, ok := cache.GetReview(123)
	if ok {
		t.Error("GetReview() should return ok=false when cache disabled")
	}
}

func TestCacheTTL(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewCache(tmpDir, true)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	cache.SetTTL(1 * time.Nanosecond)

	review := CachedReview{Comment: "test"}
	cache.SetReview(123, review)

	time.Sleep(10 * time.Millisecond)

	_, ok := cache.GetReview(123)
	if ok {
		t.Error("GetReview() should return ok=false after TTL expires")
	}
}

func TestGetDiffHash(t *testing.T) {
	diff1 := "some diff content"
	diff2 := "some diff content"
	diff3 := "different content"

	hash1 := GetDiffHash(diff1)
	hash2 := GetDiffHash(diff2)
	hash3 := GetDiffHash(diff3)

	if hash1 != hash2 {
		t.Error("Same diff should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("Different diffs should produce different hashes")
	}
}
